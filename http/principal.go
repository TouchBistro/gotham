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

// TODO move these out of the library since they must be either supplied
// by the caller/user or made more generic & configurable ..

// cachePrincipal is a utility func to cache the supplied principal using the
// cache key built using concatenated prefix::key => principal
//
// the ttl is set using the expiry field value in the principal object
// func cachePrincipal(ctx context.Context, prefix, eid string, pr Principal) error {
// 	key := buildCacheKey()
// 	ttl := time.Until(pr.Expiry)
// 	log.Debugf("caching principal key=%v, ttl=%v", key, ttl)
// 	// TODO inject cache here.
// 	// if err := cache.Store().PutWithTtl(ctx, key, pr, ttl); err != nil {
// 	// 	return err
// 	// }
// 	return nil
// }

// loadPrincipalFromCache retrives the Principal for the supplied prefix::eid from cache, or
// a non-nil error if the principal is not found, or has expired
// func loadPrincipalFromCache(ctx context.Context, prefix, eid string) (*Principal, error) {
// 	key := buildCacheKey(prefix, eid)
// 	pr := Principal{
// 		Roles: Set{},
// 	}
// 	// var err error
// 	var ttl *time.Duration

// 	// TODO tmp code
// 	// if _, err := cache.Store().Delete(ctx, key); err != nil {
// 	// 	return nil, err
// 	// }
// 	// TODO end

// 	// TODO inject cache here
// 	// if ttl, err = cache.Store().FetchWithTtl(ctx, key, &pr); err != nil {
// 	// 	log.Debugf("cache miss: key=%v", key)
// 	// 	return nil, err
// 	// }
// 	log.Debugf("cache hit: key=%v, ttl=%v expiry=%v", key, time.Now().Add(*ttl), pr.Expiry)
// 	return &pr, nil
// }

// loadPrincipalFromClaims builds a Principal from the supplied claims map
// func loadPrincipalFromClaims(cfg Config, claims map[string]any) (*Principal, error) {

// 	// sub
// 	var sub string
// 	if v, ok := claims["sub"]; !ok {
// 		return nil, errors.Errorf("no sub claim in JWT, cannot create principal")
// 	} else if _, ok := v.(string); !ok {
// 		return nil, errors.Errorf("invalid sub claim in JWT, cannot create principal")
// 	} else {
// 		sub = v.(string)
// 	}

// 	// if login exists
// 	var login, alias string
// 	if _, ok := claims["login"]; ok {
// 		login = claims["login"].(string)
// 		alias = login
// 		// if login was an email, then just use the alias part of that email addr
// 		if strings.Contains(login, "@") {
// 			alias = login[0:strings.Index(login, "@")]
// 		}
// 		alias = strings.Replace(alias, "+", "_", -1) // replace any + with _
// 		alias = strings.Replace(alias, ".", "_", -1) // replace any . with _
// 	}

// 	var fname, lname, email string
// 	if v, ok := claims["fname"]; ok {
// 		vstr := v.(string)
// 		fname = vstr
// 	}

// 	if v, ok := claims["lname"]; ok {
// 		vstr := v.(string)
// 		lname = vstr
// 	}

// 	if v, ok := claims["eml"]; ok {
// 		vstr := v.(string)
// 		email = vstr
// 	}

// 	// groups is available
// 	var grps []string
// 	var rols Set
// 	var isSuper, isAdmin bool
// 	if _, ok := claims["groups"]; ok {
// 		grps = []string{}
// 		gr_any := claims["groups"]
// 		if gr_arr, ok := gr_any.([]any); ok {
// 			for _, v := range gr_arr {
// 				grps = append(grps, v.(string))
// 			}
// 		}
// 		rols, isSuper, isAdmin = rolesFromGroups(cfg, grps)
// 	}

// 	var managerId, managerName string

// 	// manager
// 	if v, ok := claims["managerId"]; ok {
// 		vstr := v.(string)
// 		managerId = vstr
// 	}

// 	// managerId
// 	if v, ok := claims["manager"]; ok {
// 		vstr := v.(string)
// 		managerName = vstr
// 	}

// 	// expiry
// 	var exp time.Time
// 	if v, ok := claims["exp"]; ok {
// 		if exp, ok = v.(time.Time); !ok {
// 			exp = time.Now().Add(2 * time.Hour) // if can't format exp to time.Time, then use Now() + 2hr
// 		}
// 	}

// 	return &Principal{
// 		Id:           sub,
// 		Login:        login,
// 		Alias:        alias,
// 		FirstName:    fname,
// 		LastName:     lname,
// 		Email:        email,
// 		ManagerId:    managerId,
// 		ManagerName:  managerName,
// 		Groups:       grps,
// 		Roles:        rols,
// 		Expiry:       exp,
// 		IsSuperAdmin: isSuper,
// 		IsAdmin:      isAdmin,
// 	}, nil
// }

// // loadPrincipalFromDb loads the principal from a set of sre.auth_* database tables
// func loadPrincipalFromDb(ctx context.Context, cfg Config, externalId string) (*Principal, error) {

// 	// check if database result is cached
// 	cacheKey := "db-principal"
// 	if pr, err := loadPrincipalFromCache(ctx, cacheKey, externalId); pr != nil && err == nil {
// 		return pr, err // return is found in cache with a non-nil err
// 	}

// 	return nil, fmt.Errorf("not found in cache")

// 	// // re init the principal & load from database
// 	// pr := Principal{
// 	// 	Id: externalId,
// 	// }

// 	// pr_stmt := `
// 	// 		SELECT
// 	// 		u.user_alias,
// 	// 		u.user_name,
// 	// 		u.first_name,
// 	// 		u.last_name,
// 	// 		u.email1,
// 	// 		m.user_alias,
// 	// 		m.display_name
// 	// 	FROM sre.auth_identities u LEFT OUTER JOIN sre.auth_identities m
// 	// 	on u.manager_alias = m.user_alias
// 	// 	WHERE u.external_id = $1;
// 	// `

// 	// gr_stmt := `
// 	// SELECT g.name
// 	// 	FROM
// 	// 		(SELECT u.display_name AS nom, m.display_name AS m_nom, unnest(u.memberships) AS gid
// 	// 		FROM sre.auth_identities u
// 	// 				LEFT JOIN sre.auth_identities m ON u.manager_alias = m.user_alias
// 	// 		WHERE u.external_id = $1
// 	// 		) ix LEFT JOIN sre.auth_groups g ON ix.gid = g.group_id
// 	// 	;
// 	// `

// 	// // get config db
// 	// configDb, err := dbapi.GetConfigDatabase()
// 	// if err != nil {
// 	// 	log.Error("error retrieving configuration database connection")
// 	// 	return nil, err
// 	// }

// 	// conn, err := configDb.SwitchDatabase("sre")
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	// rows, err := conn.Query(pr_stmt, externalId)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// defer rows.Close()

// 	// if !rows.Next() {
// 	// 	return nil, fmt.Errorf("user not found for external id %v", externalId)
// 	// }

// 	// err = rows.Scan(&pr.Alias, &pr.Login, &pr.FirstName, &pr.LastName, &pr.Email, &pr.ManagerId, &pr.ManagerName)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }

// 	// // fetch group memberships
// 	// rows2, err := conn.Query(gr_stmt, externalId)
// 	// if err != nil {
// 	// 	return nil, err
// 	// }
// 	// defer rows2.Close()

// 	// groups := make([]string, 0)
// 	// var group string
// 	// for rows2.Next() {
// 	// 	err = rows2.Scan(&group)
// 	// 	if err != nil {
// 	// 		return nil, err
// 	// 	}
// 	// 	groups = append(groups, group)
// 	// }

// 	// pr.Expiry = time.Now().Add(12 * time.Hour) // set expiry for this to 12 hours
// 	// pr.Groups = groups
// 	// pr.Roles, pr.IsSuperAdmin, pr.IsAdmin = rolesFromGroups(cfg, pr.Groups)

// 	// // cache before returning
// 	// cachePrincipal(ctx, cacheKey, externalId, pr)

// 	// return &pr, nil
// }
