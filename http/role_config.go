package http

// RolesConfig defines a mapping between external groups/group-sets & application-specific roles to
// be used for auth configuration. Also defines a list of roles that are considered admin or super-admin
type RolesConfig struct {
	AdminRoles      Set            `json:"admins"`      // Roles that are considered admins
	SuperAdminRoles Set            `json:"superAdmins"` // Roles that are considered super-admins
	Definitions     map[string]Set `json:"def"`         // map for Role->GroupSet
}
