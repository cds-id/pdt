package weaviate

import (
	"context"
	"fmt"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

// ListJira returns all JiraCardEmbedding objects for a user (and optional workspace)
// without a nearText semantic query, using only where filters.
// Note: JiraCardEmbedding has no date field, so date-range filtering is not possible
// at the Weaviate level; callers must post-filter if needed.
func (c *Client) ListJira(ctx context.Context, userID int, workspaceID *int, limit int) ([]JiraSearchResult, error) {
	if !c.available {
		return nil, nil
	}
	if limit <= 0 {
		limit = 500
	}

	operands := []*filters.WhereBuilder{
		filters.Where().
			WithPath([]string{"user_id"}).
			WithOperator(filters.Equal).
			WithValueNumber(float64(userID)),
	}
	if workspaceID != nil {
		operands = append(operands, filters.Where().
			WithPath([]string{"workspace_id"}).
			WithOperator(filters.Equal).
			WithValueNumber(float64(*workspaceID)),
		)
	}

	var whereFilter *filters.WhereBuilder
	if len(operands) == 1 {
		whereFilter = operands[0]
	} else {
		whereFilter = filters.Where().WithOperator(filters.And).WithOperands(operands)
	}

	fields := []graphql.Field{
		{Name: "card_key"},
		{Name: "content"},
		{Name: "status"},
		{Name: "assignee"},
		{Name: "source_type"},
	}

	result, err := c.client.GraphQL().Get().
		WithClassName(jiraCollectionName).
		WithFields(fields...).
		WithWhere(whereFilter).
		WithLimit(limit).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("jira list failed: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("jira list error: %v", result.Errors[0].Message)
	}

	raw, err := marshalAndParse(result.Data, jiraCollectionName)
	if err != nil || raw == nil {
		return nil, err
	}

	var out []JiraSearchResult
	for _, obj := range raw {
		r := JiraSearchResult{}
		if v, ok := obj["card_key"].(string); ok {
			r.CardKey = v
		}
		if v, ok := obj["content"].(string); ok {
			r.Content = v
		}
		if v, ok := obj["status"].(string); ok {
			r.Status = v
		}
		if v, ok := obj["assignee"].(string); ok {
			r.Assignee = v
		}
		if v, ok := obj["source_type"].(string); ok {
			r.SourceType = v
		}
		out = append(out, r)
	}
	return out, nil
}

// ListCommits returns all CommitEmbedding objects for a user within [start, end]
// without a nearText semantic query, using only where filters.
func (c *Client) ListCommits(ctx context.Context, userID int, startDate, endDate time.Time, limit int) ([]CommitSearchResult, error) {
	if !c.available {
		return nil, nil
	}
	if limit <= 0 {
		limit = 500
	}

	operands := []*filters.WhereBuilder{
		filters.Where().
			WithPath([]string{"user_id"}).
			WithOperator(filters.Equal).
			WithValueNumber(float64(userID)),
		filters.Where().
			WithPath([]string{"committed_at"}).
			WithOperator(filters.GreaterThanEqual).
			WithValueDate(startDate),
		filters.Where().
			WithPath([]string{"committed_at"}).
			WithOperator(filters.LessThanEqual).
			WithValueDate(endDate),
	}

	whereFilter := filters.Where().WithOperator(filters.And).WithOperands(operands)

	fields := []graphql.Field{
		{Name: "commit_id"},
		{Name: "sha"},
		{Name: "content"},
		{Name: "repo_name"},
		{Name: "author"},
		{Name: "committed_at"},
	}

	result, err := c.client.GraphQL().Get().
		WithClassName(commitCollectionName).
		WithFields(fields...).
		WithWhere(whereFilter).
		WithLimit(limit).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("commit list failed: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("commit list error: %v", result.Errors[0].Message)
	}

	raw, err := marshalAndParse(result.Data, commitCollectionName)
	if err != nil || raw == nil {
		return nil, err
	}

	var out []CommitSearchResult
	for _, obj := range raw {
		r := CommitSearchResult{}
		if v, ok := obj["commit_id"].(float64); ok {
			r.CommitID = v
		}
		if v, ok := obj["sha"].(string); ok {
			r.SHA = v
		}
		if v, ok := obj["content"].(string); ok {
			r.Content = v
		}
		if v, ok := obj["repo_name"].(string); ok {
			r.RepoName = v
		}
		if v, ok := obj["author"].(string); ok {
			r.Author = v
		}
		if v, ok := obj["committed_at"].(string); ok {
			r.CommittedAt = v
		}
		out = append(out, r)
	}
	return out, nil
}

// ListWAMessages returns all WaMessageEmbedding objects for a user within [start, end]
// without a nearText semantic query, using only where filters.
func (c *Client) ListWAMessages(ctx context.Context, userID int, listenerID *int, startDate, endDate time.Time, limit int) ([]SearchResult, error) {
	if !c.available {
		return nil, nil
	}
	if limit <= 0 {
		limit = 500
	}

	whereFilter := buildWhereFilter(userID, listenerID, &startDate, &endDate)

	fields := []graphql.Field{
		{Name: "message_id"},
		{Name: "listener_id"},
		{Name: "content"},
		{Name: "sender_name"},
		{Name: "timestamp"},
	}

	result, err := c.client.GraphQL().Get().
		WithClassName(collectionName).
		WithFields(fields...).
		WithWhere(whereFilter).
		WithLimit(limit).
		Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("wa message list failed: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("wa message list error: %v", result.Errors[0].Message)
	}

	return parseWAListResults(result.Data)
}

func parseWAListResults(data map[string]models.JSONObject) ([]SearchResult, error) {
	objects, err := marshalAndParse(data, collectionName)
	if err != nil || objects == nil {
		return nil, err
	}

	results := make([]SearchResult, 0, len(objects))
	for _, obj := range objects {
		sr := SearchResult{}
		if v, ok := obj["message_id"].(float64); ok {
			sr.MessageID = v
		}
		if v, ok := obj["listener_id"].(float64); ok {
			sr.ListenerID = v
		}
		if v, ok := obj["content"].(string); ok {
			sr.Content = v
		}
		if v, ok := obj["sender_name"].(string); ok {
			sr.SenderName = v
		}
		if v, ok := obj["timestamp"].(string); ok {
			sr.Timestamp = v
		}
		results = append(results, sr)
	}
	return results, nil
}
