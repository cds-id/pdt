package weaviate

import (
	"encoding/json"
	"fmt"

	"github.com/weaviate/weaviate/entities/models"
)

// marshalAndParse is a shared helper to parse GraphQL response data for a given class.
func marshalAndParse(data map[string]models.JSONObject, className string) ([]map[string]interface{}, error) {
	b, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("marshal graphql data: %w", err)
	}

	var parsed struct {
		Get map[string][]map[string]interface{} `json:"Get"`
	}
	if err := json.Unmarshal(b, &parsed); err != nil {
		return nil, fmt.Errorf("unmarshal graphql data: %w", err)
	}

	objects, ok := parsed.Get[className]
	if !ok {
		return nil, nil
	}
	return objects, nil
}
