package cereport

import (
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// maxGroupBy is the number of GroupBy entries GetCostAndUsage accepts.
const maxGroupBy = 2

var (
	// knownMetrics is hand-kept: GetCostAndUsage takes these CamelCase names,
	// while the SDK's Metric enum spells them UNBLENDED_COST etc. for other
	// operations.
	knownMetrics = set(MetricUnblendedCost, MetricBlendedCost, MetricAmortizedCost,
		MetricNetUnblendedCost, MetricNetAmortizedCost, MetricUsageQuantity, MetricNormalizedUsageAmount)

	// The rest come from the SDK, so they track the service/costexplorer
	// version in go.mod rather than a list here.
	knownGranularities = enumSet(cetypes.Granularity("").Values())
	knownTypes         = enumSet(cetypes.GroupDefinitionType("").Values())
	knownDimensions    = enumSet(cetypes.Dimension("").Values())
)

func set(vals ...string) map[string]bool {
	m := make(map[string]bool, len(vals))
	for _, v := range vals {
		m[v] = true
	}
	return m
}

func enumSet[E ~string](vals []E) map[string]bool {
	m := make(map[string]bool, len(vals))
	for _, v := range vals {
		m[string(v)] = true
	}
	return m
}

func sortedKeys(m map[string]bool) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// Metrics returns the metric names a Spec may use, sorted.
func Metrics() []string { return sortedKeys(knownMetrics) }

// Granularities returns the granularities a Spec may use, sorted.
func Granularities() []string { return sortedKeys(knownGranularities) }

// Dimensions returns the keys a DIMENSION group or filter may use, sorted.
func Dimensions() []string { return sortedKeys(knownDimensions) }

// Validate checks the spec against every rule Cost Explorer is known to
// enforce, plus what this package needs, and returns all problems joined into
// one error (errors.Join), or nil. GetCostAndUsageInput and Run call it first,
// so an invalid spec never costs an API request.
//
// Rules:
//   - Name is set.
//   - Metric is one of the Metric* constants (see Metrics).
//   - Granularity is one of the Granularity* constants (see Granularities).
//   - GroupBy has at most two entries. Each has a Type from the Type*
//     constants and a Key; a DIMENSION Key must be a Cost Explorer dimension
//     (see Dimensions).
//   - Each Filter has a valid Type and Key (same rules) and at least one value.
//   - TimeRange.Relative is CUSTOM (or empty), YEAR_TO_DATE, MONTH_TO_DATE,
//     LAST_<n>_DAYS or LAST_<n>_MONTHS with n ≥ 1. A CUSTOM range has Start
//     and End. Any Start or End given is yyyy-MM-dd, with Start before End.
func (s Spec) Validate() error {
	var errs []error
	if strings.TrimSpace(s.Name) == "" {
		errs = append(errs, errors.New("name is required"))
	}
	if !knownMetrics[s.Metric] {
		errs = append(errs, fmt.Errorf("metric %q: want one of %s", s.Metric, strings.Join(Metrics(), ", ")))
	}
	if !knownGranularities[s.Granularity] {
		errs = append(errs, fmt.Errorf("granularity %q: want one of %s", s.Granularity, strings.Join(Granularities(), ", ")))
	}
	if len(s.GroupBy) > maxGroupBy {
		errs = append(errs, fmt.Errorf("groupBy has %d entries; Cost Explorer allows at most %d", len(s.GroupBy), maxGroupBy))
	}
	for i, g := range s.GroupBy {
		if err := validateKey(fmt.Sprintf("groupBy[%d]", i), g.Type, g.Key); err != nil {
			errs = append(errs, err)
		}
	}
	for i, f := range s.Filters {
		field := fmt.Sprintf("filters[%d]", i)
		if err := validateKey(field, f.Type, f.Key); err != nil {
			errs = append(errs, err)
		}
		if len(f.Values) == 0 {
			errs = append(errs, fmt.Errorf("%s: values is empty", field))
		}
	}
	errs = append(errs, s.TimeRange.validate()...)
	if len(errs) == 0 {
		return nil
	}
	return fmt.Errorf("report %q: %w", s.Name, errors.Join(errs...))
}

// validateKey checks a Group or Filter's type/key pair.
func validateKey(field, typ, key string) error {
	if !knownTypes[typ] {
		return fmt.Errorf("%s: unknown type %q; want %s, %s or %s", field, typ, TypeDimension, TypeTag, TypeCostCategory)
	}
	if typ == TypeDimension {
		if !knownDimensions[key] {
			return fmt.Errorf("%s: unknown dimension %q; known: %s", field, key, strings.Join(Dimensions(), ", "))
		}
		return nil
	}
	if key == "" {
		return fmt.Errorf("%s: key is required for type %s", field, typ)
	}
	return nil
}

func (t TimeRange) validate() []error {
	var errs []error
	switch r := t.Relative; {
	case t.IsCustom():
		if t.Start == "" || t.End == "" {
			errs = append(errs, errors.New("timeRange: a CUSTOM range needs start and end"))
		}
	case r == RangeYearToDate, r == RangeMonthToDate:
	case lastNRange.MatchString(r):
		if n, _ := strconv.Atoi(lastNRange.FindStringSubmatch(r)[1]); n < 1 {
			errs = append(errs, fmt.Errorf("timeRange.relative %q: n must be at least 1", r))
		}
	default:
		errs = append(errs, fmt.Errorf("timeRange.relative %q: want %s, %s, %s, LAST_<n>_DAYS or LAST_<n>_MONTHS",
			r, RangeCustom, RangeYearToDate, RangeMonthToDate))
	}

	var start, end time.Time
	if t.Start != "" {
		d, err := time.Parse(dateLayout, t.Start)
		if err != nil {
			errs = append(errs, fmt.Errorf("timeRange.start %q: want yyyy-MM-dd", t.Start))
		}
		start = d
	}
	if t.End != "" {
		d, err := time.Parse(dateLayout, t.End)
		if err != nil {
			errs = append(errs, fmt.Errorf("timeRange.end %q: want yyyy-MM-dd", t.End))
		}
		end = d
	}
	if !start.IsZero() && !end.IsZero() && !start.Before(end) {
		errs = append(errs, fmt.Errorf("timeRange: start %s must be before end %s", t.Start, t.End))
	}
	return errs
}
