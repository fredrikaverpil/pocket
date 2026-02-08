package golang

import (
	"context"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
)

// Flag names for the Test task.
const (
	FlagTestRace         = "race"
	FlagTestRun          = "run"
	FlagTestTimeout      = "timeout"
	FlagTestCoverage     = "coverage"
	FlagTestCoverageHTML = "coverage-html"
	FlagTestCPUProfile   = "cpuprofile"
	FlagTestMemProfile   = "memprofile"
	FlagTestBlockProfile = "blockprofile"
	FlagTestMutexProfile = "mutexprofile"
	FlagTestArgs         = "args"
)

// Test runs Go tests.
var Test = &pk.Task{
	Name:  "go-test",
	Usage: "run go tests",
	Flags: map[string]pk.FlagDef{
		FlagTestRace:         {Default: true, Usage: "enable race detector"},
		FlagTestRun:          {Default: "", Usage: "run only tests matching regexp"},
		FlagTestTimeout:      {Default: "", Usage: "test timeout (e.g., 5m, 30s)"},
		FlagTestCoverage:     {Default: false, Usage: "enable coverage and write to coverage.out"},
		FlagTestCoverageHTML: {Default: false, Usage: "enable coverage and generate coverage.html"},
		FlagTestCPUProfile:   {Default: "", Usage: "write CPU profile to file (e.g., cpu.prof)"},
		FlagTestMemProfile:   {Default: "", Usage: "write memory profile to file (e.g., mem.prof)"},
		FlagTestBlockProfile: {Default: "", Usage: "write block profile to file (e.g., block.prof)"},
		FlagTestMutexProfile: {Default: "", Usage: "write mutex profile to file (e.g., mutex.prof)"},
		FlagTestArgs:         {Default: "", Usage: "additional arguments to pass to go test"},
	},
	Do: func(ctx context.Context) error {
		args := []string{"test"}
		if pk.GetFlag[bool](ctx, FlagTestRace) {
			args = append(args, "-race")
		}
		coverageHTML := pk.GetFlag[bool](ctx, FlagTestCoverageHTML)
		if pk.GetFlag[bool](ctx, FlagTestCoverage) || coverageHTML {
			args = append(args, "-coverprofile=coverage.out", "-covermode=atomic")
		}
		if pattern := pk.GetFlag[string](ctx, FlagTestRun); pattern != "" {
			args = append(args, "-run", pattern)
		}
		if timeout := pk.GetFlag[string](ctx, FlagTestTimeout); timeout != "" {
			args = append(args, "-timeout", timeout)
		}
		if cpuprofile := pk.GetFlag[string](ctx, FlagTestCPUProfile); cpuprofile != "" {
			args = append(args, "-cpuprofile="+cpuprofile)
		}
		if memprofile := pk.GetFlag[string](ctx, FlagTestMemProfile); memprofile != "" {
			args = append(args, "-memprofile="+memprofile)
		}
		if blockprofile := pk.GetFlag[string](ctx, FlagTestBlockProfile); blockprofile != "" {
			args = append(args, "-blockprofile="+blockprofile)
		}
		if mutexprofile := pk.GetFlag[string](ctx, FlagTestMutexProfile); mutexprofile != "" {
			args = append(args, "-mutexprofile="+mutexprofile)
		}
		if extraArgs := pk.GetFlag[string](ctx, FlagTestArgs); extraArgs != "" {
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
