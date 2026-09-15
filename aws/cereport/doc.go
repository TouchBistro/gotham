// Package cereport runs AWS Cost Explorer reports and lays the results out as
// a grid: one row per group, one column per period, a trailing Total.
//
// A report is a Spec — metric, granularity, group-by, filters and a time
// range. Spec.Validate checks it against the rules Cost Explorer enforces;
// Spec.GetCostAndUsageInput translates it into a GetCostAndUsage request; Run
// follows every page of results into a Result; Result.WriteCSV writes the
// grid. The Metric*, Granularity*, Type* and Range* constants name the
// accepted values, and Spec's JSON tags let definitions live in a checked-in
// file.
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
//	spec := cereport.Spec{
//	    Name:        "ytd_aws_by_service_unblended",
//	    Metric:      cereport.MetricUnblendedCost,
//	    Granularity: cereport.GranularityMonthly,
//	    GroupBy:     []cereport.Group{{Type: cereport.TypeDimension, Key: "SERVICE"}},
//	    Filters: []cereport.Filter{{
//	        Type: cereport.TypeDimension, Key: "RECORD_TYPE", Exclude: true,
//	        Values: []string{"Distributor Discount", "Refund", "Tax"},
//	    }},
//	    TimeRange: cereport.TimeRange{Relative: cereport.RangeYearToDate},
//	}
//	res, err := cereport.Run(ctx, api, &spec, time.Now().UTC(), cereport.ExcludeCurrentDay())
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
//
// See README.md in this directory for more Spec and Result examples.
package cereport
