package main

import "github.com/fredrikaverpil/pocket"

// Config defines the build configuration for this project.
var Config = pocket.Config{
	Shim: &pocket.ShimConfig{
		Posix:      true,
		Windows:    true,
		PowerShell: true,
	},
	Go: &pocket.GoConfig{
		Modules: map[string]pocket.GoModuleOptions{
			".": {},
		},
	},
	Lua: &pocket.LuaConfig{
		Modules: map[string]pocket.LuaModuleOptions{
			".": {},
		},
	},
	Markdown: &pocket.MarkdownConfig{
		Modules: map[string]pocket.MarkdownModuleOptions{
			".": {},
		},
	},
}
