package cereport

// relativeCustom is the TimeRange.Relative value for a fixed window.
const relativeCustom = "CUSTOM"

// Spec is the checked-in definition of a Cost Explorer report and the input to
// GetCostAndUsageInput and Run. Specs are stored as a JSON array (see
// LoadSpecs) and looked up by Name (see Find).
type Spec struct {
	Name string `json:"name" yaml:"name"`

	// Metric is the Cost Explorer metric name, e.g. UnblendedCost,
	// AmortizedCost, UsageQuantity.
	Metric string `json:"metric" yaml:"metric"`
	// Granularity is HOURLY, DAILY or MONTHLY.
	Granularity string   `json:"granularity" yaml:"granularity"`
	GroupBy     []Group  `json:"groupBy,omitempty" yaml:"groupBy,omitempty"`
	Filters     []Filter `json:"filters,omitempty" yaml:"filters,omitempty"`

	TimeRange TimeRange `json:"timeRange" yaml:"timeRange"`
}

// Group is a group-by dimension, tag key or cost category, in the form the
// Cost Explorer API expects.
type Group struct {
	// Type is DIMENSION, TAG or COST_CATEGORY.
	Type string `json:"type" yaml:"type"`
	// Key is the API key, e.g. SERVICE for a DIMENSION, or the tag key itself
	// for a TAG.
	Key string `json:"key" yaml:"key"`
}

// Filter is one filter clause. Clauses combine with AND.
type Filter struct {
	// Type is DIMENSION, TAG or COST_CATEGORY.
	Type string `json:"type" yaml:"type"`
	// Key is the API dimension key (e.g. RECORD_TYPE), tag key or cost
	// category name.
	Key string `json:"key" yaml:"key"`
	// Exclude inverts the match: the clause becomes a NOT around the
	// expression.
	Exclude bool `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	// Values are API-side values (e.g. the RECORD_TYPE value "Tax"), not
	// display labels.
	Values []string `json:"values" yaml:"values"`
}

// TimeRange is the report's period. Relative names a rolling range that is
// recomputed on every run (YEAR_TO_DATE, MONTH_TO_DATE, LAST_N_DAYS,
// LAST_N_MONTHS); Start and End are read only when the range is CUSTOM.
type TimeRange struct {
	// Relative is the range name, e.g. YEAR_TO_DATE, LAST_6_MONTHS, CUSTOM.
	// Empty is treated as CUSTOM.
	Relative string `json:"relative" yaml:"relative"`
	// Start is inclusive, End exclusive, both yyyy-MM-dd. Authoritative only
	// when the range is CUSTOM.
	Start string `json:"start" yaml:"start"`
	End   string `json:"end" yaml:"end"`
}

// IsCustom reports whether the range is a fixed window rather than one that
// should be recomputed at run time.
func (t TimeRange) IsCustom() bool { return t.Relative == "" || t.Relative == relativeCustom }
