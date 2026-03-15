package pk

import (
	"runtime/debug"
)

// version returns the current version of Pocket.
// It reads version info embedded by Go 1.18+ during `go build`.
func version() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	// Check if Pocket is a dependency (user's .pocket/ importing us)
	for _, dep := range info.Deps {
		if dep.Path == "github.com/fredrikaverpil/pocket" {
			// v0.0.0 means replace directive - fall through to VCS check
			if dep.Version != "" && dep.Version != "v0.0.0" {
				return dep.Version
			}
			break
		}
	}

	// Try to get VCS info (works when building in the Pocket repo)
	var revision, dirty string
	for _, s := range info.Settings {
		switch s.Key {
		case "vcs.revision":
			if len(s.Value) >= 7 {
				revision = s.Value[:7]
			} else {
				revision = s.Value
			}
		case "vcs.modified":
			if s.Value == "true" {
				dirty = "-dirty"
			}
		}
	}
	if revision != "" {
		return "dev-" + revision + dirty
	}

	return "dev"
}
