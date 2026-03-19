package docs

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/zensical"
)

// ZensicalFlags holds flags for the Zensical task.
type ZensicalFlags struct {
	Serve bool `flag:"serve" usage:"serve documentation locally"`
	Build bool `flag:"build" usage:"build documentation"`
}

// Zensical generates or serves documentation using zensical.
// Automatically installs zensical if not present. Builds documentation if no flags are passed.
var Zensical = &pk.Task{
	Name:  "docs",
	Usage: "generate or serve documentation with zensical",
	Flags: ZensicalFlags{},
	Body:  pk.Serial(zensical.Install, zensicalCmd()),
}

func zensicalCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[ZensicalFlags](ctx)

		serve := f.Serve
		build := f.Build

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
