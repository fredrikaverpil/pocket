package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
)

// Flag names for the Pprof task.
const (
	FlagPprofFile = "file"
	FlagPprofPort = "port"
)

// Pprof launches the pprof web UI for profile analysis.
var Pprof = &pk.Task{
	Name:  "go-pprof",
	Usage: "launch pprof web UI for profile analysis",
	Flags: map[string]pk.FlagDef{
		FlagPprofFile: {Default: "cpu.prof", Usage: "profile file to analyze"},
		FlagPprofPort: {Default: "8080", Usage: "port for pprof HTTP server"},
	},
	Do: func(ctx context.Context) error {
		file := pk.GetFlag[string](ctx, FlagPprofFile)
		port := pk.GetFlag[string](ctx, FlagPprofPort)
		return pk.Exec(ctx, "go", "tool", "pprof", "-http=:"+port, file)
	},
}
