package golang

import (
	"context"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
)

// Test runs Go tests.
var Test = &pk.Task{
	Name:  "go-test",
	Usage: "run go tests",
	Flags: map[string]pk.FlagDef{
		"race":          {Default: true, Usage: "enable race detector"},
		"run":           {Default: "", Usage: "run only tests matching regexp"},
		"timeout":       {Default: "", Usage: "test timeout (e.g., 5m, 30s)"},
		"coverage":      {Default: false, Usage: "enable coverage and write to coverage.out"},
		"coverage-html": {Default: false, Usage: "enable coverage and generate coverage.html"},
		"cpuprofile":    {Default: "", Usage: "write CPU profile to file (e.g., cpu.prof)"},
		"memprofile":    {Default: "", Usage: "write memory profile to file (e.g., mem.prof)"},
		"blockprofile":  {Default: "", Usage: "write block profile to file (e.g., block.prof)"},
		"mutexprofile":  {Default: "", Usage: "write mutex profile to file (e.g., mutex.prof)"},
		"args":          {Default: "", Usage: "additional arguments to pass to go test"},
	},
	Do: func(ctx context.Context) error {
		args := []string{"test"}
		if pk.GetFlag[bool](ctx, "race") {
			args = append(args, "-race")
		}
		coverageHTML := pk.GetFlag[bool](ctx, "coverage-html")
		if pk.GetFlag[bool](ctx, "coverage") || coverageHTML {
			args = append(args, "-coverprofile=coverage.out", "-covermode=atomic")
		}
		if pattern := pk.GetFlag[string](ctx, "run"); pattern != "" {
			args = append(args, "-run", pattern)
		}
		if timeout := pk.GetFlag[string](ctx, "timeout"); timeout != "" {
			args = append(args, "-timeout", timeout)
		}
		if cpuprofile := pk.GetFlag[string](ctx, "cpuprofile"); cpuprofile != "" {
			args = append(args, "-cpuprofile="+cpuprofile)
		}
		if memprofile := pk.GetFlag[string](ctx, "memprofile"); memprofile != "" {
			args = append(args, "-memprofile="+memprofile)
		}
		if blockprofile := pk.GetFlag[string](ctx, "blockprofile"); blockprofile != "" {
			args = append(args, "-blockprofile="+blockprofile)
		}
		if mutexprofile := pk.GetFlag[string](ctx, "mutexprofile"); mutexprofile != "" {
			args = append(args, "-mutexprofile="+mutexprofile)
		}
		if extraArgs := pk.GetFlag[string](ctx, "args"); extraArgs != "" {
			args = append(args, strings.Fields(extraArgs)...)
		}
		args = append(args, "./...")
		if err := pk.Exec(ctx, "go", args...); err != nil {
			return err
		}
		if coverageHTML {
			return pk.Exec(ctx, "go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
		}
		return nil
	},
}
