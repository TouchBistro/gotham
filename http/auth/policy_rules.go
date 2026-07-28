package auth

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"github.com/TouchBistro/gotham/ds"
	"github.com/TouchBistro/goutils/color"
	log "github.com/sirupsen/logrus"
)

// Matching-related constants. Wildcard ("*") is treated as "match anything"
// when present as an element of a rule subject set, or as the value of method
// / path on a rule.
const (
	Wildcard   string = "*"
	Anything   string = Wildcard
	AllMethods string = Wildcard
	AllPaths   string = Wildcard
	Everyone   string = Wildcard
)

type Effect string

const (
	EffectAllow Effect = "allow"
	EffectDeny  Effect = "deny"
)

// Item is a single matching rule in a Policies list.
type Item struct {
	Priority   int64  `json:"-"`
	Name       string `json:"name"`
	HttpMethod string `json:"method"`
	HttpPath   string `json:"url"`
	Effect     Effect `json:"effect"`
	Subjects   ds.Set `json:"subjects"`
}

// Policies is an ordered list of Items.
type Policies []Item

// Match walks the rules in order and returns the first Item whose method,
// path and subjects all match the supplied request and principal. The
// principal's Identifier and Roles are consulted via its Principal methods.
// If no rule matches, a non-nil error is returned.
func (p Policies) Match(ctx context.Context, pr Principal, req http.Request) (*Item, error) {
	id := pr.Identifier(ctx)
	rolesSet := ds.From(pr.Roles(ctx)...)

	for _, item := range p {
		log.Tracef("matching: %v %v %v to %v (%v)", id, req.Method, req.URL.Path, item.Name, item.Priority)
		if item.HttpMethod == AllMethods || strings.EqualFold(item.HttpMethod, req.Method) {
			if item.HttpPath == AllPaths || strings.EqualFold(item.HttpPath, req.URL.Path) {
				if containsSetWithWildcard(item.Subjects, rolesSet) {
					log.Debugf("auth match found: %v %v %v to %v (%v)", color.Green(id), req.Method, req.URL.Path, color.Green(item.Name), item.Priority)
					return &item, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("subject %v not explicitly authorized to %v", color.Red(id), req.URL)
}

// RoleSetFrom builds a role Set from the supplied role names. If none are
// supplied the returned set contains the Everyone wildcard.
func RoleSetFrom(roles ...string) ds.Set {
	if len(roles) == 0 {
		roles = []string{Everyone}
	}
	return ds.From(roles...)
}

// pathMatches reports whether a rule url matches a request path. Three url
// forms are supported, evaluated in order:
//
//  1. the bare "*" sentinel (AllPaths), which matches every path;
//  2. exact case-insensitive equality with the request path;
//  3. a url ending in "*", which matches any request path that begins with
//     the url minus the trailing "*", compared case-insensitively. The
//     prefix must be non-empty; the empty-prefix case is only reachable as
//     the bare "*" sentinel handled above.
//
// No other wildcard form (mid-path "*", "**", regex, ":param") is supported.
func pathMatches(rulePath, reqPath string) bool {
	if rulePath == AllPaths || strings.EqualFold(rulePath, reqPath) {
		return true
	}
	if prefix, ok := strings.CutSuffix(rulePath, "*"); ok && prefix != "" {
		return len(reqPath) >= len(prefix) && strings.EqualFold(prefix, reqPath[:len(prefix)])
	}
	return false
}

// containsSetWithWildcard reports whether s intersects with other, treating
// the Wildcard sentinel in s as matching anything.
func containsSetWithWildcard(s, other ds.Set) bool {
	if _, ok := s[Wildcard]; ok {
		return true
	}
	return s.ContainsSet(other)
}
