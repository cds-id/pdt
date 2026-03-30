package weaviate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"log"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

const (
	collectionName = "WaMessageEmbedding"
)

// Client wraps the Weaviate client with availability tracking.
type Client struct {
	client    *weaviate.Client
	apiKey    string
	available bool
}

// SearchResult holds the result of a semantic search.
type SearchResult struct {
	MessageID  float64
	ListenerID float64
	Content    string
	SenderName string
	Timestamp  string
	Distance   float32
}

// NewClient creates a new Weaviate client service.
// If Weaviate is unreachable at init, available is set to false and no crash occurs.
func NewClient(url, geminiAPIKey string) *Client {
	c := &Client{
		apiKey:    geminiAPIKey,
		available: false,
	}

	cfg := weaviate.Config{
		Host:   url,
		Scheme: "http",
		Headers: map[string]string{
			"X-Google-Api-Key": geminiAPIKey,
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	weaviateClient, err := weaviate.NewClient(cfg)
	if err != nil {
		log.Printf("[weaviate] failed to create client: %v", err)
		return c
	}

	// Verify connectivity
	_, err = weaviateClient.Misc().MetaGetter().Do(ctx)
	if err != nil {
		log.Printf("[weaviate] service unavailable at %s: %v", url, err)
		return c
	}

	c.client = weaviateClient
	c.available = true

	if err := c.ensureSchema(context.Background()); err != nil {
		log.Printf("[weaviate] WA message schema setup failed: %v", err)
		c.available = false
	}
	if err := c.ensureJiraSchema(context.Background()); err != nil {
		log.Printf("[weaviate] Jira schema setup failed: %v", err)
	}
	if err := c.ensureCommitSchema(context.Background()); err != nil {
		log.Printf("[weaviate] Commit schema setup failed: %v", err)
	}

	return c
}

// IsAvailable returns whether the Weaviate service is reachable and ready.
func (c *Client) IsAvailable() bool {
	return c.available
}

// ensureSchema creates the WaMessageEmbedding collection if it does not already exist.
func (c *Client) ensureSchema(ctx context.Context) error {
	_, err := c.client.Schema().ClassGetter().WithClassName(collectionName).Do(ctx)
	if err == nil {
		// Class already exists
		return nil
	}

	trueVal := true
	falseVal := false

	class := &models.Class{
		Class:      collectionName,
		Vectorizer: "text2vec-google",
		ModuleConfig: map[string]interface{}{
			"text2vec-google": map[string]interface{}{
				"projectId":   "google",
				"apiEndpoint": "generativelanguage.googleapis.com",
				"modelId":     "gemini-embedding-001",
			},
		},
		Properties: []*models.Property{
			{
				Name:     "message_id",
				DataType: []string{"number"},
				ModuleConfig: map[string]interface{}{
					"text2vec-google": map[string]interface{}{
						"skip": true,
					},
				},
			},
			{
				Name:     "listener_id",
				DataType: []string{"number"},
				ModuleConfig: map[string]interface{}{
					"text2vec-google": map[string]interface{}{
						"skip": true,
					},
				},
			},
			{
				Name:     "user_id",
				DataType: []string{"number"},
				ModuleConfig: map[string]interface{}{
					"text2vec-google": map[string]interface{}{
						"skip": true,
					},
				},
			},
			{
				Name:          "content",
				DataType:      []string{"text"},
				IndexInverted: &trueVal,
				Tokenization:  models.PropertyTokenizationWord,
				ModuleConfig: map[string]interface{}{
					"text2vec-google": map[string]interface{}{
						"skip": false,
					},
				},
			},
			{
				Name:          "sender_name",
				DataType:      []string{"text"},
				IndexInverted: &falseVal,
				// Skip vectorization for sender_name
				ModuleConfig: map[string]interface{}{
					"text2vec-google": map[string]interface{}{
						"skip": true,
					},
				},
			},
			{
				Name:     "timestamp",
				DataType: []string{"date"},
				ModuleConfig: map[string]interface{}{
					"text2vec-google": map[string]interface{}{
						"skip": true,
					},
				},
			},
		},
	}

	return c.client.Schema().ClassCreator().WithClass(class).Do(ctx)
}

// deterministicUUID generates a consistent UUID from a message ID using SHA-256.
// This ensures idempotent upserts using the same UUID for the same message.
func deterministicUUID(messageID int) string {
	data := fmt.Sprintf("wa-message-%d", messageID)
	h := sha256.Sum256([]byte(data))
	// Set version 5
	h[6] = (h[6] & 0x0f) | 0x50
	// Set variant bits
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

// Upsert inserts or updates a message embedding in Weaviate using a deterministic UUID.
func (c *Client) Upsert(ctx context.Context, messageID, listenerID, userID int, content, senderName string, timestamp time.Time) error {
	if !c.available {
		return nil
	}

	uuid := deterministicUUID(messageID)

	properties := map[string]interface{}{
		"message_id":  float64(messageID),
		"listener_id": float64(listenerID),
		"user_id":     float64(userID),
		"content":     content,
		"sender_name": senderName,
		"timestamp":   timestamp.UTC().Format(time.RFC3339),
	}

	// Try merge-update first; if object doesn't exist, create it.
	err := c.client.Data().Updater().
		WithClassName(collectionName).
		WithID(uuid).
		WithProperties(properties).
		WithMerge().
		Do(ctx)

	if err != nil {
		_, createErr := c.client.Data().Creator().
			WithClassName(collectionName).
			WithID(uuid).
			WithProperties(properties).
			Do(ctx)
		if createErr != nil {
			return fmt.Errorf("weaviate upsert failed (create): %w", createErr)
		}
	}

	return nil
}

// Search performs a semantic nearText search with optional filters.
func (c *Client) Search(ctx context.Context, query string, userID int, listenerID *int, startDate, endDate *time.Time, limit int) ([]SearchResult, error) {
	if !c.available {
		return nil, nil
	}

	if limit <= 0 {
		limit = 10
	}

	whereFilter := buildWhereFilter(userID, listenerID, startDate, endDate)

	nearText := c.client.GraphQL().NearTextArgBuilder().
		WithConcepts([]string{query})

	fields := []graphql.Field{
		{Name: "message_id"},
		{Name: "listener_id"},
		{Name: "content"},
		{Name: "sender_name"},
		{Name: "timestamp"},
		{
			Name: "_additional",
			Fields: []graphql.Field{
				{Name: "distance"},
			},
		},
	}

	builder := c.client.GraphQL().Get().
		WithClassName(collectionName).
		WithNearText(nearText).
		WithFields(fields...).
		WithLimit(limit)

	if whereFilter != nil {
		builder = builder.WithWhere(whereFilter)
	}

	result, err := builder.Do(ctx)
	if err != nil {
		return nil, fmt.Errorf("weaviate search failed: %w", err)
	}

	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("weaviate graphql error: %v", result.Errors[0].Message)
	}

	return parseGraphQLResponse(result.Data)
}

// buildWhereFilter constructs a Weaviate where filter combining user_id (required)
// with optional listener_id and date range filters.
func buildWhereFilter(userID int, listenerID *int, startDate, endDate *time.Time) *filters.WhereBuilder {
	operands := []*filters.WhereBuilder{
		filters.Where().
			WithPath([]string{"user_id"}).
			WithOperator(filters.Equal).
			WithValueNumber(float64(userID)),
	}

	if listenerID != nil {
		operands = append(operands, filters.Where().
			WithPath([]string{"listener_id"}).
			WithOperator(filters.Equal).
			WithValueNumber(float64(*listenerID)),
		)
	}

	if startDate != nil {
		operands = append(operands, filters.Where().
			WithPath([]string{"timestamp"}).
			WithOperator(filters.GreaterThanEqual).
			WithValueDate(*startDate),
		)
	}

	if endDate != nil {
		operands = append(operands, filters.Where().
			WithPath([]string{"timestamp"}).
			WithOperator(filters.LessThanEqual).
			WithValueDate(*endDate),
		)
	}

	if len(operands) == 1 {
		return operands[0]
	}

	return filters.Where().
		WithOperator(filters.And).
		WithOperands(operands)
}

// parseGraphQLResponse parses the raw GraphQL data map into SearchResult slice.
func parseGraphQLResponse(data map[string]models.JSONObject) ([]SearchResult, error) {
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
		if additional, ok := obj["_additional"].(map[string]interface{}); ok {
			if d, ok := additional["distance"].(float64); ok {
				sr.Distance = float32(d)
			}
		}

		results = append(results, sr)
	}

	return results, nil
}
