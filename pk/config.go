package pk

// Config represents the Pocket configuration.
// It holds the root of the task graph and manual tasks.
type Config struct {
	// Root is the top-level runnable that composes all tasks.
	// This is typically created using Serial() to compose multiple tasks.
	// Tasks in Root are executed on bare `./pok` invocation.
	Root Runnable

	// Manual contains tasks that only run when explicitly invoked.
	// These tasks are not executed as part of Root on bare `./pok`.
	// Example: deploy tasks, setup scripts, or tasks requiring specific flags.
	//
	//	Manual: []pk.Runnable{
	//	    Deploy,
	//	    Hello.Manual(),
	//	}
	Manual []Runnable
}
