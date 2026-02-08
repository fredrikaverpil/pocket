package docs

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/zensical"
)

// Flag names for the Zensical task.
const (
	FlagServe = "serve"
	FlagBuild = "build"
)

// Zensical generates or serves documentation using zensical.
// Automatically installs zensical if not present. Builds documentation if no flags are passed.
var Zensical = &pk.Task{
	Name:  "docs",
	Usage: "generate or serve documentation with zensical",
	Flags: map[string]pk.FlagDef{
		FlagServe: {Default: false, Usage: "serve documentation locally"},
		FlagBuild: {Default: false, Usage: "build documentation"},
	},
	Body: pk.Serial(zensical.Install, zensicalCmd()),
}

func zensicalCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		serve := pk.GetFlag[bool](ctx, FlagServe)
		build := pk.GetFlag[bool](ctx, FlagBuild)

		// Default to build if neither flag is specified.
		if !serve && !build {
			build = true
		}

		var args []string
		if serve {
			args = []string{"serve"}
		} else if build {
			args = []string{"build"}
		}

		return zensical.Exec(ctx, args...)
	})
}
