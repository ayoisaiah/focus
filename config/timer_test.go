package config

import (
	"testing"

	"github.com/davecgh/go-spew/spew"
	"github.com/urfave/cli/v2"

	"github.com/ayoisaiah/focus"
)

func TestTimerConfig(t *testing.T) {
	app := focus.GetApp()
	spew.Dump(app)

	cli.NewContext(app, nil, nil)
}
