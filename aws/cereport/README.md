# cereport

Package `cereport` runs AWS Cost Explorer reports from checked-in definitions and
writes the results as a CSV grid — one row per group, one column per period, a
trailing `Total`. It is the library half of the `cer` CLI (repo `cerep`), which
holds TouchBistro's report definitions. This package holds no report data and
does no credential handling.

The package takes a one-method `CostExplorerAPI` interface rather than building a
client, so callers own credentials, region and retries. Importing it adds
`aws-sdk-go-v2` core and `service/costexplorer` to your module — not `config`.

---

## Installation

```bash
go get github.com/TouchBistro/gotham/aws/cereport
```

---

## Report definitions (`Spec`)

A report is a `Spec`, stored as one element of a JSON array:

```json
[
  {
    "name": "ytd_aws_by_service_unblended",
    "metric": "UnblendedCost",
    "granularity": "MONTHLY",
    "groupBy": [{ "type": "DIMENSION", "key": "SERVICE" }],
    "filters": [
      {
        "type": "DIMENSION", "key": "RECORD_TYPE", "exclude": true,
        "values": ["Distributor Discount", "Refund", "Tax"]
      }
    ],
    "timeRange": { "relative": "YEAR_TO_DATE" }
  }
]
```

| Field | Values |
|-------|--------|
| `name` | Lookup key for `Find`. The `cer` CLI also uses it as the output filename. |
| `metric` | `UnblendedCost`, `BlendedCost`, `AmortizedCost`, `NetUnblendedCost`, `NetAmortizedCost`, `UsageQuantity`, `NormalizedUsageAmount`. One metric per report. |
| `granularity` | `HOURLY`, `DAILY`, `MONTHLY`. |
| `groupBy[]` | `type` is `DIMENSION`, `TAG` or `COST_CATEGORY`; `key` is the API dimension (`SERVICE`, `LINKED_ACCOUNT`, `REGION`, …), the tag key, or the cost category name. Omit for period totals only. |
| `filters[]` | Same `type`/`key` shape plus `values` (API-side values, not console display labels) and optional `exclude`. Clauses combine with AND; `exclude` wraps the clause in NOT. |
| `timeRange.relative` | `YEAR_TO_DATE`, `MONTH_TO_DATE`, `LAST_<n>_DAYS`, `LAST_<n>_MONTHS`, or `CUSTOM`. |
| `timeRange.start` / `end` | `yyyy-MM-dd`; start inclusive, end exclusive. Read only when `relative` is `CUSTOM` (or empty). |

Load a file and pick a report:

```go
specs, err := cereport.LoadSpecs("reports/specs.json")
if err != nil {
    log.Fatal(err)
}
spec, err := cereport.Find(specs, "ytd_aws_by_service_unblended") // a miss lists the available names
```

Specs are normally captured once from a saved report in the Cost Explorer
console. That tooling (console URL → `Spec` JSON) lives with the report data in
`cerep`, not in this package.

---

## Running a report

```go
import (
    "github.com/TouchBistro/gotham/aws/cereport"
    "github.com/aws/aws-sdk-go-v2/config"
    "github.com/aws/aws-sdk-go-v2/service/costexplorer"
)

// Cost Explorer is a global service fronted in us-east-1.
cfg, err := config.LoadDefaultConfig(ctx,
    config.WithSharedConfigProfile("touchbistro"),
    config.WithRegion("us-east-1"),
)
if err != nil {
    log.Fatal(err)
}
api := costexplorer.NewFromConfig(cfg)

res, err := cereport.Run(ctx, api, spec, time.Now().UTC())
```

`Run` builds the request with `spec.GetCostAndUsageInput`, follows every
`NextPageToken`, and accumulates into a `Result`:

| Field | Meaning |
|-------|---------|
| `Periods` | Period start dates, chronological. |
| `Rows` | Group key → per-period amounts, one entry per period. |
| `Unit` | Currency or unit reported by Cost Explorer, e.g. `USD`. |
| `Start`, `End` | The resolved period actually requested. |

A report with no `groupBy` gets a single `(total)` row.

### Time ranges and "today"

Relative ranges resolve against the `now` you pass, in UTC — Cost Explorer
reckons days in UTC. `End` is exclusive, so a to-date range ends on tomorrow's
date to include today, matching the console. Today's data is still settling
(Savings Plan fees are amortized a day later), so for a stable, re-runnable
snapshot exclude it:

```go
res, err := cereport.Run(ctx, api, spec, time.Now().UTC(), cereport.ExcludeCurrentDay())
```

Pass a fixed `now` — say `time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)` — to
reproduce a month-end report later. `CUSTOM` ranges ignore `now` entirely.

---

## Writing CSV

```go
if err := res.WriteCSV(os.Stdout); err != nil {
    log.Fatal(err)
}
```

```
SERVICE,2026-01-01,2026-02-01,2026-03-01,Total
Amazon Elastic Compute Cloud - Compute,41230.11,39877.4,42011.09,123118.6
Amazon Relational Database Service,20112.5,19980.02,20455.7,60548.22
Total,61342.61,59857.42,62466.79,183666.82
```

- **Header:** the group-by keys (`SERVICE`, `SERVICE / repo`, or `Total` when
  ungrouped), then period start dates, then `Total`.
- **Rows** sorted by descending total, ties broken by name. Each row ends with
  its own total; a final `Total` row sums every column.
- **Group keys** are exactly what Cost Explorer returned: tag groups arrive as
  `tagkey$value`.
- **Full precision.** Amounts are written as shortest-round-trip `float64`,
  never rounded to cents. The CSV is an interchange format; a consumer that
  re-aggregates cells lands on the figure Cost Explorer reports. Rounding is the
  presentation layer's job.

`Result` also exposes `Total(key)`, `GrandTotal()` and `SortedKeys()` for callers
that render their own output.

---

## Testing without AWS

`CostExplorerAPI` is the one method this package calls:

```go
type CostExplorerAPI interface {
    GetCostAndUsage(context.Context, *costexplorer.GetCostAndUsageInput,
        ...func(*costexplorer.Options)) (*costexplorer.GetCostAndUsageOutput, error)
}
```

Hand `Run` a fake that returns canned `GetCostAndUsageOutput` pages. The
package's own tests do exactly that and need no credentials.

---

## Permissions and cost

The caller's credentials need the `ce:GetCostAndUsage` action. AWS bills Cost
Explorer API calls per request (see the Cost Explorer pricing page); a report
costs one request per page of groups.

## Limitations

- One metric per report.
- The console's "show only untagged / uncategorized" toggles (an `ABSENT` match
  option) cannot be expressed in a `Spec`.
- Reservation and Savings Plans report modes use different Cost Explorer APIs
  and are out of scope.
