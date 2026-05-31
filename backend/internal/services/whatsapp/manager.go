package whatsapp

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	waLog "go.mau.fi/whatsmeow/util/log"
	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/ai/mistral"
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
	VisionClient    *mistral.VisionClient
	Hub             *Hub
	clients         map[uint]*whatsmeow.Client
	mu              sync.RWMutex
	container       *sqlstore.Container
}

func NewManager(ctx context.Context, db *gorm.DB, r2 *storage.R2Client, ew *wvClient.EmbeddingWorker, whatsmeowDBPath string, mistralAPIKey string) (*Manager, error) {
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

	var visionClient *mistral.VisionClient
	if mistralAPIKey != "" {
		visionClient = mistral.NewVisionClient(mistralAPIKey)
		log.Printf("[wa-manager] Mistral vision client configured")
	}

	return &Manager{
		DB:              db,
		R2:              r2,
		EmbeddingWorker: ew,
		VisionClient:    visionClient,
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
	// Load device JID from our DB
	var waNumber models.WaNumber
	if err := m.DB.First(&waNumber, numberID).Error; err != nil {
		log.Printf("[wa-manager] number %d not found in DB", numberID)
		return
	}

	var device *store.Device

	if waNumber.DeviceJID != "" {
		// Try to find the specific device by JID
		deviceJID, err := types.ParseJID(waNumber.DeviceJID)
		if err == nil {
			device, err = m.container.GetDevice(ctx, deviceJID)
			if err != nil {
				log.Printf("[wa-manager] get device by JID %s failed: %v", waNumber.DeviceJID, err)
			}
		}
	}

	if device == nil {
		// Fallback: try first available device
		devices, err := m.container.GetAllDevices(ctx)
		if err != nil {
			log.Printf("[wa-manager] get devices error: %v", err)
			return
		}
		for _, d := range devices {
			if d.ID != nil {
				device = d
				break
			}
		}
	}

	if device == nil {
		log.Printf("[wa-manager] number %d has no device, needs pairing", numberID)
		m.DB.Model(&models.WaNumber{}).Where("id = ?", numberID).Update("status", "disconnected")
		return
	}

	client := whatsmeow.NewClient(device, waLog.Noop)
	handler := NewMessageHandler(m.DB, m.R2, m.EmbeddingWorker, m.VisionClient, numberID)
	handler.SetHub(m.Hub)
	handler.SetClient(client)
	client.AddEventHandler(handler.HandleEvent)

	if client.Store.ID == nil {
		log.Printf("[wa-manager] number %d device has no ID, needs pairing", numberID)
		m.DB.Model(&models.WaNumber{}).Where("id = ?", numberID).Update("status", "disconnected")
		return
	}

	if err := client.Connect(); err != nil {
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

func (m *Manager) SendMessage(ctx context.Context, numberID uint, jid string, text string) (string, error) {
	return m.SendMediaMessage(ctx, numberID, jid, text, "")
}

// SendMediaMessage sends text and, if mediaURL is set, an attachment. The media
// type is auto-detected from the URL's Content-Type. Used by the outbox worker
// and as the low-level primitive for inbox sends.
func (m *Manager) SendMediaMessage(ctx context.Context, numberID uint, jid string, text string, mediaURL string) (string, error) {
	return m.SendTyped(ctx, numberID, jid, text, mediaURL, "")
}

// SendTyped sends a message with an optional attachment of an explicit type
// ("image"|"video"|"audio"|"document"|"sticker"). Empty mediaType auto-detects.
func (m *Manager) SendTyped(ctx context.Context, numberID uint, jid string, text string, mediaURL string, mediaType string) (string, error) {
	client, ok := m.GetClient(numberID)
	if !ok {
		return "", fmt.Errorf("number %d not connected", numberID)
	}

	targetJID, err := parseJID(jid)
	if err != nil {
		return "", err
	}

	if mediaURL != "" {
		return m.sendMediaMessage(ctx, client, targetJID, text, mediaURL, mediaType)
	}

	resp, err := client.SendMessage(ctx, targetJID, &waE2E.Message{
		Conversation: &text,
	})
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// sendMediaMessage downloads media from a URL, uploads it to WhatsApp, and sends
// the appropriate typed message.
func (m *Manager) sendMediaMessage(ctx context.Context, client *whatsmeow.Client, targetJID types.JID, caption, mediaURL, mediaType string) (string, error) {
	httpResp, err := http.Get(mediaURL)
	if err != nil {
		return "", fmt.Errorf("download media: %w", err)
	}
	defer httpResp.Body.Close()

	data, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return "", fmt.Errorf("read media data: %w", err)
	}

	mimeType := httpResp.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = "application/octet-stream"
	}
	if mediaType == "" {
		mediaType = mediaTypeFromMime(mimeType)
	}

	fileName := path.Base(mediaURL)
	fileLen := uint64(len(data))

	var msg *waE2E.Message
	switch mediaType {
	case "image":
		up, err := client.Upload(ctx, data, whatsmeow.MediaImage)
		if err != nil {
			return "", fmt.Errorf("upload image: %w", err)
		}
		msg = &waE2E.Message{ImageMessage: &waE2E.ImageMessage{
			Caption: &caption, Mimetype: &mimeType, URL: &up.URL, DirectPath: &up.DirectPath,
			MediaKey: up.MediaKey, FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &up.FileLength,
		}}
	case "video":
		up, err := client.Upload(ctx, data, whatsmeow.MediaVideo)
		if err != nil {
			return "", fmt.Errorf("upload video: %w", err)
		}
		msg = &waE2E.Message{VideoMessage: &waE2E.VideoMessage{
			Caption: &caption, Mimetype: &mimeType, URL: &up.URL, DirectPath: &up.DirectPath,
			MediaKey: up.MediaKey, FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &up.FileLength,
		}}
	case "audio":
		up, err := client.Upload(ctx, data, whatsmeow.MediaAudio)
		if err != nil {
			return "", fmt.Errorf("upload audio: %w", err)
		}
		ptt := true
		msg = &waE2E.Message{AudioMessage: &waE2E.AudioMessage{
			Mimetype: &mimeType, URL: &up.URL, DirectPath: &up.DirectPath, PTT: &ptt,
			MediaKey: up.MediaKey, FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &up.FileLength,
		}}
	case "sticker":
		up, err := client.Upload(ctx, data, whatsmeow.MediaImage)
		if err != nil {
			return "", fmt.Errorf("upload sticker: %w", err)
		}
		msg = &waE2E.Message{StickerMessage: &waE2E.StickerMessage{
			Mimetype: &mimeType, URL: &up.URL, DirectPath: &up.DirectPath,
			MediaKey: up.MediaKey, FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &up.FileLength,
		}}
	default: // document
		up, err := client.Upload(ctx, data, whatsmeow.MediaDocument)
		if err != nil {
			return "", fmt.Errorf("upload document: %w", err)
		}
		msg = &waE2E.Message{DocumentMessage: &waE2E.DocumentMessage{
			Caption: &caption, Mimetype: &mimeType, FileName: &fileName, URL: &up.URL, DirectPath: &up.DirectPath,
			MediaKey: up.MediaKey, FileEncSHA256: up.FileEncSHA256, FileSHA256: up.FileSHA256, FileLength: &fileLen,
		}}
	}

	resp, err := client.SendMessage(ctx, targetJID, msg)
	if err != nil {
		return "", err
	}
	return resp.ID, nil
}

// SendInboxMessage sends a live inbox message (bypassing outbox approval),
// persists it as an outgoing WaMessage, updates conversation metadata, and
// broadcasts it to realtime clients. Returns the saved message.
func (m *Manager) SendInboxMessage(ctx context.Context, numberID uint, chatJID, text, mediaURL, mediaType string) (*models.WaMessage, error) {
	waMsgID, err := m.SendTyped(ctx, numberID, chatJID, text, mediaURL, mediaType)
	if err != nil {
		return nil, err
	}

	handler := NewMessageHandler(m.DB, m.R2, m.EmbeddingWorker, m.VisionClient, numberID)
	listener, err := handler.findOrCreateListener(chatJID, "")
	if err != nil {
		return nil, fmt.Errorf("resolve conversation: %w", err)
	}

	msgType := "text"
	content := text
	if mediaURL != "" {
		if mediaType != "" {
			msgType = mediaType
		} else {
			msgType = "document"
		}
		if content == "" {
			content = "[" + msgType + "]"
		}
	}

	now := time.Now()
	msg := &models.WaMessage{
		WaListenerID: listener.ID,
		MessageID:    waMsgID,
		SenderJID:    "",
		SenderName:   "You",
		Content:      content,
		MessageType:  msgType,
		HasMedia:     mediaURL != "",
		FromMe:       true,
		ChatJID:      chatJID,
		Status:       "sent",
		Timestamp:    now,
	}
	if err := m.DB.Create(msg).Error; err != nil {
		return nil, fmt.Errorf("persist message: %w", err)
	}

	// Attach a media record pointing at the already-hosted R2 URL.
	if mediaURL != "" {
		media := models.WaMedia{
			WaMessageID: msg.ID,
			FileName:    path.Base(mediaURL),
			MimeType:    mediaType,
			FileURL:     mediaURL,
		}
		m.DB.Create(&media)
		msg.Media = []models.WaMedia{media}
	}

	handler.SetHub(m.Hub)
	handler.touchListener(listener.ID, content, now, true)
	handler.broadcast("message", msg)
	return msg, nil
}

// mediaTypeFromMime maps a MIME type to a WhatsApp media category.
func mediaTypeFromMime(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return "image"
	case strings.HasPrefix(mime, "video/"):
		return "video"
	case strings.HasPrefix(mime, "audio/"):
		return "audio"
	default:
		return "document"
	}
}
