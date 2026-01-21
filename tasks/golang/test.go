package golang

import (
	"context"
	"flag"

	"github.com/fredrikaverpil/pocket/pk"
)

var (
	testFlags = flag.NewFlagSet("go-test", flag.ContinueOnError)
	testRace  = testFlags.Bool("race", true, "enable race detector")
)

// Test runs Go tests.
var Test = pk.NewTask("go-test", "run go tests", testFlags, pk.Do(func(ctx context.Context) error {
	args := []string{"test"}
	if *testRace {
		args = append(args, "-race")
	}
	args = append(args, "./...")

	return pk.Exec(ctx, "go", args...)
}))
