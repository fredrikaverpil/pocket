package main

import (
	"os"
	"os/exec"

	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tasks"
	"github.com/goyek/goyek/v3"
	"github.com/goyek/x/boot"
)

// All tasks are automatically created based on Config.
var t = tasks.New(Config)

// Update updates bld dependency.
var _ = goyek.Define(goyek.Task{
	Name:  "update",
	Usage: "update bld dependency",
	Action: func(a *goyek.A) {
		cmd := exec.CommandContext(a.Context(), "go", "run", "github.com/fredrikaverpil/bld/cmd/bld@latest", "update")
		cmd.Dir = bld.FromGitRoot()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			a.Fatalf("bld update: %v", err)
		}
	},
})

func main() {
	goyek.SetDefault(t.All)
	boot.Main()
}
