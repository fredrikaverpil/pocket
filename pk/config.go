package pk

// Config represents the Pocket configuration.
// It holds the task graph composition and manual tasks.
type Config struct {
	// Auto is the top-level runnable that composes all tasks.
	// This is typically created using Serial() to compose multiple tasks.
	// Tasks in Auto are executed on bare `./pok` invocation.
	Auto Runnable

	// Manual contains tasks that only run when explicitly invoked.
	// These tasks are not executed as part of Auto on bare `./pok`.
	// Example: deploy tasks, setup scripts, or tasks requiring specific flags.
	//
	//	Manual: []pk.Runnable{
	//	    Deploy,
	//	    Hello.Manual(),
	//	}
	Manual []Runnable
}
