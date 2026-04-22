package auth

import (
	"context"
	"time"

	"github.com/TouchBistro/gotham/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// default claim names.
const (
	defaultSubClaim    = "sub"
	defaultExpiryClaim = "exp"
)

// JwtPrincipal is the concrete Principal produced by JwtPrincipalLoader. It
// embeds BasicPrincipal and retains the raw JWT along with its parsed claims.
type JwtPrincipal struct {
	BasicPrincipal

	// Raw is the original JWT token string.
	Raw string `json:"-"`

	// Claims is the parsed JWT claim set.
	Claims map[string]any `json:"-"`
}

// JwtPrincipalLoader implements PrincipalLoader by reading a JWT token from a
// configured HTTP header on the incoming request and parsing its claims.
// Construct via NewJwtPrincipalLoader and configure the header name with
// WithHttpHeaderName.
type JwtPrincipalLoader struct {
	httpHeaderName  string
	subClaimName    string
	expiryClaimName string

	// Reserved for future JWT signature validation (not yet supported).
	validateJwtSignature bool     //nolint:unused
	jwks                 []string //nolint:unused
	jwksUri              string   //nolint:unused
}

// JwtPrincipalLoaderBuilder builds a JwtPrincipalLoader.
type JwtPrincipalLoaderBuilder struct {
	l JwtPrincipalLoader
}

// NewJwtPrincipalLoader returns a builder pre-populated with defaults
// (subClaimName="sub", expiryClaimName="exp").
func NewJwtPrincipalLoader() *JwtPrincipalLoaderBuilder {
	return &JwtPrincipalLoaderBuilder{
		l: JwtPrincipalLoader{
			subClaimName:    defaultSubClaim,
			expiryClaimName: defaultExpiryClaim,
		},
	}
}

// WithHttpHeaderName sets the HTTP request header whose first value
// FetchPrincipal will read to obtain the JWT token.
func (b *JwtPrincipalLoaderBuilder) WithHttpHeaderName(name string) *JwtPrincipalLoaderBuilder {
	b.l.httpHeaderName = name
	return b
}

// WithSubClaimName overrides the claim name used to extract the principal's
// identifier. Defaults to "sub".
func (b *JwtPrincipalLoaderBuilder) WithSubClaimName(name string) *JwtPrincipalLoaderBuilder {
	if name != "" {
		b.l.subClaimName = name
	}
	return b
}

// WithExpiryClaimName overrides the claim name used to extract the
// principal's expiry. Defaults to "exp".
func (b *JwtPrincipalLoaderBuilder) WithExpiryClaimName(name string) *JwtPrincipalLoaderBuilder {
	if name != "" {
		b.l.expiryClaimName = name
	}
	return b
}

// Build returns the configured JwtPrincipalLoader.
func (b *JwtPrincipalLoaderBuilder) Build() JwtPrincipalLoader {
	return b.l
}

// FetchPrincipal implements PrincipalLoader. The JWT token is read from the
// first value of the configured HTTP header on in.Request, parsed, and the
// configured sub/expiry claims are lifted onto the returned *JwtPrincipal.
// The Id field of the input is ignored — the subject is sourced from the
// configured sub claim in the JWT.
func (l JwtPrincipalLoader) FetchPrincipal(ctx context.Context, in FetchPrincipalInput) (FetchPrincipalOutput, error) {
	if l.httpHeaderName == "" {
		return FetchPrincipalOutput{}, errors.New("no http header name configured on JwtPrincipalLoader")
	}

	jwt := in.Request.Header.Get(l.httpHeaderName)
	if jwt == "" {
		return FetchPrincipalOutput{}, errors.Errorf("no JWT found in request header %v", l.httpHeaderName)
	}

	claims, err := util.ClaimsFromJwt(jwt)
	if err != nil {
		return FetchPrincipalOutput{}, err
	}

	v, ok := claims[l.subClaimName]
	if !ok {
		return FetchPrincipalOutput{}, errors.Errorf("no %v claim in JWT, cannot create principal", l.subClaimName)
	}
	id, ok := v.(string)
	if !ok {
		return FetchPrincipalOutput{}, errors.Errorf("invalid %v claim in JWT, cannot create principal", l.subClaimName)
	}
	log.Debugf("%v claim value %v found in JWT, creating principal", l.subClaimName, id)

	var exp time.Time
	if v, ok := claims[l.expiryClaimName]; ok {
		if t, ok := v.(time.Time); ok {
			exp = t
		}
	}

	return FetchPrincipalOutput{
		Principal: &JwtPrincipal{
			BasicPrincipal: BasicPrincipal{
				Id:       id,
				ExpiryAt: exp,
			},
			Raw:    jwt,
			Claims: claims,
		},
	}, nil
}
