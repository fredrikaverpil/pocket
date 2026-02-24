package golang

import (
	"context"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/goreleaser"
)

// Flag names for the Release task.
const (
	FlagReleaseSnapshot = "snapshot"
	FlagReleaseClean    = "clean"
	FlagReleaseArgs     = "args"
)

// Release builds and releases Go binaries with goreleaser.
// Defaults to snapshot mode (safe for local development).
var Release = &pk.Task{
	Name:  "go-release",
	Usage: "build and release Go binaries with goreleaser",
	Flags: map[string]pk.FlagDef{
		FlagReleaseSnapshot: {Default: true, Usage: "build without publishing (local/CI preview)"},
		FlagReleaseClean:    {Default: true, Usage: "remove dist/ before build"},
		FlagReleaseArgs:     {Default: "", Usage: "additional goreleaser arguments"},
	},
	Body: pk.Serial(goreleaser.Install, releaseCmd()),
}

func releaseCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		args := []string{"release"}
		if pk.GetFlag[bool](ctx, FlagReleaseSnapshot) {
			args = append(args, "--snapshot")
		}
		if pk.GetFlag[bool](ctx, FlagReleaseClean) {
			args = append(args, "--clean")
		}
		if pk.Verbose(ctx) {
			args = append(args, "--verbose")
		}
		if extraArgs := pk.GetFlag[string](ctx, FlagReleaseArgs); extraArgs != "" {
			args = append(args, strings.Fields(extraArgs)...)
		}
		return pk.Exec(ctx, goreleaser.Name, args...)
	})
}
