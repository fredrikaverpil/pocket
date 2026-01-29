package github

import "github.com/fredrikaverpil/pocket/pk"

// Tasks returns the Workflows task.
func Tasks() pk.Runnable {
	return Workflows
}
