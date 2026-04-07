package circleci

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"time"
)

const (
	defaultV1BaseURL = "https://circleci.com/api/v1.1"
	defaultV2BaseURL = "https://circleci.com/api/v2"
	defaultTimeout   = 30 * time.Second
)

// Client is an HTTP client for the CircleCI API. It holds configuration for
// both v1.1 and v2 API base URLs, an API token, and an underlying *http.Client.
// Use NewClientBuilder to construct a Client.
type Client struct {
	v1BaseURL  string
	v2BaseURL  string
	token      string
	httpClient *http.Client
}

// ClientBuilder constructs a Client using a chainable builder pattern.
type ClientBuilder struct {
	token      string
	httpClient *http.Client
	v1BaseURL  string
	v2BaseURL  string
}

// NewClientBuilder returns a new ClientBuilder with default settings.
func NewClientBuilder() *ClientBuilder {
	return &ClientBuilder{}
}

// WithToken sets the CircleCI API token used for authentication.
func (b *ClientBuilder) WithToken(token string) *ClientBuilder {
	b.token = token
	return b
}

// WithHTTPClient sets a custom *http.Client for the CircleCI client.
func (b *ClientBuilder) WithHTTPClient(c *http.Client) *ClientBuilder {
	b.httpClient = c
	return b
}

// WithBaseURLs overrides the default CircleCI v1.1 and v2 base URLs.
// This is useful for testing with httptest servers.
func (b *ClientBuilder) WithBaseURLs(v1, v2 string) *ClientBuilder {
	b.v1BaseURL = v1
	b.v2BaseURL = v2
	return b
}

// Build creates a Client from the builder configuration. Fields not explicitly
// set are populated with sensible defaults: CircleCI cloud URLs and a 30-second
// timeout HTTP client. It returns an error if no API token was provided.
func (b *ClientBuilder) Build() (*Client, error) {
	if b.token == "" {
		return nil, fmt.Errorf("circleci: API token must not be empty")
	}

	c := &Client{
		token:      b.token,
		v1BaseURL:  b.v1BaseURL,
		v2BaseURL:  b.v2BaseURL,
		httpClient: b.httpClient,
	}

	if c.v1BaseURL == "" {
		c.v1BaseURL = defaultV1BaseURL
	}
	if c.v2BaseURL == "" {
		c.v2BaseURL = defaultV2BaseURL
	}
	if c.httpClient == nil {
		c.httpClient = &http.Client{Timeout: defaultTimeout}
	}

	return c, nil
}

// doRequest executes an HTTP request with the given method, URL, and optional
// body. It sets the Circle-Token and Accept headers, reads the response body,
// and returns the raw bytes. On non-2xx responses it returns an error containing
// the status code and a body excerpt.
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, fmt.Errorf("circleci: creating request: %w", err)
	}

	req.Header.Set("Circle-Token", c.token)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("circleci: executing request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("circleci: reading response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		excerpt := string(respBody)
		if len(excerpt) > 200 {
			excerpt = excerpt[:200]
		}
		return nil, fmt.Errorf("circleci: request %s %s returned status %d: %s", method, url, resp.StatusCode, excerpt)
	}

	return respBody, nil
}
