package weaviate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"time"

	"github.com/weaviate/weaviate-go-client/v4/weaviate/filters"
	"github.com/weaviate/weaviate-go-client/v4/weaviate/graphql"
	"github.com/weaviate/weaviate/entities/models"
)

const commitCollectionName = "CommitEmbedding"

func (c *Client) ensureCommitSchema(ctx context.Context) error {
	_, err := c.client.Schema().ClassGetter().WithClassName(commitCollectionName).Do(ctx)
	if err == nil {
		return nil
	}

	trueVal := true
	skip := map[string]interface{}{"text2vec-google": map[string]interface{}{"skip": true}}
	noSkip := map[string]interface{}{"text2vec-google": map[string]interface{}{"skip": false}}

	class := &models.Class{
		Class:      commitCollectionName,
		Vectorizer: "text2vec-google",
		ModuleConfig: map[string]interface{}{
			"text2vec-google": map[string]interface{}{
				"projectId":   "google",
				"apiEndpoint": "generativelanguage.googleapis.com",
				"modelId":     "gemini-embedding-001",
			},
		},
		Properties: []*models.Property{
			{Name: "commit_id", DataType: []string{"number"}, ModuleConfig: skip},
			{Name: "user_id", DataType: []string{"number"}, ModuleConfig: skip},
			{Name: "sha", DataType: []string{"text"}, ModuleConfig: skip},
			{Name: "content", DataType: []string{"text"}, IndexInverted: &trueVal, Tokenization: models.PropertyTokenizationWord, ModuleConfig: noSkip},
			{Name: "repo_name", DataType: []string{"text"}, ModuleConfig: skip},
			{Name: "author", DataType: []string{"text"}, ModuleConfig: skip},
			{Name: "committed_at", DataType: []string{"date"}, ModuleConfig: skip},
		},
	}

	return c.client.Schema().ClassCreator().WithClass(class).Do(ctx)
}

func commitUUID(commitID int) string {
	data := fmt.Sprintf("commit-%d", commitID)
	h := sha256.Sum256([]byte(data))
	h[6] = (h[6] & 0x0f) | 0x50
	h[8] = (h[8] & 0x3f) | 0x80
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		h[0:4], h[4:6], h[6:8], h[8:10], h[10:16])
}

// UpsertCommit embeds a git commit message.
func (c *Client) UpsertCommit(ctx context.Context, commitID, userID int, sha, message, repoName, author string, committedAt time.Time) error {
	if !c.available || message == "" {
		return nil
	}

	uuid := commitUUID(commitID)
	properties := map[string]interface{}{
		"commit_id":    float64(commitID),
		"user_id":      float64(userID),
		"sha":          sha,
		"content":      message,
		"repo_name":    repoName,
		"author":       author,
		"committed_at": committedAt.UTC().Format(time.RFC3339),
	}

	err := c.client.Data().Updater().
		WithClassName(commitCollectionName).
		WithID(uuid).
		WithProperties(properties).
		WithMerge().
		Do(ctx)

	if err != nil {
		_, createErr := c.client.Data().Creator().
			WithClassName(commitCollectionName).
			WithID(uuid).
			WithProperties(properties).
			Do(ctx)
		if createErr != nil {
			return fmt.Errorf("commit upsert failed: %w", createErr)
		}
	}
	return nil
}

// SearchCommits performs semantic search across git commits.
func (c *Client) SearchCommits(ctx context.Context, query string, userID int, limit int) ([]CommitSearchResult, error) {
	if !c.available {
		return nil, nil
	}
	if limit <= 0 {
		limit = 10
	}

	whereFilter := filters.Where().
		WithPath([]string{"user_id"}).
		WithOperator(filters.Equal).
		WithValueNumber(float64(userID))

	nearText := c.client.GraphQL().NearTextArgBuilder().WithConcepts([]string{query})

	fields := []graphql.Field{
		{Name: "commit_id"},
		{Name: "sha"},
		{Name: "content"},
		{Name: "repo_name"},
		{Name: "author"},
		{Name: "committed_at"},
		{Name: "_additional", Fields: []graphql.Field{{Name: "distance"}}},
	}

	result, err := c.client.GraphQL().Get().
		WithClassName(commitCollectionName).
		WithNearText(nearText).
		WithFields(fields...).
		WithWhere(whereFilter).
		WithLimit(limit).
		Do(ctx)

	if err != nil {
		return nil, fmt.Errorf("commit search failed: %w", err)
	}
	if len(result.Errors) > 0 {
		return nil, fmt.Errorf("commit search error: %v", result.Errors[0].Message)
	}

	return parseCommitResults(result.Data)
}

type CommitSearchResult struct {
	CommitID    float64 `json:"commit_id"`
	SHA         string  `json:"sha"`
	Content     string  `json:"content"`
	RepoName    string  `json:"repo_name"`
	Author      string  `json:"author"`
	CommittedAt string  `json:"committed_at"`
	Distance    float32 `json:"distance"`
}

func parseCommitResults(data map[string]models.JSONObject) ([]CommitSearchResult, error) {
	raw, err := marshalAndParse(data, commitCollectionName)
	if err != nil || raw == nil {
		return nil, err
	}

	var results []CommitSearchResult
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
		if additional, ok := obj["_additional"].(map[string]interface{}); ok {
			if d, ok := additional["distance"].(float64); ok {
				r.Distance = float32(d)
			}
		}
		results = append(results, r)
	}
	return results, nil
}
