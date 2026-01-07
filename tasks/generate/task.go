// Package generate provides the generate task for regenerating all generated files.
package generate

import (
	pocket "github.com/fredrikaverpil/pocket"
	"github.com/fredrikaverpil/pocket/internal/scaffold"
	"github.com/goyek/goyek/v3"
)

// Task returns a goyek task that regenerates all generated files.
func Task(cfg pocket.Config) *goyek.DefinedTask {
	return goyek.Define(goyek.Task{
		Name:  "generate",
		Usage: "regenerate all generated files (main.go, shim)",
		Action: func(a *goyek.A) {
			if err := scaffold.GenerateAll(&cfg); err != nil {
				a.Fatal(err)
			}
			a.Log("Generated .pocket/main.go and shim")
		},
	})
}
