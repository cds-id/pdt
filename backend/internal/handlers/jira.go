package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/helpers"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/jira"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type JiraHandler struct {
	DB        *gorm.DB
	Encryptor *crypto.Encryptor
}

// getClientForWorkspace creates a Jira client for a specific workspace.
func (h *JiraHandler) getClientForWorkspace(userID uint, workspaceID uint) (*jira.Client, *models.JiraWorkspaceConfig, error) {
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	if user.JiraToken == "" || user.JiraEmail == "" {
		return nil, nil, fmt.Errorf("jira credentials not configured")
	}

	var ws models.JiraWorkspaceConfig
	if err := h.DB.Where("id = ? AND user_id = ?", workspaceID, userID).First(&ws).Error; err != nil {
		return nil, nil, fmt.Errorf("workspace not found")
	}

	token, err := h.Encryptor.Decrypt(user.JiraToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt jira token")
	}

	return jira.New(ws.Workspace, user.JiraEmail, token), &ws, nil
}

// getDefaultClient creates a Jira client using the first active workspace.
func (h *JiraHandler) getDefaultClient(userID uint) (*jira.Client, *models.JiraWorkspaceConfig, error) {
	var ws models.JiraWorkspaceConfig
	if err := h.DB.Where("user_id = ? AND is_active = ?", userID, true).First(&ws).Error; err != nil {
		// Fallback to legacy User fields
		return h.getLegacyClient(userID)
	}
	return h.getClientForWorkspace(userID, ws.ID)
}

// getLegacyClient falls back to User model fields (backwards compat).
func (h *JiraHandler) getLegacyClient(userID uint) (*jira.Client, *models.JiraWorkspaceConfig, error) {
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		return nil, nil, fmt.Errorf("user not found")
	}

	if user.JiraToken == "" || user.JiraWorkspace == "" || user.JiraEmail == "" {
		return nil, nil, fmt.Errorf("jira not configured")
	}

	token, err := h.Encryptor.Decrypt(user.JiraToken)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to decrypt jira token")
	}

	return jira.New(user.JiraWorkspace, user.JiraEmail, token), nil, nil
}

// resolveWorkspaceID parses optional workspace_id query param.
func resolveWorkspaceID(c *gin.Context) *uint {
	wsParam := c.Query("workspace_id")
	if wsParam == "" {
		return nil
	}
	id, err := strconv.ParseUint(wsParam, 10, 32)
	if err != nil {
		return nil
	}
	v := uint(id)
	return &v
}

// --- Workspace CRUD ---

func (h *JiraHandler) ListWorkspaces(c *gin.Context) {
	userID := c.GetUint("user_id")

	var workspaces []models.JiraWorkspaceConfig
	h.DB.Where("user_id = ?", userID).Order("created_at").Find(&workspaces)

	c.JSON(http.StatusOK, workspaces)
}

func (h *JiraHandler) AddWorkspace(c *gin.Context) {
	userID := c.GetUint("user_id")

	var req struct {
		Workspace   string `json:"workspace" binding:"required"`
		Name        string `json:"name"`
		ProjectKeys string `json:"project_keys"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Name == "" {
		req.Name = req.Workspace
	}

	ws := models.JiraWorkspaceConfig{
		UserID:      userID,
		Workspace:   req.Workspace,
		Name:        req.Name,
		ProjectKeys: req.ProjectKeys,
		IsActive:    true,
	}
	if err := h.DB.Create(&ws).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create workspace"})
		return
	}

	c.JSON(http.StatusCreated, ws)
}

func (h *JiraHandler) UpdateWorkspace(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID := c.Param("id")

	var ws models.JiraWorkspaceConfig
	if err := h.DB.Where("id = ? AND user_id = ?", wsID, userID).First(&ws).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	var req struct {
		Workspace   *string `json:"workspace"`
		Name        *string `json:"name"`
		ProjectKeys *string `json:"project_keys"`
		IsActive    *bool   `json:"is_active"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	updates := map[string]interface{}{}
	if req.Workspace != nil {
		updates["workspace"] = *req.Workspace
	}
	if req.Name != nil {
		updates["name"] = *req.Name
	}
	if req.ProjectKeys != nil {
		updates["project_keys"] = *req.ProjectKeys
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	h.DB.Model(&ws).Updates(updates)
	h.DB.First(&ws, ws.ID)

	c.JSON(http.StatusOK, ws)
}

func (h *JiraHandler) DeleteWorkspace(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID := c.Param("id")

	result := h.DB.Where("id = ? AND user_id = ?", wsID, userID).Delete(&models.JiraWorkspaceConfig{})
	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "workspace not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

// --- Sprint / Card / Comment endpoints (workspace-aware) ---

func (h *JiraHandler) ListSprints(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID := resolveWorkspaceID(c)

	// If workspace_id provided, sync from that workspace
	if wsID != nil {
		client, ws, err := h.getClientForWorkspace(userID, *wsID)
		if err == nil {
			h.syncSprintsFromJira(client, userID, ws)
		}
	} else {
		// Sync from first active workspace
		client, ws, err := h.getDefaultClient(userID)
		if err == nil {
			h.syncSprintsFromJira(client, userID, ws)
		}
	}

	query := h.DB.Where("user_id = ?", userID)
	if wsID != nil {
		query = query.Where("workspace_id = ?", *wsID)
	}

	var sprints []models.Sprint
	query.Order("start_date desc").Find(&sprints)

	c.JSON(http.StatusOK, sprints)
}

func (h *JiraHandler) syncSprintsFromJira(client *jira.Client, userID uint, ws *models.JiraWorkspaceConfig) {
	boards, err := client.FetchBoards()
	if err != nil {
		return
	}

	for _, boardID := range boards {
		sprints, err := client.FetchSprints(boardID)
		if err != nil {
			continue
		}
		for _, s := range sprints {
			sprint := models.Sprint{
				UserID:       userID,
				JiraSprintID: strconv.Itoa(s.ID),
				Name:         s.Name,
				State:        models.SprintState(s.State),
				StartDate:    s.StartDate,
				EndDate:      s.EndDate,
			}
			if ws != nil {
				sprint.WorkspaceID = &ws.ID
			}
			h.DB.Where("jira_sprint_id = ?", sprint.JiraSprintID).
				Assign(sprint).FirstOrCreate(&sprint)
		}
	}
}

func (h *JiraHandler) GetSprint(c *gin.Context) {
	userID := c.GetUint("user_id")
	sprintID := c.Param("id")

	var sprint models.Sprint
	if err := h.DB.Where("id = ? AND user_id = ?", sprintID, userID).
		Preload("Cards").First(&sprint).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "sprint not found"})
		return
	}

	c.JSON(http.StatusOK, sprint)
}

func (h *JiraHandler) GetActiveSprint(c *gin.Context) {
	userID := c.GetUint("user_id")
	wsID := resolveWorkspaceID(c)

	query := h.DB.Where("user_id = ? AND state = ?", userID, models.SprintActive)
	if wsID != nil {
		query = query.Where("workspace_id = ?", *wsID)
	}

	var sprint models.Sprint
	if err := query.Order("start_date DESC").First(&sprint).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active sprint found"})
		return
	}

	// Load cards with project key filter from workspace
	cardQuery := h.DB.Where("sprint_id = ?", sprint.ID)
	if sprint.WorkspaceID != nil {
		var ws models.JiraWorkspaceConfig
		if h.DB.First(&ws, *sprint.WorkspaceID).Error == nil && ws.ProjectKeys != "" {
			clause, args := helpers.BuildProjectKeyWhereClauses(ws.ProjectKeys, "card_key")
			if clause != "" {
				cardQuery = cardQuery.Where(clause, args...)
			}
		}
	} else {
		// Legacy fallback
		var user models.User
		if err := h.DB.First(&user, userID).Error; err == nil && user.JiraProjectKeys != "" {
			clause, args := helpers.BuildProjectKeyWhereClauses(user.JiraProjectKeys, "card_key")
			if clause != "" {
				cardQuery = cardQuery.Where(clause, args...)
			}
		}
	}
	cardQuery.Find(&sprint.Cards)

	c.JSON(http.StatusOK, sprint)
}

func (h *JiraHandler) ListCards(c *gin.Context) {
	userID := c.GetUint("user_id")
	sprintIDParam := c.Query("sprint_id")
	wsID := resolveWorkspaceID(c)

	var sprint models.Sprint
	if sprintIDParam != "" {
		if err := h.DB.Where("id = ? AND user_id = ?", sprintIDParam, userID).First(&sprint).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "sprint not found"})
			return
		}
	} else {
		query := h.DB.Where("user_id = ? AND state = ?", userID, models.SprintActive)
		if wsID != nil {
			query = query.Where("workspace_id = ?", *wsID)
		}
		if err := query.Order("start_date DESC").First(&sprint).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active sprint found"})
			return
		}
	}

	// Sync from Jira if possible
	if sprint.WorkspaceID != nil {
		client, ws, err := h.getClientForWorkspace(userID, *sprint.WorkspaceID)
		if err == nil {
			jiraSprintID, _ := strconv.Atoi(sprint.JiraSprintID)
			cards, err := client.FetchSprintIssues(jiraSprintID)
			if err == nil {
				for _, card := range cards {
					jiraCard := models.JiraCard{
						UserID:      userID,
						WorkspaceID: sprint.WorkspaceID,
						Key:         card.Key,
						Summary:     card.Summary,
						Status:      card.Status,
						Assignee:    card.Assignee,
						SprintID:    &sprint.ID,
					}
					h.DB.Where("user_id = ? AND card_key = ?", userID, card.Key).
						Assign(jiraCard).FirstOrCreate(&jiraCard)
				}
			}
			// Apply project key filter from workspace
			_ = ws // ws used for filtering below
		}
	}

	cardQuery := h.DB.Where("user_id = ? AND sprint_id = ?", userID, sprint.ID)

	// Apply project key filtering
	projectKeys := ""
	if sprint.WorkspaceID != nil {
		var ws models.JiraWorkspaceConfig
		if h.DB.First(&ws, *sprint.WorkspaceID).Error == nil {
			projectKeys = ws.ProjectKeys
		}
	}
	if projectKeys == "" {
		var user models.User
		if h.DB.First(&user, userID).Error == nil {
			projectKeys = user.JiraProjectKeys
		}
	}
	if clause, args := helpers.BuildProjectKeyWhereClauses(projectKeys, "card_key"); clause != "" {
		cardQuery = cardQuery.Where(clause, args...)
	}

	var dbCards []models.JiraCard
	cardQuery.Find(&dbCards)

	c.JSON(http.StatusOK, dbCards)
}

func (h *JiraHandler) GetCard(c *gin.Context) {
	userID := c.GetUint("user_id")
	cardKey := c.Param("key")

	findCommits := func(key string) []models.Commit {
		var commits []models.Commit
		h.DB.Joins("JOIN repositories ON repositories.id = commits.repo_id").
			Where("repositories.user_id = ? AND (commits.jira_card_key = ? OR commits.id IN (SELECT commit_id FROM commit_card_links WHERE jira_card_key = ?))",
				userID, key, key).
			Order("commits.date desc").
			Find(&commits)
		return commits
	}

	commits := findCommits(cardKey)

	// Try to fetch from Jira — find workspace for this card
	var issueDetail *jira.IssueDetail
	var card models.JiraCard
	if h.DB.Where("user_id = ? AND card_key = ?", userID, cardKey).First(&card).Error == nil && card.WorkspaceID != nil {
		client, _, err := h.getClientForWorkspace(userID, *card.WorkspaceID)
		if err == nil {
			issueDetail, _ = client.FetchIssue(cardKey)
		}
	} else {
		client, _, err := h.getDefaultClient(userID)
		if err == nil {
			issueDetail, _ = client.FetchIssue(cardKey)
		}
	}

	type subtaskWithCommits struct {
		Key     string          `json:"key"`
		Summary string          `json:"summary"`
		Status  string          `json:"status"`
		Type    string          `json:"type"`
		Commits []models.Commit `json:"commits"`
	}

	var subtasks []subtaskWithCommits
	if issueDetail != nil {
		for _, st := range issueDetail.Subtasks {
			stCommits := findCommits(st.Key)
			subtasks = append(subtasks, subtaskWithCommits{
				Key:     st.Key,
				Summary: st.Summary,
				Status:  st.Status,
				Type:    st.Type,
				Commits: stCommits,
			})
		}
	}

	if issueDetail == nil && len(commits) == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "card not found"})
		return
	}

	result := gin.H{
		"key":     cardKey,
		"commits": commits,
	}

	if issueDetail != nil {
		result["summary"] = issueDetail.Summary
		result["description"] = issueDetail.Description
		result["status"] = issueDetail.Status
		result["assignee"] = issueDetail.Assignee
		result["issue_type"] = issueDetail.IssueType
		result["parent"] = issueDetail.Parent
		result["subtasks"] = subtasks
		result["changelog"] = issueDetail.Changelog
	}

	c.JSON(http.StatusOK, result)
}

func (h *JiraHandler) GetCardComments(c *gin.Context) {
	userID := c.GetUint("user_id")
	cardKey := c.Param("key")

	var comments []models.JiraComment
	h.DB.Where("user_id = ? AND card_key = ?", userID, cardKey).
		Order("commented_at asc").
		Find(&comments)

	c.JSON(http.StatusOK, comments)
}
