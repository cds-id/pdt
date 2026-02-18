package handlers

import (
	"net/http"
	"time"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type CommitHandler struct {
	DB *gorm.DB
}

func (h *CommitHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")

	query := h.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ?", userID)

	if repoID := c.Query("repo_id"); repoID != "" {
		query = query.Where("commits.repo_id = ?", repoID)
	}
	if cardKey := c.Query("jira_card_key"); cardKey != "" {
		query = query.Where("commits.jira_card_key = ?", cardKey)
	}
	if hasLink := c.Query("has_link"); hasLink != "" {
		query = query.Where("commits.has_link = ?", hasLink == "true")
	}

	var commits []models.Commit
	if err := query.Order("commits.date desc").Find(&commits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch commits"})
		return
	}

	c.JSON(http.StatusOK, commits)
}

func (h *CommitHandler) Missing(c *gin.Context) {
	userID := c.GetUint("user_id")

	var commits []models.Commit
	if err := h.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.has_link = false", userID).
		Order("commits.date desc").
		Find(&commits).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch commits"})
		return
	}

	c.JSON(http.StatusOK, commits)
}

type linkRequest struct {
	JiraCardKey string `json:"jira_card_key" binding:"required"`
}

func (h *CommitHandler) Link(c *gin.Context) {
	userID := c.GetUint("user_id")
	sha := c.Param("sha")

	var req linkRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var commit models.Commit
	if err := h.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
		Where("repositories.user_id = ? AND commits.sha = ?", userID, sha).
		First(&commit).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "commit not found"})
		return
	}

	link := models.CommitCardLink{
		CommitID:    commit.ID,
		JiraCardKey: req.JiraCardKey,
		LinkedAt:    time.Now(),
	}

	if err := h.DB.Create(&link).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create link"})
		return
	}

	// Update has_link on commit
	h.DB.Model(&commit).Update("has_link", true)

	c.JSON(http.StatusCreated, link)
}
