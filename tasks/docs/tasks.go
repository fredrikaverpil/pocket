// Package docs provides documentation tasks.
package docs

import (
	"github.com/fredrikaverpil/pocket/pk"
)

// Tasks returns all docs tasks composed as a Runnable.
func Tasks() pk.Runnable {
	return Zensical
}
