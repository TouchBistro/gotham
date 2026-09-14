package cereport

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type fakeCE struct {
	out []*costexplorer.GetCostAndUsageOutput
	// tokens records the NextPageToken of each request, so pagination tests
	// can check that page N+1 asked for the token page N returned.
	tokens []string
	// err, when set, is returned from every call.
	err error
}

func (f *fakeCE) GetCostAndUsage(_ context.Context, in *costexplorer.GetCostAndUsageInput,
	_ ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	f.tokens = append(f.tokens, aws.ToString(in.NextPageToken))
	if f.err != nil {
		return nil, f.err
	}
	o := f.out[0]
	f.out = f.out[1:]
	return o, nil
}

func metric(v string) map[string]cetypes.MetricValue {
	return map[string]cetypes.MetricValue{
		"AmortizedCost": {Amount: aws.String(v), Unit: aws.String("USD")},
	}
}

func period(start, end string, groups ...cetypes.Group) cetypes.ResultByTime {
	return cetypes.ResultByTime{
		TimePeriod: &cetypes.DateInterval{Start: aws.String(start), End: aws.String(end)},
		Total:      metric("0"),
		Groups:     groups,
	}
}

// TestRun_EmptyPeriodInGroupedReport pins the distinction between a report that
// has no group-by and a grouped report that simply had no spend in a period.
// Only the former gets a total row; the latter must not gain a phantom group.
func TestRun_EmptyPeriodInGroupedReport(t *testing.T) {
	spec := &Spec{
		Name: "grouped", Metric: "AmortizedCost", Granularity: "MONTHLY",
		GroupBy:   []Group{{Type: "TAG", Key: "repo"}},
		TimeRange: TimeRange{Relative: "CUSTOM", Start: "2026-01-01", End: "2026-03-01"},
	}
	api := &fakeCE{out: []*costexplorer.GetCostAndUsageOutput{{
		ResultsByTime: []cetypes.ResultByTime{
			period("2026-01-01", "2026-02-01"), // no spend at all this month
			period("2026-02-01", "2026-03-01",
				cetypes.Group{Keys: []string{"repo$tb-pos"}, Metrics: metric("12.50")}),
		},
	}}}
	res, err := Run(context.Background(), api, spec, time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if _, ok := res.Rows["(total)"]; ok {
		t.Error(`grouped report gained a phantom "(total)" row from an empty period`)
	}
	if len(res.Rows) != 1 {
		t.Fatalf("rows = %v, want just the one group", res.Rows)
	}
	row := res.Rows["repo$tb-pos"]
	if len(row) != 2 || row[0] != 0 || row[1] != 12.50 {
		t.Errorf("row = %v, want [0 12.5]", row)
	}
}

// TestRun_UngroupedReportKeepsTotalRow is the other half: with no group-by the
// period totals are the only data there is.
func TestRun_UngroupedReportKeepsTotalRow(t *testing.T) {
	spec := &Spec{
		Name: "ungrouped", Metric: "AmortizedCost", Granularity: "MONTHLY",
		TimeRange: TimeRange{Relative: "CUSTOM", Start: "2026-01-01", End: "2026-02-01"},
	}
	p := period("2026-01-01", "2026-02-01")
	p.Total = metric("90438.93")
	api := &fakeCE{out: []*costexplorer.GetCostAndUsageOutput{{
		ResultsByTime: []cetypes.ResultByTime{p},
	}}}
	res, err := Run(context.Background(), api, spec, time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := res.Total("(total)"); got != 90438.93 {
		t.Errorf(`"(total)" = %v, want 90438.93 (rows: %v)`, got, res.Rows)
	}
}

// metricNamed is metric with a caller-chosen key, for the error paths where
// the response does not carry the metric the spec asked for.
func metricNamed(name, v string) map[string]cetypes.MetricValue {
	return map[string]cetypes.MetricValue{
		name: {Amount: aws.String(v), Unit: aws.String("USD")},
	}
}

// TestRun_Pagination pins that every page is followed and that a group seen on
// two pages is summed rather than overwritten. Cost Explorer paginates group
// results, so a report with many groups is silently short otherwise.
func TestRun_Pagination(t *testing.T) {
	spec := &Spec{
		Name: "paged", Metric: "AmortizedCost", Granularity: "MONTHLY",
		GroupBy:   []Group{{Type: "DIMENSION", Key: "SERVICE"}},
		TimeRange: TimeRange{Relative: "CUSTOM", Start: "2026-01-01", End: "2026-03-01"},
	}
	api := &fakeCE{out: []*costexplorer.GetCostAndUsageOutput{
		{
			ResultsByTime: []cetypes.ResultByTime{
				period("2026-01-01", "2026-02-01",
					cetypes.Group{Keys: []string{"EC2"}, Metrics: metric("10")}),
			},
			NextPageToken: aws.String("page-2"),
		},
		{
			// Page two continues January's groups and introduces February; an
			// empty token ends the walk just as a nil one would.
			ResultsByTime: []cetypes.ResultByTime{
				period("2026-01-01", "2026-02-01",
					cetypes.Group{Keys: []string{"EC2"}, Metrics: metric("5")}),
				period("2026-02-01", "2026-03-01",
					cetypes.Group{Keys: []string{"RDS"}, Metrics: metric("7")}),
			},
			NextPageToken: aws.String(""),
		},
	}}

	res, err := Run(context.Background(), api, spec, time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	if got := api.tokens; len(got) != 2 || got[0] != "" || got[1] != "page-2" {
		t.Errorf("request tokens = %q, want [\"\" \"page-2\"]", got)
	}
	if len(res.Periods) != 2 || res.Periods[0] != "2026-01-01" || res.Periods[1] != "2026-02-01" {
		t.Errorf("Periods = %v, want [2026-01-01 2026-02-01]", res.Periods)
	}
	if row := res.Rows["EC2"]; len(row) != 2 || row[0] != 15 || row[1] != 0 {
		t.Errorf("EC2 = %v, want [15 0] (summed across pages, zero-filled for February)", row)
	}
	if row := res.Rows["RDS"]; len(row) != 2 || row[0] != 0 || row[1] != 7 {
		t.Errorf("RDS = %v, want [0 7] (back-filled for January)", row)
	}
	if res.Unit != "USD" || res.Start != "2026-01-01" || res.End != "2026-03-01" {
		t.Errorf("Unit/Start/End = %s/%s/%s", res.Unit, res.Start, res.End)
	}
	if got := res.GrandTotal(); got != 22 {
		t.Errorf("GrandTotal = %v, want 22", got)
	}
}

func TestRun_APIError(t *testing.T) {
	spec := &Spec{
		Name: "e", Metric: "AmortizedCost", Granularity: "MONTHLY",
		TimeRange: TimeRange{Relative: "CUSTOM", Start: "2026-01-01", End: "2026-02-01"},
	}
	want := errors.New("AccessDeniedException")
	res, err := Run(context.Background(), &fakeCE{err: want}, spec, time.Now())
	if !errors.Is(err, want) {
		t.Fatalf("err = %v, want %v", err, want)
	}
	if res != nil {
		t.Errorf("Result = %+v, want nil on error", res)
	}
}

// TestRun_SpecError: a spec that cannot become a request fails before any API
// call is made.
func TestRun_SpecError(t *testing.T) {
	spec := &Spec{Name: "bad", Metric: "AmortizedCost", TimeRange: TimeRange{Relative: "CUSTOM"}}
	api := &fakeCE{}
	if _, err := Run(context.Background(), api, spec, time.Now()); err == nil {
		t.Fatal("expected an error for a custom range without dates")
	}
	if len(api.tokens) != 0 {
		t.Errorf("API was called %d times, want 0", len(api.tokens))
	}
}

func TestRun_MetricErrors(t *testing.T) {
	custom := TimeRange{Relative: "CUSTOM", Start: "2026-01-01", End: "2026-02-01"}
	grouped := &Spec{
		Name: "g", Metric: "AmortizedCost", Granularity: "MONTHLY",
		GroupBy: []Group{{Type: "DIMENSION", Key: "SERVICE"}}, TimeRange: custom,
	}
	ungrouped := &Spec{Name: "u", Metric: "AmortizedCost", Granularity: "MONTHLY", TimeRange: custom}

	totalWith := func(m map[string]cetypes.MetricValue) cetypes.ResultByTime {
		p := period("2026-01-01", "2026-02-01")
		p.Total = m
		return p
	}
	groupWith := func(m map[string]cetypes.MetricValue) cetypes.ResultByTime {
		return period("2026-01-01", "2026-02-01", cetypes.Group{Keys: []string{"EC2"}, Metrics: m})
	}

	tests := []struct {
		name    string
		spec    *Spec
		period  cetypes.ResultByTime
		wantErr string
	}{
		{"ungrouped total lacks the metric", ungrouped,
			totalWith(metricNamed("BlendedCost", "1")), `no metric "AmortizedCost"`},
		{"group amount is nil", grouped,
			groupWith(map[string]cetypes.MetricValue{"AmortizedCost": {Unit: aws.String("USD")}}),
			"has no amount"},
		{"group amount is not a number", grouped,
			groupWith(metric("twelve")), `parsing AmortizedCost amount "twelve"`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			api := &fakeCE{out: []*costexplorer.GetCostAndUsageOutput{
				{ResultsByTime: []cetypes.ResultByTime{tt.period}},
			}}
			_, err := Run(context.Background(), api, tt.spec, time.Now())
			if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
				t.Fatalf("err = %v, want one containing %q", err, tt.wantErr)
			}
		})
	}
}

func TestMetricValue_UnitOptional(t *testing.T) {
	m := map[string]cetypes.MetricValue{"UsageQuantity": {Amount: aws.String("3.5")}}
	amt, unit, err := metricValue(m, "UsageQuantity")
	if err != nil || amt != 3.5 || unit != "" {
		t.Errorf("got %v/%q/%v, want 3.5/\"\"/nil", amt, unit, err)
	}
}

func TestLoadSpecs(t *testing.T) {
	dir := t.TempDir()
	write := func(name, body string) string {
		t.Helper()
		p := filepath.Join(dir, name)
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
		return p
	}

	good := write("specs.json", `[
	  {"name":"a","metric":"UnblendedCost","granularity":"MONTHLY",
	   "groupBy":[{"type":"DIMENSION","key":"SERVICE"}],
	   "filters":[{"type":"DIMENSION","key":"RECORD_TYPE","exclude":true,"values":["Tax"]}],
	   "timeRange":{"relative":"YEAR_TO_DATE","start":"2026-01-01","end":"2026-08-30"}},
	  {"name":"b","metric":"AmortizedCost","granularity":"DAILY",
	   "timeRange":{"relative":"CUSTOM","start":"2025-01-01","end":"2025-02-01"}}
	]`)
	specs, err := LoadSpecs(good)
	if err != nil {
		t.Fatalf("LoadSpecs: %v", err)
	}
	if len(specs) != 2 || specs[0].Name != "a" || specs[1].Name != "b" {
		t.Fatalf("specs = %+v, want a and b", specs)
	}
	if len(specs[0].GroupBy) != 1 || specs[0].GroupBy[0].Key != "SERVICE" {
		t.Errorf("a.GroupBy = %+v", specs[0].GroupBy)
	}
	if len(specs[0].Filters) != 1 || !specs[0].Filters[0].Exclude || specs[0].Filters[0].Values[0] != "Tax" {
		t.Errorf("a.Filters = %+v", specs[0].Filters)
	}
	if specs[0].TimeRange.IsCustom() || !specs[1].TimeRange.IsCustom() {
		t.Errorf("IsCustom: a=%v b=%v, want false/true",
			specs[0].TimeRange.IsCustom(), specs[1].TimeRange.IsCustom())
	}

	if _, err := LoadSpecs(filepath.Join(dir, "missing.json")); err == nil {
		t.Error("missing file: expected an error")
	}

	bad := write("bad.json", `{"name": "not an array"}`)
	if _, err := LoadSpecs(bad); err == nil || !strings.Contains(err.Error(), bad) {
		t.Errorf("bad json: err = %v, want one naming %s", err, bad)
	}
}

func TestFind(t *testing.T) {
	specs := []*Spec{{Name: "zeta"}, {Name: "alpha"}}
	got, err := Find(specs, "alpha")
	if err != nil || got != specs[1] {
		t.Fatalf("Find(alpha) = %v, %v; want the second spec", got, err)
	}
	_, err = Find(specs, "nope")
	if err == nil {
		t.Fatal("Find(nope): expected an error")
	}
	// The message lists what is available, sorted, so a typo is easy to fix.
	if !strings.Contains(err.Error(), `"nope"`) || !strings.Contains(err.Error(), "[alpha zeta]") {
		t.Errorf("err = %q, want it to name the miss and list [alpha zeta]", err)
	}
}

func TestResult_SortedKeys_GrandTotal(t *testing.T) {
	r := &Result{Rows: map[string][]float64{
		"EC2": {0.5, 0.25}, // 0.75
		"S3":  {3, 4},      // 7
		"RDS": {3, 4},      // 7: ties with S3, so alphabetical order decides
	}}
	want := []string{"RDS", "S3", "EC2"}
	got := r.SortedKeys()
	if len(got) != len(want) {
		t.Fatalf("SortedKeys = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("SortedKeys[%d] = %s, want %s", i, got[i], want[i])
		}
	}
	if gt := r.GrandTotal(); gt != 14.75 {
		t.Errorf("GrandTotal = %v, want 14.75", gt)
	}
	if r.Total("missing") != 0 {
		t.Errorf("Total(missing) = %v, want 0", r.Total("missing"))
	}
}

func TestGroupHeader(t *testing.T) {
	tests := []struct {
		groups []Group
		want   string
	}{
		{nil, "Total"},
		{[]Group{{Type: "DIMENSION", Key: "SERVICE"}}, "SERVICE"},
		{[]Group{{Type: "DIMENSION", Key: "SERVICE"}, {Type: "TAG", Key: "repo"}}, "SERVICE / repo"},
	}
	for _, tt := range tests {
		if got := groupHeader(&Spec{GroupBy: tt.groups}); got != tt.want {
			t.Errorf("groupHeader(%v) = %q, want %q", tt.groups, got, tt.want)
		}
	}
}

func TestWriteCSV_Grouped(t *testing.T) {
	r := &Result{
		Spec:    &Spec{GroupBy: []Group{{Type: "DIMENSION", Key: "SERVICE"}}},
		Periods: []string{"2026-01-01", "2026-02-01"},
		Rows:    map[string][]float64{"EC2": {0.5, 0.25}, "S3": {3, 4}, "RDS": {3, 4}},
		Unit:    "USD",
	}
	var buf bytes.Buffer
	if err := r.WriteCSV(&buf); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	// Rows by descending total (name breaks the tie), then a Total row; every
	// row ends in its own total.
	want := "SERVICE,2026-01-01,2026-02-01,Total\n" +
		"RDS,3,4,7\n" +
		"S3,3,4,7\n" +
		"EC2,0.5,0.25,0.75\n" +
		"Total,6.5,8.25,14.75\n"
	if buf.String() != want {
		t.Errorf("csv =\n%s\nwant\n%s", buf.String(), want)
	}
}

func TestWriteCSV_Ungrouped(t *testing.T) {
	r := &Result{
		Spec:    &Spec{},
		Periods: []string{"2026-01-01"},
		Rows:    map[string][]float64{"(total)": {90438.93}},
	}
	var buf bytes.Buffer
	if err := r.WriteCSV(&buf); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	want := "Total,2026-01-01,Total\n" +
		"(total),90438.93,90438.93\n" +
		"Total,90438.93,90438.93\n"
	if buf.String() != want {
		t.Errorf("csv =\n%s\nwant\n%s", buf.String(), want)
	}
}

// TestWriteCSV_FullPrecision pins the interchange contract: cells are written
// at full float64 precision, never rounded to cents, so a consumer that
// re-aggregates them lands on the same figure Cost Explorer reports. Tag group
// keys are written exactly as the API returned them ("tagkey$value").
func TestWriteCSV_FullPrecision(t *testing.T) {
	r := &Result{
		Spec:    &Spec{GroupBy: []Group{{Type: "TAG", Key: "repo"}}},
		Periods: []string{"2026-01-01", "2026-02-01"},
		Rows:    map[string][]float64{"repo$tb-pos": {0.1, 0.2}},
	}
	var buf bytes.Buffer
	if err := r.WriteCSV(&buf); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	want := "repo,2026-01-01,2026-02-01,Total\n" +
		"repo$tb-pos,0.1,0.2,0.30000000000000004\n" +
		"Total,0.1,0.2,0.30000000000000004\n"
	if buf.String() != want {
		t.Errorf("csv =\n%s\nwant\n%s", buf.String(), want)
	}
}

type failWriter struct{}

func (failWriter) Write([]byte) (int, error) { return 0, errors.New("disk full") }

// TestWriteCSV_WriterErrors: the writer's error surfaces whether it hits during
// the header, a row, or the final flush. A field longer than csv.Writer's
// buffer forces a flush (and so the failure) at that exact write.
func TestWriteCSV_WriterErrors(t *testing.T) {
	long := strings.Repeat("x", 8192)
	tests := []struct {
		name string
		r    *Result
	}{
		{"header", &Result{
			Spec: &Spec{GroupBy: []Group{{Type: "DIMENSION", Key: long}}},
			Rows: map[string][]float64{},
		}},
		{"row", &Result{
			Spec: &Spec{}, Periods: []string{"p"},
			Rows: map[string][]float64{long: {1}},
		}},
		{"flush", &Result{
			Spec: &Spec{}, Periods: []string{"p"},
			Rows: map[string][]float64{"(total)": {1}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := tt.r.WriteCSV(failWriter{}); err == nil {
				t.Fatal("expected the writer's error to surface")
			}
		})
	}
}
