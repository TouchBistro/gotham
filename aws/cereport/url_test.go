package cereport

import (
	"net/url"
	"reflect"
	"strings"
	"testing"
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
	if spec.Metric != "UnblendedCost" {
		t.Errorf("Metric = %q, want UnblendedCost", spec.Metric)
	}
	if spec.Granularity != "MONTHLY" {
		t.Errorf("Granularity = %q, want MONTHLY", spec.Granularity)
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

// TestResolvePeriod_ExcludeCurrentDay pins the opt-out used for stable
// fragmentURL builds a console-shaped URL from report-state parameters, so a
// fixture reads as key/value pairs instead of several KB of escaped JSON.
// Parameters are merged over a minimal valid report (monthly, unblended,
// year-to-date), so each test spells out only what it is about.
func fragmentURL(params map[string]string) string {
	q := url.Values{
		"reportName":              {"r"},
		"costAggregate":           {"unBlendedCost"},
		"granularity":             {"Monthly"},
		"historicalRelativeRange": {"YEAR_TO_DATE"},
		"startDate":               {"2026-01-01"},
		"endDate":                 {"2026-08-30"},
	}
	for k, v := range params {
		q.Set(k, v)
	}
	return "/costmanagement/home?region=us-east-1#/cost-explorer?" + q.Encode()
}

// TestParseURL_FullURL: scheme and host are irrelevant, only the fragment
// carries report state.
func TestParseURL_FullURL(t *testing.T) {
	spec, err := ParseURL("https://us-east-1.console.aws.amazon.com" + fragmentURL(nil))
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	if spec.Name != "r" || spec.Metric != "UnblendedCost" || spec.Granularity != "MONTHLY" {
		t.Errorf("spec = %+v", spec)
	}
	if spec.GroupBy != nil || spec.Filters != nil {
		t.Errorf("GroupBy/Filters = %v/%v, want none", spec.GroupBy, spec.Filters)
	}
	if spec.TimeRange != (TimeRange{Relative: "YEAR_TO_DATE", Start: "2026-01-01", End: "2026-08-30"}) {
		t.Errorf("TimeRange = %+v", spec.TimeRange)
	}
}

// TestParseURL_TagGroupBy pins the console's "TagKeyValue:<key>" encoding for
// grouping by tag, previously covered only by the ytd_ecs_by_repo_amortized
// golden report.
func TestParseURL_TagGroupBy(t *testing.T) {
	spec, err := ParseURL(fragmentURL(map[string]string{"groupBy": `["TagKeyValue:repo","Service"]`}))
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	want := []Group{{Type: "TAG", Key: "repo"}, {Type: "DIMENSION", Key: "SERVICE"}}
	if !reflect.DeepEqual(spec.GroupBy, want) {
		t.Errorf("GroupBy = %+v, want %+v", spec.GroupBy, want)
	}
}

// TestParseURL_TagFilterRow pins the tag filter row shape: the console puts the
// tag key in growableValue and the tag's values in values. Inverting them
// yields a filter that matches nothing and reports zero without erroring.
// Previously covered only by the ytd_venue_ark_by_service_amortized golden
// report.
func TestParseURL_TagFilterRow(t *testing.T) {
	filter := `[
	  {"dimension":{"id":"TagKey","displayValue":"Tag"},"operator":"INCLUDES",
	   "values":[{"value":"venue","displayValue":"venue"}],
	   "growableValue":{"value":"groupid","displayValue":"groupid"}},
	  {"dimension":{"id":"TagKey","displayValue":"Tag"},"operator":"EXCLUDES",
	   "values":[{"value":"ark","displayValue":"ark"},{"value":"pos","displayValue":"pos"}],
	   "growableValue":{"value":"serviceid","displayValue":"serviceid"}},
	  {"dimension":{"id":"Region","displayValue":"Region"},"operator":"INCLUDES",
	   "values":[{"value":"us-east-1","displayValue":"US East (N. Virginia)"}]}
	]`
	spec, err := ParseURL(fragmentURL(map[string]string{"filter": filter}))
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	want := []Filter{
		{Type: "TAG", Key: "groupid", Values: []string{"venue"}},
		{Type: "TAG", Key: "serviceid", Exclude: true, Values: []string{"ark", "pos"}},
		{Type: "DIMENSION", Key: "REGION", Values: []string{"us-east-1"}},
	}
	if !reflect.DeepEqual(spec.Filters, want) {
		t.Errorf("Filters = %+v, want %+v", spec.Filters, want)
	}
}

// TestParseURL_TrimsName: several reports were saved with a leading space in
// the console; the name becomes a filename downstream.
func TestParseURL_TrimsName(t *testing.T) {
	spec, err := ParseURL(fragmentURL(map[string]string{"reportName": "  ytd_by_service "}))
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	if spec.Name != "ytd_by_service" {
		t.Errorf("Name = %q, want trimmed", spec.Name)
	}
}

func TestParseURL_EmptyGroupByAndFilter(t *testing.T) {
	spec, err := ParseURL(fragmentURL(map[string]string{"groupBy": "", "filter": "[]"}))
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	if spec.GroupBy != nil || spec.Filters != nil {
		t.Errorf("GroupBy/Filters = %v/%v, want nil (totals-only report)", spec.GroupBy, spec.Filters)
	}
}

// TestParseURL_NormalizedUnits: useNormalizedUnits wins over costAggregate,
// which the console leaves as "undefined" in that case.
func TestParseURL_NormalizedUnits(t *testing.T) {
	spec, err := ParseURL(fragmentURL(map[string]string{
		"useNormalizedUnits": "true", "costAggregate": "undefined",
	}))
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	if spec.Metric != "NormalizedUsageAmount" {
		t.Errorf("Metric = %q, want NormalizedUsageAmount", spec.Metric)
	}
}

func TestParseURL_CustomRange(t *testing.T) {
	spec, err := ParseURL(fragmentURL(map[string]string{
		"historicalRelativeRange": "CUSTOM", "startDate": "2025-01-01", "endDate": "2025-02-01",
	}))
	if err != nil {
		t.Fatalf("ParseURL: %v", err)
	}
	if !spec.TimeRange.IsCustom() || spec.TimeRange.Start != "2025-01-01" || spec.TimeRange.End != "2025-02-01" {
		t.Errorf("TimeRange = %+v, want a custom 2025-01-01..2025-02-01 window", spec.TimeRange)
	}
}

func TestParseURL_RejectsMore(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"fragment without query", "/costmanagement/home#/cost-explorer"},
		{"invalid query escape", "/x#/cost-explorer?reportName=%zz"},
		{"malformed groupBy json", fragmentURL(map[string]string{"groupBy": `["Service"`})},
		{"malformed filter json", fragmentURL(map[string]string{"filter": `[{`})},
		{"tag group-by with empty key", fragmentURL(map[string]string{"groupBy": `["TagKeyValue:"]`})},
		{"tag filter row without growableValue", fragmentURL(map[string]string{
			"filter": `[{"dimension":{"id":"TagKey"},"operator":"INCLUDES","values":[{"value":"x"}]}]`})},
		{"filter on unknown dimension", fragmentURL(map[string]string{
			"filter": `[{"dimension":{"id":"Bogus"},"operator":"INCLUDES","values":[{"value":"x"}]}]`})},
		{"filter with unknown operator", fragmentURL(map[string]string{
			"filter": `[{"dimension":{"id":"Service"},"operator":"CONTAINS","values":[{"value":"x"}]}]`})},
		{"filter with no values", fragmentURL(map[string]string{
			"filter": `[{"dimension":{"id":"Service"},"operator":"INCLUDES","values":[]}]`})},
		{"uncategorized only", fragmentURL(map[string]string{"showOnlyUncategorized": "true"})},
		{"savings plans report", fragmentURL(map[string]string{"reportMode": "SAVINGS_PLANS"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := ParseURL(tt.url); err == nil {
				t.Fatal("expected an error, got none")
			}
		})
	}
}

// TestConsoleDimensions_AreAPIDimensions: every console id must map to a real
// Cost Explorer dimension, or Validate rejects the parsed spec. The original
// table mapped AZ to AVAILABILITY_ZONE, which the API does not know.
func TestConsoleDimensions_AreAPIDimensions(t *testing.T) {
	for id, dim := range consoleDimensions {
		if !knownDimensions[dim] {
			t.Errorf("console id %q maps to %q, which is not a Cost Explorer dimension", id, dim)
		}
	}
	if got := consoleDimensions["AZ"]; got != "AZ" {
		t.Errorf("AZ maps to %q, want AZ", got)
	}
}

// TestParseURL_ValidatesResult: a URL that parses but yields an invalid spec is
// rejected by ParseURL itself, not later by Run.
func TestParseURL_ValidatesResult(t *testing.T) {
	_, err := ParseURL(fragmentURL(map[string]string{"startDate": "2026-08-30", "endDate": "2026-01-01"}))
	if err == nil || !strings.Contains(err.Error(), "must be before") {
		t.Fatalf("err = %v, want the validation error", err)
	}
}
