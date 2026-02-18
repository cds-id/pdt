package worker

import (
	"fmt"
	"log"
	"time"

	"github.com/cds-id/pdt/backend/internal/crypto"
	"github.com/cds-id/pdt/backend/internal/models"
	"github.com/cds-id/pdt/backend/internal/services"
	"github.com/cds-id/pdt/backend/internal/services/github"
	"github.com/cds-id/pdt/backend/internal/services/gitlab"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type CommitSyncResult struct {
	RepoID   uint   `json:"repo_id"`
	RepoName string `json:"repo_name"`
	Provider string `json:"provider"`
	New      int    `json:"new_commits"`
	Total    int    `json:"total_fetched"`
	Error    string `json:"error,omitempty"`
}

func SyncUserCommits(db *gorm.DB, enc *crypto.Encryptor, userID uint) ([]CommitSyncResult, error) {
	var user models.User
	if err := db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	var repos []models.Repository
	if err := db.Where("user_id = ?", userID).Find(&repos).Error; err != nil {
		return nil, fmt.Errorf("failed to fetch repositories: %w", err)
	}

	if len(repos) == 0 {
		return nil, nil
	}

	since := time.Now().AddDate(0, 0, -30)
	var results []CommitSyncResult

	for _, repo := range repos {
		result := CommitSyncResult{
			RepoID:   repo.ID,
			RepoName: fmt.Sprintf("%s/%s", repo.Owner, repo.Name),
			Provider: string(repo.Provider),
		}

		var provider services.CommitProvider
		var token string

		switch repo.Provider {
		case models.ProviderGitHub:
			provider = github.New()
			decrypted, err := enc.Decrypt(user.GithubToken)
			if err != nil {
				result.Error = "failed to decrypt github token"
				results = append(results, result)
				continue
			}
			token = decrypted
		case models.ProviderGitLab:
			provider = gitlab.New(user.GitlabURL)
			decrypted, err := enc.Decrypt(user.GitlabToken)
			if err != nil {
				result.Error = "failed to decrypt gitlab token"
				results = append(results, result)
				continue
			}
			token = decrypted
		}

		if token == "" {
			result.Error = fmt.Sprintf("no %s token configured", repo.Provider)
			results = append(results, result)
			continue
		}

		commits, err := provider.FetchCommits(repo.Owner, repo.Name, token, since)
		if err != nil {
			result.Error = err.Error()
			db.Model(&repo).Update("is_valid", false)
			results = append(results, result)
			continue
		}

		result.Total = len(commits)

		for _, ci := range commits {
			jiraKey := services.ExtractJiraKey(ci.Message)
			commit := models.Commit{
				RepoID:      repo.ID,
				SHA:         ci.SHA,
				Message:     ci.Message,
				Author:      ci.Author,
				AuthorEmail: ci.AuthorEmail,
				Branch:      ci.Branch,
				Date:        ci.Date,
				JiraCardKey: jiraKey,
				HasLink:     jiraKey != "",
			}

			res := db.Clauses(clause.OnConflict{
				Columns:   []clause.Column{{Name: "sha"}},
				DoNothing: true,
			}).Create(&commit)

			if res.RowsAffected > 0 {
				result.New++
			}
		}

		now := time.Now()
		db.Model(&repo).Updates(map[string]interface{}{
			"is_valid":       true,
			"last_synced_at": &now,
		})

		results = append(results, result)
	}

	return results, nil
}

func SyncAllUsersCommits(db *gorm.DB, enc *crypto.Encryptor) {
	var userIDs []uint
	db.Model(&models.Repository{}).Distinct("user_id").Pluck("user_id", &userIDs)

	for _, uid := range userIDs {
		results, err := SyncUserCommits(db, enc, uid)
		if err != nil {
			log.Printf("[worker] commit sync failed for user %d: %v", uid, err)
			continue
		}
		for _, r := range results {
			if r.Error != "" {
				log.Printf("[worker] commit sync user=%d repo=%s error=%s", uid, r.RepoName, r.Error)
			} else if r.New > 0 {
				log.Printf("[worker] commit sync user=%d repo=%s new=%d total=%d", uid, r.RepoName, r.New, r.Total)
			}
		}
	}
}
