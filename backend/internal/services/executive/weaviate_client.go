package executive

import (
	"context"
	"time"
)

// WeaviateClient is the narrow interface the correlator depends on. The
// production implementation wraps backend/internal/services/weaviate.
type WeaviateClient interface {
	// ListJiraCards returns Jira cards for a user/workspace whose primary
	// timestamp falls in [start, end]. Empty query, ordered by most recent.
	ListJiraCards(ctx context.Context, userID uint, workspaceID *uint, start, end time.Time, limit int) ([]JiraCard, error)

	// ListCommits returns commits for a user in [start, end]. Empty query.
	ListCommits(ctx context.Context, userID uint, workspaceID *uint, start, end time.Time) ([]Commit, error)

	// ListWAMessages returns WA messages for a user in [start, end]. Empty query.
	ListWAMessages(ctx context.Context, userID uint, workspaceID *uint, start, end time.Time) ([]WAMessage, error)

	// SemanticCommits runs a near-text query on CommitEmbedding filtered by user + date.
	// Returns results sorted by distance ascending. The float is the distance.
	SemanticCommits(ctx context.Context, userID uint, workspaceID *uint, query string, start, end time.Time, limit int) ([]CommitHit, error)

	// SemanticWA runs a near-text query on WaMessageEmbedding filtered by user + date.
	SemanticWA(ctx context.Context, userID uint, workspaceID *uint, query string, start, end time.Time, limit int) ([]WAHit, error)
}

type CommitHit struct {
	Commit   Commit
	Distance float64
}

type WAHit struct {
	Message  WAMessage
	Distance float64
}
