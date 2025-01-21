package http

// JwtConfig contains all JWT related values, e.g request headers to read ID & Access tokens, the
// header that holds the "sub" claim & Jwt validation related parameters
type JwtConfig struct {
	IdTokenHeader        string   `json:"idTokenHeader"`
	AccessTokenHeader    string   `json:"accessTokenHeader"`
	SubClaimHeader       string   `json:"subClaimHeader"`
	ValidateJwtSignature bool     `json:"validateJwtSignature"`
	Jwks                 []string `json:"jwks"`
	JwksUri              string   `json:"jwksUri"`
}
