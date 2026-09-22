package cereport

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

const (
	// consoleTagGroupPrefix is how the console encodes "group by tag" in the
	// groupBy array, e.g. "TagKeyValue:repo".
	consoleTagGroupPrefix = "TagKeyValue:"
	// consoleTagFilterID is the dimension id the console uses for a tag filter
	// row. Such a row is shaped differently from a dimension row: see
	// parseFilters.
	consoleTagFilterID = "TagKey"
)

// consoleDimensions maps the dimension ids the Cost Explorer console uses in
// its URLs to the dimension keys the Cost Explorer API expects (most are the
// SCREAMING_SNAKE form of the console id; AZ is the same on both sides). Unmapped ids
// are rejected rather than guessed at, so that an unfamiliar report surfaces as
// an error instead of a silently wrong query.
var consoleDimensions = map[string]string{
	"AZ":                 "AZ",
	"BillingEntity":      "BILLING_ENTITY",
	"CacheEngine":        "CACHE_ENGINE",
	"DatabaseEngine":     "DATABASE_ENGINE",
	"DeploymentOption":   "DEPLOYMENT_OPTION",
	"InstanceType":       "INSTANCE_TYPE",
	"InstanceTypeFamily": "INSTANCE_TYPE_FAMILY",
	"InvoicingEntity":    "INVOICING_ENTITY",
	"LegalEntityName":    "LEGAL_ENTITY_NAME",
	"LinkedAccount":      "LINKED_ACCOUNT",
	"LinkedAccountName":  "LINKED_ACCOUNT_NAME",
	"OperatingSystem":    "OPERATING_SYSTEM",
	"Operation":          "OPERATION",
	"Platform":           "PLATFORM",
	"PurchaseType":       "PURCHASE_TYPE",
	"RecordTypeV2":       "RECORD_TYPE",
	"Region":             "REGION",
	"ReservationId":      "RESERVATION_ID",
	"ResourceId":         "RESOURCE_ID",
	"SavingsPlanArn":     "SAVINGS_PLAN_ARN",
	"SavingsPlansType":   "SAVINGS_PLANS_TYPE",
	"Scope":              "SCOPE",
	"Service":            "SERVICE",
	"ServiceCode":        "SERVICE_CODE",
	"Tenancy":            "TENANCY",
	"UsageType":          "USAGE_TYPE",
	"UsageTypeGroup":     "USAGE_TYPE_GROUP",
}

// consoleMetrics maps the console's costAggregate values to API metric names.
var consoleMetrics = map[string]string{
	"unBlendedCost":         MetricUnblendedCost,
	"blendedCost":           MetricBlendedCost,
	"amortizedCost":         MetricAmortizedCost,
	"netUnblendedCost":      MetricNetUnblendedCost,
	"netAmortizedCost":      MetricNetAmortizedCost,
	"usageQuantity":         MetricUsageQuantity,
	"normalizedUsageAmount": MetricNormalizedUsageAmount,
}

var consoleGranularities = map[string]string{
	"Hourly":  GranularityHourly,
	"Daily":   GranularityDaily,
	"Monthly": GranularityMonthly,
}

// consoleFilter mirrors one entry of the URL's `filter` JSON array.
type consoleFilter struct {
	Dimension struct {
		ID           string `json:"id"`
		DisplayValue string `json:"displayValue"`
	} `json:"dimension"`
	Operator string `json:"operator"`
	Values   []struct {
		Value        string `json:"value"`
		DisplayValue string `json:"displayValue"`
	} `json:"values"`
	// GrowableValue carries the tag key on a tag filter row; it is absent on
	// dimension rows.
	GrowableValue struct {
		Value        string `json:"value"`
		DisplayValue string `json:"displayValue"`
	} `json:"growableValue"`
}

// ParseURL converts a Cost Explorer saved-report console URL into a Spec. The
// input may be a full URL or just the part from the path onwards; what matters
// is the fragment, which is where the console keeps the report state.
func ParseURL(raw string) (*Spec, error) {
	q, err := fragmentQuery(raw)
	if err != nil {
		return nil, err
	}

	if mode := q.Get("reportMode"); mode != "" && mode != "STANDARD" {
		return nil, fmt.Errorf("report mode %q is not a cost-and-usage report; "+
			"reservation and savings-plans reports use different Cost Explorer APIs", mode)
	}
	// These two are filter toggles rather than presentation, so honouring them
	// wrong would silently change the numbers.
	for _, k := range []string{"showOnlyUntagged", "showOnlyUncategorized"} {
		if q.Get(k) == "true" {
			return nil, fmt.Errorf("%s=true is not supported yet: it needs an MatchOptionAbsent filter", k)
		}
	}

	spec := &Spec{
		Name: strings.TrimSpace(q.Get("reportName")),
		TimeRange: TimeRange{
			Relative: q.Get("historicalRelativeRange"),
			Start:    q.Get("startDate"),
			End:      q.Get("endDate"),
		},
	}
	if spec.Name == "" {
		return nil, fmt.Errorf("url has no reportName; is this a saved report url?")
	}

	// useNormalizedUnits switches the usage metric to normalized units; it wins
	// over costAggregate, which the console leaves as "undefined" in that case.
	agg := q.Get("costAggregate")
	if q.Get("useNormalizedUnits") == "true" {
		spec.Metric = MetricNormalizedUsageAmount
	} else {
		m, ok := consoleMetrics[agg]
		if !ok {
			return nil, fmt.Errorf("unknown costAggregate %q (known: %s)", agg, strings.Join(sortedKeys(consoleMetrics), ", "))
		}
		spec.Metric = m
	}

	g, ok := consoleGranularities[q.Get("granularity")]
	if !ok {
		return nil, fmt.Errorf("unknown granularity %q (known: %s)", q.Get("granularity"), strings.Join(sortedKeys(consoleGranularities), ", "))
	}
	spec.Granularity = g

	if spec.GroupBy, err = parseGroupBy(q.Get("groupBy")); err != nil {
		return nil, err
	}
	if spec.Filters, err = parseFilters(q.Get("filter")); err != nil {
		return nil, err
	}
	// The console ids above are mapped by hand; validating the result keeps a
	// wrong mapping from producing a spec that Run would reject later.
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	return spec, nil
}

// fragmentQuery pulls the query string out of the URL fragment. The console
// puts the report state after the fragment's own "?", e.g.
// /costmanagement/home?region=us-east-1#/cost-explorer?chartStyle=STACK&...
func fragmentQuery(raw string) (url.Values, error) {
	raw = strings.TrimSpace(raw)
	_, frag, ok := strings.Cut(raw, "#")
	if !ok {
		return nil, fmt.Errorf("url has no '#' fragment; the report state lives in the fragment")
	}
	_, qs, ok := strings.Cut(frag, "?")
	if !ok {
		return nil, fmt.Errorf("url fragment %q has no query string", frag)
	}
	q, err := url.ParseQuery(qs)
	if err != nil {
		return nil, fmt.Errorf("parsing report query string: %w", err)
	}
	return q, nil
}

// parseGroupBy handles the `groupBy` param, a JSON array of console dimension
// ids, e.g. ["Service"].
func parseGroupBy(raw string) ([]Group, error) {
	if raw == "" {
		return nil, nil
	}
	var ids []string
	if err := json.Unmarshal([]byte(raw), &ids); err != nil {
		return nil, fmt.Errorf("parsing groupBy %q: %w", raw, err)
	}
	groups := make([]Group, 0, len(ids))
	for _, id := range ids {
		g, err := resolveKey(id)
		if err != nil {
			return nil, fmt.Errorf("groupBy: %w", err)
		}
		groups = append(groups, g)
	}
	return groups, nil
}

// parseFilters handles the `filter` param, a JSON array of filter rows. Rows
// combine with AND, matching the console's behaviour.
func parseFilters(raw string) ([]Filter, error) {
	if raw == "" || raw == "[]" {
		return nil, nil
	}
	var cfs []consoleFilter
	if err := json.Unmarshal([]byte(raw), &cfs); err != nil {
		return nil, fmt.Errorf("parsing filter %q: %w", raw, err)
	}
	filters := make([]Filter, 0, len(cfs))
	for _, cf := range cfs {
		var f Filter
		if cf.Dimension.ID == consoleTagFilterID {
			// A tag filter row inverts what you would expect: growableValue
			// holds the tag key and values holds that tag's values. Reading it
			// the other way round produces a filter that matches nothing and
			// reports zero without erroring, so it is worth being explicit.
			if cf.GrowableValue.Value == "" {
				return nil, fmt.Errorf("tag filter row has no growableValue naming the tag key")
			}
			f = Filter{Type: TypeTag, Key: cf.GrowableValue.Value}
		} else {
			g, err := resolveKey(cf.Dimension.ID)
			if err != nil {
				return nil, fmt.Errorf("filter: %w", err)
			}
			f = Filter{Type: g.Type, Key: g.Key}
		}
		switch cf.Operator {
		case "INCLUDES":
			f.Exclude = false
		case "EXCLUDES":
			f.Exclude = true
		default:
			return nil, fmt.Errorf("filter on %s: unknown operator %q", cf.Dimension.ID, cf.Operator)
		}
		for _, v := range cf.Values {
			f.Values = append(f.Values, v.Value)
		}
		if len(f.Values) == 0 {
			return nil, fmt.Errorf("filter on %s has no values", cf.Dimension.ID)
		}
		filters = append(filters, f)
	}
	return filters, nil
}

// resolveKey turns a console dimension id into the API type/key pair. Grouping
// by a tag arrives as "TagKeyValue:<tag key>"; everything else is a dimension.
// Cost-category grouping is deliberately absent: none of the reports migrated
// so far use it, and guessing its encoding risks a silently wrong query.
func resolveKey(id string) (Group, error) {
	if k, ok := strings.CutPrefix(id, consoleTagGroupPrefix); ok {
		if k == "" {
			return Group{}, fmt.Errorf("tag group-by %q names no tag key", id)
		}
		return Group{Type: TypeTag, Key: k}, nil
	}
	if dim, ok := consoleDimensions[id]; ok {
		return Group{Type: TypeDimension, Key: dim}, nil
	}
	return Group{}, fmt.Errorf("unknown console dimension %q; add it to consoleDimensions", id)
}
