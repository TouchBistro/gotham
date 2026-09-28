# cereport

`cereport` runs AWS Cost Explorer reports from Go and hands back the numbers as a
grid — one row per group, one column per period, with totals. You describe the
report as a `Spec` (metric, granularity, group-by, filters, time range); the
package validates it, builds the `GetCostAndUsage` request, follows every page
of results, and returns a `Result` you can write as CSV or read directly.

It is the library behind the `cer` CLI (repo `cerep`), which holds TouchBistro's
report definitions. This package holds no report data and does no credential
handling: you pass in any client that implements `CostExplorerAPI`. Importing it
adds `aws-sdk-go-v2` core and `service/costexplorer` to your module, not
`config`. `ParseURL` turns an existing saved report's console URL into a `Spec`,
so reports built in the console can be captured once and replayed from code.

---

## Install

```bash
go get github.com/TouchBistro/gotham/aws/cereport
```

---

## Quick start

```go
package main

import (
	"context"
	"log"
	"os"
	"time"

	"github.com/TouchBistro/gotham/aws/cereport"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
)

func main() {
	ctx := context.Background()

	// Cost Explorer is a global service fronted in us-east-1.
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithSharedConfigProfile("touchbistro"),
		config.WithRegion("us-east-1"),
	)
	if err != nil {
		log.Fatal(err)
	}
	api := costexplorer.NewFromConfig(cfg)

	spec := cereport.Spec{
		Name:        "ytd_aws_by_service_unblended",
		Metric:      cereport.MetricUnblendedCost,
		Granularity: cereport.GranularityMonthly,
		GroupBy:     []cereport.Group{{Type: cereport.TypeDimension, Key: "SERVICE"}},
		Filters: []cereport.Filter{{
			Type: cereport.TypeDimension, Key: "RECORD_TYPE", Exclude: true,
			Values: []string{"Distributor Discount", "Refund", "Tax"},
		}},
		TimeRange: cereport.TimeRange{Relative: cereport.RangeYearToDate},
	}

	res, err := cereport.Run(ctx, api, &spec, time.Now().UTC())
	if err != nil {
		log.Fatal(err)
	}
	if err := res.WriteCSV(os.Stdout); err != nil {
		log.Fatal(err)
	}
}
```

```
SERVICE,2026-01-01,2026-02-01,…,2026-09-01,Total
Amazon Elastic Compute Cloud - Compute,41230.11,39877.4,…,123118.6
Amazon Relational Database Service,20112.5,19980.02,…,60548.22
…
Total,61342.61,59857.42,…,183666.82
```

---

## Writing a Spec

| Field | Values |
|-------|--------|
| `Name` | Required. Names the report in errors; CLIs use it for output filenames. |
| `Metric` | One `Metric*` constant: `UnblendedCost`, `BlendedCost`, `AmortizedCost`, `NetUnblendedCost`, `NetAmortizedCost`, `UsageQuantity`, `NormalizedUsageAmount`. |
| `Granularity` | `GranularityHourly`, `GranularityDaily` or `GranularityMonthly`. |
| `GroupBy` | Up to two `Group`s. `Type` is `TypeDimension`, `TypeTag` or `TypeCostCategory`; `Key` is the dimension name (`SERVICE`, `LINKED_ACCOUNT`, `REGION`, `USAGE_TYPE`, … — `cereport.Dimensions()` lists all 35), the tag key, or the cost-category name. Omit for period totals only. |
| `Filters` | Same `Type`/`Key` shape plus `Values` (API-side values, not console labels) and `Exclude`. Clauses AND together; `Exclude` wraps a clause in NOT. In a multi-set report they apply to every set. |
| `FilterSets` | Optional. Two or more named `FilterSet`s (`Name`, `Filters`) for a report that is a union of slices, with a sub-total per set. See [Example 5](#example-5--a-union-of-named-slices-filter-sets). |
| `TimeRange.Relative` | `RangeYearToDate`, `RangeMonthToDate`, `RangeLastDays(n)`, `RangeLastMonths(n)` or `RangeCustom`. |
| `TimeRange.Start`, `End` | `yyyy-MM-dd`; start inclusive, end exclusive. Required for `RangeCustom`, ignored otherwise. |

JSON tags are the lowerCamel field names (`name`, `metric`, `granularity`,
`groupBy[].type/key`, `filters[].type/key/exclude/values`,
`filterSets[].name/filters`, `timeRange.relative/start/end`), so specs can live in a checked-in file — see
[Loading specs from JSON](#loading-specs-from-json). To start from a report that
already exists in the console, see
[Capturing a Spec from a console URL](#capturing-a-spec-from-a-console-url).

### Example 1 — monthly spend by service, year to date

The quick-start spec. Excluding `RECORD_TYPE` values `Tax`, `Refund` and
`Distributor Discount` leaves what was consumed, which is what a service
breakdown should show.

### Example 2 — daily amortized cost of one repo, last 30 days, by region

```go
spec := cereport.Spec{
	Name:        "tb_pos_daily_by_region",
	Metric:      cereport.MetricAmortizedCost,
	Granularity: cereport.GranularityDaily,
	GroupBy:     []cereport.Group{{Type: cereport.TypeDimension, Key: "REGION"}},
	Filters: []cereport.Filter{
		{Type: cereport.TypeTag, Key: "repo", Values: []string{"tb-pos"}},
	},
	TimeRange: cereport.TimeRange{Relative: cereport.RangeLastDays(30)},
}
```

Amortized cost spreads Savings Plan and Reserved Instance up-front fees across
the period. Use it for "what does this workload really cost"; use unblended for
"what did the invoice say".

### Example 3 — one number: month-to-date net spend of a team

No `GroupBy`, so the result has a single `(total)` row.

```go
spec := cereport.Spec{
	Name:        "sre_mtd_net",
	Metric:      cereport.MetricNetAmortizedCost,
	Granularity: cereport.GranularityMonthly,
	Filters: []cereport.Filter{
		{Type: cereport.TypeCostCategory, Key: "Team", Values: []string{"SRE"}},
		{Type: cereport.TypeDimension, Key: "RECORD_TYPE", Exclude: true, Values: []string{"Tax", "Refund"}},
	},
	TimeRange: cereport.TimeRange{Relative: cereport.RangeMonthToDate},
}
```

### Example 4 — fixed window, two group-by keys

Cost Explorer allows two group-by keys. Groups come back joined with ` / `,
e.g. `Amazon Elastic Container Service / repo$tb-pos`.

```go
spec := cereport.Spec{
	Name:        "q1_2026_ecs_by_service_and_repo",
	Metric:      cereport.MetricAmortizedCost,
	Granularity: cereport.GranularityMonthly,
	GroupBy: []cereport.Group{
		{Type: cereport.TypeDimension, Key: "SERVICE"},
		{Type: cereport.TypeTag, Key: "repo"},
	},
	Filters: []cereport.Filter{
		{Type: cereport.TypeDimension, Key: "SERVICE", Values: []string{"Amazon Elastic Container Service"}},
	},
	TimeRange: cereport.TimeRange{Relative: cereport.RangeCustom, Start: "2026-01-01", End: "2026-04-01"},
}
```

### Example 5 — a union of named slices (filter sets)

Some reports are a union of differently-shaped slices: everything in one
account plus one service's tagged resources elsewhere, or an app's databases,
its Kafka clusters and part of its EC2. The console's filter panel ANDs every
clause, so no single filter list describes such a union without pulling in
costs you did not mean. `FilterSets` does: name each slice, and the report runs
one request per set and stacks the results.

```json
{
  "name": "12mo_stack_amortized",
  "metric": "AmortizedCost",
  "granularity": "MONTHLY",
  "groupBy": [{ "type": "DIMENSION", "key": "SERVICE" }],
  "filters": [
    { "type": "DIMENSION", "key": "RECORD_TYPE", "exclude": true,
      "values": ["Distributor Discount", "Tax", "Refund"] }
  ],
  "filterSets": [
    { "name": "app-account", "filters": [
      { "type": "DIMENSION", "key": "LINKED_ACCOUNT", "values": ["111122223333"] }
    ]},
    { "name": "ansible-awx", "filters": [
      { "type": "TAG", "key": "serviceid", "values": ["awx"] },
      { "type": "TAG", "key": "groupid", "values": ["ansible"] },
      { "type": "DIMENSION", "key": "LINKED_ACCOUNT", "exclude": true, "values": ["111122223333"] }
    ]}
  ],
  "timeRange": { "relative": "LAST_12_MONTHS" }
}
```

- `filters` apply to every set, and each set ANDs its own on top. Metric,
  granularity, group-by and time range are shared, so every set has the same
  columns.
- One `GetCostAndUsage` request per set, plus one per extra page of groups.
- In Go: `FilterSets: []cereport.FilterSet{{Name: "app-account", Filters: …}, …}`.

The CSV stacks each set's rows, then a `<set> Total` sub-total row, then the
grand `Total`:

```
SERVICE,2025-09-01,…,2026-09-01,Total
Amazon Relational Database Service,812.4,…,10561.2
Amazon Elastic Compute Cloud - Compute,640.11,…,8321.43
…
app-account Total,2210.93,…,28742.09
Amazon Elastic Compute Cloud - Compute,95.2,…,1237.6
EC2 - Other,12.08,…,157.04
ansible-awx Total,107.28,…,1394.64
Total,2318.21,…,30136.73
```

A service can appear under more than one set, each time with that set's share.
Without `groupBy` each set's only row would repeat its sub-total, so the report
is one row per set under a `Filter set` header:

```
Filter set,2025-09-01,…,2026-09-01,Total
app-account,2210.93,…,28742.09
ansible-awx,107.28,…,1394.64
Total,2318.21,…,30136.73
```

(Numbers are illustrative.)

Every aggregate row's label is `Total` or ends in ` Total`, so a consumer can
skip them all with one rule and keep only data rows. In the ungrouped layout the
set rows are the data, which is why a set name may not be `Total` or end in
` Total`. The rule misreads only a group key that itself ends in ` Total`: AWS
service names don't, but a tag value could.

> **Keep the sets disjoint.** Sets are summed, not de-duplicated. A cost that
> matches two sets is counted in both, and `Total` overstates the union. Here,
> AWX resources running inside the app account would match both sets, which is
> why `ansible-awx` excludes that account. The reliable pattern is a
> single-valued dimension (`LINKED_ACCOUNT`, `SERVICE`, `REGION`, `USAGE_TYPE`,
> …) that one set includes and the others exclude, or that each set includes
> with values no other set uses. Every line item has exactly one value per
> dimension, so such sets cannot overlap. `Validate` cannot check disjointness;
> tags in particular can overlap in ways only the data shows.

### Validation

`Spec.Validate` checks every rule Cost Explorer is known to enforce and reports
all problems at once. `Run` and `GetCostAndUsageInput` call it first, so a bad
spec never costs an API request. Call it yourself when loading checked-in specs
so the whole file fails up front:

```go
if err := spec.Validate(); err != nil {
	log.Fatal(err)
}
```

```
report "tb_pos_daily": metric "AMORTIZED_COST": want one of AmortizedCost, BlendedCost, NetAmortizedCost, NetUnblendedCost, NormalizedUsageAmount, UnblendedCost, UsageQuantity
groupBy[0]: unknown dimension "AVAILABILITY_ZONE"; known: AGREEMENT_END_DATE_TIME_AFTER, …, AZ, …, USAGE_TYPE_GROUP
timeRange.relative "LAST_WEEK": want CUSTOM, YEAR_TO_DATE, MONTH_TO_DATE, LAST_<n>_DAYS or LAST_<n>_MONTHS
```

Rules: `Name` set · `Metric` and `Granularity` from the constants · at most two
`GroupBy` · every `Group`/`Filter` has a known `Type` and a `Key`, and
`DIMENSION` keys are real Cost Explorer dimensions (taken from the SDK, so they
track the `service/costexplorer` version in `go.mod`) · every `Filter` has at
least one value · `Relative` is a known range, with `n ≥ 1` for rolling ranges ·
a `CUSTOM` range has `Start` and `End` · any dates given are `yyyy-MM-dd` with
`Start` before `End` · `FilterSets`, if given, has at least two sets, each with
a name (present, unique, not `Total` and not ending in ` Total`) and at least
one filter of its own.
`cereport.Metrics()`, `Granularities()` and `Dimensions()` return the accepted
values, sorted, for help text.

### Capturing a Spec from a console URL

The Cost Explorer API has no operation that reads saved reports, but the console
URL of a saved report carries its whole definition in the fragment query string.
Open the report in the console, copy the browser URL, and parse it once:

```go
spec, err := cereport.ParseURL(consoleURL) // full URL, or just "/costmanagement/home?…#/cost-explorer?…"
if err != nil {
	log.Fatal(err)
}
b, _ := json.MarshalIndent(spec, "", "  ")
os.WriteFile("reports/"+spec.Name+".json", b, 0o644)
```

What it reads: `reportName` → `Name` (trimmed), `costAggregate` /
`useNormalizedUnits` → `Metric`, `granularity`, `groupBy` (console dimension ids
and `TagKeyValue:<key>` tag groups), `filter` rows (dimension, tag via
`growableValue`, includes/excludes), `historicalRelativeRange` + `startDate` /
`endDate` → `TimeRange`. The result is validated before it is returned.

What it rejects, on purpose: report modes other than `STANDARD` (reservation and
Savings Plans reports use different APIs), `showOnlyUntagged` /
`showOnlyUncategorized` (would need an `ABSENT` match the `Spec` cannot express),
and console dimension ids not in its mapping table — add the id rather than let
a guess produce a silently wrong query. Cost-category *group-by* is not parsed
from URLs (no captured report uses it); cost-category *filters* are.

### Loading specs from JSON

Decode a checked-in file into `[]cereport.Spec` with the standard library and
validate:

```json
[
  {
    "name": "ytd_aws_by_service_unblended",
    "metric": "UnblendedCost",
    "granularity": "MONTHLY",
    "groupBy": [{ "type": "DIMENSION", "key": "SERVICE" }],
    "filters": [
      { "type": "DIMENSION", "key": "RECORD_TYPE", "exclude": true,
        "values": ["Distributor Discount", "Refund", "Tax"] }
    ],
    "timeRange": { "relative": "YEAR_TO_DATE" }
  },
  {
    "name": "q1_2026_total",
    "metric": "AmortizedCost",
    "granularity": "MONTHLY",
    "timeRange": { "relative": "CUSTOM", "start": "2026-01-01", "end": "2026-04-01" }
  }
]
```

```go
b, err := os.ReadFile("reports/specs.json")
if err != nil {
	log.Fatal(err)
}
var specs []cereport.Spec
if err := json.Unmarshal(b, &specs); err != nil {
	log.Fatal(err)
}
for _, s := range specs {
	if err := s.Validate(); err != nil {
		log.Fatal(err)
	}
}
```

---

## Running a report and reading the Result

```go
func Run(ctx context.Context, api CostExplorerAPI, spec *Spec, now time.Time, opts ...PeriodOption) (*Result, error)
```

`Run` validates the spec, resolves the time range against `now`, builds the
request, follows every `NextPageToken`, and accumulates into a `Result`:

| Field | Meaning |
|-------|---------|
| `Periods []string` | Period start dates, chronological (`2026-01-01`, …). |
| `Rows map[string][]float64` | Group key → amount per period, aligned with `Periods`. An ungrouped report has one key, `(total)`. |
| `Unit string` | `USD` for cost metrics; the usage unit otherwise. |
| `Start`, `End string` | The resolved period actually requested. |
| `Spec *Spec` | The spec that produced it. |
| `Sets []*Result` | Multi-set reports only: one ordinary `Result` per filter set, in spec order, all on the same `Periods`. `Sets[i].Spec.Name` is the set's name and `Sets[i].GrandTotal()` its sub-total. `Rows` then sums the sets by group key. |

Methods: `Total(key)`, `GrandTotal()`, `PeriodTotals()` (the `Total` row's
per-period sums), `SortedKeys()` (descending total, ties by name), `WriteCSV(w)`.
Totals add rows in key order, so the same `Result` writes the same bytes every
time.

### Example A — CSV to a file

```go
f, err := os.Create(filepath.Join("out", spec.Name+".csv"))
if err != nil {
	log.Fatal(err)
}
defer f.Close()
if err := res.WriteCSV(f); err != nil {
	log.Fatal(err)
}
```

Shape: header is the group keys (`SERVICE`, `SERVICE / repo`, or `Total` when
ungrouped), then period start dates, then `Total`. Rows are sorted by
descending total and each ends with its own total; a final `Total` row sums
every column. Amounts are shortest-round-trip `float64`, never rounded to cents:
the CSV is an interchange format and rounding is the presentation layer's job.
Tag groups arrive from the API as `tagkey$value` and are written as-is. A
multi-set report stacks its sets instead ([Example 5](#example-5--a-union-of-named-slices-filter-sets)).
Its sub-total rows sit among the group rows, labelled `<set> Total`: a consumer
that skips every label equal to `Total` or ending in ` Total` keeps only data
rows.

### Example B — top five groups and their share

```go
grand := res.GrandTotal()
for i, key := range res.SortedKeys() {
	if i == 5 {
		break
	}
	fmt.Printf("%-45s %12.2f %5.1f%%\n", key, res.Total(key), 100*res.Total(key)/grand)
}
```

### Example C — one number for a Slack message

```go
// Ungrouped report (Example 3): the only row is "(total)".
mtd := res.Total("(total)")
fmt.Printf("SRE month-to-date: %.2f %s (%s → %s)\n", mtd, res.Unit, res.Start, res.End)
```

### Example D — tag groups without the `key$` prefix

```go
for _, key := range res.SortedKeys() {
	repo := strings.TrimPrefix(key, "repo$") // "" for resources with no repo tag
	fmt.Printf("%-30s %.2f\n", repo, res.Total(key))
}
```

### Example E — change against the previous period

```go
n := len(res.Periods)
for _, key := range res.SortedKeys() {
	row := res.Rows[key]
	if n < 2 || row[n-2] == 0 {
		continue
	}
	fmt.Printf("%-45s %+6.1f%%\n", key, 100*(row[n-1]-row[n-2])/row[n-2])
}
```

### Example F — per-set sub-totals

```go
// Multi-set report (Example 5): one Result per set, all on the same columns.
for _, set := range res.Sets {
	fmt.Printf("%-15s %12.2f\n", set.Spec.Name, set.GrandTotal())
	for _, key := range set.SortedKeys()[:min(3, len(set.Rows))] {
		fmt.Printf("  %-40s %12.2f\n", key, set.Total(key))
	}
}
fmt.Printf("%-15s %12.2f %s\n", "Total", res.GrandTotal(), res.Unit)
```

### Time ranges and "today"

| `Relative` | Resolves to, with `now` = 2026-09-14 UTC |
|------------|------------------------------------------|
| `YEAR_TO_DATE` | 2026-01-01 → 2026-09-15 |
| `MONTH_TO_DATE` | 2026-09-01 → 2026-09-15 |
| `LAST_7_DAYS` | 2026-09-07 → 2026-09-15 |
| `LAST_6_MONTHS` | 2026-03-01 → 2026-09-15 (first of the month, so buckets align) |
| `CUSTOM` | `Start` → `End` as given; `now` is ignored |

Ranges resolve in UTC; Cost Explorer reckons days in UTC. `End` is exclusive,
so to-date ranges end on tomorrow's date to include today, matching the
console. Today's data is still settling (Savings Plan fees are amortized a day
later), so for a stable, re-runnable snapshot exclude it:

```go
res, err := cereport.Run(ctx, api, &spec, time.Now().UTC(), cereport.ExcludeCurrentDay())
```

Pass a fixed `now` — say `time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)` — to
reproduce a month-end report later.

---

## Testing without AWS

`CostExplorerAPI` is the one method this package calls:

```go
type CostExplorerAPI interface {
	GetCostAndUsage(context.Context, *costexplorer.GetCostAndUsageInput,
		...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
}
```

Hand `Run` a fake that returns canned pages:

```go
type fakeCE struct{ pages []*costexplorer.GetCostAndUsageOutput }

func (f *fakeCE) GetCostAndUsage(context.Context, *costexplorer.GetCostAndUsageInput,
	...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error) {
	p := f.pages[0]
	f.pages = f.pages[1:]
	return p, nil
}
```

The package's own tests work this way and need no credentials.

---

## Permissions, cost, limits

- Credentials need `ce:GetCostAndUsage`. AWS bills Cost Explorer API calls per
  request; a report costs one request per page of groups, and a multi-set report
  that for each set.
- Two group-by keys at most (enforced by `Validate`).
- `HOURLY` needs hourly granularity enabled on the payer account and covers only
  recent days (14 at the time of writing).
- One metric per report.
- Filter sets are summed, not de-duplicated: keep them disjoint (Example 5).
- The console's "show only untagged / uncategorized" toggles have no `Spec`
  equivalent.
- Reservation and Savings Plans report modes use different Cost Explorer APIs
  and are out of scope.
