package circleci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// ListPipelines retrieves pipelines for the given GitHub owner and repository
// from the CircleCI v2 API. It returns the first page of results.
func (c *Client) ListPipelines(ctx context.Context, owner, repo string) (*ListPipelinesResponse, error) {
	rawURL := fmt.Sprintf("%s/project/gh/%s/%s/pipeline", c.v2BaseURL, url.PathEscape(owner), url.PathEscape(repo))

	body, err := c.doRequest(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	var resp ListPipelinesResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("circleci: decoding ListPipelines response: %w", err)
	}

	return &resp, nil
}
