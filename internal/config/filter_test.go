package config

import (
	"flag"
	"slices"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

type FilterTest struct {
	Name     string
	Args     []string
	Flags    map[string]string
	Expected FilterConfig
}

var filterTestCases = []FilterTest{
	{
		Name: "Provide a valid perid",
		Args: []string{"-period 7days"},
		Flags: map[string]string{
			"period": "7days",
		},
	},
}

func TestFilter(t *testing.T) {
	for _, tc := range filterTestCases {
		t.Run(tc.Name, func(t *testing.T) {
			f := flag.NewFlagSet("stats", flag.PanicOnError)
			for k, v := range tc.Flags {
				_ = f.String(k, "", "")

				err := f.Set(k, v)
				if err != nil {
					t.Log(err)
				}
			}

			ctx := cli.NewContext(&cli.App{}, f, nil)

			cfg := Filter(ctx)

			var expectedTags []string

			if _, ok := tc.Flags["tag"]; ok {
				expectedTags = strings.Split(tc.Flags["tag"], ",")
			}

			if !slices.Equal(cfg.Tags, expectedTags) {
				t.Errorf(
					"expected tags to be: %v, but got: %v",
					expectedTags,
					cfg.Tags,
				)
			}
		})
	}
}
