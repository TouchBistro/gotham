package auth

import (
	"testing"
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
