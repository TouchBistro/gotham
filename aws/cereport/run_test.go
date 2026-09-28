package cereport

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"reflect"
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
	// ins records a copy of each request.
	ins []costexplorer.GetCostAndUsageInput
	// err, when set, is returned from call number failCall (1-based), or from
	// every call when failCall is 0.
	err      error
	failCall int
}

func (f *fakeCE) GetCostAndUsage(_ context.Context, in *costexplorer.GetCostAndUsageInput,
	_ ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	f.tokens = append(f.tokens, aws.ToString(in.NextPageToken))
	f.ins = append(f.ins, *in)
	if f.err != nil && (f.failCall == 0 || f.failCall == len(f.tokens)) {
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
		{"filter set row", &Result{
			Spec:    &Spec{GroupBy: []Group{{Type: TypeDimension, Key: "SERVICE"}}, FilterSets: twoSets()},
			Periods: []string{"p"}, Rows: map[string][]float64{},
			Sets: []*Result{{Spec: &Spec{Name: "db"}, Periods: []string{"p"}, Rows: map[string][]float64{long: {1}}}},
		}},
		{"filter set sub-total row", &Result{
			Spec:    &Spec{GroupBy: []Group{{Type: TypeDimension, Key: "SERVICE"}}, FilterSets: twoSets()},
			Periods: []string{"p"}, Rows: map[string][]float64{},
			Sets: []*Result{{Spec: &Spec{Name: long}, Periods: []string{"p"}, Rows: map[string][]float64{"x": {1}}}},
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

// TestWriteCSV_Deterministic: the same Result always writes the same bytes.
// Totals used to be summed in map order; floating-point addition is not
// associative, so the last digits of the Total row changed from run to run and
// two back-to-back runs of the same report could not be diffed.
func TestWriteCSV_Deterministic(t *testing.T) {
	rows := map[string][]float64{}
	for i := 0; i < 71; i++ { // as many services as the ytd by-service report
		rows[fmt.Sprintf("svc%02d", i)] = []float64{float64(i)*123.4567891 + 0.1, float64(i%7)*0.3333333 + 0.2}
	}
	r := &Result{
		Spec:    &Spec{GroupBy: []Group{{Type: TypeDimension, Key: "SERVICE"}}},
		Periods: []string{"2026-01-01", "2026-02-01"},
		Rows:    rows,
	}
	var first string
	for n := 0; n < 50; n++ {
		var buf bytes.Buffer
		if err := r.WriteCSV(&buf); err != nil {
			t.Fatalf("WriteCSV: %v", err)
		}
		if n == 0 {
			first = buf.String()
			continue
		}
		if buf.String() != first {
			t.Fatalf("call %d wrote different bytes from call 0", n)
		}
	}
}

func TestPeriodTotals(t *testing.T) {
	r := &Result{
		Periods: []string{"2026-01-01", "2026-02-01"},
		// A short row (fewer amounts than periods) contributes what it has.
		Rows: map[string][]float64{"a": {1, 2}, "b": {3, 4}, "c": {5}},
	}
	got := r.PeriodTotals()
	if len(got) != 2 || got[0] != 9 || got[1] != 6 {
		t.Errorf("PeriodTotals = %v, want [9 6]", got)
	}
	if got := (&Result{}).PeriodTotals(); len(got) != 0 {
		t.Errorf("empty Result: PeriodTotals = %v, want []", got)
	}
}

// filterSetsSpec is a two-set report grouped by service: RDS, and MSK tagged
// app=singleapp, with Tax excluded from both.
func filterSetsSpec() *Spec {
	return &Spec{
		Name: "singleapp", Metric: MetricAmortizedCost, Granularity: GranularityMonthly,
		GroupBy: []Group{{Type: TypeDimension, Key: "SERVICE"}},
		Filters: []Filter{{Type: TypeDimension, Key: "RECORD_TYPE", Exclude: true, Values: []string{"Tax"}}},
		FilterSets: []FilterSet{
			{Name: "db", Filters: []Filter{{Type: TypeDimension, Key: "SERVICE", Values: []string{"RDS"}}}},
			{Name: "stream", Filters: []Filter{
				{Type: TypeDimension, Key: "SERVICE", Values: []string{"MSK"}},
				{Type: TypeTag, Key: "app", Values: []string{"singleapp"}},
			}},
		},
		TimeRange: TimeRange{Relative: RangeCustom, Start: "2026-01-01", End: "2026-03-01"},
	}
}

func grp(key, amount string) cetypes.Group {
	return cetypes.Group{Keys: []string{key}, Metrics: metric(amount)}
}

// filterSetsAPI answers the two requests filterSetsSpec makes. The second set
// returns no January period at all, and "EC2 - Other" shows up in both sets.
func filterSetsAPI() *fakeCE {
	return &fakeCE{out: []*costexplorer.GetCostAndUsageOutput{
		{ResultsByTime: []cetypes.ResultByTime{
			period("2026-01-01", "2026-02-01", grp("RDS", "10"), grp("EC2 - Other", "1")),
			period("2026-02-01", "2026-03-01", grp("RDS", "20")),
		}},
		{ResultsByTime: []cetypes.ResultByTime{
			period("2026-02-01", "2026-03-01", grp("MSK", "5"), grp("EC2 - Other", "2")),
		}},
	}}
}

func TestRun_FilterSets(t *testing.T) {
	api := filterSetsAPI()
	res, err := Run(context.Background(), api, filterSetsSpec(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	// One request per set, each ANDing the common filter with the set's own.
	if len(api.ins) != 2 {
		t.Fatalf("made %d requests, want 2 (one per set)", len(api.ins))
	}
	type clauses struct {
		n       int
		service string
		tag     bool
	}
	for i, want := range []clauses{{2, "RDS", false}, {3, "MSK", true}} {
		f := api.ins[i].Filter
		if f == nil || len(f.And) != want.n {
			t.Fatalf("request %d: filter = %+v, want an And of %d", i, f, want.n)
		}
		if n := f.And[0].Not; n == nil || n.Dimensions == nil || string(n.Dimensions.Key) != "RECORD_TYPE" {
			t.Errorf("request %d: first clause = %+v, want the common NOT RECORD_TYPE", i, f.And[0])
		}
		if d := f.And[1].Dimensions; d == nil || string(d.Key) != "SERVICE" || d.Values[0] != want.service {
			t.Errorf("request %d: second clause = %+v, want SERVICE %s", i, f.And[1], want.service)
		}
		if want.tag && (f.And[2].Tags == nil || *f.And[2].Tags.Key != "app") {
			t.Errorf("request %d: third clause = %+v, want TAG app", i, f.And[2])
		}
	}

	if len(res.Sets) != 2 || res.Sets[0].Spec.Name != "db" || res.Sets[1].Spec.Name != "stream" {
		t.Fatalf("Sets = %+v, want db then stream", res.Sets)
	}
	// Every set sits on the same columns; the set with no January gets zeros.
	periods := []string{"2026-01-01", "2026-02-01"}
	for _, r := range append([]*Result{res}, res.Sets...) {
		if !reflect.DeepEqual(r.Periods, periods) {
			t.Errorf("%s: Periods = %v, want %v", r.Spec.Name, r.Periods, periods)
		}
	}
	if got, want := res.Sets[1].Rows, map[string][]float64{"MSK": {0, 5}, "EC2 - Other": {0, 2}}; !reflect.DeepEqual(got, want) {
		t.Errorf("stream rows = %v, want %v", got, want)
	}
	// Rows sums the sets by group key.
	if want := map[string][]float64{"RDS": {10, 20}, "MSK": {0, 5}, "EC2 - Other": {1, 2}}; !reflect.DeepEqual(res.Rows, want) {
		t.Errorf("Rows = %v, want %v", res.Rows, want)
	}
	if res.Sets[0].GrandTotal() != 31 || res.Sets[1].GrandTotal() != 7 || res.GrandTotal() != 38 {
		t.Errorf("sub-totals %v + %v, grand total %v; want 31 + 7 = 38",
			res.Sets[0].GrandTotal(), res.Sets[1].GrandTotal(), res.GrandTotal())
	}
	if res.Unit != "USD" || res.Start != "2026-01-01" || res.End != "2026-03-01" {
		t.Errorf("Unit/Start/End = %s/%s/%s", res.Unit, res.Start, res.End)
	}
}

// TestRun_FilterSetError: a failing set fails the report, and the error says
// which set.
func TestRun_FilterSetError(t *testing.T) {
	want := errors.New("ThrottlingException")
	api := filterSetsAPI()
	api.err, api.failCall = want, 2
	res, err := Run(context.Background(), api, filterSetsSpec(), time.Now())
	if !errors.Is(err, want) || !strings.Contains(err.Error(), `report "singleapp", filter set "stream"`) {
		t.Fatalf("err = %v, want %v naming the stream set", err, want)
	}
	if res != nil {
		t.Errorf("Result = %+v, want nil", res)
	}
}

func TestRun_FilterSetsInvalid(t *testing.T) {
	spec := filterSetsSpec()
	spec.FilterSets[1].Name = "db"
	api := filterSetsAPI()
	if _, err := Run(context.Background(), api, spec, time.Now()); err == nil || !strings.Contains(err.Error(), "duplicate name") {
		t.Fatalf("err = %v, want the duplicate-name validation error", err)
	}
	if len(api.ins) != 0 {
		t.Errorf("made %d requests for an invalid spec, want 0", len(api.ins))
	}
}

func TestWriteCSV_FilterSets(t *testing.T) {
	res, err := Run(context.Background(), filterSetsAPI(), filterSetsSpec(), time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	var buf bytes.Buffer
	if err := res.WriteCSV(&buf); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	// Each set's rows by descending total, then its "<set> Total" sub-total
	// row; the grand Total row last. A group key can recur across sets.
	want := "SERVICE,2026-01-01,2026-02-01,Total\n" +
		"RDS,10,20,30\n" +
		"EC2 - Other,1,0,1\n" +
		"db Total,11,20,31\n" +
		"MSK,0,5,5\n" +
		"EC2 - Other,0,2,2\n" +
		"stream Total,0,7,7\n" +
		"Total,11,27,38\n"
	if buf.String() != want {
		t.Errorf("csv =\n%s\nwant\n%s", buf.String(), want)
	}

	// One set on its own writes as an ordinary report.
	buf.Reset()
	if err := res.Sets[1].WriteCSV(&buf); err != nil {
		t.Fatal(err)
	}
	if want := "SERVICE,2026-01-01,2026-02-01,Total\nMSK,0,5,5\nEC2 - Other,0,2,2\nTotal,0,7,7\n"; buf.String() != want {
		t.Errorf("single set csv =\n%s\nwant\n%s", buf.String(), want)
	}
}

// TestWriteCSV_FilterSets_Ungrouped: with no group-by each set's only row
// would repeat its sub-total, so the sets themselves are the rows.
func TestWriteCSV_FilterSets_Ungrouped(t *testing.T) {
	spec := filterSetsSpec()
	spec.GroupBy = nil
	spec.TimeRange = TimeRange{Relative: RangeCustom, Start: "2026-01-01", End: "2026-02-01"}
	totalOnly := func(amount string) cetypes.ResultByTime {
		p := period("2026-01-01", "2026-02-01")
		p.Total = metric(amount)
		return p
	}
	api := &fakeCE{out: []*costexplorer.GetCostAndUsageOutput{
		{ResultsByTime: []cetypes.ResultByTime{totalOnly("10.5")}},
		{ResultsByTime: []cetypes.ResultByTime{totalOnly("5")}},
	}}
	res, err := Run(context.Background(), api, spec, time.Now())
	if err != nil {
		t.Fatalf("Run: %v", err)
	}
	var buf bytes.Buffer
	if err := res.WriteCSV(&buf); err != nil {
		t.Fatalf("WriteCSV: %v", err)
	}
	want := "Filter set,2026-01-01,Total\n" +
		"db,10.5,10.5\n" +
		"stream,5,5\n" +
		"Total,15.5,15.5\n"
	if buf.String() != want {
		t.Errorf("csv =\n%s\nwant\n%s", buf.String(), want)
	}
}

// TestIsTotalLabel pins the one rule a CSV consumer needs to skip every
// aggregate row: the label is "Total", or ends in " Total", in any case.
func TestIsTotalLabel(t *testing.T) {
	tests := map[string]bool{
		"Total":             true,
		" total ":           true,
		"db Total":          true,
		"ansible-awx total": true,
		"Grand Total":       true,
		"(total)":           false, // an ungrouped single-set report's data row
		"Totals":            false,
		"db subtotal":       false,
		"total-spend":       false,
		"EC2 - Other":       false,
		"":                  false,
	}
	for label, want := range tests {
		if got := isTotalLabel(label); got != want {
			t.Errorf("isTotalLabel(%q) = %v, want %v", label, got, want)
		}
	}
	if got := subTotalLabel("db"); got != "db Total" || !isTotalLabel(got) {
		t.Errorf("subTotalLabel(db) = %q, want a total label \"db Total\"", got)
	}
}
