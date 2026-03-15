// Package pk is the core engine for Pocket, a composable task runner framework.
//
// Tasks are the fundamental units of work. Compose them into execution trees
// using [Serial] and [Parallel], then configure path filtering and flag
// overrides with [WithOptions].
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
// Define tasks as struct literals with a Do function or composed Body:
//
//	var Lint = &pk.Task{
//	    Name:  "lint",
//	    Usage: "run linters",
//	    Do: func(ctx context.Context) error {
//	        return pk.Exec(ctx, "golangci-lint", "run")
//	    },
//	}
//
// # Execution
//
// Use [Exec] to run external commands and [Do] to wrap Go functions as
// [Runnable] values. Use [Printf], [Println], and [Errorf] for output
// that works correctly in parallel contexts.
//
// # Path Filtering
//
// Use [WithOptions] with [WithPath], [WithSkipPath], or [WithDetect] to
// control which directories tasks execute in. Use [WithFlags] and
// [WithNameSuffix] to create task variants.
//
// # Plan Introspection
//
// [NewPlan] builds an execution plan from a [Config]. Access it at runtime
// via [PlanFromContext] to generate CI workflows or custom tooling.
package pk
