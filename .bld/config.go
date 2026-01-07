package main

import "github.com/fredrikaverpil/bld"

// Config defines the build configuration for this project.
var Config = bld.Config{
	Go: &bld.GoConfig{
		Modules: map[string]bld.GoModuleOptions{
			".": {},
		},
	},
	Lua: &bld.LuaConfig{
		Modules: map[string]bld.LuaModuleOptions{
			".": {},
		},
	},
	Markdown: &bld.MarkdownConfig{
		Modules: map[string]bld.MarkdownModuleOptions{
			".": {},
		},
	},
	GitHub: &bld.GitHubConfig{
		SkipSync: true, // This is the bld repo itself, no need to sync
	},
}
