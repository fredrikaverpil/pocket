package renovate

import "github.com/fredrikaverpil/pocket/pk"

// Tasks returns Renovate-related tasks.
func Tasks() pk.Runnable {
	return Validate
}
