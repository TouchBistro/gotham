package shipit

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"time"
)

// Client holds the configuration and HTTP client required to interact
// with the Shipit deployment engine API.
type Client struct {
	// baseURI is the base URL for all Shipit API requests (e.g. "https://shipit.example.com").
	baseURI string
	// apiPassword is the password used for Basic Auth on all API requests.
	// The username is always an empty string.
	apiPassword string
	// httpClient is the underlying HTTP client used to execute requests.
	httpClient *http.Client
}

// NewClient constructs a Client with the supplied base URI and API password.
// The returned client is ready to use immediately with no additional setup.
// apiPassword is used as the Basic Auth password; the username is always empty.
func NewClient(baseURI, apiPassword string) *Client {
	return &Client{
		baseURI:     baseURI,
		apiPassword: apiPassword,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
	}
}

// setAuth sets the Basic Auth header on req using an empty username and
// the client's configured API password.
func (c *Client) setAuth(req *http.Request) {
	req.SetBasicAuth("", c.apiPassword)
}

// linkNextSince parses a standard HTTP Link header value and returns the value
// of the `since` query parameter from the URL with rel="next".
// Returns an empty string if no rel=next link is present.
var reLinkNext = regexp.MustCompile(`<([^>]+)>;\s*rel="next"`)

func parseLinkNextSince(header string) string {
	m := reLinkNext.FindStringSubmatch(header)
	if m == nil {
		return ""
	}
	u, err := url.Parse(m[1])
	if err != nil {
		return ""
	}
	return u.Query().Get("since")
}

// ListAllStacks retrieves all stacks from the Shipit API, following pagination
// automatically. It returns the full slice of stacks across all pages.
func (c *Client) ListAllStacks() ([]Stack, error) {
	var all []Stack
	endpoint := fmt.Sprintf("%s/api/stacks?page_size=50", c.baseURI)

	for endpoint != "" {
		req, err := http.NewRequest(http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("shipit: creating list stacks request: %w", err)
		}
		c.setAuth(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("shipit: executing list stacks request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		resp.Body.Close()
		if err != nil {
			return nil, fmt.Errorf("shipit: reading list stacks response body: %w", err)
		}

		if resp.StatusCode < 200 || resp.StatusCode >= 300 {
			return nil, fmt.Errorf("shipit: list stacks returned status %d: %s", resp.StatusCode, string(body))
		}

		var page []Stack
		if err := json.Unmarshal(body, &page); err != nil {
			return nil, fmt.Errorf("shipit: decoding list stacks response: %w", err)
		}
		all = append(all, page...)

		// Determine next page URL from Link header.
		since := parseLinkNextSince(resp.Header.Get("Link"))
		if since == "" {
			break
		}
		endpoint = fmt.Sprintf("%s/api/stacks?page_size=50&since=%s", c.baseURI, since)
	}

	if all == nil {
		all = []Stack{}
	}
	return all, nil
}

// LockStack locks the stack identified by stackID (repo_owner/repo_name/environment)
// with the supplied reason. It sends a POST to {base_uri}/api/stacks/{stack_id}/lock
// with a JSON body containing the reason.
func (c *Client) LockStack(stackID, reason string) error {
	payload, err := json.Marshal(struct {
		Reason string `json:"reason"`
	}{Reason: reason})
	if err != nil {
		return fmt.Errorf("shipit: marshaling lock request body: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/stacks/%s/lock", c.baseURI, stackID)
	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("shipit: creating lock stack request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shipit: executing lock stack request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("shipit: reading lock stack response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("shipit: lock stack %q returned status %d: %s", stackID, resp.StatusCode, string(body))
	}
	return nil
}

// UnlockStack unlocks the stack identified by stackID (repo_owner/repo_name/environment).
// It sends a DELETE to {base_uri}/api/stacks/{stack_id}/lock with Basic Auth.
func (c *Client) UnlockStack(stackID string) error {
	endpoint := fmt.Sprintf("%s/api/stacks/%s/lock", c.baseURI, stackID)
	req, err := http.NewRequest(http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("shipit: creating unlock stack request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shipit: executing unlock stack request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	resp.Body.Close()
	if err != nil {
		return fmt.Errorf("shipit: reading unlock stack response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("shipit: unlock stack %q returned status %d: %s", stackID, resp.StatusCode, string(body))
	}
	return nil
}
