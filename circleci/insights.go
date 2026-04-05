package circleci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// GetProjectInsights retrieves workflow insights and metrics for the given
// GitHub owner and repository from the CircleCI v2 API.
func (c *Client) GetProjectInsights(ctx context.Context, owner, repo string) (*ProjectInsights, error) {
	url := fmt.Sprintf("%s/insights/gh/%s/%s/workflows", c.v2BaseURL, owner, repo)

	body, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var insights ProjectInsights
	if err := json.Unmarshal(body, &insights); err != nil {
		return nil, fmt.Errorf("circleci: decoding GetProjectInsights response: %w", err)
	}

	return &insights, nil
}
