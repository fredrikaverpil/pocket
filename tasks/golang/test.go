package golang

import (
	"context"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
)

// TestFlags holds flags for the Test task.
type TestFlags struct {
	Race         bool   `flag:"race"          usage:"enable race detector"`
	Run          string `flag:"run"           usage:"run only tests matching regexp"`
	Timeout      string `flag:"timeout"       usage:"test timeout (e.g., 5m, 30s)"`
	Coverage     bool   `flag:"coverage"      usage:"enable coverage and write to coverage.out"`
	CoverageHTML bool   `flag:"coverage-html" usage:"enable coverage and generate coverage.html"`
	CPUProfile   string `flag:"cpuprofile"    usage:"write CPU profile to file (e.g., cpu.prof)"`
	MemProfile   string `flag:"memprofile"    usage:"write memory profile to file (e.g., mem.prof)"`
	BlockProfile string `flag:"blockprofile"  usage:"write block profile to file (e.g., block.prof)"`
	MutexProfile string `flag:"mutexprofile"  usage:"write mutex profile to file (e.g., mutex.prof)"`
	Pkg          string `flag:"pkg"           usage:"package pattern to test (e.g., ./pk)"`
	Args         string `flag:"args"          usage:"additional arguments to pass to go test"`
}

// Test runs Go tests.
var Test = &pk.Task{
	Name:  "go-test",
	Usage: "run go tests",
	Flags: TestFlags{Race: true, Pkg: "./..."},
	Do: func(ctx context.Context) error {
		f := pk.GetFlags[TestFlags](ctx)
		args := []string{"test"}
		if f.Race {
			args = append(args, "-race")
		}
		if f.Coverage || f.CoverageHTML {
			args = append(args, "-coverprofile=coverage.out", "-covermode=atomic")
		}
		if f.Run != "" {
			args = append(args, "-run", f.Run)
		}
		if f.Timeout != "" {
			args = append(args, "-timeout", f.Timeout)
		}
		if f.CPUProfile != "" {
			args = append(args, "-cpuprofile="+f.CPUProfile)
		}
		if f.MemProfile != "" {
			args = append(args, "-memprofile="+f.MemProfile)
		}
		if f.BlockProfile != "" {
			args = append(args, "-blockprofile="+f.BlockProfile)
		}
		if f.MutexProfile != "" {
			args = append(args, "-mutexprofile="+f.MutexProfile)
		}
		if f.Args != "" {
			args = append(args, strings.Fields(f.Args)...)
		}
		args = append(args, f.Pkg)
		if err := pk.Exec(ctx, "go", args...); err != nil {
			return err
		}
		if f.CoverageHTML {
			return pk.Exec(ctx, "go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
		}
		return nil
	},
}
