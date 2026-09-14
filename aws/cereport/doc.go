// Package cereport parses legacy AWS Cost Explorer saved-report console URLs
// into a durable spec that can be checked into git, translates that spec into
// Cost Explorer API input, runs the report and lays the result out as a CSV
// grid: one row per group, one column per period, a trailing Total.
//
// The Cost Explorer API exposes no way to read saved reports (there is no
// ListSavedReports/GetSavedReport operation), so the console URL — which
// carries the complete report state in its fragment query string — is the only
// machine-readable source for an existing report's definition. ParseURL turns
// that URL into a Spec once; the Spec is what gets versioned, diffed and
// replayed.
//
// The package takes a CostExplorerAPI interface rather than building a client,
// so callers own credentials and configuration and this package adds no
// dependency on aws-sdk-go-v2/config. Cost Explorer is a global service
// fronted in us-east-1.
//
// Basic usage:
//
//	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-1"))
//	if err != nil {
//	    log.Fatal(err)
//	}
//	api := costexplorer.NewFromConfig(cfg)
//
//	specs, err := cereport.LoadSpecs("reports/specs.json")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	spec, err := cereport.Find(specs, "ytd_aws_by_service_unblended")
//	if err != nil {
//	    log.Fatal(err)
//	}
//	res, err := cereport.Run(ctx, api, spec, time.Now().UTC(), cereport.ExcludeCurrentDay())
//	if err != nil {
//	    log.Fatal(err)
//	}
//	if err := res.WriteCSV(os.Stdout); err != nil {
//	    log.Fatal(err)
//	}
//
// To capture a report from the console once and check it in:
//
//	spec, err := cereport.ParseURL(consoleURL) // full URL or the part from the path onwards
//	b, _ := json.MarshalIndent([]*cereport.Spec{spec}, "", "  ")
//	_ = os.WriteFile("reports/specs.json", b, 0o644)
//
// Relative ranges (YEAR_TO_DATE, MONTH_TO_DATE, LAST_N_DAYS, LAST_N_MONTHS)
// resolve against the time passed to Run, so a year-to-date report stays
// year-to-date on every replay; CUSTOM ranges keep their captured dates. The
// console includes today's still-settling data in to-date ranges and so does
// this package by default; pass ExcludeCurrentDay for a stable, re-runnable
// snapshot. The caller's credentials need the ce:GetCostAndUsage action.
package cereport
