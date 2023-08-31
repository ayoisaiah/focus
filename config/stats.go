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
		"the end date must not be earlier than the start date",
	)

	errInvalidPeriod = errors.New(
		"please provide a valid time period",
	)

	errInvalidStartDate = errors.New(
		"please provide a valid start date",
	)
)

var statsCfg *StatsConfig

type StatsConfig struct {
	StartTime time.Time
	EndTime   time.Time
	PathToDB  string
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

// setStatsConfig updates the stats configuration from command-line arguments.
func setStatsConfig(ctx *cli.Context) error {
	if (ctx.String("tag")) != "" {
		statsCfg.Tags = strings.Split(ctx.String("tag"), ",")
	}

	period := timeutil.Period(strings.TrimSpace(ctx.String("period")))

	if period != "" && !slices.Contains(timeutil.PeriodCollection, period) {
		return errInvalidPeriod
	}

	if period != "" {
		statsCfg.StartTime, statsCfg.EndTime = getTimeRange(period)

		return nil
	}

	start := ctx.String("start")
	if start != "" {
		dateTime, err := dateparse.ParseAny(start)
		if err != nil {
			return err
		}

		statsCfg.StartTime = dateTime
	}

	statsCfg.EndTime = time.Now()

	end := ctx.String("end")
	if end != "" {
		dateTime, err := dateparse.ParseAny(end)
		if err != nil {
			return err
		}

		statsCfg.EndTime = dateTime
	}

	if statsCfg.StartTime.IsZero() {
		return errInvalidStartDate
	}

	if int(statsCfg.EndTime.Sub(statsCfg.StartTime).Seconds()) < 0 {
		return errInvalidDateRange
	}

	return nil
}

// Stats initializes and returns the stats configuration from
// command-line arguments.
func Stats(ctx *cli.Context) *StatsConfig {
	once.Do(func() {
		statsCfg = &StatsConfig{
			PathToDB: dbFilePath,
		}

		if err := setStatsConfig(ctx); err != nil {
			pterm.Error.Println(err)
			os.Exit(1)
		}
	})

	return statsCfg
}
