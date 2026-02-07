package docs

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/uv"
	"github.com/fredrikaverpil/pocket/tools/zensical"
)

// Zensical generates or serves documentation using zensical.
// Automatically installs zensical if not present. Builds documentation if no flags are passed.
var Zensical = &pk.Task{
	Name:  "docs",
	Usage: "generate or serve documentation with zensical",
	Flags: map[string]pk.FlagDef{
		"serve": {Default: false, Usage: "serve documentation locally"},
		"build": {Default: false, Usage: "build documentation"},
	},
	Body: pk.Serial(zensical.Install, zensicalCmd()),
}

func zensicalCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		serve := pk.GetFlag[bool](ctx, "serve")
		build := pk.GetFlag[bool](ctx, "build")

		// Default to build if neither flag is specified
		if !serve && !build {
			build = true
		}

		var args []string
		if serve {
			args = []string{"serve"}
		} else if build {
			args = []string{"build"}
		}

		// Pass verbose flag to zensical
		if pk.Verbose(ctx) {
			args = append(args, "--verbose")
		}

		// Run zensical via uv from its isolated venv
		return uv.Run(ctx, uv.RunOptions{
			PythonVersion: uv.DefaultPythonVersion,
			VenvPath:      zensical.VenvPath(),
			ProjectDir:    zensical.InstallDir(),
		}, zensical.Name, args...)
	})
}
