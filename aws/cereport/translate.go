package cereport

import (
	"fmt"
	"regexp"
	"strconv"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/costexplorer"
	cetypes "github.com/aws/aws-sdk-go-v2/service/costexplorer/types"
)

// lastNRange matches the console's rolling ranges, e.g. LAST_6_MONTHS.
var lastNRange = regexp.MustCompile(`^LAST_(\d+)_(DAYS|MONTHS)$`)

// PeriodOption adjusts how a relative range resolves.
type PeriodOption func(*periodConfig)

type periodConfig struct{ excludeCurrentDay bool }

// ExcludeCurrentDay ends a to-date range at the start of today rather than
// including today's partial data.
//
// The console includes today: a year-to-date report downloaded on the 30th
// covers through the 30th. Matching that is the default, since these specs
// exist to reproduce those reports. But the current day is still being
// written — its Savings Plan fees have not been amortized yet, so they appear
// un-amortized and net out a day later — so a report meant to be stable and
// re-runnable should exclude it.
func ExcludeCurrentDay() PeriodOption {
	return func(c *periodConfig) { c.excludeCurrentDay = true }
}

// ResolvePeriod turns the spec's time range into the absolute start/end dates
// the Cost Explorer API takes. Start is inclusive and End is exclusive, so a
// to-date range ends on tomorrow's date in order to include today, which is
// what the console does.
//
// Pass the current time in UTC; Cost Explorer reckons days in UTC.
func (s Spec) ResolvePeriod(now time.Time, opts ...PeriodOption) (start, end string, err error) {
	var cfg periodConfig
	for _, o := range opts {
		o(&cfg)
	}
	now = now.UTC()
	const layout = "2006-01-02"

	if s.TimeRange.IsCustom() {
		if s.TimeRange.Start == "" || s.TimeRange.End == "" {
			return "", "", fmt.Errorf("report %q has a custom range with no start/end dates", s.Name)
		}
		return s.TimeRange.Start, s.TimeRange.End, nil
	}

	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	// End is exclusive, so including today means ending on tomorrow's date.
	last := today.AddDate(0, 0, 1)
	if cfg.excludeCurrentDay {
		last = today
	}
	switch r := s.TimeRange.Relative; {
	case r == "YEAR_TO_DATE":
		return time.Date(now.Year(), time.January, 1, 0, 0, 0, 0, time.UTC).Format(layout), last.Format(layout), nil
	case r == "MONTH_TO_DATE":
		return time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).Format(layout), last.Format(layout), nil
	case lastNRange.MatchString(r):
		m := lastNRange.FindStringSubmatch(r)
		n, _ := strconv.Atoi(m[1])
		if m[2] == "DAYS" {
			return today.AddDate(0, 0, -n).Format(layout), last.Format(layout), nil
		}
		// Rolling months run from the first of the month N months back, so the
		// buckets line up with calendar months under MONTHLY granularity.
		from := time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC).AddDate(0, -n, 0)
		return from.Format(layout), last.Format(layout), nil
	default:
		return "", "", fmt.Errorf("report %q: unhandled relative range %q; add it to ResolvePeriod", s.Name, r)
	}
}

// GetCostAndUsageInput builds the Cost Explorer request for this report as of
// the given time. Reports with a relative range resolve against `now`, so a
// year-to-date report stays year-to-date on every replay.
func (s Spec) GetCostAndUsageInput(now time.Time, opts ...PeriodOption) (*costexplorer.GetCostAndUsageInput, error) {
	start, end, err := s.ResolvePeriod(now, opts...)
	if err != nil {
		return nil, err
	}

	in := &costexplorer.GetCostAndUsageInput{
		TimePeriod:  &cetypes.DateInterval{Start: aws.String(start), End: aws.String(end)},
		Granularity: cetypes.Granularity(s.Granularity),
		Metrics:     []string{s.Metric},
	}
	for _, g := range s.GroupBy {
		in.GroupBy = append(in.GroupBy, cetypes.GroupDefinition{
			Type: cetypes.GroupDefinitionType(g.Type),
			Key:  aws.String(g.Key),
		})
	}
	if in.Filter, err = s.Expression(); err != nil {
		return nil, err
	}
	return in, nil
}

// Expression builds the Cost Explorer filter expression. Filter rows combine
// with AND, matching how the console's filter panel behaves, and an excluding
// row is wrapped in a NOT.
func (s Spec) Expression() (*cetypes.Expression, error) {
	if len(s.Filters) == 0 {
		return nil, nil
	}
	exprs := make([]cetypes.Expression, 0, len(s.Filters))
	for _, f := range s.Filters {
		e, err := f.expression()
		if err != nil {
			return nil, err
		}
		exprs = append(exprs, *e)
	}
	// Cost Explorer rejects an And with a single element, so a lone filter is
	// passed through on its own.
	if len(exprs) == 1 {
		return &exprs[0], nil
	}
	return &cetypes.Expression{And: exprs}, nil
}

func (f Filter) expression() (*cetypes.Expression, error) {
	var e cetypes.Expression
	switch f.Type {
	case "DIMENSION":
		e.Dimensions = &cetypes.DimensionValues{
			Key:          cetypes.Dimension(f.Key),
			Values:       f.Values,
			MatchOptions: []cetypes.MatchOption{cetypes.MatchOptionEquals},
		}
	case "TAG":
		e.Tags = &cetypes.TagValues{
			Key:          aws.String(f.Key),
			Values:       f.Values,
			MatchOptions: []cetypes.MatchOption{cetypes.MatchOptionEquals},
		}
	case "COST_CATEGORY":
		e.CostCategories = &cetypes.CostCategoryValues{
			Key:          aws.String(f.Key),
			Values:       f.Values,
			MatchOptions: []cetypes.MatchOption{cetypes.MatchOptionEquals},
		}
	default:
		return nil, fmt.Errorf("filter on %q: unknown type %q", f.Key, f.Type)
	}
	if f.Exclude {
		return &cetypes.Expression{Not: &e}, nil
	}
	return &e, nil
}
