package worker

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services/jira"
	"gorm.io/gorm"
)

func SyncUserJira(db *gorm.DB, enc *crypto.Encryptor, userID uint) error {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return fmt.Errorf("user not found: %w", err)
	}

	if user.JiraToken == "" || user.JiraWorkspace == "" || user.JiraEmail == "" {
		return nil
	}

	token, err := enc.Decrypt(user.JiraToken)
	if err != nil {
		return fmt.Errorf("failed to decrypt jira token: %w", err)
	}

	client := jira.New(user.JiraWorkspace, user.JiraEmail, token)

	boards, err := client.FetchBoards()
	if err != nil {
		return fmt.Errorf("failed to fetch boards: %w", err)
	}

	for _, boardID := range boards {
		sprints, err := client.FetchSprints(boardID)
		if err != nil {
			log.Printf("[worker] jira sync user=%d board=%d sprint fetch error: %v", userID, boardID, err)
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
			db.Where("jira_sprint_id = ?", sprint.JiraSprintID).
				Assign(sprint).FirstOrCreate(&sprint)

			if s.State == "active" {
				cards, err := client.FetchSprintIssues(s.ID)
				if err != nil {
					log.Printf("[worker] jira sync user=%d sprint=%d issue fetch error: %v", userID, s.ID, err)
					continue
				}

				for _, card := range cards {
					jiraCard := models.JiraCard{
						UserID:   userID,
						Key:      card.Key,
						Summary:  card.Summary,
						Status:   card.Status,
						Assignee: card.Assignee,
						SprintID: &sprint.ID,
					}

					detail, err := client.FetchIssue(card.Key)
					if err == nil && detail != nil {
						detailJSON, _ := json.Marshal(detail)
						jiraCard.DetailsJSON = string(detailJSON)
					}

					db.Where("user_id = ? AND card_key = ?", userID, card.Key).
						Assign(jiraCard).FirstOrCreate(&jiraCard)
				}
			}
		}
	}

	return nil
}

func SyncAllUsersJira(db *gorm.DB, enc *crypto.Encryptor) {
	var users []models.User
	db.Where("jira_token != '' AND jira_workspace != '' AND jira_email != ''").Find(&users)

	for _, user := range users {
		if err := SyncUserJira(db, enc, user.ID); err != nil {
			log.Printf("[worker] jira sync failed for user %d: %v", user.ID, err)
		} else {
			log.Printf("[worker] jira sync completed for user %d", user.ID)
		}
	}
}
