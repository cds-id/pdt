package handlers

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/jira"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type JiraHandler struct {
	DB        *gorm.DB
	Encryptor *crypto.Encryptor
}

func (h *JiraHandler) getClient(userID uint) (*jira.Client, error) {
	var user models.User
	if err := h.DB.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found")
	}

	if user.JiraToken == "" || user.JiraWorkspace == "" || user.JiraEmail == "" {
		return nil, fmt.Errorf("jira not configured")
	}

	token, err := h.Encryptor.Decrypt(user.JiraToken)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt jira token")
	}

	return jira.New(user.JiraWorkspace, user.JiraEmail, token), nil
}

func (h *JiraHandler) ListSprints(c *gin.Context) {
	userID := c.GetUint("user_id")

	client, err := h.getClient(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	boards, err := client.FetchBoards()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch boards: " + err.Error()})
		return
	}

	var allSprints []jira.SprintInfo
	for _, boardID := range boards {
		sprints, err := client.FetchSprints(boardID)
		if err != nil {
			continue
		}
		allSprints = append(allSprints, sprints...)
	}

	// Sync to DB
	for _, s := range allSprints {
		sprint := models.Sprint{
			UserID:       userID,
			JiraSprintID: strconv.Itoa(s.ID),
			Name:         s.Name,
			State:        models.SprintState(s.State),
			StartDate:    s.StartDate,
			EndDate:      s.EndDate,
		}
		h.DB.Where("jira_sprint_id = ?", sprint.JiraSprintID).
			Assign(sprint).FirstOrCreate(&sprint)
	}

	var sprints []models.Sprint
	h.DB.Where("user_id = ?", userID).Order("start_date desc").Find(&sprints)

	c.JSON(http.StatusOK, sprints)
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

	var sprint models.Sprint
	if err := h.DB.Where("user_id = ? AND state = ?", userID, models.SprintActive).
		Preload("Cards").First(&sprint).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "no active sprint found"})
		return
	}

	c.JSON(http.StatusOK, sprint)
}

func (h *JiraHandler) ListCards(c *gin.Context) {
	userID := c.GetUint("user_id")
	sprintIDParam := c.Query("sprint_id")

	client, err := h.getClient(userID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var sprint models.Sprint
	if sprintIDParam != "" {
		if err := h.DB.Where("id = ? AND user_id = ?", sprintIDParam, userID).First(&sprint).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "sprint not found"})
			return
		}
	} else {
		// Default to active sprint
		if err := h.DB.Where("user_id = ? AND state = ?", userID, models.SprintActive).First(&sprint).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "no active sprint found"})
			return
		}
	}

	jiraSprintID, _ := strconv.Atoi(sprint.JiraSprintID)
	cards, err := client.FetchSprintIssues(jiraSprintID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to fetch cards: " + err.Error()})
		return
	}

	// Sync cards to DB
	for _, card := range cards {
		jiraCard := models.JiraCard{
			UserID:   userID,
			Key:      card.Key,
			Summary:  card.Summary,
			Status:   card.Status,
			Assignee: card.Assignee,
			SprintID: &sprint.ID,
		}
		h.DB.Where("user_id = ? AND card_key = ?", userID, card.Key).
			Assign(jiraCard).FirstOrCreate(&jiraCard)
	}

	var dbCards []models.JiraCard
	h.DB.Where("user_id = ? AND sprint_id = ?", userID, sprint.ID).Find(&dbCards)

	c.JSON(http.StatusOK, dbCards)
}

func (h *JiraHandler) GetCard(c *gin.Context) {
	userID := c.GetUint("user_id")
	cardKey := c.Param("key")

	// Find commits linked to this card key
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

	// Try to fetch issue details from Jira (parent + subtasks)
	var issueDetail *jira.IssueDetail
	client, err := h.getClient(userID)
	if err == nil {
		issueDetail, _ = client.FetchIssue(cardKey)
	}

	// Also collect commits from subtasks
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
		result["status"] = issueDetail.Status
		result["assignee"] = issueDetail.Assignee
		result["issue_type"] = issueDetail.IssueType
		result["parent"] = issueDetail.Parent
		result["subtasks"] = subtasks
		result["changelog"] = issueDetail.Changelog
	}

	c.JSON(http.StatusOK, result)
}
