// Package commits provides commit message validation tasks.
package commits

import (
	"github.com/fredrikaverpil/pocket/pk"
)

// Tasks returns all commit tasks composed as a Runnable.
func Tasks() pk.Runnable {
	return Validate
}
