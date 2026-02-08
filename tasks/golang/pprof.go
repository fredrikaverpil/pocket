package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
)

// Pprof launches the pprof web UI for profile analysis.
var Pprof = &pk.Task{
	Name:  "go-pprof",
	Usage: "launch pprof web UI for profile analysis",
	Flags: map[string]pk.FlagDef{
		"file": {Default: "cpu.prof", Usage: "profile file to analyze"},
		"port": {Default: "8080", Usage: "port for pprof HTTP server"},
	},
	Do: func(ctx context.Context) error {
		file := pk.GetFlag[string](ctx, "file")
		port := pk.GetFlag[string](ctx, "port")
		return pk.Exec(ctx, "go", "tool", "pprof", "-http=:"+port, file)
	},
}
