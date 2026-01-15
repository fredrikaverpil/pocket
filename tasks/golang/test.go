package golang

import (
	"context"

	"github.com/fredrikaverpil/pocket"
)

// TestOptions configures the go-test task.
type TestOptions struct {
	SkipRace     bool `arg:"skip-race"     usage:"disable race detection"`
	SkipCoverage bool `arg:"skip-coverage" usage:"disable coverage generation"`
	Short        bool `arg:"short"         usage:"run short tests only"`
	Verbose      bool `arg:"verbose"       usage:"verbose output"`
}

// Test runs tests with race detection and coverage by default.
var Test = pocket.Func("go-test", "run Go tests", test).
	With(TestOptions{})

func test(ctx context.Context) error {
	opts := pocket.Options[TestOptions](ctx)

	args := []string{"test"}
	if opts.Verbose {
		args = append(args, "-v")
	}
	if !opts.SkipRace {
		args = append(args, "-race")
	}
	if !opts.SkipCoverage {
		coverPath := pocket.FromGitRoot("coverage.out")
		args = append(args, "-coverprofile="+coverPath)
	}
	if opts.Short {
		args = append(args, "-short")
	}
	args = append(args, "./...")

	return pocket.Exec(ctx, "go", args...)
}
