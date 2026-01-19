package pk

// Config represents the Pocket configuration.
// It holds the root of the task graph to be executed.
type Config struct {
	// Root is the top-level runnable that composes all tasks.
	// This is typically created using Serial() to compose multiple tasks.
	Root Runnable
}
