package http

import (
	"time"
)

// Principal
type Principal struct {
	// The identifier for the principal, normally the sub
	Id string `json:"id" claim:"sub"` // sub

	// User alias
	Alias string `json:"alias"` // deried alias

	// Attributes mapped from claims
	// login
	Login string `json:"login" claim:"login"` // unique login

	// fname
	FirstName string `json:"fname" claim:"fname"` // first name
	// lname
	LastName string `json:"lname" claim:"lname"` // last name
	// eml
	Email string `json:"email" claim:"eml"` // email address
	// groups
	Groups []string `json:"groups" claim:"groups"` // groups assignments
	// managerId
	ManagerId string `json:"managerId" claim:"managerId"` // manager id
	// managerName
	ManagerName string `json:"managerName" claim:"manager"` // manager name

	// raw claims
	RawClaims    map[string]any `json:"claims"`
	Roles        Set            `json:"roles"`        // roles assigned, mapped from groups
	RawToken     string         `json:"raw"`          // raw id token awsalb token
	IsAdmin      bool           `json:"isAdmin"`      // is admiistrator
	IsSuperAdmin bool           `json:"isSuperAdmin"` // is super adming
	Expiry       time.Time
}

// Merge does a field-by-field merge, by taking the non-zero value from the other
// principal (arg) if it exists. If the other field is zero, then the original value
// is retained...
//
// The value for Id field must be supplied & be the same in both principals
func (p *Principal) Merge(other Principal) {

	if p.Id != other.Id {
		return
	}

	p.Alias = returnFirstNonZero(p.Alias, other.Alias)
	p.Login = returnFirstNonZero(p.Login, other.Login)
	p.FirstName = returnFirstNonZero(p.FirstName, other.FirstName)
	p.LastName = returnFirstNonZero(p.LastName, other.LastName)
	p.Email = returnFirstNonZero(p.Email, other.Email)
	p.ManagerId = returnFirstNonZero(p.ManagerId, other.ManagerId)
	p.ManagerName = returnFirstNonZero(p.ManagerName, other.ManagerName)
	p.RawToken = returnFirstNonZero(p.RawToken, other.RawToken)
	p.IsAdmin = p.IsAdmin || other.IsAdmin
	p.IsSuperAdmin = p.IsSuperAdmin || other.IsSuperAdmin

	// if groups /roles exists in the other, use them instead
	if len(p.Groups) == 0 {
		p.Groups = other.Groups
		p.Roles = other.Roles
		p.IsAdmin = other.IsAdmin
		p.IsSuperAdmin = other.IsSuperAdmin
	}

	if p.Expiry.IsZero() {
		p.Expiry = other.Expiry
	}
}
