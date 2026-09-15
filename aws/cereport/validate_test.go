package cereport

import (
	"errors"
	"strings"
	"testing"
)

// validSpec is a fully valid report; each rule test breaks exactly one thing.
func validSpec() Spec {
	return Spec{
		Name: "ytd_by_service", Metric: MetricUnblendedCost, Granularity: GranularityMonthly,
		GroupBy:   []Group{{Type: TypeDimension, Key: "SERVICE"}},
		Filters:   []Filter{{Type: TypeDimension, Key: "RECORD_TYPE", Exclude: true, Values: []string{"Tax"}}},
		TimeRange: TimeRange{Relative: RangeYearToDate},
	}
}

func TestValidate_OK(t *testing.T) {
	specs := map[string]Spec{
		"dimension group, dimension filter, year to date": validSpec(),
		"two groups incl. tag; tag and cost-category filters; custom window": {
			Name: "q1", Metric: MetricAmortizedCost, Granularity: GranularityMonthly,
			GroupBy: []Group{{Type: TypeDimension, Key: "SERVICE"}, {Type: TypeTag, Key: "repo"}},
			Filters: []Filter{
				{Type: TypeTag, Key: "repo", Values: []string{"tb-pos"}},
				{Type: TypeCostCategory, Key: "Team", Values: []string{"SRE"}},
			},
			TimeRange: TimeRange{Relative: RangeCustom, Start: "2026-01-01", End: "2026-04-01"},
		},
		"totals only, last 30 days, hourly usage": {
			Name: "usage", Metric: MetricUsageQuantity, Granularity: GranularityHourly,
			TimeRange: TimeRange{Relative: RangeLastDays(30)},
		},
		"empty relative with dates is custom": {
			Name: "fixed", Metric: MetricBlendedCost, Granularity: GranularityDaily,
			TimeRange: TimeRange{Start: "2026-02-01", End: "2026-03-01"},
		},
		"relative range with informational dates": {
			Name: "six", Metric: MetricNetAmortizedCost, Granularity: GranularityMonthly,
			TimeRange: TimeRange{Relative: RangeLastMonths(6), Start: "2026-02-01", End: "2026-08-31"},
		},
	}
	for name, s := range specs {
		t.Run(name, func(t *testing.T) {
			if err := s.Validate(); err != nil {
				t.Fatalf("Validate: %v", err)
			}
		})
	}
}

func TestValidate_Rules(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*Spec)
		want   string
	}{
		{"name required", func(s *Spec) { s.Name = "  " }, "name is required"},
		// The SDK's Metric enum spelling is not what GetCostAndUsage accepts.
		{"unknown metric", func(s *Spec) { s.Metric = "UNBLENDED_COST" }, `metric "UNBLENDED_COST": want one of AmortizedCost, BlendedCost`},
		{"unknown granularity", func(s *Spec) { s.Granularity = "Monthly" }, `granularity "Monthly": want one of DAILY, HOURLY, MONTHLY`},
		{"too many groups", func(s *Spec) {
			s.GroupBy = []Group{{Type: TypeDimension, Key: "SERVICE"}, {Type: TypeTag, Key: "repo"}, {Type: TypeDimension, Key: "REGION"}}
		}, "groupBy has 3 entries; Cost Explorer allows at most 2"},
		{"group unknown type", func(s *Spec) { s.GroupBy[0].Type = "USAGE_TYPE_GROUP" }, `groupBy[0]: unknown type "USAGE_TYPE_GROUP"`},
		// The console's id for availability zone is not the API's.
		{"group unknown dimension", func(s *Spec) { s.GroupBy[0].Key = "AVAILABILITY_ZONE" }, `groupBy[0]: unknown dimension "AVAILABILITY_ZONE"; known: `},
		{"tag group without key", func(s *Spec) { s.GroupBy[0] = Group{Type: TypeTag} }, "groupBy[0]: key is required for type TAG"},
		{"filter unknown type", func(s *Spec) { s.Filters[0].Type = "BOGUS" }, `filters[0]: unknown type "BOGUS"`},
		{"filter unknown dimension", func(s *Spec) { s.Filters[0].Key = "Charge type" }, `filters[0]: unknown dimension "Charge type"`},
		{"cost category filter without key", func(s *Spec) { s.Filters[0] = Filter{Type: TypeCostCategory, Values: []string{"x"}} }, "filters[0]: key is required for type COST_CATEGORY"},
		{"filter without values", func(s *Spec) { s.Filters[0].Values = nil }, "filters[0]: values is empty"},
		{"custom without dates", func(s *Spec) { s.TimeRange = TimeRange{Relative: RangeCustom} }, "a CUSTOM range needs start and end"},
		{"unknown relative", func(s *Spec) { s.TimeRange.Relative = "LAST_WEEK" }, `timeRange.relative "LAST_WEEK": want CUSTOM, YEAR_TO_DATE, MONTH_TO_DATE`},
		{"zero-length rolling range", func(s *Spec) { s.TimeRange.Relative = RangeLastDays(0) }, `timeRange.relative "LAST_0_DAYS": n must be at least 1`},
		{"bad start", func(s *Spec) { s.TimeRange = TimeRange{Relative: RangeCustom, Start: "2026-1-5", End: "2026-02-01"} }, `timeRange.start "2026-1-5": want yyyy-MM-dd`},
		{"bad end", func(s *Spec) { s.TimeRange = TimeRange{Relative: RangeCustom, Start: "2026-01-01", End: "02/01/2026"} }, `timeRange.end "02/01/2026": want yyyy-MM-dd`},
		{"start not before end", func(s *Spec) { s.TimeRange = TimeRange{Relative: RangeCustom, Start: "2026-02-01", End: "2026-02-01"} }, "start 2026-02-01 must be before end 2026-02-01"},
		{"informational dates still checked", func(s *Spec) { s.TimeRange.Start = "yesterday" }, `timeRange.start "yesterday"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := validSpec()
			tt.mutate(&s)
			err := s.Validate()
			if err == nil {
				t.Fatal("expected an error, got none")
			}
			if !strings.Contains(err.Error(), tt.want) {
				t.Errorf("err = %q\nwant it to contain %q", err, tt.want)
			}
			if !strings.HasPrefix(err.Error(), "report ") {
				t.Errorf("err = %q, want it to name the report", err)
			}
		})
	}
}

// TestValidate_ReportsEveryProblem: a checked-in file with several mistakes
// should surface all of them at once, not one per run.
func TestValidate_ReportsEveryProblem(t *testing.T) {
	err := Spec{}.Validate() // name, metric, granularity, custom range without dates
	if err == nil {
		t.Fatal("expected errors")
	}
	joined, ok := errors.Unwrap(err).(interface{ Unwrap() []error })
	if !ok {
		t.Fatalf("err = %T, want a wrapped errors.Join", errors.Unwrap(err))
	}
	if got := len(joined.Unwrap()); got != 4 {
		t.Errorf("got %d problems, want 4:\n%v", got, err)
	}
}

func TestRangeHelpers(t *testing.T) {
	if got := RangeLastDays(30); got != "LAST_30_DAYS" {
		t.Errorf("RangeLastDays(30) = %q", got)
	}
	if got := RangeLastMonths(6); got != "LAST_6_MONTHS" {
		t.Errorf("RangeLastMonths(6) = %q", got)
	}
}

func TestKnownValueLists(t *testing.T) {
	has := func(list []string, v string) bool {
		for _, x := range list {
			if x == v {
				return true
			}
		}
		return false
	}
	if m := Metrics(); len(m) != 7 || !has(m, MetricUnblendedCost) || !has(m, MetricNormalizedUsageAmount) {
		t.Errorf("Metrics() = %v", m)
	}
	if g := Granularities(); len(g) != 3 || g[0] != "DAILY" || g[1] != "HOURLY" || g[2] != "MONTHLY" {
		t.Errorf("Granularities() = %v, want sorted DAILY HOURLY MONTHLY", g)
	}
	d := Dimensions()
	if len(d) < 30 || !has(d, "SERVICE") || !has(d, "AZ") || has(d, "AVAILABILITY_ZONE") {
		t.Errorf("Dimensions() = %v", d)
	}
	if !sortedStrings(d) {
		t.Error("Dimensions() is not sorted")
	}
}

func sortedStrings(ss []string) bool {
	for i := 1; i < len(ss); i++ {
		if ss[i-1] > ss[i] {
			return false
		}
	}
	return true
}

// TestGetCostAndUsageInput_ValidatesFirst: an invalid spec is rejected before
// any request is built, with the validation message.
func TestGetCostAndUsageInput_ValidatesFirst(t *testing.T) {
	s := validSpec()
	s.Metric = "Cost"
	_, err := s.GetCostAndUsageInput(timeNow())
	if err == nil || !strings.Contains(err.Error(), `metric "Cost"`) {
		t.Fatalf("err = %v, want the validation error", err)
	}
}
