package http

import (
	"encoding/json"
	"os"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// The auth middleware configuration that contains Jwt & role configuration
// pre/post actions & auth middleware
type AuthPolicy struct {
	Config        Config         `json:"config"`
	PreActions    []PolicyAction `json:"preActions"`
	AuthrPolicies Policies       `json:"authrPolicy"`
	PostActions   []PolicyAction `json:"postActions"`
}

// The auth middleware configuration
type Config struct {
	JwtConfig JwtConfig   `json:"jwt"`   // jwt related configuration;
	Roles     RolesConfig `json:"roles"` // role definition
}

// LoadPolicyFromFile reads the auth policies from the supplied file path
func LoadPolicyFromFile(path string) (*AuthPolicy, error) {

	cfg := path
	ap := &AuthPolicy{}

	bytes, err := os.ReadFile(cfg)
	if err != nil {
		return loadPolicyDefault(), errors.Wrapf(err, "error reading auth policy file: %v", cfg)
	}

	err = json.Unmarshal(bytes, &ap)
	if err != nil {
		return loadPolicyDefault(), errors.Wrapf(err, "error reading auth policy file: %v", cfg)
	}

	ap.Config.Roles.AdminRoles, err = RoleSetFromRawJsonArray(ap.Config.Roles.AdminGroupsRaw)
	if err != nil {
		return nil, err
	}
	ap.Config.Roles.SuperAdminRoles, err = RoleSetFromRawJsonArray(ap.Config.Roles.SuperAdminGroupsRaw)
	if err != nil {
		return nil, err
	}

	ap.Config.Roles.Definitions, err = RoleDefsFromRawJsonMap(ap.Config.Roles.DefinitionsRaw)
	if err != nil {
		return nil, err
	}

	ap.Config.Roles.AdminGroupsRaw = nil
	ap.Config.Roles.SuperAdminGroupsRaw = nil

	for n, pol := range ap.AuthrPolicies {
		if r, err := RoleSetFromRawJsonArray(pol.SubjectsRaw); err != nil {
			return nil, err
		} else {
			ap.AuthrPolicies[n].Subjects = r
		}
		ap.AuthrPolicies[n].Priority = int64(n)
		ap.AuthrPolicies[n].SubjectsRaw = nil
	}

	log.Infof("loaded %v policies", len(ap.AuthrPolicies))
	return ap, nil
}

// loadPolicyDefault loads the default auth policy (hardcoded)
func loadPolicyDefault() *AuthPolicy {
	return &AuthPolicy{
		Config: Config{
			Roles: RolesConfig{
				AdminRoles:      nil,
				SuperAdminRoles: nil,
			},
			JwtConfig: JwtConfig{
				IdTokenHeader: "x-jwt-data",
			},
		},
		PreActions:  nil,
		PostActions: nil,
		AuthrPolicies: []PolicyItem{
			{
				Priority:   0,
				Name:       "default_allow_all_to_admins",
				HttpMethod: AllMethods,
				HttpPath:   AllPaths,
				Effect:     PolicyEffectAllow,
				Subjects:   RoleSetFrom("admin"),
			},
			{
				Priority:   0,
				Name:       "default_deny_all_to_all",
				HttpMethod: AllMethods,
				HttpPath:   AllPaths,
				Effect:     PolicyEffectDeny,
				Subjects:   RoleSetFrom(Everyone),
			},
		},
	}
}
