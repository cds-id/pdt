package executive

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/cds-id/pdt/backend/internal/models"
	weaviatesvc "github.com/cds-id/pdt/backend/internal/services/weaviate"
)

// weaviateAdapter adapts the production weaviate.Client to the executive.WeaviateClient
// interface. Jira cards are sourced from Postgres (date-filterable); commit and WA
// queries use the Weaviate client.
type weaviateAdapter struct {
	db *gorm.DB
	c  *weaviatesvc.Client
}

// NewWeaviateAdapter returns a WeaviateClient backed by both Postgres (for Jira cards)
// and Weaviate (for commits and WA messages).
func NewWeaviateAdapter(db *gorm.DB, wv *weaviatesvc.Client) WeaviateClient {
	return &weaviateAdapter{db: db, c: wv}
}

// ListJiraCards returns Jira cards from Postgres filtered by userID, optional
// workspaceID, and the [start, end] updated_at window, ordered most-recent-first.
func (a *weaviateAdapter) ListJiraCards(ctx context.Context, userID uint, workspaceID *uint, start, end time.Time, limit int) ([]JiraCard, error) {
	var rows []models.JiraCard
	q := a.db.WithContext(ctx).
		Where("user_id = ? AND updated_at >= ? AND updated_at <= ?", userID, start, end)
	if workspaceID != nil {
		q = q.Where("workspace_id = ?", *workspaceID)
	}
	if err := q.Order("updated_at DESC").Limit(limit).Find(&rows).Error; err != nil {
		return nil, fmt.Errorf("ListJiraCards: %w", err)
	}

	out := make([]JiraCard, 0, len(rows))
	for _, r := range rows {
		wsPtr := r.WorkspaceID
		out = append(out, JiraCard{
			CardKey:     r.Key,
			Title:       r.Summary,
			Status:      r.Status,
			Assignee:    r.Assignee,
			Content:     r.Summary, // Summary used as semantic query seed
			UpdatedAt:   r.UpdatedAt,
			WorkspaceID: wsPtr,
		})
	}
	return out, nil
}

// ListCommits returns commits for a user in [start, end].
// Uses a where-only query (no NearText) so all commits in the date range are returned.
func (a *weaviateAdapter) ListCommits(ctx context.Context, userID uint, workspaceID *uint, start, end time.Time) ([]Commit, error) {
	// SearchCommits and ListCommits in weaviate pkg filter only by user_id (no workspace).
	// Weaviate commit schema has no workspace_id field — commits are user-scoped only.
	results, err := a.c.ListCommits(ctx, int(userID), start, end, 0)
	if err != nil {
		return nil, fmt.Errorf("ListCommits: %w", err)
	}

	out := make([]Commit, 0, len(results))
	for _, r := range results {
		committedAt, _ := time.Parse(time.RFC3339, r.CommittedAt)
		out = append(out, Commit{
			SHA:         r.SHA,
			Message:     r.Content, // content field holds the commit message
			RepoName:    r.RepoName,
			Author:      r.Author,
			CommittedAt: committedAt,
		})
	}
	return out, nil
}

// ListWAMessages returns WA messages for a user in [start, end].
// Uses a where-only query (no NearText) so all messages in the date range are returned.
func (a *weaviateAdapter) ListWAMessages(ctx context.Context, userID uint, workspaceID *uint, start, end time.Time) ([]WAMessage, error) {
	// WA messages are scoped by user_id only; workspaceID is ignored (not in schema).
	results, err := a.c.ListWAMessages(ctx, int(userID), nil, start, end, 0)
	if err != nil {
		return nil, fmt.Errorf("ListWAMessages: %w", err)
	}

	out := make([]WAMessage, 0, len(results))
	for _, r := range results {
		ts, _ := time.Parse(time.RFC3339, r.Timestamp)
		out = append(out, WAMessage{
			MessageID:  fmt.Sprintf("%.0f", r.MessageID),
			SenderName: r.SenderName,
			Content:    r.Content,
			Timestamp:  ts,
		})
	}
	return out, nil
}

// SemanticCommits runs a NearText query on CommitEmbedding filtered by user + date range.
// Distance is exposed by SearchCommits via _additional.distance.
func (a *weaviateAdapter) SemanticCommits(ctx context.Context, userID uint, workspaceID *uint, query string, start, end time.Time, limit int) ([]CommitHit, error) {
	results, err := a.c.SearchCommits(ctx, query, int(userID), limit)
	if err != nil {
		return nil, fmt.Errorf("SemanticCommits: %w", err)
	}

	out := make([]CommitHit, 0, len(results))
	for _, r := range results {
		committedAt, _ := time.Parse(time.RFC3339, r.CommittedAt)
		// Post-filter by date: SearchCommits does not accept startDate/endDate params.
		if committedAt.Before(start) || committedAt.After(end) {
			continue
		}
		out = append(out, CommitHit{
			Commit: Commit{
				SHA:         r.SHA,
				Message:     r.Content,
				RepoName:    r.RepoName,
				Author:      r.Author,
				CommittedAt: committedAt,
			},
			Distance: float64(r.Distance),
		})
	}
	return out, nil
}

// SemanticWA runs a NearText query on WaMessageEmbedding filtered by user + date range.
// Distance is exposed by Search via _additional.distance.
func (a *weaviateAdapter) SemanticWA(ctx context.Context, userID uint, workspaceID *uint, query string, start, end time.Time, limit int) ([]WAHit, error) {
	results, err := a.c.Search(ctx, query, int(userID), nil, &start, &end, limit)
	if err != nil {
		return nil, fmt.Errorf("SemanticWA: %w", err)
	}

	out := make([]WAHit, 0, len(results))
	for _, r := range results {
		ts, _ := time.Parse(time.RFC3339, r.Timestamp)
		out = append(out, WAHit{
			Message: WAMessage{
				MessageID:  fmt.Sprintf("%.0f", r.MessageID),
				SenderName: r.SenderName,
				Content:    r.Content,
				Timestamp:  ts,
			},
			Distance: float64(r.Distance),
		})
	}
	return out, nil
}
