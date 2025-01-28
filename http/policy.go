package http

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/TouchBistro/goutils/color"
	log "github.com/sirupsen/logrus"
)

// a policy document defines the authorization for api resources
//
//
// matcher           | sub                  | effect |
// -----------------------------------------------------
// GET /api/v1/dbs   | *                    | allow  |
// * /api/v1/dbs     | admin,managers       | allow  |
// * /api/v1/*       | admin                | allow  |
// * *               | *                    | deny   |
//

type PolicyEffect string

const (
	Wildcard   string = "*"
	Anything   string = Wildcard // const denoting any matcher, wildcard regex
	AllMethods string = Wildcard
	AllPaths   string = Wildcard
	Everyone   string = Wildcard
)

const (
	PolicyEffectAllow PolicyEffect = "allow"
	PolicyEffectDeny  PolicyEffect = "deny"
)

type PolicyItem struct {
	Priority   int64        `json:"-"`        // assigned at parse
	Name       string       `json:"name"`     // human-readable name for this policy item
	HttpMethod string       `json:"method"`   // an http method, or everything if not supplied
	HttpPath   string       `json:"url"`      // a pattern or regex that matches the path/object
	Effect     PolicyEffect `json:"effect"`   // allow | deny
	Subjects   Set          `json:"subjects"` // a set of subjects to whom this applies; uses custom unmarshall logic
	// SubjectsRaw json.RawMessage `json:"subjects"`
}

type Policies []PolicyItem

// Match matches the supplied sub with the policies & returns a
// matching policy based on the pre-defined rules. If no match is found, a non-nil
// error is returns. Also, if an error occurs during matching, a non-nil error
// specifying the details is returned.
func (p Policies) Match(pr Principal, req http.Request) (*PolicyItem, error) {
	for _, item := range p {
		log.Tracef("matching: %v %v %v to %v (%v)", pr, req.Method, req.URL.Path, item.Name, item.Priority)
		if item.HttpMethod == AllMethods || strings.EqualFold(item.HttpMethod, req.Method) {
			if item.HttpPath == AllPaths || strings.EqualFold(item.HttpPath, req.URL.Path) {
				if item.Subjects.ContainsSet(pr.Roles) {
					log.Debugf("auth match found: %v %v %v to %v (%v)", color.Green(pr.Login), req.Method, req.URL.Path, color.Green(item.Name), item.Priority)
					return &item, nil
				}
			}
		}
	}
	return nil, fmt.Errorf("subject %v not explicitly authorized to %v", color.Red(pr.Login), req.URL)
}
