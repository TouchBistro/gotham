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

// CancelWorkflow cancels a running workflow by its ID using the CircleCI v2
// API. It returns nil on success or a descriptive error on failure.
func (c *Client) CancelWorkflow(ctx context.Context, workflowID string) error {
	url := fmt.Sprintf("%s/workflow/%s/cancel", c.v2BaseURL, workflowID)

	_, err := c.doRequest(ctx, http.MethodPost, url, nil)
	return err
}
