package main

import (
	"github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/tasks/golang"
	"github.com/fredrikaverpil/pocket/tasks/lua"
	"github.com/fredrikaverpil/pocket/tasks/markdown"
)

var Config = pocket.Config{
	TaskGroups: []pocket.TaskGroup{
		golang.New(golang.Config{
			Modules: map[string]golang.Options{
				".": {},
			},
		}),
		lua.New(lua.Config{
			Modules: map[string]lua.Options{
				".": {},
			},
		}),
		markdown.New(markdown.Config{
			Modules: map[string]markdown.Options{
				".": {},
			},
		}),
	},
	Shim: &pocket.ShimConfig{
		Posix:      true,
		Windows:    true,
		PowerShell: true,
	},
}
