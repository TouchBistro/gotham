// Package cereport runs AWS Cost Explorer reports from checked-in definitions
// and lays the results out as a CSV grid: one row per group, one column per
// period, a trailing Total.
//
// A report is a Spec — metric, granularity, group-by, filters and a time
// range — stored as JSON and loaded with LoadSpecs:
//
//	[{"name": "ytd_aws_by_service_unblended",
//	  "metric": "UnblendedCost", "granularity": "MONTHLY",
//	  "groupBy": [{"type": "DIMENSION", "key": "SERVICE"}],
//	  "filters": [{"type": "DIMENSION", "key": "RECORD_TYPE", "exclude": true,
//	               "values": ["Distributor Discount", "Refund", "Tax"]}],
//	  "timeRange": {"relative": "YEAR_TO_DATE"}}]
//
// Spec.GetCostAndUsageInput translates a Spec into a GetCostAndUsage request,
// Run follows every page of results into a Result, and Result.WriteCSV writes
// the grid.
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
// Relative ranges (YEAR_TO_DATE, MONTH_TO_DATE, LAST_N_DAYS, LAST_N_MONTHS)
// resolve against the time passed to Run, so a year-to-date report stays
// year-to-date on every replay; CUSTOM ranges use their Start and End. To-date
// ranges include today by default, matching the Cost Explorer console; today's
// data is still settling, so pass ExcludeCurrentDay for a stable, re-runnable
// snapshot. The caller's credentials need the ce:GetCostAndUsage action.
package cereport
