package golang

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/pk/run"
)

// PprofFlags holds flags for the Pprof task.
type PprofFlags struct {
	File string `flag:"file" usage:"profile file to analyze"`
	Port string `flag:"port" usage:"port for pprof HTTP server"`
}

// Pprof launches the pprof web UI for profile analysis.
var Pprof = &pk.Task{
	Name:  "go-pprof",
	Usage: "launch pprof web UI for profile analysis",
	Flags: PprofFlags{File: "cpu.prof", Port: "8080"},
	Do: func(ctx context.Context) error {
		if _, err := exec.LookPath("dot"); err != nil {
			return fmt.Errorf(
				"graphviz is required for pprof web UI\n  brew install graphviz\n  nix shell nixpkgs#graphviz",
			)
		}
		f := run.GetFlags[PprofFlags](ctx)
		return run.Exec(ctx, "go", "tool", "pprof", "-http=:"+f.Port, f.File)
	},
}
