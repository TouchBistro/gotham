package cereport

import "fmt"

// Group and Filter types.
const (
	TypeDimension    = "DIMENSION"
	TypeTag          = "TAG"
	TypeCostCategory = "COST_CATEGORY"
)

// Metrics accepted by GetCostAndUsage. A Spec names exactly one.
const (
	MetricUnblendedCost         = "UnblendedCost"
	MetricBlendedCost           = "BlendedCost"
	MetricAmortizedCost         = "AmortizedCost"
	MetricNetUnblendedCost      = "NetUnblendedCost"
	MetricNetAmortizedCost      = "NetAmortizedCost"
	MetricUsageQuantity         = "UsageQuantity"
	MetricNormalizedUsageAmount = "NormalizedUsageAmount"
)

// Granularities.
const (
	GranularityHourly  = "HOURLY"
	GranularityDaily   = "DAILY"
	GranularityMonthly = "MONTHLY"
)

// Relative time ranges. Rolling ranges are built with RangeLastDays and
// RangeLastMonths.
const (
	RangeCustom      = "CUSTOM"
	RangeYearToDate  = "YEAR_TO_DATE"
	RangeMonthToDate = "MONTH_TO_DATE"
)

// RangeLastDays returns the LAST_<n>_DAYS range name: the n days before today,
// plus today unless ExcludeCurrentDay is passed.
func RangeLastDays(n int) string { return fmt.Sprintf("LAST_%d_DAYS", n) }

// RangeLastMonths returns the LAST_<n>_MONTHS range name: from the first of
// the month n months ago through today, so buckets line up with calendar
// months under MONTHLY granularity.
func RangeLastMonths(n int) string { return fmt.Sprintf("LAST_%d_MONTHS", n) }

// dateLayout is the yyyy-MM-dd form Cost Explorer uses for period bounds.
const dateLayout = "2006-01-02"

// Spec is the definition of a Cost Explorer report and the input to
// GetCostAndUsageInput and Run. Its JSON tags let definitions live in a
// checked-in file; Validate checks one against the rules Cost Explorer
// enforces.
type Spec struct {
	// Name identifies the report in errors and, for CLIs, output filenames.
	Name string `json:"name" yaml:"name"`

	// Metric is one of the Metric* constants.
	Metric string `json:"metric" yaml:"metric"`
	// Granularity is one of the Granularity* constants.
	Granularity string `json:"granularity" yaml:"granularity"`
	// GroupBy has at most two entries. Omit it for period totals only.
	GroupBy []Group `json:"groupBy,omitempty" yaml:"groupBy,omitempty"`
	// Filters combine with AND.
	Filters []Filter `json:"filters,omitempty" yaml:"filters,omitempty"`

	TimeRange TimeRange `json:"timeRange" yaml:"timeRange"`
}

// Group is a group-by dimension, tag key or cost category, in the form the
// Cost Explorer API expects.
type Group struct {
	// Type is one of the Type* constants.
	Type string `json:"type" yaml:"type"`
	// Key is the dimension name (see Dimensions) for a DIMENSION, the tag key
	// for a TAG, or the cost category name for a COST_CATEGORY.
	Key string `json:"key" yaml:"key"`
}

// Filter is one filter clause. Clauses combine with AND.
type Filter struct {
	// Type is one of the Type* constants.
	Type string `json:"type" yaml:"type"`
	// Key is the dimension name (see Dimensions) for a DIMENSION, the tag key
	// for a TAG, or the cost category name for a COST_CATEGORY.
	Key string `json:"key" yaml:"key"`
	// Exclude inverts the match: the clause becomes a NOT around the
	// expression.
	Exclude bool `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	// Values are API-side values (e.g. the RECORD_TYPE value "Tax"), not
	// console display labels. At least one is required.
	Values []string `json:"values" yaml:"values"`
}

// TimeRange is the report's period. Relative names a range that is recomputed
// on every run (see the Range* constants and RangeLastDays/RangeLastMonths);
// Start and End are read only when the range is CUSTOM.
type TimeRange struct {
	// Relative is one of the Range* constants or a RangeLastDays /
	// RangeLastMonths value. Empty is treated as CUSTOM.
	Relative string `json:"relative" yaml:"relative"`
	// Start is inclusive, End exclusive, both yyyy-MM-dd. Authoritative only
	// when the range is CUSTOM.
	Start string `json:"start" yaml:"start"`
	End   string `json:"end" yaml:"end"`
}

// IsCustom reports whether the range is a fixed window rather than one that
// should be recomputed at run time.
func (t TimeRange) IsCustom() bool { return t.Relative == "" || t.Relative == RangeCustom }
