package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket"
)

// TestOptions configures the go-test task.
type TestOptions struct {
	Race     bool `arg:"race"     usage:"enable race detection"`
	Coverage bool `arg:"coverage" usage:"generate coverage.out in git root"`
	Short    bool `arg:"short"    usage:"run short tests only"`
	Verbose  bool `arg:"verbose"  usage:"verbose output"`
}

// Test runs tests with race detection and coverage.
var Test = pocket.Func("go-test", "run Go tests", test).
	With(TestOptions{Race: true, Coverage: true})

func test(ctx context.Context) error {
	opts := pocket.Options[TestOptions](ctx)

	args := []string{"test"}
	if opts.Verbose {
		args = append(args, "-v")
	}
	if opts.Race {
		args = append(args, "-race")
	}
	if opts.Coverage {
		coverPath := pocket.FromGitRoot("coverage.out")
		args = append(args, "-coverprofile="+coverPath)
	}
	if opts.Short {
		args = append(args, "-short")
	}
	args = append(args, "./...")

	return pocket.Exec(ctx, "go", args...)
}
