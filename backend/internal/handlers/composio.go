package handlers

import (
	"fmt"
	"net/http"

	"github.com/cds-id/pdt/backend/internal/ai/composio"
	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type ComposioHandler struct {
	DB             *gorm.DB
	Encryptor      *crypto.Encryptor
	ComposioClient *composio.Client
}

// GetConfig returns whether the user has a Composio API key configured.
func (h *ComposioHandler) GetConfig(c *gin.Context) {
	userID := c.GetUint("user_id")

	var cfg models.ComposioConfig
	err := h.DB.Where("user_id = ?", userID).First(&cfg).Error
	hasKey := err == nil

	c.JSON(http.StatusOK, gin.H{
		"configured": hasKey,
	})
}

// SaveConfig saves or updates the user's Composio API key.
func (h *ComposioHandler) SaveConfig(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		APIKey string `json:"api_key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate the API key by making a test call
	_, err := h.ComposioClient.GetConnectedAccounts(req.APIKey, fmt.Sprintf("pdt-user-%d", userID))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid Composio API key"})
		return
	}

	encrypted, err := h.Encryptor.Encrypt(req.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "encryption failed"})
		return
	}

	var cfg models.ComposioConfig
	result := h.DB.Where("user_id = ?", userID).First(&cfg)
	if result.Error != nil {
		cfg = models.ComposioConfig{
			UserID: userID,
			APIKey: encrypted,
		}
		h.DB.Create(&cfg)
	} else {
		h.DB.Model(&cfg).Update("api_key", encrypted)
	}

	c.JSON(http.StatusOK, gin.H{"configured": true})
}

// DeleteConfig removes the user's Composio API key and all connections.
func (h *ComposioHandler) DeleteConfig(c *gin.Context) {
	userID := c.GetUint("user_id")

	h.DB.Where("user_id = ?", userID).Delete(&models.ComposioConnection{})
	h.DB.Where("user_id = ?", userID).Delete(&models.ComposioConfig{})

	c.JSON(http.StatusOK, gin.H{"message": "Composio configuration removed"})
}

// ListConnections returns the user's Composio service connections.
func (h *ComposioHandler) ListConnections(c *gin.Context) {
	userID := c.GetUint("user_id")

	var connections []models.ComposioConnection
	h.DB.Where("user_id = ?", userID).Find(&connections)

	c.JSON(http.StatusOK, connections)
}

// SaveAuthConfigID saves the Composio auth config ID for a toolkit.
func (h *ComposioHandler) SaveAuthConfigID(c *gin.Context) {
	userID := c.GetUint("user_id")
	toolkit := c.Param("toolkit")

	var req struct {
		AuthConfigID string `json:"auth_config_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var conn models.ComposioConnection
	result := h.DB.Where("user_id = ? AND toolkit = ?", userID, toolkit).First(&conn)
	if result.Error != nil {
		conn = models.ComposioConnection{
			UserID:       userID,
			Toolkit:      toolkit,
			AuthConfigID: req.AuthConfigID,
		}
		h.DB.Create(&conn)
	} else {
		h.DB.Model(&conn).Update("auth_config_id", req.AuthConfigID)
	}

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("Auth config saved for %s", toolkit)})
}

// InitiateConnection starts the OAuth flow for a toolkit.
func (h *ComposioHandler) InitiateConnection(c *gin.Context) {
	userID := c.GetUint("user_id")
	toolkit := c.Param("toolkit")

	var req struct {
		RedirectURI string `json:"redirect_uri" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var conn models.ComposioConnection
	if err := h.DB.Where("user_id = ? AND toolkit = ?", userID, toolkit).First(&conn).Error; err != nil || conn.AuthConfigID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Auth Config ID not set for this toolkit. Please configure it first."})
		return
	}

	var cfg models.ComposioConfig
	if err := h.DB.Where("user_id = ?", userID).First(&cfg).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Composio API key not configured"})
		return
	}

	apiKey, err := h.Encryptor.Decrypt(cfg.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "decryption failed"})
		return
	}

	entityID := fmt.Sprintf("pdt-user-%d", userID)
	result, err := h.ComposioClient.InitiateConnection(apiKey, conn.AuthConfigID, req.RedirectURI, entityID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	h.DB.Model(&conn).Updates(map[string]any{
		"account_id": result.ConnectedAccountID,
		"status":     result.ConnectionStatus,
	})

	c.JSON(http.StatusOK, gin.H{
		"redirect_url": result.RedirectURL,
		"account_id":   result.ConnectedAccountID,
		"status":       result.ConnectionStatus,
	})
}

// SyncConnections refreshes connection statuses from Composio API.
func (h *ComposioHandler) SyncConnections(c *gin.Context) {
	userID := c.GetUint("user_id")

	var cfg models.ComposioConfig
	if err := h.DB.Where("user_id = ?", userID).First(&cfg).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Composio API key not configured"})
		return
	}

	apiKey, err := h.Encryptor.Decrypt(cfg.APIKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "decryption failed"})
		return
	}

	entityID := fmt.Sprintf("pdt-user-%d", userID)
	accounts, err := h.ComposioClient.GetConnectedAccounts(apiKey, entityID)
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error()})
		return
	}

	for _, acc := range accounts {
		h.DB.Model(&models.ComposioConnection{}).
			Where("user_id = ? AND account_id = ?", userID, acc.ID).
			Update("status", acc.Status)
	}

	var connections []models.ComposioConnection
	h.DB.Where("user_id = ?", userID).Find(&connections)

	c.JSON(http.StatusOK, connections)
}

// DeleteConnection removes a service connection.
func (h *ComposioHandler) DeleteConnection(c *gin.Context) {
	userID := c.GetUint("user_id")
	toolkit := c.Param("toolkit")

	h.DB.Where("user_id = ? AND toolkit = ?", userID, toolkit).Delete(&models.ComposioConnection{})

	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%s disconnected", toolkit)})
}
