package http

import "encoding/json"

// RolesConfig defines a mapping between external groups/group-sets & application-specific roles to
// be used for auth configuration. Also defines a list of roles that are considered admin or super-admin
type RolesConfig struct {
	AdminRoles          Set             // Roles that are considered admins
	SuperAdminRoles     Set             // Roles that are considered super-admins
	Definitions         map[string]Set  // map for Role->GroupSet
	AdminGroupsRaw      json.RawMessage `json:"admins"`
	SuperAdminGroupsRaw json.RawMessage `json:"superAdmins"`
	DefinitionsRaw      json.RawMessage `json:"def"`
}
