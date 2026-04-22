package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/TouchBistro/gotham/ds"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// userMember renders the per-user membership token used to grant a role to a
// specific principal by id (e.g. "user(alice)"). A role definition whose
// member set contains this token grants the role to the principal with the
// matching Identifier, in addition to any group-based matches.
func userMember(id string) string {
	return fmt.Sprintf("user(%s)", id)
}

// Config is the top-level auth configuration. It bundles role configuration,
// pre/post actions and the authorization policy list.
type Config struct {
	AdminRoles      ds.Set            `json:"admins"`
	SuperAdminRoles ds.Set            `json:"superAdmins"`
	Definitions     map[string]ds.Set `json:"def"`
	PreActions      []Action          `json:"preActions"`
	AuthrPolicies   Policies          `json:"authrPolicy"`
	PostActions     []Action          `json:"postActions"`
}

// LoadConfigFromFile reads an auth Config from the supplied file path. If the
// file cannot be read or parsed, the default hardcoded config is returned
// along with a non-nil error.
func LoadConfigFromFile(path string) (*Config, error) {
	c := &Config{}

	bytes, err := os.ReadFile(path)
	if err != nil {
		return defaultConfig(), errors.Wrapf(err, "error reading auth policy file: %v", path)
	}

	if err := json.Unmarshal(bytes, c); err != nil {
		return defaultConfig(), errors.Wrapf(err, "error reading auth policy file: %v", path)
	}

	log.Debugf("loaded %v policies", len(c.AuthrPolicies))
	return c, nil
}

// defaultConfig returns the hardcoded default Config.
func defaultConfig() *Config {
	return &Config{
		AdminRoles:      nil,
		SuperAdminRoles: nil,
		PreActions:      nil,
		PostActions:     nil,
		AuthrPolicies: []Item{
			{
				Priority:   0,
				Name:       "default_allow_all_to_admins",
				HttpMethod: AllMethods,
				HttpPath:   AllPaths,
				Effect:     EffectAllow,
				Subjects:   RoleSetFrom("admin"),
			},
			{
				Priority:   0,
				Name:       "default_deny_all_to_all",
				HttpMethod: AllMethods,
				HttpPath:   AllPaths,
				Effect:     EffectDeny,
				Subjects:   RoleSetFrom(Everyone),
			},
		},
	}
}

// RolesForPrincipal resolves the role names conferred on the supplied
// Principal by this Config, and reports whether the resolved role set confers
// super-admin and/or admin privileges.
//
// A role (as keyed in c.Definitions) is granted when its member set matches
// the principal in any of the following ways:
//   - Any of the role's member groups matches one of the principal's groups
//     (sourced from p.Groups(ctx)).
//   - The role's member set contains the Wildcard sentinel ("*"), which
//     matches any group.
//   - The role's member set contains the per-user token "user(<id>)" where
//     <id> is the principal's Identifier. This grants the role directly to a
//     specific principal regardless of group membership.
//
// A resolved role triggers the corresponding return flag when it also appears
// in c.AdminRoles or c.SuperAdminRoles.
//
// Returns:
//   - roles: sorted slice of resolved role names (nil when the principal has
//     a nil groups slice and an empty id — nothing can match).
//   - isSuperAdmin: true if any resolved role is in c.SuperAdminRoles.
//   - isAdmin: true if any resolved role is in c.AdminRoles.
func (c Config) RolesForPrincipal(ctx context.Context, p Principal) (roles []string, isSuperAdmin bool, isAdmin bool) {
	groups := p.Groups(ctx)
	id := p.Identifier(ctx)

	if groups == nil && id == "" {
		return nil, false, false
	}

	userTok := userMember(id)

	resolved := ds.Set{}
	for roleName, members := range c.Definitions {
		matched := containsWithWildcard(members, groups...)
		if !matched && id != "" {
			matched = members.Contains(userTok)
		}
		if !matched {
			continue
		}
		resolved.Insert(roleName)
		if _, ok := c.AdminRoles[roleName]; ok {
			isAdmin = true
		}
		if _, ok := c.SuperAdminRoles[roleName]; ok {
			isSuperAdmin = true
		}
	}
	return resolved.ToStringSlice(), isSuperAdmin, isAdmin
}

// containsWithWildcard reports whether s contains any of items, treating the
// Wildcard sentinel in s as matching anything.
func containsWithWildcard(s ds.Set, items ...string) bool {
	if _, ok := s[Wildcard]; ok {
		return true
	}
	return s.Contains(items...)
}
