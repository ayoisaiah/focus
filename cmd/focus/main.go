package main

import (
	"log/slog"
	"os"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ayoisaiah/focus/app"
	_ "github.com/ayoisaiah/focus/internal/static"
	"github.com/ayoisaiah/focus/report"
)

func initLogger() {
	out := &lumberjack.Logger{
		Filename:   "app.log",
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     14,
		Compress:   false,
	}

	opts := &slog.HandlerOptions{}

	l := slog.New(slog.NewTextHandler(out, opts))

	slog.SetDefault(l)
}

func run(args []string) error {
	return app.Get().Run(args)
}

func main() {
	initLogger()

	err := run(os.Args)
	if err != nil {
		report.Quit(err)
	}
}
