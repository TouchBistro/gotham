package auth

import (
	"context"
	"net/http/httptest"
	"testing"

	"github.com/TouchBistro/gotham/ds"
)

func TestPathMatches(t *testing.T) {
	tests := []struct {
		name     string
		rulePath string
		reqPath  string
		want     bool
	}{
		// Global sentinel: bare "*" matches everything, including the empty
		// path — proving it resolves through the sentinel branch and never
		// reaches the prefix branch (an empty prefix is guarded against).
		{"global sentinel matches any path", "*", "/anything", true},
		{"global sentinel matches root", "*", "/", true},
		{"global sentinel matches empty path", "*", "", true},

		// Exact case-insensitive equality.
		{"exact match", "/cr7/reviews", "/cr7/reviews", true},
		{"exact match rejects trailing slash", "/cr7/reviews", "/cr7/reviews/", false},
		{"exact match is case-insensitive (rule upper)", "/CR7/Reviews", "/cr7/reviews", true},
		{"exact match is case-insensitive (request upper)", "/cr7/reviews", "/CR7/Reviews", true},

		// Trailing-"*" prefix matching.
		{"prefix matches bare prefix with trailing slash", "/cr7/reviews/*", "/cr7/reviews/", true},
		{"prefix matches sub-path", "/cr7/reviews/*", "/cr7/reviews/abc", true},
		{"prefix matches nested sub-path", "/cr7/reviews/*", "/cr7/reviews/6f1c9e2a-0000-0000-0000-000000000000/content", true},

		// Prefix negatives.
		{"prefix rejects path without trailing slash", "/cr7/reviews/*", "/cr7/reviews", false},
		{"prefix rejects shorter sibling", "/cr7/reviews/*", "/cr7/review", false},
		{"prefix rejects ancestor", "/cr7/reviews/*", "/cr7", false},
		{"prefix rejects mid-path occurrence", "/cr7/reviews/*", "/other/cr7/reviews/x", false},

		// Bare-prefix edge: request path exactly equal to the prefix.
		{"bare prefix matches exact prefix", "/x*", "/x", true},
		{"bare prefix matches extension", "/x*", "/xyz", true},
		{"bare prefix rejects different path", "/x*", "/y", false},
		{"bare prefix rejects empty path", "/x*", "", false},

		// Prefix case-insensitivity.
		{"prefix match is case-insensitive", "/CR7/Reviews/*", "/cr7/reviews/abc", true},

		// Empty-prefix guard: request path shorter than prefix never matches.
		{"request shorter than prefix", "/cr7/reviews/long-prefix/*", "/cr7", false},

		// A "*" anywhere other than the trailing position is not a wildcard;
		// only exact-equality semantics apply.
		{"mid-string star is not a wildcard", "/a*b", "/aXb", false},
		{"mid-string star matches only literally", "/a*b", "/a*b", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := pathMatches(tt.rulePath, tt.reqPath); got != tt.want {
				t.Errorf("pathMatches(%q, %q) = %v; want %v", tt.rulePath, tt.reqPath, got, tt.want)
			}
		})
	}
}

func TestPolicies_Match(t *testing.T) {
	prefixRule := Item{
		Name:       "reviews-prefix",
		HttpMethod: "GET",
		HttpPath:   "/cr7/reviews/*",
		Effect:     EffectAllow,
		Subjects:   ds.From(Everyone),
	}
	adminPrefixRule := Item{
		Name:       "admin-prefix",
		HttpMethod: "GET",
		HttpPath:   "/cr7/reviews/*",
		Effect:     EffectAllow,
		Subjects:   ds.From("admin"),
	}
	globalRule := Item{
		Name:       "global",
		HttpMethod: AllMethods,
		HttpPath:   AllPaths,
		Effect:     EffectAllow,
		Subjects:   ds.From(Everyone),
	}
	exactRule := Item{
		Name:       "exact-mixed-case",
		HttpMethod: "GET",
		HttpPath:   "/CR7/Reviews",
		Effect:     EffectAllow,
		Subjects:   ds.From(Everyone),
	}

	tests := []struct {
		name     string
		policies Policies
		roles    []string
		method   string
		path     string
		wantName string // empty means no match expected
	}{
		{
			name:     "prefix rule matches nested sub-path",
			policies: Policies{prefixRule},
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/cr7/reviews/6f1c9e2a-0000-0000-0000-000000000000/content",
			wantName: "reviews-prefix",
		},
		{
			name:     "prefix rule does not match shorter sibling path",
			policies: Policies{prefixRule},
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/cr7/review",
			wantName: "",
		},
		{
			name:     "method still gates on a matching prefix",
			policies: Policies{prefixRule},
			roles:    []string{"viewer"},
			method:   "POST",
			path:     "/cr7/reviews/abc",
			wantName: "",
		},
		{
			name:     "subject still gates on a matching prefix",
			policies: Policies{adminPrefixRule},
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/cr7/reviews/abc",
			wantName: "",
		},
		{
			name:     "global rule matches any method and path",
			policies: Policies{globalRule},
			roles:    []string{"viewer"},
			method:   "DELETE",
			path:     "/anything/at/all",
			wantName: "global",
		},
		{
			name:     "exact-path rule matches case-insensitively",
			policies: Policies{exactRule},
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/cr7/reviews",
			wantName: "exact-mixed-case",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &BasicPrincipal{Id: "test-user", RoleSet: ds.From(tt.roles...)}
			req := httptest.NewRequest(tt.method, tt.path, nil)

			item, err := tt.policies.Match(context.Background(), pr, *req)

			if tt.wantName == "" {
				if err == nil {
					t.Fatalf("Match(%v %v) error = nil; want no-match error", tt.method, tt.path)
				}
				if item != nil {
					t.Fatalf("Match(%v %v) item = %+v; want nil", tt.method, tt.path, item)
				}
				return
			}
			if err != nil {
				t.Fatalf("Match(%v %v) unexpected error: %v", tt.method, tt.path, err)
			}
			if item == nil {
				t.Fatalf("Match(%v %v) item = nil; want rule %q", tt.method, tt.path, tt.wantName)
			}
			if item.Name != tt.wantName {
				t.Errorf("Match(%v %v) matched rule %q; want %q", tt.method, tt.path, item.Name, tt.wantName)
			}
		})
	}
}
