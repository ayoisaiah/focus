package main

import (
	"log/slog"
	"os"

	"github.com/pterm/pterm"
	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/ayoisaiah/focus/app"
	"github.com/ayoisaiah/focus/config"
)

func initLogger() {
	out := &lumberjack.Logger{
		Filename:   config.LogFilePath(),
		MaxSize:    10,
		MaxBackups: 5,
		MaxAge:     14,
		Compress:   false,
	}

	opts := &slog.HandlerOptions{}

	l := slog.New(slog.NewTextHandler(out, opts)).With(
		slog.Int("pid", os.Getpid()),
	)

	slog.SetDefault(l)
}

func run(args []string) error {
	return app.Get().Run(args)
}

func main() {
	config.InitializePaths()

	initLogger()

	err := run(os.Args)
	if err != nil {
		slog.Error("an error occurred", slog.Any("error", err))
		pterm.Error.Println(err)
		os.Exit(1)
	}
}
