package cereport

import (
	"strings"
	"testing"
	"time"

	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

func TestExpression(t *testing.T) {
	dim := Filter{Type: "DIMENSION", Key: "SERVICE", Values: []string{"Amazon EC2"}}
	tag := Filter{Type: "TAG", Key: "repo", Values: []string{"tb-pos"}}
	cat := Filter{Type: "COST_CATEGORY", Key: "Team", Values: []string{"SRE"}}

	t.Run("no filters means no expression", func(t *testing.T) {
		e, err := Spec{}.Expression()
		if err != nil || e != nil {
			t.Fatalf("got %+v, %v; want nil, nil", e, err)
		}
	})

	t.Run("single filter is passed through without And", func(t *testing.T) {
		// Cost Explorer rejects an And with one element.
		e, err := Spec{Filters: []Filter{dim}}.Expression()
		if err != nil {
			t.Fatal(err)
		}
		if e.And != nil || e.Dimensions == nil {
			t.Fatalf("expression = %+v, want a bare Dimensions", e)
		}
		if string(e.Dimensions.Key) != "SERVICE" || len(e.Dimensions.Values) != 1 ||
			len(e.Dimensions.MatchOptions) != 1 || e.Dimensions.MatchOptions[0] != cetypes.MatchOptionEquals {
			t.Errorf("Dimensions = %+v", e.Dimensions)
		}
	})

	t.Run("filters combine with And in order", func(t *testing.T) {
		e, err := Spec{Filters: []Filter{dim, tag, cat}}.Expression()
		if err != nil {
			t.Fatal(err)
		}
		if len(e.And) != 3 {
			t.Fatalf("And has %d elements, want 3", len(e.And))
		}
		if e.And[0].Dimensions == nil {
			t.Errorf("And[0] = %+v, want Dimensions", e.And[0])
		}
		if e.And[1].Tags == nil || *e.And[1].Tags.Key != "repo" || e.And[1].Tags.Values[0] != "tb-pos" ||
			e.And[1].Tags.MatchOptions[0] != cetypes.MatchOptionEquals {
			t.Errorf("And[1] = %+v, want Tags repo=tb-pos", e.And[1])
		}
		if e.And[2].CostCategories == nil || *e.And[2].CostCategories.Key != "Team" ||
			e.And[2].CostCategories.Values[0] != "SRE" {
			t.Errorf("And[2] = %+v, want CostCategories Team=SRE", e.And[2])
		}
	})

	t.Run("exclude wraps the row in Not", func(t *testing.T) {
		x := tag
		x.Exclude = true
		e, err := Spec{Filters: []Filter{x}}.Expression()
		if err != nil {
			t.Fatal(err)
		}
		if e.Not == nil || e.Not.Tags == nil || e.Tags != nil {
			t.Fatalf("expression = %+v, want Not{Tags}", e)
		}
	})

	t.Run("unknown filter type is an error", func(t *testing.T) {
		_, err := Spec{Filters: []Filter{{Type: "USAGE_TYPE_GROUP", Key: "k", Values: []string{"v"}}}}.Expression()
		if err == nil || !strings.Contains(err.Error(), "unknown type") {
			t.Fatalf("err = %v, want an unknown-type error", err)
		}
	})
}

func TestResolvePeriod_Errors(t *testing.T) {
	now := time.Date(2026, 8, 30, 14, 0, 0, 0, time.UTC)
	tests := []struct {
		name string
		tr   TimeRange
		want string
	}{
		{"custom without dates", TimeRange{Relative: "CUSTOM"}, "custom range with no start/end"},
		{"empty relative without dates", TimeRange{Start: "2026-01-01"}, "custom range with no start/end"},
		{"unhandled relative range", TimeRange{Relative: "LAST_WEEK"}, `unhandled relative range "LAST_WEEK"`},
		{"malformed rolling range", TimeRange{Relative: "LAST_X_DAYS"}, "unhandled relative range"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := Spec{Name: "rpt", TimeRange: tt.tr}.ResolvePeriod(now)
			if err == nil || !strings.Contains(err.Error(), tt.want) || !strings.Contains(err.Error(), `"rpt"`) {
				t.Fatalf("err = %v, want one naming the report and containing %q", err, tt.want)
			}
		})
	}
}

// TestResolvePeriod_RollingRanges_UTC: rolling ranges resolve against the UTC
// date, since Cost Explorer reckons days in UTC. Late evening in Toronto is
// already the next day in UTC.
func TestResolvePeriod_RollingRanges_UTC(t *testing.T) {
	toronto := time.FixedZone("EDT", -4*3600)
	now := time.Date(2026, 8, 30, 23, 59, 0, 0, toronto) // 2026-08-31 03:59 UTC
	tests := []struct {
		relative   string
		start, end string
	}{
		{"LAST_30_DAYS", "2026-08-01", "2026-08-31"},
		{"LAST_3_MONTHS", "2026-05-01", "2026-08-31"},
		{"MONTH_TO_DATE", "2026-08-01", "2026-08-31"},
	}
	for _, tt := range tests {
		t.Run(tt.relative, func(t *testing.T) {
			s := Spec{Name: "t", TimeRange: TimeRange{Relative: tt.relative}}
			start, end, err := s.ResolvePeriod(now, ExcludeCurrentDay())
			if err != nil {
				t.Fatalf("ResolvePeriod: %v", err)
			}
			if start != tt.start || end != tt.end {
				t.Errorf("got %s..%s, want %s..%s", start, end, tt.start, tt.end)
			}
		})
	}
}

func TestGetCostAndUsageInput_MultiGroupNoFilter(t *testing.T) {
	s := Spec{
		Name: "g", Metric: "AmortizedCost", Granularity: "DAILY",
		GroupBy:   []Group{{Type: "DIMENSION", Key: "SERVICE"}, {Type: "TAG", Key: "repo"}},
		TimeRange: TimeRange{Relative: "CUSTOM", Start: "2026-01-01", End: "2026-02-01"},
	}
	in, err := s.GetCostAndUsageInput(time.Now())
	if err != nil {
		t.Fatalf("GetCostAndUsageInput: %v", err)
	}
	if in.Filter != nil {
		t.Errorf("Filter = %+v, want nil for a report with no filters", in.Filter)
	}
	if in.Granularity != cetypes.GranularityDaily || in.Metrics[0] != "AmortizedCost" {
		t.Errorf("Granularity/Metrics = %s/%v", in.Granularity, in.Metrics)
	}
	if *in.TimePeriod.Start != "2026-01-01" || *in.TimePeriod.End != "2026-02-01" {
		t.Errorf("period = %s..%s", *in.TimePeriod.Start, *in.TimePeriod.End)
	}
	if len(in.GroupBy) != 2 ||
		in.GroupBy[0].Type != cetypes.GroupDefinitionTypeDimension || *in.GroupBy[0].Key != "SERVICE" ||
		in.GroupBy[1].Type != cetypes.GroupDefinitionTypeTag || *in.GroupBy[1].Key != "repo" {
		t.Errorf("GroupBy = %+v, want DIMENSION/SERVICE then TAG/repo", in.GroupBy)
	}
	if in.NextPageToken != nil {
		t.Errorf("NextPageToken = %q on the first request, want nil", *in.NextPageToken)
	}
}

func TestGetCostAndUsageInput_Errors(t *testing.T) {
	t.Run("period", func(t *testing.T) {
		_, err := Spec{Name: "p", Metric: "AmortizedCost", TimeRange: TimeRange{Relative: "CUSTOM"}}.
			GetCostAndUsageInput(time.Now())
		if err == nil {
			t.Fatal("expected the period error to propagate")
		}
	})
	t.Run("filter", func(t *testing.T) {
		s := Spec{
			Name: "f", Metric: "AmortizedCost", Granularity: "MONTHLY",
			TimeRange: TimeRange{Relative: "CUSTOM", Start: "2026-01-01", End: "2026-02-01"},
			Filters:   []Filter{{Type: "BOGUS", Key: "k", Values: []string{"v"}}},
		}
		_, err := s.GetCostAndUsageInput(time.Now())
		if err == nil || !strings.Contains(err.Error(), "unknown type") {
			t.Fatalf("err = %v, want the expression error to propagate", err)
		}
	})
}

func TestTimeRange_IsCustom(t *testing.T) {
	tests := []struct {
		relative string
		want     bool
	}{
		{"", true},
		{"CUSTOM", true},
		{"YEAR_TO_DATE", false},
		{"LAST_6_MONTHS", false},
	}
	for _, tt := range tests {
		if got := (TimeRange{Relative: tt.relative}).IsCustom(); got != tt.want {
			t.Errorf("IsCustom(%q) = %v, want %v", tt.relative, got, tt.want)
		}
	}
}

// ytdByService mirrors the ytd_aws_by_service_unblended report: monthly
// unblended cost by service, year to date, with non-usage charge types
// excluded. It doubles as the end-to-end fixture for input building.
var ytdByService = Spec{
	Name: "ytd_aws_by_service_unblended", Metric: "UnblendedCost", Granularity: "MONTHLY",
	GroupBy: []Group{{Type: "DIMENSION", Key: "SERVICE"}},
	Filters: []Filter{{
		Type: "DIMENSION", Key: "RECORD_TYPE", Exclude: true,
		Values: []string{"Distributor Discount", "Refund", "Tax"},
	}},
	TimeRange: TimeRange{Relative: "YEAR_TO_DATE", Start: "2026-01-01", End: "2026-08-30"},
}

func TestResolvePeriod(t *testing.T) {
	now := time.Date(2026, 8, 30, 14, 0, 0, 0, time.UTC)
	// End is exclusive and to-date ranges include today, matching the console:
	// a report run on the 30th covers through the 30th, so End is the 31st.
	tests := []struct {
		relative, start, end string
	}{
		{"YEAR_TO_DATE", "2026-01-01", "2026-08-31"},
		{"MONTH_TO_DATE", "2026-08-01", "2026-08-31"},
		{"LAST_7_DAYS", "2026-08-23", "2026-08-31"},
		{"LAST_6_MONTHS", "2026-02-01", "2026-08-31"},
	}
	for _, tt := range tests {
		t.Run(tt.relative, func(t *testing.T) {
			s := Spec{Name: "t", TimeRange: TimeRange{Relative: tt.relative}}
			start, end, err := s.ResolvePeriod(now)
			if err != nil {
				t.Fatalf("ResolvePeriod: %v", err)
			}
			if start != tt.start || end != tt.end {
				t.Errorf("got %s..%s, want %s..%s", start, end, tt.start, tt.end)
			}
		})
	}
}

// TestResolvePeriod_ExcludeCurrentDay pins the opt-out used for stable
// snapshots, where today's still-settling data would otherwise move the
// numbers between runs.
func TestResolvePeriod_ExcludeCurrentDay(t *testing.T) {
	s := Spec{Name: "t", TimeRange: TimeRange{Relative: "YEAR_TO_DATE"}}
	start, end, err := s.ResolvePeriod(time.Date(2026, 8, 30, 14, 0, 0, 0, time.UTC), ExcludeCurrentDay())
	if err != nil {
		t.Fatalf("ResolvePeriod: %v", err)
	}
	if start != "2026-01-01" || end != "2026-08-30" {
		t.Errorf("got %s..%s, want 2026-01-01..2026-08-30", start, end)
	}
}

// TestResolvePeriod_CustomUsesStartEnd: a CUSTOM range is a fixed window and
// ignores the current time entirely.
func TestResolvePeriod_CustomUsesStartEnd(t *testing.T) {
	s := Spec{Name: "t", TimeRange: TimeRange{Relative: "CUSTOM", Start: "2025-01-01", End: "2025-02-01"}}
	start, end, err := s.ResolvePeriod(time.Now())
	if err != nil {
		t.Fatalf("ResolvePeriod: %v", err)
	}
	if start != "2025-01-01" || end != "2025-02-01" {
		t.Errorf("got %s..%s, want the spec's own window", start, end)
	}
}

// TestGetCostAndUsageInput_ExcludeBecomesNot: for a relative range the spec's
// stored Start/End are ignored in favour of the resolved period, and an
// excluding filter becomes a NOT around the dimension.
func TestGetCostAndUsageInput_ExcludeBecomesNot(t *testing.T) {
	in, err := ytdByService.GetCostAndUsageInput(time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("GetCostAndUsageInput: %v", err)
	}
	if *in.TimePeriod.Start != "2026-01-01" || *in.TimePeriod.End != "2026-08-31" {
		t.Errorf("period = %s..%s", *in.TimePeriod.Start, *in.TimePeriod.End)
	}
	if string(in.Granularity) != "MONTHLY" {
		t.Errorf("granularity = %s", in.Granularity)
	}
	if len(in.Metrics) != 1 || in.Metrics[0] != "UnblendedCost" {
		t.Errorf("metrics = %v", in.Metrics)
	}
	if len(in.GroupBy) != 1 || *in.GroupBy[0].Key != "SERVICE" {
		t.Errorf("groupBy = %+v", in.GroupBy)
	}
	if in.Filter == nil || in.Filter.Not == nil || in.Filter.Not.Dimensions == nil {
		t.Fatalf("filter = %+v, want a NOT around a dimension", in.Filter)
	}
	if got := string(in.Filter.Not.Dimensions.Key); got != "RECORD_TYPE" {
		t.Errorf("filter dimension = %s", got)
	}
	if len(in.Filter.Not.Dimensions.Values) != 3 {
		t.Errorf("filter values = %v", in.Filter.Not.Dimensions.Values)
	}
}
