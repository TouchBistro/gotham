package http

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/TouchBistro/gotham/cache"
	"github.com/TouchBistro/gotham/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// PrincipalLoader supplies an interface for principal loading implementations
type PrincipalLoader interface {
	FetchPrincipal(ctx context.Context, sub string) (*Principal, error)
}

// PrincipalLoaderFn defines a adanpter func type that matches the PrincipalLoader method signature
type PrincipalLoaderFunc func(context.Context, string) (*Principal, error)

// implements the PrincipalLoader interface
func (f PrincipalLoaderFunc) FetchPrincipal(ctx context.Context, sub string) (*Principal, error) {
	return f(ctx, sub)
}

// JwtClaimsPrincipalLoader implements PrincipalLoader from claims of a JWT token
type JwtClaimsPrincipalLoader struct {
	config Config
	jwt    string
}

// FetchPrincipal implements the interface method
func (l JwtClaimsPrincipalLoader) FetchPrincipal(ctx context.Context, subject string) (*Principal, error) {

	log.Debugf("loading principal for sub %v from jwt claims", subject)

	// get claism from jwt
	claims, err := util.ClaimsFromJwt(l.jwt)
	if err != nil {
		return nil, err
	}

	// sub
	var sub string
	if v, ok := claims["sub"]; !ok {
		log.Debug("no sub claim in JWT, cannot create principal")
		return nil, errors.Errorf("no sub claim in JWT, cannot create principal")
	} else if _, ok := v.(string); !ok {
		log.Debug("invalid datatype for sub claim in JWT, cannot create principal")
		return nil, errors.Errorf("invalid sub claim in JWT, cannot create principal")
	} else {
		sub = v.(string)
	}

	// if subject value was supplied, we also compared
	// with the sub claim fond in the JWT
	if subject != "" {
		if sub != subject {
			log.Debugf("sub claim value %v found in JWT is differnt from %v, cannot create principal", sub, subject)
			return nil, errors.Errorf("incorrect sub claim %v found in JWT", subject)
		}
	}

	log.Debugf("sub claim value %v found in JWT, creating principal", sub)

	// if login exists
	var login, alias string
	if _, ok := claims["login"]; ok {
		login = claims["login"].(string)
		alias = login
		// if login was an email, then just use the alias part of that email addr
		if strings.Contains(login, "@") {
			alias = login[0:strings.Index(login, "@")]
		}
		alias = strings.Replace(alias, "+", "_", -1) // replace any + with _
		alias = strings.Replace(alias, ".", "_", -1) // replace any . with _
	}

	var fname, lname, email string
	if v, ok := claims["fname"]; ok {
		vstr := v.(string)
		fname = vstr
	}

	if v, ok := claims["lname"]; ok {
		vstr := v.(string)
		lname = vstr
	}

	if v, ok := claims["eml"]; ok {
		vstr := v.(string)
		email = vstr
	}

	// groups is available
	var grps []string
	var rols Set
	var isSuper, isAdmin bool
	if _, ok := claims["groups"]; ok {
		grps = []string{}
		gr_any := claims["groups"]
		if gr_arr, ok := gr_any.([]any); ok {
			for _, v := range gr_arr {
				grps = append(grps, v.(string))
			}
		}
		rols, isSuper, isAdmin = rolesFromGroups(l.config, grps)
	}

	var managerId, managerName string

	// manager
	if v, ok := claims["managerId"]; ok {
		vstr := v.(string)
		managerId = vstr
	}

	// managerId
	if v, ok := claims["manager"]; ok {
		vstr := v.(string)
		managerName = vstr
	}

	// expiry
	var exp time.Time
	if v, ok := claims["exp"]; ok {
		if exp, ok = v.(time.Time); !ok {
			exp = time.Now().Add(2 * time.Hour) // if can't format exp to time.Time, then use Now() + 2hr
		}
	}

	return &Principal{
		Id:           sub,
		Login:        login,
		Alias:        alias,
		FirstName:    fname,
		LastName:     lname,
		Email:        email,
		ManagerId:    managerId,
		ManagerName:  managerName,
		Groups:       grps,
		Roles:        rols,
		Expiry:       exp,
		IsSuperAdmin: isSuper,
		IsAdmin:      isAdmin,
	}, nil
}

// CachePrincipalLoader implements PrincipalLoader from a memory cache
type CachePrincipalLoader struct {
	KeyPrefix string
	Cache     cache.MemoryCache
}

// FetchPrincipal implements the interface method
func (l CachePrincipalLoader) FetchPrincipal(ctx context.Context, subject string) (*Principal, error) {

	key := l.buildCacheKey(subject)
	pr := Principal{
		Roles: Set{},
	}

	ttl, err := l.Cache.FetchWithTtl(ctx, key, &pr)
	if err != nil {
		log.Debugf("cache miss for key=%v (sub) when fetching cached principal", key)
		return nil, err
	}

	log.Debugf("cache hit for key=%v (sub) ttl=%v expiry=%v when fetching cached principal", key, time.Now().Add(*ttl), pr.Expiry)
	return &pr, nil
}

func (l CachePrincipalLoader) Persist(ctx context.Context, pr Principal) error {
	key := l.buildCacheKey(pr.Id) // Id is the value of "sub" claim
	ttl := time.Until(pr.Expiry)
	log.Debugf("caching principal key=%v (sub), ttl=%v", key, ttl)

	if err := l.Cache.PutWithTtl(ctx, key, pr, ttl); err != nil {
		return err
	}
	return nil
}

func (l CachePrincipalLoader) buildCacheKey(sub string) string {
	if l.KeyPrefix != "" {
		return fmt.Sprintf("%v::%v", l.KeyPrefix, sub)
	}
	return sub
}

// StaticPrincipalLoader is a mocking helper function that returns a PrincipalLoader that
// returns the supplied principal struct, with the Id set to supplied `sub`
func StaticPrincipalLoader(pr Principal) PrincipalLoaderFunc {
	return PrincipalLoaderFunc(func(ctx context.Context, sub string) (*Principal, error) {
		p := pr
		p.Id = sub // set sub as external Id, everything else stays as-is
		return &p, nil
	})
}

// helper functions

// rolesFromGroups using mapping defined in config, this func
// returns a RoleSet from the list of groups that these supplied groups
// lie into; also return if any of these roles are superAdmins & Admins
func rolesFromGroups(cfg Config, grps []string) (Set, bool, bool) {

	var issa, isa bool

	if grps == nil {
		return nil, issa, isa
	}

	roles := Set{}
	for roleName, membersGroupset := range cfg.Roles.Definitions {
		// if any of the groups are part of this role...
		if membersGroupset.Contains(grps...) {
			// add the role to this list
			roles.Insert(roleName)

			// check if this role is admin; mark user admin
			if _, ok := cfg.Roles.AdminRoles[roleName]; ok {
				isa = ok
			}

			// check if this role is super admin; mark user super admin
			if _, ok := cfg.Roles.SuperAdminRoles[roleName]; ok {
				issa = ok
			}
		}
	}
	return roles, issa, isa
}
