package circleci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
)

// GetProject retrieves project information for the given GitHub owner and
// repository from the CircleCI v2 API.
func (c *Client) GetProject(ctx context.Context, owner, repo string) (*Project, error) {
	rawURL := fmt.Sprintf("%s/project/gh/%s/%s", c.v2BaseURL, url.PathEscape(owner), url.PathEscape(repo))

	body, err := c.doRequest(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}

	var project Project
	if err := json.Unmarshal(body, &project); err != nil {
		return nil, fmt.Errorf("circleci: decoding GetProject response: %w", err)
	}

	return &project, nil
}
