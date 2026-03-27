package whatsapp

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	waLog "go.mau.fi/whatsmeow/util/log"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/storage"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"

	// sqlite3 driver for whatsmeow sqlstore
	_ "modernc.org/sqlite"
)

type Manager struct {
	DB              *gorm.DB
	R2              *storage.R2Client
	EmbeddingWorker *wvClient.EmbeddingWorker
	clients         map[uint]*whatsmeow.Client
	mu              sync.RWMutex
	container       *sqlstore.Container
}

func NewManager(ctx context.Context, db *gorm.DB, r2 *storage.R2Client, ew *wvClient.EmbeddingWorker, whatsmeowDBPath string) (*Manager, error) {
	if whatsmeowDBPath == "" {
		whatsmeowDBPath = "data/whatsmeow.db"
	}

	// Ensure parent directory exists
	dir := filepath.Dir(whatsmeowDBPath)
	log.Printf("[wa-manager] ensuring directory exists: %s", dir)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create dir %s: %w", dir, err)
	}
	log.Printf("[wa-manager] directory ready: %s", dir)

	dbURI := fmt.Sprintf("file:%s?_pragma=foreign_keys(1)", whatsmeowDBPath)
	log.Printf("[wa-manager] opening device store at %s", dbURI)

	container, err := sqlstore.New(ctx, "sqlite", dbURI, waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("create sqlstore at %s: %w", dbURI, err)
	}

	return &Manager{
		DB:              db,
		R2:              r2,
		EmbeddingWorker: ew,
		clients:         make(map[uint]*whatsmeow.Client),
		container:       container,
	}, nil
}

func (m *Manager) Start(ctx context.Context) {
	// Load connected numbers and reconnect them
	var numbers []models.WaNumber
	m.DB.Where("status = ?", "connected").Find(&numbers)

	for _, num := range numbers {
		go m.connectNumber(ctx, num.ID)
	}

	log.Printf("[wa-manager] started with %d numbers", len(numbers))
}

func (m *Manager) Shutdown() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, client := range m.clients {
		client.Disconnect()
		log.Printf("[wa-manager] disconnected number %d", id)
	}
	m.clients = make(map[uint]*whatsmeow.Client)
}

func (m *Manager) connectNumber(ctx context.Context, numberID uint) {
	devices, err := m.container.GetAllDevices(ctx)
	if err != nil {
		log.Printf("[wa-manager] get devices error: %v", err)
		return
	}

	var device *store.Device
	for _, d := range devices {
		if d.ID != nil {
			device = d
			break
		}
	}
	if device == nil {
		device = m.container.NewDevice()
	}

	client := whatsmeow.NewClient(device, waLog.Noop)
	handler := NewMessageHandler(m.DB, m.R2, m.EmbeddingWorker, numberID)
	handler.SetClient(client)
	client.AddEventHandler(handler.HandleEvent)

	if client.Store.ID == nil {
		log.Printf("[wa-manager] number %d needs pairing", numberID)
		m.DB.Model(&models.WaNumber{}).Where("id = ?", numberID).Update("status", "disconnected")
		return
	}

	err = client.Connect()
	if err != nil {
		log.Printf("[wa-manager] connect error for number %d: %v", numberID, err)
		m.reconnectWithBackoff(ctx, numberID, client)
		return
	}

	m.mu.Lock()
	m.clients[numberID] = client
	m.mu.Unlock()
}

func (m *Manager) reconnectWithBackoff(ctx context.Context, numberID uint, client *whatsmeow.Client) {
	delays := []time.Duration{5 * time.Second, 10 * time.Second, 30 * time.Second, 60 * time.Second, 60 * time.Second}

	for attempt, delay := range delays {
		select {
		case <-ctx.Done():
			return
		case <-time.After(delay):
		}

		err := client.Connect()
		if err == nil {
			m.mu.Lock()
			m.clients[numberID] = client
			m.mu.Unlock()
			m.DB.Model(&models.WaNumber{}).Where("id = ?", numberID).Update("status", "connected")
			log.Printf("[wa-manager] reconnected number %d after %d attempts", numberID, attempt+1)
			return
		}
	}

	m.DB.Model(&models.WaNumber{}).Where("id = ?", numberID).Update("status", "disconnected")
}

func (m *Manager) GetClient(numberID uint) (*whatsmeow.Client, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	c, ok := m.clients[numberID]
	return c, ok
}

func (m *Manager) GetContainer() *sqlstore.Container {
	return m.container
}

func (m *Manager) RegisterClient(numberID uint, client *whatsmeow.Client) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.clients[numberID] = client
}

func (m *Manager) RemoveClient(numberID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	if client, ok := m.clients[numberID]; ok {
		client.Disconnect()
		delete(m.clients, numberID)
	}
}

// GetGroups returns joined WhatsApp groups for a number.
func (m *Manager) GetGroups(ctx context.Context, numberID uint) ([]GroupInfo, error) {
	client, ok := m.GetClient(numberID)
	if !ok {
		return nil, fmt.Errorf("number %d not connected", numberID)
	}

	groups, err := client.GetJoinedGroups(ctx)
	if err != nil {
		return nil, fmt.Errorf("get groups: %w", err)
	}

	var result []GroupInfo
	for _, g := range groups {
		result = append(result, GroupInfo{
			JID:  g.JID.String(),
			Name: g.Name,
			Topic: g.Topic,
			ParticipantCount: len(g.Participants),
		})
	}
	return result, nil
}

// GetContacts returns cached contacts for a number.
func (m *Manager) GetContacts(ctx context.Context, numberID uint) ([]ContactInfo, error) {
	client, ok := m.GetClient(numberID)
	if !ok {
		return nil, fmt.Errorf("number %d not connected", numberID)
	}

	store := client.Store
	contacts, err := store.Contacts.GetAllContacts(ctx)
	if err != nil {
		return nil, fmt.Errorf("get contacts: %w", err)
	}

	var result []ContactInfo
	for jid, contact := range contacts {
		name := contact.PushName
		if contact.FullName != "" {
			name = contact.FullName
		}
		if name == "" {
			name = contact.BusinessName
		}
		result = append(result, ContactInfo{
			JID:      jid.String(),
			Name:     name,
			PushName: contact.PushName,
		})
	}
	return result, nil
}

type GroupInfo struct {
	JID              string `json:"jid"`
	Name             string `json:"name"`
	Topic            string `json:"topic,omitempty"`
	ParticipantCount int    `json:"participant_count"`
}

type ContactInfo struct {
	JID      string `json:"jid"`
	Name     string `json:"name"`
	PushName string `json:"push_name,omitempty"`
}

func (m *Manager) SendMessage(ctx context.Context, numberID uint, jid string, text string) error {
	client, ok := m.GetClient(numberID)
	if !ok {
		return fmt.Errorf("number %d not connected", numberID)
	}

	targetJID, err := parseJID(jid)
	if err != nil {
		return err
	}

	_, err = client.SendMessage(ctx, targetJID, &waE2E.Message{
		Conversation: &text,
	})
	return err
}
