package config

import (
	"errors"
	"io"
	"os"
	"strings"
	"time"

	"github.com/araddon/dateparse"
	"github.com/pterm/pterm"
	"github.com/urfave/cli/v2"
	"golang.org/x/exp/slices"

	"github.com/ayoisaiah/focus/internal/timeutil"
)

var errInvalidDateRange = errors.New(
	"The end date must not be earlier than the start date",
)

var statsCfg *StatsConfig

type StatsConfig struct {
	Stderr    io.Writer `json:"-"`
	Stdout    io.Writer `json:"-"`
	Stdin     io.Reader `json:"-"`
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

	//nolint:exhaustive // delibrate inexhaustive switch
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
	}

	start = now.AddDate(0, 0, timeutil.Range[period])

	return
}

// setStatsConfig updates the stats configuration from command-line arguments.
func setStatsConfig(ctx *cli.Context) error {
	statsCfg.Stderr = os.Stderr
	statsCfg.Stdout = os.Stdout
	statsCfg.Stdin = os.Stdin

	if (ctx.String("tag")) != "" {
		statsCfg.Tags = strings.Split(ctx.String("tag"), ",")
	}

	period := timeutil.Period(ctx.String("period"))

	if !slices.Contains(timeutil.PeriodCollection, period) {
		period = timeutil.Period7Days
	}

	statsCfg.StartTime, statsCfg.EndTime = getTimeRange(period)

	start := ctx.String("start")
	if start != "" {
		dateTime, err := dateparse.ParseAny(start)
		if err != nil {
			return err
		}

		statsCfg.StartTime = dateTime
	}

	end := ctx.String("end")
	if end != "" {
		dateTime, err := dateparse.ParseAny(end)
		if err != nil {
			return err
		}

		statsCfg.EndTime = dateTime
	}

	if int(statsCfg.EndTime.Sub(statsCfg.StartTime).Seconds()) < 0 {
		return errInvalidDateRange
	}

	return nil
}

// GetStats initializes and returns the stats configuration from
// command-line arguments.
func GetStats(ctx *cli.Context) *StatsConfig {
	once.Do(func() {
		now := time.Now()
		start := timeutil.RoundToStart(now)

		statsCfg = &StatsConfig{
			StartTime: start.AddDate(0, 0, -6),
			EndTime:   timeutil.RoundToEnd(start),
			PathToDB:  pathToDB,
		}

		if err := setStatsConfig(ctx); err != nil {
			pterm.Error.Println(err)
			os.Exit(1)
		}
	})

	return statsCfg
}
