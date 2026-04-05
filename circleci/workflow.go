package circleci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// ListWorkflows retrieves workflows for the given pipeline ID from the
// CircleCI v2 API. It returns the first page of results.
func (c *Client) ListWorkflows(ctx context.Context, pipelineID string) (*ListWorkflowsResponse, error) {
	url := fmt.Sprintf("%s/pipeline/%s/workflow", c.v2BaseURL, pipelineID)

	body, err := c.doRequest(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	var resp ListWorkflowsResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("circleci: decoding ListWorkflows response: %w", err)
	}

	return &resp, nil
}

