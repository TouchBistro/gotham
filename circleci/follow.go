package circleci

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
)

// FollowProject follows the given GitHub project on CircleCI for the
// authenticated user. It calls the v1.1 follow endpoint.
func (c *Client) FollowProject(ctx context.Context, owner, repo string) (*FollowProjectResponse, error) {
	url := fmt.Sprintf("%s/project/github/%s/%s/follow", c.v1BaseURL, owner, repo)

	body, err := c.doRequest(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	var resp FollowProjectResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("circleci: decoding FollowProject response: %w", err)
	}

	return &resp, nil
}

// UnfollowProject unfollows the given GitHub project on CircleCI for the
// authenticated user. It calls the v1.1 unfollow endpoint.
func (c *Client) UnfollowProject(ctx context.Context, owner, repo string) (*UnfollowProjectResponse, error) {
	url := fmt.Sprintf("%s/project/github/%s/%s/unfollow", c.v1BaseURL, owner, repo)

	body, err := c.doRequest(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, err
	}

	var resp UnfollowProjectResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("circleci: decoding UnfollowProject response: %w", err)
	}

	return &resp, nil
}
