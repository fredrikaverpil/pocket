// Package pk is the config-authoring API for Pocket, a composable task
// runner framework.
//
// Use this package to define task composition trees in .pocket/config.go.
// For task implementation utilities (command execution, output, flags),
// see [github.com/fredrikaverpil/pocket/pk/run].
//
// # Configuration
//
// Define your task tree using [Config]:
//
//	var Config = &pk.Config{
//	    Auto: pk.Serial(
//	        Format,
//	        pk.Parallel(Lint, Test),
//	        Build,
//	    ),
//	}
//
// # Task Definition
//
// Define tasks as struct literals:
//
//	var Lint = &pk.Task{
//	    Name:  "lint",
//	    Usage: "run linters",
//	    Do: func(ctx context.Context) error {
//	        return run.Exec(ctx, "golangci-lint", "run")
//	    },
//	}
//
// # Composition
//
// Use [Serial] and [Parallel] to compose tasks into execution trees.
// Use [WithOptions] with [WithPath], [WithDetect], and [WithFlags] to
// control path filtering and flag overrides.
//
// # Plan Introspection
//
// Access the execution plan at runtime via [PlanFromContext] to
// generate CI workflows or custom tooling.
package pk
