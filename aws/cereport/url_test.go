package cereport

import (
	"testing"
	"time"
)

// ytdByServiceURL is the console URL for the ytd_aws_by_service_unblended saved
// report, captured verbatim from the Cost Explorer console.
const ytdByServiceURL = `/costmanagement/home?region=us-east-1#/cost-explorer?chartStyle=STACK&costAggregate=unBlendedCost&endDate=2026-08-30&excludeForecasting=false&filter=%5B%7B"dimension":%7B"id":"RecordTypeV2","displayValue":"Charge%20type"%7D,"operator":"EXCLUDES","values":%5B%7B"value":"Distributor%20Discount","displayValue":"Distributor%20Discount"%7D,%7B"value":"Refund","displayValue":"Refund"%7D,%7B"value":"Tax","displayValue":"Tax"%7D%5D%7D%5D&futureRelativeRange=CUSTOM&granularity=Monthly&groupBy=%5B"Service"%5D&historicalRelativeRange=YEAR_TO_DATE&isDefault=false&reportId=11454ee4-7df0-43d1-bef4-d1b59257105c&reportMode=STANDARD&reportName=ytd_aws_by_service_unblended&showOnlyUncategorized=false&showOnlyUntagged=false&startDate=2026-01-01&usageAggregate=undefined&useNormalizedUnits=false&reportArn=arn:aws:ce::651264383976:ce-saved-report%2F11454ee4-7df0-43d1-bef4-d1b59257105c`

func TestParseURL_YTDByService(t *testing.T) {
	spec, err := ParseURL(ytdByServiceURL)
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}

	if spec.Name != "ytd_aws_by_service_unblended" {
		t.Errorf("Name = %q", spec.Name)
	}
	if spec.ReportID != "11454ee4-7df0-43d1-bef4-d1b59257105c" {
		t.Errorf("ReportID = %q", spec.ReportID)
	}
	if spec.Metric != "UnblendedCost" {
		t.Errorf("Metric = %q, want UnblendedCost", spec.Metric)
	}
	if spec.Granularity != "MONTHLY" {
		t.Errorf("Granularity = %q, want MONTHLY", spec.Granularity)
	}
	if spec.ChartStyle != "STACK" {
		t.Errorf("ChartStyle = %q", spec.ChartStyle)
	}

	if len(spec.GroupBy) != 1 || spec.GroupBy[0].Type != "DIMENSION" || spec.GroupBy[0].Key != "SERVICE" {
		t.Fatalf("GroupBy = %+v, want one DIMENSION/SERVICE", spec.GroupBy)
	}

	if len(spec.Filters) != 1 {
		t.Fatalf("Filters = %+v, want one", spec.Filters)
	}
	f := spec.Filters[0]
	if f.Type != "DIMENSION" || f.Key != "RECORD_TYPE" {
		t.Errorf("filter key = %s/%s, want DIMENSION/RECORD_TYPE", f.Type, f.Key)
	}
	if !f.Exclude {
		t.Error("filter should be an exclusion (console operator EXCLUDES)")
	}
	want := []string{"Distributor Discount", "Refund", "Tax"}
	if len(f.Values) != len(want) {
		t.Fatalf("filter values = %v, want %v", f.Values, want)
	}
	for i, v := range want {
		if f.Values[i] != v {
			t.Errorf("filter value %d = %q, want %q", i, f.Values[i], v)
		}
	}

	if spec.TimeRange.Relative != "YEAR_TO_DATE" {
		t.Errorf("TimeRange.Relative = %q", spec.TimeRange.Relative)
	}
	if spec.TimeRange.Start != "2026-01-01" || spec.TimeRange.End != "2026-08-30" {
		t.Errorf("captured range = %s..%s", spec.TimeRange.Start, spec.TimeRange.End)
	}
}

func TestParseURL_Rejects(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"no fragment", "/costmanagement/home?region=us-east-1"},
		{"no report name", "/x#/cost-explorer?granularity=Monthly"},
		{"unknown metric", `/x#/cost-explorer?reportName=r&costAggregate=bogusCost&granularity=Monthly`},
		{"unknown granularity", `/x#/cost-explorer?reportName=r&costAggregate=unBlendedCost&granularity=Yearly`},
		{"unknown dimension", `/x#/cost-explorer?reportName=r&costAggregate=unBlendedCost&granularity=Monthly&groupBy=%5B"Nonsense"%5D`},
		{"reservation report", `/x#/cost-explorer?reportName=r&reportMode=RESERVATIONS`},
		{"untagged only", `/x#/cost-explorer?reportName=r&showOnlyUntagged=true`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseURL(tt.url); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
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

func TestResolvePeriod_CustomKeepsCapturedDates(t *testing.T) {
	s := Spec{Name: "t", TimeRange: TimeRange{Relative: "CUSTOM", Start: "2025-01-01", End: "2025-02-01"}}
	start, end, err := s.ResolvePeriod(time.Now())
	if err != nil {
		t.Fatalf("ResolvePeriod: %v", err)
	}
	if start != "2025-01-01" || end != "2025-02-01" {
		t.Errorf("got %s..%s, want the captured window", start, end)
	}
}

func TestGetCostAndUsageInput_ExcludeBecomesNot(t *testing.T) {
	spec, err := ParseURL(ytdByServiceURL)
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	in, err := spec.GetCostAndUsageInput(time.Date(2026, 8, 30, 0, 0, 0, 0, time.UTC))
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
