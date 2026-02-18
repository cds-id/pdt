package handlers

import (
	"net/http"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/worker"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type SyncHandler struct {
	DB        *gorm.DB
	Encryptor *crypto.Encryptor
	Status    *worker.SyncStatus
}

func (h *SyncHandler) SyncCommits(c *gin.Context) {
	userID := c.GetUint("user_id")

	results, err := worker.SyncUserCommits(h.DB, h.Encryptor, userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if results == nil {
		c.JSON(http.StatusOK, gin.H{"message": "no repositories to sync", "results": []worker.CommitSyncResult{}})
		return
	}

	c.JSON(http.StatusOK, gin.H{"results": results})
}

func (h *SyncHandler) SyncStatus(c *gin.Context) {
	userID := c.GetUint("user_id")

	response := gin.H{
		"commits": h.Status.GetCommitStatus(userID),
		"jira":    h.Status.GetJiraStatus(userID),
	}

	c.JSON(http.StatusOK, response)
}
