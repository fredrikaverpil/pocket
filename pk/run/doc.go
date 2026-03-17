// Package run provides the task-authoring API for Pocket.
//
// Task and tool authors use this package for command execution, output,
// flag handling, and context accessors. Config authors typically only
// need [github.com/fredrikaverpil/pocket/pk].
//
// # Command Execution
//
// Use [Exec] to run external commands with automatic PATH setup,
// output buffering, and graceful shutdown:
//
//	run.Exec(ctx, "golangci-lint", "run", "./...")
//
// # Output
//
// Use [Printf], [Println], and [Errorf] for output that works correctly
// in parallel task execution:
//
//	run.Printf(ctx, "  running: %s\n", cmd)
//
// # Flags
//
// Use [GetFlags] to retrieve resolved flag values from context:
//
//	f := run.GetFlags[TestFlags](ctx)
//
// # Context
//
// Use [PathFromContext], [Verbose], and the ContextWith* functions to
// read and modify execution context:
//
//	if run.Verbose(ctx) {
//	    run.Printf(ctx, "  verbose output\n")
//	}
//
// For plan introspection, use [github.com/fredrikaverpil/pocket/pk.PlanFromContext].
package run
