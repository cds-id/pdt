package handlers

import (
	"net/http"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type UserHandler struct {
	DB        *gorm.DB
	Encryptor *crypto.Encryptor
}

type profileResponse struct {
	ID            uint   `json:"id"`
	Email         string `json:"email"`
	HasGithub     bool   `json:"has_github_token"`
	HasGitlab     bool   `json:"has_gitlab_token"`
	GitlabURL     string `json:"gitlab_url"`
	JiraEmail     string `json:"jira_email"`
	HasJiraToken  bool   `json:"has_jira_token"`
	JiraWorkspace string `json:"jira_workspace"`
	JiraUsername     string `json:"jira_username"`
	JiraProjectKeys string `json:"jira_project_keys"`
}

type updateProfileRequest struct {
	GithubToken   *string `json:"github_token"`
	GitlabToken   *string `json:"gitlab_token"`
	GitlabURL     *string `json:"gitlab_url"`
	JiraEmail     *string `json:"jira_email"`
	JiraToken     *string `json:"jira_token"`
	JiraWorkspace *string `json:"jira_workspace"`
	JiraUsername     *string `json:"jira_username"`
	JiraProjectKeys *string `json:"jira_project_keys"`
}

func (h *UserHandler) GetProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	c.JSON(http.StatusOK, profileResponse{
		ID:            user.ID,
		Email:         user.Email,
		HasGithub:     user.GithubToken != "",
		HasGitlab:     user.GitlabToken != "",
		GitlabURL:     user.GitlabURL,
		JiraEmail:     user.JiraEmail,
		HasJiraToken:  user.JiraToken != "",
		JiraWorkspace: user.JiraWorkspace,
		JiraUsername:     user.JiraUsername,
		JiraProjectKeys: user.JiraProjectKeys,
	})
}

func (h *UserHandler) UpdateProfile(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req updateProfileRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	updates := map[string]interface{}{}

	if req.GithubToken != nil {
		encrypted, err := h.Encryptor.Encrypt(*req.GithubToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt token"})
			return
		}
		updates["github_token"] = encrypted
	}
	if req.GitlabToken != nil {
		encrypted, err := h.Encryptor.Encrypt(*req.GitlabToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt token"})
			return
		}
		updates["gitlab_token"] = encrypted
	}
	if req.GitlabURL != nil {
		updates["gitlab_url"] = *req.GitlabURL
	}
	if req.JiraEmail != nil {
		updates["jira_email"] = *req.JiraEmail
	}
	if req.JiraToken != nil {
		encrypted, err := h.Encryptor.Encrypt(*req.JiraToken)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to encrypt token"})
			return
		}
		updates["jira_token"] = encrypted
	}
	if req.JiraWorkspace != nil {
		updates["jira_workspace"] = *req.JiraWorkspace
	}
	if req.JiraUsername != nil {
		updates["jira_username"] = *req.JiraUsername
	}
	if req.JiraProjectKeys != nil {
		updates["jira_project_keys"] = *req.JiraProjectKeys
	}

	if len(updates) > 0 {
		if err := h.DB.Model(&user).Updates(updates).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update profile"})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "profile updated"})
}

func (h *UserHandler) ValidateConnections(c *gin.Context) {
	userID := c.GetUint("user_id")

	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}

	results := map[string]interface{}{
		"github": map[string]interface{}{"configured": user.GithubToken != ""},
		"gitlab": map[string]interface{}{"configured": user.GitlabToken != ""},
		"jira":   map[string]interface{}{"configured": user.JiraToken != "" && user.JiraWorkspace != ""},
	}

	c.JSON(http.StatusOK, results)
}
