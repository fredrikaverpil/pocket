package lua

import "github.com/fredrikaverpil/pocket"

// AutoConfig defines configuration for auto-detecting Lua files.
type AutoConfig struct {
	// Options are the default options applied to all auto-detected modules.
	Options Options
	// Overrides specifies custom options for specific module paths.
	Overrides map[string]Options
}

// Auto creates a Lua task group that auto-detects modules by finding directories with .lua files.
func Auto(cfg ...AutoConfig) pocket.TaskGroup {
	var config AutoConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}
	return pocket.NewAutoTaskGroup(
		name,
		func() []string { return pocket.DetectByExtension(".lua") },
		config.Options,
		config.Overrides,
		func(modules map[string]Options) pocket.TaskGroup {
			return &taskGroup{modules: modules}
		},
	)
}
