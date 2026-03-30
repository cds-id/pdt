package weaviate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"

	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

const jiraCollectionName = "JiraCardEmbedding"

func (c *Client) ensureJiraSchema(ctx context.Context) error {
	_, err := c.client.Schema().ClassGetter().WithClassName(jiraCollectionName).Do(ctx)
	if err == nil {
		return nil
	}

	trueVal := true
	skip := map[string]interface{}{"text2vec-google": map[string]interface{}{"skip": true}}
	noSkip := map[string]interface{}{"text2vec-google": map[string]interface{}{"skip": false}}

	class := &models.Class{
		Class:      jiraCollectionName,
		Vectorizer: "text2vec-google",
		ModuleConfig: map[string]interface{}{
			"text2vec-google": map[string]interface{}{
				"projectId":   "google",
				"apiEndpoint": "generativelanguage.googleapis.com",
				"modelId":     "gemini-embedding-001",
			},
		},
		Properties: []*models.Property{
			{Name: "card_key", DataType: []string{"text"}, ModuleConfig: skip},
			{Name: "user_id", DataType: []string{"number"}, ModuleConfig: skip},
			{Name: "workspace_id", DataType: []string{"number"}, ModuleConfig: skip},
			{Name: "content", DataType: []string{"text"}, IndexInverted: &trueVal, Tokenization: models.PropertyTokenizationWord, ModuleConfig: noSkip},
			{Name: "status", DataType: []string{"text"}, ModuleConfig: skip},
			{Name: "assignee", DataType: []string{"text"}, ModuleConfig: skip},
			{Name: "source_type", DataType: []string{"text"}, ModuleConfig: skip}, // "card" or "comment"
		},
	}

	return c.client.Schema().ClassCreator().WithClass(class).Do(ctx)
}

func jiraUUID(prefix string, id string) string {
	data := fmt.Sprintf("jira-%s-%s", prefix, id)
	h := sha256.Sum256([]byte(data))
	h[6] = (h[6] & 0x0f) | 0x50
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

// UpsertJiraCard embeds a Jira card's summary + description.
func (c *Client) UpsertJiraCard(ctx context.Context, cardKey string, userID, workspaceID int, content, status, assignee string) error {
	if !c.available || content == "" {
		return nil
	}

	uuid := jiraUUID("card", cardKey)
	properties := map[string]interface{}{
		"card_key":     cardKey,
		"user_id":      float64(userID),
		"workspace_id": float64(workspaceID),
		"content":      content,
		"status":       status,
		"assignee":     assignee,
		"source_type":  "card",
	}

	err := c.client.Data().Updater().
		WithClassName(jiraCollectionName).
		WithID(uuid).
		WithProperties(properties).
		WithMerge().
		Do(ctx)

	if err != nil {
		_, createErr := c.client.Data().Creator().
			WithClassName(jiraCollectionName).
			WithID(uuid).
			WithProperties(properties).
			Do(ctx)
		if createErr != nil {
			return fmt.Errorf("jira card upsert failed: %w", createErr)
		}
	}
	return nil
}

// UpsertJiraComment embeds a Jira comment.
func (c *Client) UpsertJiraComment(ctx context.Context, commentID string, cardKey string, userID, workspaceID int, content, author string) error {
	if !c.available || content == "" {
		return nil
	}

	uuid := jiraUUID("comment", commentID)
	properties := map[string]interface{}{
		"card_key":     cardKey,
		"user_id":      float64(userID),
		"workspace_id": float64(workspaceID),
		"content":      fmt.Sprintf("[%s] %s", author, content),
		"status":       "",
		"assignee":     author,
		"source_type":  "comment",
	}

	err := c.client.Data().Updater().
		WithClassName(jiraCollectionName).
		WithID(uuid).
		WithProperties(properties).
		WithMerge().
		Do(ctx)

	if err != nil {
		_, createErr := c.client.Data().Creator().
			WithClassName(jiraCollectionName).
			WithID(uuid).
			WithProperties(properties).
			Do(ctx)
		if createErr != nil {
			return fmt.Errorf("jira comment upsert failed: %w", createErr)
		}
	}
	return nil
}

// SearchJira performs semantic search across Jira cards and comments.
func (c *Client) SearchJira(ctx context.Context, query string, userID int, workspaceID *int, limit int) ([]JiraSearchResult, error) {
	if !c.available {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
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

	nearText := c.client.GraphQL().NearTextArgBuilder().WithConcepts([]string{query})

	fields := []graphql.Field{
		{Name: "card_key"},
		{Name: "content"},
		{Name: "status"},
		{Name: "assignee"},
		{Name: "source_type"},
		{Name: "_additional", Fields: []graphql.Field{{Name: "distance"}}},
	}

	builder := c.client.GraphQL().Get().
		WithClassName(jiraCollectionName).
		WithNearText(nearText).
		WithFields(fields...).
		WithWhere(whereFilter).
		WithLimit(limit)

	result, err := builder.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("jira search failed: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("jira search error: %v", result.Errors[0].Message)
	}

	return parseJiraResults(result.Data)
}

type JiraSearchResult struct {
	CardKey    string  `json:"card_key"`
	Content    string  `json:"content"`
	Status     string  `json:"status"`
	Assignee   string  `json:"assignee"`
	SourceType string  `json:"source_type"`
	Distance   float32 `json:"distance"`
}

func parseJiraResults(data map[string]models.JSONObject) ([]JiraSearchResult, error) {
	raw, err := marshalAndParse(data, jiraCollectionName)
	if err != nil || raw == nil {
		return nil, err
	}

	var results []JiraSearchResult
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
		if additional, ok := obj["_additional"].(map[string]interface{}); ok {
			if d, ok := additional["distance"].(float64); ok {
				r.Distance = float32(d)
			}
		}
		results = append(results, r)
	}
	return results, nil
}

func init() {
	log.SetFlags(log.LstdFlags)
}
