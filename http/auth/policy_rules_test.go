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

// TestPolicies_Match_Ordering proves first-match-wins ordering over the
// Policies slice is unchanged by the prefix capability: the first matching
// rule in slice order is returned, with no reordering and no specificity
// preference between exact-path and prefix rules.
func TestPolicies_Match_Ordering(t *testing.T) {
	denyExact := Item{
		Name:       "deny-exact",
		HttpMethod: "GET",
		HttpPath:   "/cr7/reviews/locked",
		Effect:     EffectDeny,
		Subjects:   ds.From(Everyone),
	}
	allowPrefix := Item{
		Name:       "allow-prefix",
		HttpMethod: "GET",
		HttpPath:   "/cr7/reviews/*",
		Effect:     EffectAllow,
		Subjects:   ds.From(Everyone),
	}

	tests := []struct {
		name       string
		policies   Policies
		path       string
		wantName   string
		wantEffect Effect
	}{
		{
			name:       "deny exact rule listed first wins for the exact path",
			policies:   Policies{denyExact, allowPrefix},
			path:       "/cr7/reviews/locked",
			wantName:   "deny-exact",
			wantEffect: EffectDeny,
		},
		{
			name:       "prefix rule wins for sub-paths the exact rule does not cover",
			policies:   Policies{denyExact, allowPrefix},
			path:       "/cr7/reviews/other",
			wantName:   "allow-prefix",
			wantEffect: EffectAllow,
		},
		{
			name:       "allow prefix rule listed first wins for the overlapping path",
			policies:   Policies{allowPrefix, denyExact},
			path:       "/cr7/reviews/locked",
			wantName:   "allow-prefix",
			wantEffect: EffectAllow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &BasicPrincipal{Id: "test-user", RoleSet: ds.From("viewer")}
			req := httptest.NewRequest("GET", tt.path, nil)

			item, err := tt.policies.Match(context.Background(), pr, *req)
			if err != nil {
				t.Fatalf("Match(GET %v) unexpected error: %v", tt.path, err)
			}
			if item.Name != tt.wantName {
				t.Errorf("Match(GET %v) matched rule %q; want %q", tt.path, item.Name, tt.wantName)
			}
			if item.Effect != tt.wantEffect {
				t.Errorf("Match(GET %v) effect = %q; want %q", tt.path, item.Effect, tt.wantEffect)
			}
		})
	}
}

// TestPolicies_Match_BackwardCompatible proves no behaviour change for
// existing policy shapes. The legacy list mirrors defaultConfig() in
// policy.go (global "*" rules for admin-allow and deny-all) plus an
// exact-path rule; every case must resolve to the same rule it would have
// pre-change. The only behaviour delta introduced by prefix matching is a
// legacy rule ending in "*" (e.g. "/legacy/*") that previously matched
// nothing and now matches sub-paths; the final cases pin down that delta as
// the only one.
func TestPolicies_Match_BackwardCompatible(t *testing.T) {
	exactHealth := Item{
		Name:       "exact-health",
		HttpMethod: "GET",
		HttpPath:   "/health",
		Effect:     EffectAllow,
		Subjects:   ds.From(Everyone),
	}
	allowAdmins := Item{
		Name:       "default_allow_all_to_admins",
		HttpMethod: AllMethods,
		HttpPath:   AllPaths,
		Effect:     EffectAllow,
		Subjects:   RoleSetFrom("admin"),
	}
	denyAll := Item{
		Name:       "default_deny_all_to_all",
		HttpMethod: AllMethods,
		HttpPath:   AllPaths,
		Effect:     EffectDeny,
		Subjects:   RoleSetFrom(Everyone),
	}
	legacyStar := Item{
		Name:       "legacy-star",
		HttpMethod: "GET",
		HttpPath:   "/legacy/*",
		Effect:     EffectAllow,
		Subjects:   ds.From(Everyone),
	}

	// Only exact-path and global "*" rules — the pre-change vocabulary.
	legacy := Policies{exactHealth, allowAdmins, denyAll}
	// A legacy trailing-"*" rule that could never match pre-change.
	legacyWithStar := Policies{legacyStar, denyAll}

	tests := []struct {
		name     string
		policies Policies
		roles    []string
		method   string
		path     string
		wantName string
	}{
		// Unchanged: exact-path and global "*" rules resolve exactly as
		// they did pre-change.
		{
			name:     "exact path still resolves to the exact rule",
			policies: legacy,
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/health",
			wantName: "exact-health",
		},
		{
			name:     "exact path still matches case-insensitively",
			policies: legacy,
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/HEALTH",
			wantName: "exact-health",
		},
		{
			name:     "exact rule gains no prefix behaviour for sub-paths",
			policies: legacy,
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/health/live",
			wantName: "default_deny_all_to_all",
		},
		{
			name:     "exact rule still rejects a trailing slash",
			policies: legacy,
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/health/",
			wantName: "default_deny_all_to_all",
		},
		{
			name:     "admin still resolves to the global allow rule",
			policies: legacy,
			roles:    []string{"admin"},
			method:   "POST",
			path:     "/anything/at/all",
			wantName: "default_allow_all_to_admins",
		},
		{
			name:     "non-admin still falls through to the global deny rule",
			policies: legacy,
			roles:    []string{"viewer"},
			method:   "POST",
			path:     "/anything/at/all",
			wantName: "default_deny_all_to_all",
		},

		// The single behaviour delta: a legacy trailing-"*" rule that
		// previously matched nothing now matches sub-paths.
		{
			name:     "delta: legacy trailing-star rule now matches sub-paths",
			policies: legacyWithStar,
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/legacy/sub",
			wantName: "legacy-star",
		},
		// Everything else about the trailing-star rule is unchanged.
		{
			name:     "literal star path still matches via exact equality",
			policies: legacyWithStar,
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/legacy/*",
			wantName: "legacy-star",
		},
		{
			name:     "path equal to prefix minus slash still falls through",
			policies: legacyWithStar,
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/legacy",
			wantName: "default_deny_all_to_all",
		},
		{
			name:     "unrelated path still falls through to the deny rule",
			policies: legacyWithStar,
			roles:    []string{"viewer"},
			method:   "GET",
			path:     "/other",
			wantName: "default_deny_all_to_all",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			pr := &BasicPrincipal{Id: "test-user", RoleSet: ds.From(tt.roles...)}
			req := httptest.NewRequest(tt.method, tt.path, nil)

			item, err := tt.policies.Match(context.Background(), pr, *req)
			if err != nil {
				t.Fatalf("Match(%v %v) unexpected error: %v", tt.method, tt.path, err)
			}
			if item.Name != tt.wantName {
				t.Errorf("Match(%v %v) matched rule %q; want %q", tt.method, tt.path, item.Name, tt.wantName)
			}
		})
	}
}
