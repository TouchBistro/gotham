package cereport

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"slices"
	"sort"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// Labels the CSV uses besides group keys and filter-set names.
const (
	// totalLabel heads the per-row total column and labels the grand total
	// row. Filter sets may not use it as a name.
	totalLabel = "Total"
	// ungroupedRow is the single row of a report with no group-by.
	ungroupedRow = "(total)"
	// filterSetHeader heads the first column of an ungrouped multi-set
	// report, whose rows are the sets themselves.
	filterSetHeader = "Filter set"
)

// Result is a report's data laid out as a grid: one row per group, one column
// per time period, in the order Cost Explorer returned them.
type Result struct {
	Spec    *Spec
	Start   string
	End     string
	Periods []string             // period start dates, chronological
	Rows    map[string][]float64 // group key -> per-period amount
	Unit    string               // currency, e.g. USD

	// Sets holds one Result per filter set of a multi-set report (see
	// Spec.FilterSets), in spec order, all on the same Periods. Each is an
	// ordinary single-set Result whose Spec is the set's entry from
	// Spec.Split, so Sets[i].Spec.Name is the set's name and
	// Sets[i].GrandTotal() its sub-total. Rows then holds the sets' rows
	// summed by group key. Nil for a single-set report.
	Sets []*Result
}

// CostExplorerAPI is the subset of the Cost Explorer client this package uses.
type CostExplorerAPI interface {
	GetCostAndUsage(context.Context, *costexplorer.GetCostAndUsageInput, ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
}

// Run validates the spec (see Validate), executes the report and collects
// every page of results. Cost Explorer paginates group results, so a report
// with many groups is incomplete unless every page is followed.
//
// A multi-set spec (see Spec.FilterSets) runs one request per set, in order,
// and fails if any set does. Its Result carries each set in Sets, aligned on a
// common list of periods, and Rows summing the sets by group key.
func Run(ctx context.Context, api CostExplorerAPI, spec *Spec, now time.Time, opts ...PeriodOption) (*Result, error) {
	if err := spec.Validate(); err != nil {
		return nil, err
	}
	if len(spec.FilterSets) == 0 {
		return runOne(ctx, api, spec, now, opts...)
	}

	res := &Result{Spec: spec, Rows: map[string][]float64{}}
	subs := spec.Split()
	for i := range subs {
		r, err := runOne(ctx, api, &subs[i], now, opts...)
		if err != nil {
			return nil, fmt.Errorf("report %q, filter set %q: %w", spec.Name, subs[i].Name, err)
		}
		res.Sets = append(res.Sets, r)
	}
	res.Periods = alignPeriods(res.Sets)
	res.Start, res.End = res.Sets[0].Start, res.Sets[0].End
	for _, set := range res.Sets {
		if res.Unit == "" {
			res.Unit = set.Unit
		}
		for k, row := range set.Rows {
			sum, ok := res.Rows[k]
			if !ok {
				sum = make([]float64, len(res.Periods))
			}
			for i, v := range row {
				sum[i] += v
			}
			res.Rows[k] = sum
		}
	}
	return res, nil
}

// runOne executes a single-set spec: one request, every page.
func runOne(ctx context.Context, api CostExplorerAPI, spec *Spec, now time.Time, opts ...PeriodOption) (*Result, error) {
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
	// A report with no group-by returns period totals only; it gets a single
	// ungroupedRow so the output shape stays the same either way.
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
				res.addAt(ungroupedRow, idx, amt)
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

// alignPeriods puts every result on the union of their periods, in
// chronological order, zero-filling a period a result did not return, and
// returns that list. The sets of one report share a time range and
// granularity, so Cost Explorer normally returns the same periods for each;
// this keeps the columns lined up if it does not.
func alignPeriods(results []*Result) []string {
	seen := map[string]bool{}
	for _, r := range results {
		for _, p := range r.Periods {
			seen[p] = true
		}
	}
	periods := sortedKeys(seen) // ISO dates and timestamps sort chronologically
	for _, r := range results {
		if slices.Equal(r.Periods, periods) {
			continue
		}
		at := make(map[string]int, len(r.Periods))
		for i, p := range r.Periods {
			at[p] = i
		}
		for k, row := range r.Rows {
			aligned := make([]float64, len(periods))
			for j, p := range periods {
				if i, ok := at[p]; ok && i < len(row) {
					aligned[j] = row[i]
				}
			}
			r.Rows[k] = aligned
		}
		r.Periods = slices.Clone(periods)
	}
	return periods
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

// GrandTotal returns the sum of every row. Rows are added in key order, so
// the result is the same on every call (see PeriodTotals).
func (r *Result) GrandTotal() float64 {
	var t float64
	for _, k := range sortedKeys(r.Rows) {
		t += r.Total(k)
	}
	return t
}

// PeriodTotals returns, for each period, the sum of every row: the figures on
// the CSV's Total row.
//
// Rows are added in key order. Floating-point addition is not associative, so
// summing in map order (as this package once did) makes the last digits of a
// total vary between runs, and two back-to-back runs of the same report stop
// diffing clean.
func (r *Result) PeriodTotals() []float64 {
	t := make([]float64, len(r.Periods))
	for _, k := range sortedKeys(r.Rows) {
		for i, v := range r.Rows[k] {
			if i < len(t) {
				t[i] += v
			}
		}
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
//
// A multi-set report (see Spec.FilterSets) writes each set in spec order — its
// rows by descending total, then a sub-total row labelled with the set's name
// — and ends with the grand Total row. The same group key can appear under
// more than one set. With no group-by a set's only row would repeat its
// sub-total, so only the sub-total rows are written, under a first-column
// header of "Filter set". Sub-total rows sit among group rows: a consumer that
// charts every row except Total must also skip rows named after a set.
func (r *Result) WriteCSV(w io.Writer) error {
	cw := csv.NewWriter(w)
	defer cw.Flush()

	header := append([]string{groupHeader(r.Spec)}, r.Periods...)
	header = append(header, totalLabel)
	if err := cw.Write(header); err != nil {
		return err
	}
	if len(r.Sets) == 0 {
		if err := r.writeRows(cw); err != nil {
			return err
		}
	}
	grouped := len(r.Spec.GroupBy) > 0
	for _, set := range r.Sets {
		if grouped {
			if err := set.writeRows(cw); err != nil {
				return err
			}
		}
		if err := cw.Write(record(set.Spec.Name, set.PeriodTotals(), set.GrandTotal())); err != nil {
			return err
		}
	}
	if err := cw.Write(record(totalLabel, r.PeriodTotals(), r.GrandTotal())); err != nil {
		return err
	}
	cw.Flush()
	return cw.Error()
}

// writeRows writes one record per row, by descending total.
func (r *Result) writeRows(cw *csv.Writer) error {
	for _, k := range r.SortedKeys() {
		if err := cw.Write(record(k, r.Rows[k], r.Total(k))); err != nil {
			return err
		}
	}
	return nil
}

// record is one CSV line: a label, one cell per period, then the row total.
func record(label string, amounts []float64, total float64) []string {
	rec := make([]string, 0, len(amounts)+2)
	rec = append(rec, label)
	for _, v := range amounts {
		rec = append(rec, formatAmount(v))
	}
	return append(rec, formatAmount(total))
}

// formatAmount writes an amount at full precision: the shortest decimal that
// parses back to the same float64.
func formatAmount(v float64) string { return strconv.FormatFloat(v, 'f', -1, 64) }

func groupHeader(s *Spec) string {
	if len(s.GroupBy) == 0 {
		if len(s.FilterSets) > 0 {
			return filterSetHeader
		}
		return totalLabel
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
