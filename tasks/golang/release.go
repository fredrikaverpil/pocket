package golang

import (
	"context"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
	"github.com/fredrikaverpil/pocket/tools/goreleaser"
)

// ReleaseFlags holds flags for the Release task.
type ReleaseFlags struct {
	Snapshot bool   `flag:"snapshot" usage:"build without publishing (local/CI preview)"`
	Clean    bool   `flag:"clean"    usage:"remove dist/ before build"`
	Args     string `flag:"args"     usage:"additional goreleaser arguments"`
}

// Release builds and releases Go binaries with goreleaser.
// Defaults to snapshot mode (safe for local development).
var Release = &pk.Task{
	Name:  "go-release",
	Usage: "build and release Go binaries with goreleaser",
	Flags: ReleaseFlags{Snapshot: true, Clean: true},
	Body:  pk.Serial(goreleaser.Install, releaseCmd()),
}

func releaseCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		f := run.GetFlags[ReleaseFlags](ctx)
		args := []string{"release"}
		if f.Snapshot {
			args = append(args, "--snapshot")
		}
		if f.Clean {
			args = append(args, "--clean")
		}
		if run.Verbose(ctx) {
			args = append(args, "--verbose")
		}
		if f.Args != "" {
			args = append(args, strings.Fields(f.Args)...)
		}
		return run.Exec(ctx, goreleaser.Name, args...)
	})
}
