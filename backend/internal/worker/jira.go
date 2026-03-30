package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/helpers"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/jira"
	wvClient "github.com/cds-id/pdt/backend/internal/services/weaviate"
	"gorm.io/gorm"
)

// SyncUserJira syncs all active workspaces for a user.
func SyncUserJira(db *gorm.DB, enc *crypto.Encryptor, userID uint, wv ...*wvClient.Client) error {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.JiraToken == "" || user.JiraEmail == "" {
		return nil
	}

	token, err := enc.Decrypt(user.JiraToken)
	if err != nil {
		return fmt.Errorf("failed to decrypt jira token: %w", err)
	}

	// Get all active workspaces for this user
	var workspaces []models.JiraWorkspaceConfig
	db.Where("user_id = ? AND is_active = ?", userID, true).Find(&workspaces)

	if len(workspaces) == 0 {
		log.Printf("[jira-sync] user=%d no active workspaces", userID)
		return nil
	}

	var wvC *wvClient.Client
	if len(wv) > 0 {
		wvC = wv[0]
	}

	for _, ws := range workspaces {
		log.Printf("[jira-sync] user=%d workspace=%s starting sync", userID, ws.Workspace)
		if err := syncWorkspace(db, user, token, ws, wvC); err != nil {
			log.Printf("[jira-sync] user=%d workspace=%s sync failed: %v", userID, ws.Workspace, err)
		} else {
			log.Printf("[jira-sync] user=%d workspace=%s sync completed", userID, ws.Workspace)
		}
	}

	return nil
}

func syncWorkspace(db *gorm.DB, user models.User, token string, ws models.JiraWorkspaceConfig, wvC *wvClient.Client) error {
	client := jira.New(ws.Workspace, user.JiraEmail, token)
	userID := user.ID
	wsID := ws.ID

	boards, err := client.FetchBoards()
	if err != nil {
		return fmt.Errorf("failed to fetch boards: %w", err)
	}

	log.Printf("[jira-sync] user=%d ws=%s found %d boards", userID, ws.Workspace, len(boards))

	for _, boardID := range boards {
		sprints, err := client.FetchSprints(boardID)
		if err != nil {
			log.Printf("[jira-sync] user=%d ws=%s board=%d sprint fetch error: %v", userID, ws.Workspace, boardID, err)
			continue
		}

		for _, s := range sprints {
			sprint := models.Sprint{
				UserID:       userID,
				WorkspaceID:  &wsID,
				JiraSprintID: strconv.Itoa(s.ID),
				Name:         s.Name,
				State:        models.SprintState(s.State),
				StartDate:    s.StartDate,
				EndDate:      s.EndDate,
			}
			db.Where("jira_sprint_id = ?", sprint.JiraSprintID).
				Assign(sprint).FirstOrCreate(&sprint)

			state := models.SprintState(s.State)
			if state == models.SprintActive || state == models.SprintClosed {
				cards, err := client.FetchSprintIssues(s.ID)
				if err != nil {
					log.Printf("[jira-sync] user=%d ws=%s sprint=%d issue fetch error: %v", userID, ws.Workspace, s.ID, err)
					continue
				}

				savedCount := 0
				for _, card := range cards {
					if !helpers.FilterByProjectKeys(card.Key, ws.ProjectKeys) {
						continue
					}

					jiraCard := models.JiraCard{
						UserID:      userID,
						WorkspaceID: &wsID,
						Key:         card.Key,
						Summary:     card.Summary,
						Status:      card.Status,
						Assignee:    card.Assignee,
						SprintID:    &sprint.ID,
					}

					detail, err := client.FetchIssue(card.Key)
					if err == nil && detail != nil {
						detailJSON, _ := json.Marshal(detail)
						jiraCard.DetailsJSON = string(detailJSON)
					}

					db.Where("user_id = ? AND card_key = ?", userID, card.Key).
						Assign(jiraCard).FirstOrCreate(&jiraCard)

					// Embed card in Weaviate
					if wvC != nil {
						embedContent := card.Summary
						if detail != nil && detail.Description != "" {
							embedContent = card.Summary + "\n" + detail.Description
						}
						if err := wvC.UpsertJiraCard(context.Background(), card.Key, int(userID), int(wsID), embedContent, card.Status, card.Assignee); err != nil {
							log.Printf("[jira-sync] embed card %s error: %v", card.Key, err)
						}
					}

					comments, err := client.FetchIssueComments(card.Key)
					if err != nil {
						log.Printf("[jira-sync] user=%d ws=%s card=%s fetch comments error: %v", userID, ws.Workspace, card.Key, err)
					} else {
						for _, comment := range comments {
							jiraComment := models.JiraComment{
								UserID:      userID,
								WorkspaceID: &wsID,
								CardKey:     card.Key,
								CommentID:   comment.ID,
								Author:      comment.Author,
								AuthorEmail: comment.AuthorEmail,
								Body:        comment.Body,
								CommentedAt: comment.Created,
							}
							db.Where("comment_id = ?", comment.ID).
								Assign(jiraComment).FirstOrCreate(&jiraComment)

							// Embed comment in Weaviate
							if wvC != nil {
								if err := wvC.UpsertJiraComment(context.Background(), comment.ID, card.Key, int(userID), int(wsID), comment.Body, comment.Author); err != nil {
									log.Printf("[jira-sync] embed comment %s error: %v", comment.ID, err)
								}
							}
						}
					}

					savedCount++
				}

				log.Printf("[jira-sync] user=%d ws=%s sprint=%d (%s) found %d cards, %d after filter",
					userID, ws.Workspace, s.ID, s.State, len(cards), savedCount)
			}
		}
	}

	return nil
}

// SyncUserJiraWorkspace syncs a single workspace for a user.
func SyncUserJiraWorkspace(db *gorm.DB, enc *crypto.Encryptor, userID uint, workspaceID uint, wv ...*wvClient.Client) error {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.JiraToken == "" || user.JiraEmail == "" {
		return fmt.Errorf("jira credentials not configured")
	}

	token, err := enc.Decrypt(user.JiraToken)
	if err != nil {
		return fmt.Errorf("failed to decrypt jira token: %w", err)
	}

	var ws models.JiraWorkspaceConfig
	if err := db.Where("id = ? AND user_id = ?", workspaceID, userID).First(&ws).Error; err != nil {
		return fmt.Errorf("workspace not found")
	}

	var wvC *wvClient.Client
	if len(wv) > 0 {
		wvC = wv[0]
	}
	return syncWorkspace(db, user, token, ws, wvC)
}

func SyncAllUsersJira(db *gorm.DB, enc *crypto.Encryptor) {
	// Find users that have at least one active workspace
	var userIDs []uint
	db.Model(&models.JiraWorkspaceConfig{}).
		Where("is_active = ?", true).
		Distinct("user_id").
		Pluck("user_id", &userIDs)

	for _, userID := range userIDs {
		if err := SyncUserJira(db, enc, userID); err != nil {
			log.Printf("[jira-sync] user=%d sync failed: %v", userID, err)
		}
	}
}
