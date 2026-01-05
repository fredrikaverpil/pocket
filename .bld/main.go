package main

import (
	"os"
	"os/exec"

	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tasks/golang"
	"github.com/fredrikaverpil/bld/workflows"
	"github.com/goyek/goyek/v3"
	"github.com/goyek/x/boot"
)

// Register Go tasks
var tasks = golang.NewTasks(Config)

// Update updates bld and generates CI workflows
var update = goyek.Define(goyek.Task{
	Name:  "update",
	Usage: "update bld and generate CI workflows",
	Action: func(a *goyek.A) {
		// Update bld dependency and wrapper script
		cmd := exec.CommandContext(a.Context(), "go", "run", "github.com/fredrikaverpil/bld/cmd/bld@latest", "update")
		cmd.Dir = bld.FromGitRoot()
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			a.Fatalf("bld update: %v", err)
		}

		// Generate workflows
		if err := workflows.Generate(Config); err != nil {
			a.Fatal(err)
		}
		a.Log("Generated workflows in .github/workflows/")
	},
})

func main() {
	goyek.SetDefault(tasks.All)
	boot.Main()
}
