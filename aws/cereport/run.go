package cereport

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// LoadSpecs reads a JSON array of specs from path.
func LoadSpecs(path string) ([]*Spec, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var specs []*Spec
	if err := json.Unmarshal(b, &specs); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	return specs, nil
}

// Find returns the spec with the given name.
func Find(specs []*Spec, name string) (*Spec, error) {
	for _, s := range specs {
		if s.Name == name {
			return s, nil
		}
	}
	names := make([]string, 0, len(specs))
	for _, s := range specs {
		names = append(names, s.Name)
	}
	sort.Strings(names)
	return nil, fmt.Errorf("no report named %q; have:\n  %v", name, names)
}

// Result is a report's data laid out as a grid: one row per group, one column
// per time period, in the order Cost Explorer returned them.
type Result struct {
	Spec    *Spec
	Start   string
	End     string
	Periods []string             // period start dates, chronological
	Rows    map[string][]float64 // group key -> per-period amount
	Unit    string               // currency, e.g. USD
}

// CostExplorerAPI is the subset of the Cost Explorer client this package uses.
type CostExplorerAPI interface {
	GetCostAndUsage(context.Context, *costexplorer.GetCostAndUsageInput, ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
}

// Run executes the report and collects every page of results. Cost Explorer
// paginates group results, so a report with many groups is incomplete unless
// every page is followed.
func Run(ctx context.Context, api CostExplorerAPI, spec *Spec, now time.Time, opts ...PeriodOption) (*Result, error) {
	in, err := spec.GetCostAndUsageInput(now, opts...)
	if err != nil {
		return nil, err
	}

	res := &Result{
		Spec:  spec,
		Start: *in.TimePeriod.Start,
		End:   *in.TimePeriod.End,
		Rows:  map[string][]float64{},
	}
	// A report with no group-by returns period totals only; give it a single
	// row so the output shape stays the same either way.
	const totalRow = "(total)"

	for {
		out, err := api.GetCostAndUsage(ctx, in)
		if err != nil {
			return nil, err
		}
		for _, period := range out.ResultsByTime {
			start := *period.TimePeriod.Start
			idx := indexOf(res.Periods, start)
			if idx < 0 {
				res.Periods = append(res.Periods, start)
				idx = len(res.Periods) - 1
				for k := range res.Rows {
					res.Rows[k] = append(res.Rows[k], 0)
				}
			}
			// Whether period totals are the data is a property of the
			// spec, not of what came back. A grouped report can return a
			// period with no groups simply because nothing was spent that
			// month — reading that as "no group-by" invents a phantom
			// "(total)" row beside the real groups, which then reads as
			// another group downstream.
			if len(spec.GroupBy) == 0 {
				amt, unit, err := metricValue(period.Total, spec.Metric)
				if err != nil {
					return nil, err
				}
				res.Unit = unit
				res.addAt(totalRow, idx, amt)
				continue
			}
			for _, g := range period.Groups {
				amt, unit, err := metricValue(g.Metrics, spec.Metric)
				if err != nil {
					return nil, err
				}
				res.Unit = unit
				res.addAt(joinKeys(g.Keys), idx, amt)
			}
		}
		if out.NextPageToken == nil || *out.NextPageToken == "" {
			break
		}
		in.NextPageToken = out.NextPageToken
	}
	return res, nil
}

// addAt accumulates an amount into a row, growing the row to the current
// number of periods first.
func (r *Result) addAt(key string, idx int, amt float64) {
	row, ok := r.Rows[key]
	if !ok {
		row = make([]float64, len(r.Periods))
	}
	for len(row) < len(r.Periods) {
		row = append(row, 0)
	}
	row[idx] += amt
	r.Rows[key] = row
}

// Total returns a row's sum across all periods.
func (r *Result) Total(key string) float64 {
	var t float64
	for _, v := range r.Rows[key] {
		t += v
	}
	return t
}

// GrandTotal returns the sum of every row.
func (r *Result) GrandTotal() float64 {
	var t float64
	for k := range r.Rows {
		t += r.Total(k)
	}
	return t
}

// SortedKeys returns row keys ordered by descending total, which is how the
// console presents them.
func (r *Result) SortedKeys() []string {
	keys := make([]string, 0, len(r.Rows))
	for k := range r.Rows {
		keys = append(keys, k)
	}
	sort.Slice(keys, func(i, j int) bool {
		ti, tj := r.Total(keys[i]), r.Total(keys[j])
		if ti != tj {
			return ti > tj
		}
		return keys[i] < keys[j]
	})
	return keys
}

// WriteCSV writes the result as a grid: group key, one column per period, then
// a total. Group keys are written exactly as Cost Explorer returned them —
// tag groups arrive as "tagkey$value" — so that the output can be checked
// against the API without any relabelling in between.
//
// Amounts are written at full precision rather than rounded to cents. The file
// is an interchange format, and a consumer that re-aggregates the cells (a
// chart that recomputes column totals, say) would otherwise accumulate the
// rounding error: over 71 services one month drifts by about five cents from
// the figure Cost Explorer reports. Rounding is the presentation layer's job.
func (r *Result) WriteCSV(w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := append([]string{groupHeader(r.Spec)}, r.Periods...)
	header = append(header, "Total")
	if err := cw.Write(header); err != nil {
		return err
	}
	for _, k := range r.SortedKeys() {
		rec := make([]string, 0, len(r.Periods)+2)
		rec = append(rec, k)
		for _, v := range r.Rows[k] {
			rec = append(rec, strconv.FormatFloat(v, 'f', -1, 64))
		}
		rec = append(rec, strconv.FormatFloat(r.Total(k), 'f', -1, 64))
		if err := cw.Write(rec); err != nil {
			return err
		}
	}
	total := make([]string, 0, len(r.Periods)+2)
	total = append(total, "Total")
	for i := range r.Periods {
		var t float64
		for _, row := range r.Rows {
			t += row[i]
		}
		total = append(total, strconv.FormatFloat(t, 'f', -1, 64))
	}
	total = append(total, strconv.FormatFloat(r.GrandTotal(), 'f', -1, 64))
	if err := cw.Write(total); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

func groupHeader(s *Spec) string {
	if len(s.GroupBy) == 0 {
		return "Total"
	}
	parts := make([]string, 0, len(s.GroupBy))
	for _, g := range s.GroupBy {
		parts = append(parts, g.Key)
	}
	return joinKeys(parts)
}

func joinKeys(keys []string) string {
	out := ""
	for i, k := range keys {
		if i > 0 {
			out += " / "
		}
		out += k
	}
	return out
}

// metricValue pulls the report's metric out of a Cost Explorer metrics map.
// The amount arrives as a string and is parsed at full float64 precision:
// annual figures here run past the ~7 significant digits a float32 holds.
func metricValue(metrics map[string]cetypes.MetricValue, metric string) (float64, string, error) {
	mv, ok := metrics[metric]
	if !ok {
		return 0, "", fmt.Errorf("response has no metric %q", metric)
	}
	if mv.Amount == nil {
		return 0, "", fmt.Errorf("metric %q has no amount", metric)
	}
	amt, err := strconv.ParseFloat(*mv.Amount, 64)
	if err != nil {
		return 0, "", fmt.Errorf("parsing %s amount %q: %w", metric, *mv.Amount, err)
	}
	unit := ""
	if mv.Unit != nil {
		unit = *mv.Unit
	}
	return amt, unit, nil
}

func indexOf(ss []string, s string) int {
	for i, v := range ss {
		if v == s {
			return i
		}
	}
	return -1
}
