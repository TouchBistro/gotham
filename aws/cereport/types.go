package cereport

// Spec is the checked-in definition of a saved report. It is the source of
// truth: the console URL is parsed into a Spec once, then the Spec is what gets
// versioned, diffed and replayed.
type Spec struct {
	Name string `json:"name" yaml:"name"`
	// ReportID and ReportARN identify the console report this Spec was pulled
	// from. Kept for traceability; neither is usable via any API.
	ReportID  string `json:"reportId,omitempty" yaml:"reportId,omitempty"`
	ReportARN string `json:"reportArn,omitempty" yaml:"reportArn,omitempty"`

	Metric      string   `json:"metric" yaml:"metric"`
	Granularity string   `json:"granularity" yaml:"granularity"`
	GroupBy     []Group  `json:"groupBy,omitempty" yaml:"groupBy,omitempty"`
	Filters     []Filter `json:"filters,omitempty" yaml:"filters,omitempty"`

	TimeRange TimeRange `json:"timeRange" yaml:"timeRange"`

	// Chart style is presentation-only; retained so a re-created console report
	// or BCM dashboard widget can look like the original.
	ChartStyle string `json:"chartStyle,omitempty" yaml:"chartStyle,omitempty"`
}

// Group is a group-by dimension or tag key, already translated to the value the
// Cost Explorer API expects.
type Group struct {
	// Type is DIMENSION, TAG or COST_CATEGORY.
	Type string `json:"type" yaml:"type"`
	// Key is the API key, e.g. SERVICE for a DIMENSION, or the tag key itself
	// for a TAG.
	Key string `json:"key" yaml:"key"`
}

// Filter is one filter row from the console's filter panel.
type Filter struct {
	// Type is DIMENSION, TAG or COST_CATEGORY.
	Type string `json:"type" yaml:"type"`
	// Key is the API dimension key (e.g. RECORD_TYPE), tag key or cost
	// category name.
	Key string `json:"key" yaml:"key"`
	// Exclude inverts the match: the console's EXCLUDES operator becomes a NOT
	// around the expression.
	Exclude bool `json:"exclude,omitempty" yaml:"exclude,omitempty"`
	// Values are passed through verbatim from the console URL, which carries
	// the API-side value rather than the display label.
	Values []string `json:"values" yaml:"values"`
}

// TimeRange captures the report's period. The console stores both a relative
// range (the user's actual intent, e.g. YEAR_TO_DATE) and the absolute dates
// that range resolved to when the URL was captured. Relative wins on replay so
// that a year-to-date report stays year-to-date; the absolute dates are kept as
// a record of what the report covered at capture time.
type TimeRange struct {
	// Relative is the console range, e.g. YEAR_TO_DATE, LAST_6_MONTHS, CUSTOM.
	// When CUSTOM, Start and End are authoritative.
	Relative string `json:"relative" yaml:"relative"`
	// Start is inclusive, End exclusive, both yyyy-MM-dd, as at capture time.
	Start string `json:"start" yaml:"start"`
	End   string `json:"end" yaml:"end"`
}

// IsCustom reports whether the range is a fixed window rather than one that
// should be recomputed at run time.
func (t TimeRange) IsCustom() bool { return t.Relative == "" || t.Relative == relativeCustom }
