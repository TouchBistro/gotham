package shipit

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"sync"
	"time"

	"golang.org/x/sync/errgroup"
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

// reValidStackID matches a stack ID of the form "owner/repo/environment" where
// each segment contains only alphanumeric characters, hyphens, underscores, or dots.
var reValidStackID = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

// validateStackID returns an error if stackID does not match the expected
// "owner/repo/environment" format.
func validateStackID(stackID string) error {
	if !reValidStackID.MatchString(stackID) {
		return fmt.Errorf("shipit: invalid stack ID %q: must be in the format owner/repo/environment", stackID)
	}
	return nil
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
func (c *Client) ListAllStacks(ctx context.Context) ([]Stack, error) {
	var all []Stack
	endpoint := fmt.Sprintf("%s/api/stacks?page_size=50", c.baseURI)

	for endpoint != "" {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
		if err != nil {
			return nil, fmt.Errorf("shipit: creating list stacks request: %w", err)
		}
		c.setAuth(req)

		resp, err := c.httpClient.Do(req)
		if err != nil {
			return nil, fmt.Errorf("shipit: executing list stacks request: %w", err)
		}

		body, err := io.ReadAll(resp.Body)
		if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
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
func (c *Client) LockStack(ctx context.Context, stackID, reason string) error {
	if err := validateStackID(stackID); err != nil {
		return err
	}

	payload, err := json.Marshal(struct {
		Reason string `json:"reason"`
	}{Reason: reason})
	if err != nil {
		return fmt.Errorf("shipit: marshaling lock request body: %w", err)
	}

	endpoint := fmt.Sprintf("%s/api/stacks/%s/lock", c.baseURI, stackID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
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
	if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("shipit: reading lock stack response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("shipit: lock stack %q returned status %d: %s", stackID, resp.StatusCode, string(body))
	}
	return nil
}

// LockAll locks every stack returned by ListAllStacks concurrently, passing reason
// to each LockStack call. Concurrency is capped at 10 goroutines to avoid
// overwhelming the API. All lock operations are attempted regardless of individual
// failures. The returned error, if non-nil, is a joined error containing every
// individual failure, so callers can inspect all stacks that failed to lock.
func (c *Client) LockAll(ctx context.Context, reason string) error {
	stacks, err := c.ListAllStacks(ctx)
	if err != nil {
		return fmt.Errorf("shipit: listing stacks for LockAll: %w", err)
	}

	var (
		mu   sync.Mutex
		errs []error
	)

	var g errgroup.Group
	g.SetLimit(10)

	for _, s := range stacks {
		stackID := s.StackID()
		g.Go(func() error {
			if err := c.LockStack(ctx, stackID, reason); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
			return nil
		})
	}

	g.Wait()
	return errors.Join(errs...)
}

// UnlockAll unlocks every stack returned by ListAllStacks concurrently.
// Concurrency is capped at 10 goroutines to avoid overwhelming the API.
// All unlock operations are attempted regardless of individual failures.
// The returned error, if non-nil, is a joined error containing every
// individual failure, so callers can inspect all stacks that failed to unlock.
func (c *Client) UnlockAll(ctx context.Context) error {
	stacks, err := c.ListAllStacks(ctx)
	if err != nil {
		return fmt.Errorf("shipit: listing stacks for UnlockAll: %w", err)
	}

	var (
		mu   sync.Mutex
		errs []error
	)

	var g errgroup.Group
	g.SetLimit(10)

	for _, s := range stacks {
		stackID := s.StackID()
		g.Go(func() error {
			if err := c.UnlockStack(ctx, stackID); err != nil {
				mu.Lock()
				errs = append(errs, err)
				mu.Unlock()
			}
			return nil
		})
	}

	g.Wait()
	return errors.Join(errs...)
}

// UnlockStack unlocks the stack identified by stackID (repo_owner/repo_name/environment).
// It sends a DELETE to {base_uri}/api/stacks/{stack_id}/lock with Basic Auth.
func (c *Client) UnlockStack(ctx context.Context, stackID string) error {
	if err := validateStackID(stackID); err != nil {
		return err
	}

	endpoint := fmt.Sprintf("%s/api/stacks/%s/lock", c.baseURI, stackID)
	req, err := http.NewRequestWithContext(ctx, http.MethodDelete, endpoint, nil)
	if err != nil {
		return fmt.Errorf("shipit: creating unlock stack request: %w", err)
	}
	c.setAuth(req)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("shipit: executing unlock stack request: %w", err)
	}
	body, err := io.ReadAll(resp.Body)
	if closeErr := resp.Body.Close(); closeErr != nil && err == nil {
		err = closeErr
	}
	if err != nil {
		return fmt.Errorf("shipit: reading unlock stack response body: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("shipit: unlock stack %q returned status %d: %s", stackID, resp.StatusCode, string(body))
	}
	return nil
}
