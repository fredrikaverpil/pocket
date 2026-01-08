package markdown

import "github.com/fredrikaverpil/pocket"

// AutoConfig defines configuration for auto-detecting Markdown files.
type AutoConfig struct {
	// Options are the default options applied to all auto-detected modules.
	Options Options
	// Overrides specifies custom options for specific module paths.
	Overrides map[string]Options
}

// Auto creates a Markdown task group that runs from the repository root.
// Since markdown files are typically scattered throughout a project,
// this defaults to running mdformat from root rather than detecting individual directories.
func Auto(cfg ...AutoConfig) pocket.TaskGroup {
	var config AutoConfig
	if len(cfg) > 0 {
		config = cfg[0]
	}
	return pocket.NewAutoTaskGroup(
		name,
		func() []string { return []string{"."} }, // Just use root for markdown.
		config.Options,
		config.Overrides,
		func(modules map[string]Options) pocket.TaskGroup {
			return &taskGroup{modules: modules}
		},
	)
}
