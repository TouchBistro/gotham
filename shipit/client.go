package shipit

import (
	"net/http"
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
