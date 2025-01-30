package util

import ( // "log"
	"github.com/lestrrat-go/jwx/v2/jwt"
	log "github.com/sirupsen/logrus"
)

// ClaimsFromJwt fetches the specified claim values from the supplied jwt/jws.
// No verification is performed currently
func ClaimsFromJwt(jwtStr string, claims ...string) (map[string]any, error) {

	token, err := jwt.Parse([]byte(jwtStr), jwt.WithVerify(false))
	if err != nil {
		return nil, err
	}

	m := make(map[string]any)
	m["exp"] = token.Expiration()
	m["sub"] = token.Subject()

	if len(claims) > 0 {
		available := token.PrivateClaims()
		// fetch only selected
		for _, k := range claims {
			if v, ok := available[k]; !ok {
				log.Debugf("claim key not found %v", k)
				continue
			} else {
				m[k] = v
			}
		}
		return m, nil
	}

	// if not cliams supplied, return all + exp
	m = token.PrivateClaims()
	m["exp"] = token.Expiration()
	m["sub"] = token.Subject()
	return m, nil
}
