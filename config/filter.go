package config

import (
	"errors"
	"os"
	"slices"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus/internal/timeutil"
)

var (
	errInvalidDateRange = errors.New(
		"the start time must be earlier than the end time",
	)

	errInvalidPeriod = errors.New(
		"please provide a valid time period",
	)

	errInvalidStartDate = errors.New(
		"please provide a valid start date",
	)
)

// FilterConfig represents a configuration to filter sessions
// in the database by their start time, end time, and assigned tags.
type FilterConfig struct {
	StartTime time.Time
	EndTime   time.Time
	Tags      []string
}

// getTimeRange returns the start and end time according to the
// specified time period.
func getTimeRange(period timeutil.Period) (start, end time.Time) {
	now := time.Now()

	start = timeutil.RoundToStart(now)

	end = timeutil.RoundToEnd(now)

	//nolint:exhaustive // other cases covered by default
	switch period {
	case timeutil.PeriodToday:
		return
	case timeutil.PeriodYesterday:
		start = now.AddDate(0, 0, timeutil.Range[period])
		start = timeutil.RoundToStart(start)
		end = timeutil.RoundToEnd(start)

		return
	case timeutil.PeriodAllTime:
		start = time.Time{}
		return
	default:
		start = now.AddDate(0, 0, timeutil.Range[period])
		start = timeutil.RoundToStart(start)
	}

	return
}

// setFilterConfig updates the filter configuration from command-line arguments.
func setFilterConfig(ctx *cli.Context) (*FilterConfig, error) {
	filterCfg := &FilterConfig{}

	if (ctx.String("tag")) != "" {
		filterCfg.Tags = strings.Split(ctx.String("tag"), ",")
	}

	period := timeutil.Period(strings.TrimSpace(ctx.String("period")))

	if period != "" && !slices.Contains(timeutil.PeriodCollection, period) {
		return nil, errInvalidPeriod
	}

	if period != "" {
		filterCfg.StartTime, filterCfg.EndTime = getTimeRange(period)

		return filterCfg, nil
	}

	start := ctx.String("start")
	if start != "" {
		dateTime, err := dateparse.ParseAny(start)
		if err != nil {
			return nil, err
		}

		filterCfg.StartTime = dateTime
	}

	now := time.Now()

	if now.After(filterCfg.StartTime) {
		filterCfg.EndTime = now
	} else {
		filterCfg.EndTime = timeutil.RoundToEnd(filterCfg.StartTime)
	}

	end := ctx.String("end")
	if end != "" {
		dateTime, err := dateparse.ParseAny(end)
		if err != nil {
			return nil, err
		}

		filterCfg.EndTime = dateTime
	}

	if filterCfg.StartTime.IsZero() {
		return nil, errInvalidStartDate
	}

	if int(filterCfg.EndTime.Sub(filterCfg.StartTime).Seconds()) < 0 {
		return nil, errInvalidDateRange
	}

	return filterCfg, nil
}

// Filter initializes and returns a configuration to filter sessions from
// command-line arguments.
func Filter(ctx *cli.Context) *FilterConfig {
	cfg, err := setFilterConfig(ctx)
	if err != nil {
		pterm.Error.Println(err)
		os.Exit(1)
	}

	return cfg
}
