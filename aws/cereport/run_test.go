package cereport

import (
	"context"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

type fakeCE struct {
	out []*costexplorer.GetCostAndUsageOutput
}

func (f *fakeCE) GetCostAndUsage(_ context.Context, _ *costexplorer.GetCostAndUsageInput,
	_ ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
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
