package handlers

import (
	"net/http"
	"net/url"
	"strings"

	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type RepoHandler struct {
	DB *gorm.DB
}

type addRepoRequest struct {
	URL string `json:"url" binding:"required,url"`
}

func (h *RepoHandler) List(c *gin.Context) {
	userID := c.GetUint("user_id")

	var repos []models.Repository
	if err := h.DB.Where("user_id = ?", userID).Find(&repos).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch repositories"})
		return
	}

	c.JSON(http.StatusOK, repos)
}

func (h *RepoHandler) Add(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req addRepoRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	owner, name, provider, err := parseRepoURL(req.URL)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check for duplicates
	var existing models.Repository
	if err := h.DB.Where("user_id = ? AND owner = ? AND name = ? AND provider = ?", userID, owner, name, provider).
		First(&existing).Error; err == nil {
		c.JSON(http.StatusConflict, gin.H{"error": "repository already tracked"})
		return
	}

	repo := models.Repository{
		UserID:   userID,
		Name:     name,
		Owner:    owner,
		Provider: provider,
		URL:      req.URL,
		IsValid:  true,
	}

	if err := h.DB.Create(&repo).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to add repository"})
		return
	}

	c.JSON(http.StatusCreated, repo)
}

func (h *RepoHandler) Delete(c *gin.Context) {
	userID := c.GetUint("user_id")
	repoID := c.Param("id")

	var repo models.Repository
	if err := h.DB.Where("id = ? AND user_id = ?", repoID, userID).First(&repo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Delete associated commits first
	h.DB.Where("repo_id = ?", repo.ID).Delete(&models.Commit{})
	h.DB.Delete(&repo)

	c.JSON(http.StatusOK, gin.H{"message": "repository removed"})
}

func (h *RepoHandler) Validate(c *gin.Context) {
	userID := c.GetUint("user_id")
	repoID := c.Param("id")

	var repo models.Repository
	if err := h.DB.Where("id = ? AND user_id = ?", repoID, userID).First(&repo).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "repository not found"})
		return
	}

	// Validation will be done by the sync services during SIT
	c.JSON(http.StatusOK, gin.H{
		"id":       repo.ID,
		"url":      repo.URL,
		"provider": repo.Provider,
		"is_valid": repo.IsValid,
		"message":  "validation will be performed during sync",
	})
}

func parseRepoURL(rawURL string) (owner, name string, provider models.Provider, err error) {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "", "", "", err
	}

	host := strings.ToLower(u.Host)
	path := strings.Trim(u.Path, "/")
	path = strings.TrimSuffix(path, ".git")

	parts := strings.SplitN(path, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", "", &url.Error{Op: "parse", URL: rawURL, Err: http.ErrNotSupported}
	}

	owner = parts[0]
	name = parts[1]

	if strings.Contains(host, "github.com") {
		provider = models.ProviderGitHub
	} else {
		provider = models.ProviderGitLab
	}

	return owner, name, provider, nil
}
