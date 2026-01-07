// Package golang provides Go-related build tasks.
package golang

import (
	"github.com/fredrikaverpil/bld"
	"github.com/fredrikaverpil/bld/tools/golangcilint"
	"github.com/fredrikaverpil/bld/tools/govulncheck"
	"github.com/goyek/goyek/v3"
)

// Tasks holds the goyek tasks for Go operations.
// Create with NewTasks and register the tasks you need.
type Tasks struct {
	config bld.Config

	// All runs all Go tasks (lint, format, test, vulncheck).
	All *goyek.DefinedTask

	// Format formats Go code using go fmt.
	Format *goyek.DefinedTask

	// Test runs Go tests.
	Test *goyek.DefinedTask

	// Lint runs golangci-lint.
	Lint *goyek.DefinedTask

	// Vulncheck runs govulncheck.
	Vulncheck *goyek.DefinedTask
}

// NewTasks creates Go tasks for the given config.
func NewTasks(cfg bld.Config) *Tasks {
	cfg = cfg.WithDefaults()
	t := &Tasks{config: cfg}

	t.Format = goyek.Define(goyek.Task{
		Name:  "go-format",
		Usage: "format Go code (gofumpt, goimports, gci, golines)",
		Action: func(a *goyek.A) {
			modules := cfg.GoModulesForFormat()
			if len(modules) == 0 {
				a.Log("no modules configured for format")
				return
			}
			configPath, err := golangcilint.ConfigPath()
			if err != nil {
				a.Fatalf("get golangci-lint config: %v", err)
			}
			for _, mod := range modules {
				cmd, err := golangcilint.Command(a.Context(), "fmt", "-c", configPath, "./...")
				if err != nil {
					a.Fatalf("prepare golangci-lint: %v", err)
				}
				cmd.Dir = bld.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					a.Errorf("golangci-lint fmt failed in %s: %v", mod, err)
				}
			}
		},
	})

	t.Test = goyek.Define(goyek.Task{
		Name:  "go-test",
		Usage: "run Go tests",
		Action: func(a *goyek.A) {
			modules := cfg.GoModulesForTest()
			if len(modules) == 0 {
				a.Log("no modules configured for test")
				return
			}
			for _, mod := range modules {
				cmd := bld.Command(a.Context(), "go", "test", "-v", "-race", "./...")
				cmd.Dir = bld.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					a.Errorf("go test failed in %s: %v", mod, err)
				}
			}
		},
	})

	t.Lint = goyek.Define(goyek.Task{
		Name:  "go-lint",
		Usage: "run golangci-lint",
		Action: func(a *goyek.A) {
			modules := cfg.GoModulesForLint()
			if len(modules) == 0 {
				a.Log("no modules configured for lint")
				return
			}
			configPath, err := golangcilint.ConfigPath()
			if err != nil {
				a.Fatalf("get golangci-lint config: %v", err)
			}
			for _, mod := range modules {
				cmd, err := golangcilint.Command(
					a.Context(),
					"run",
					"--allow-parallel-runners",
					"-c",
					configPath,
					"./...",
				)
				if err != nil {
					a.Fatalf("prepare golangci-lint: %v", err)
				}
				cmd.Dir = bld.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					a.Errorf("golangci-lint failed in %s: %v", mod, err)
				}
			}
		},
	})

	t.Vulncheck = goyek.Define(goyek.Task{
		Name:  "go-vulncheck",
		Usage: "run govulncheck",
		Action: func(a *goyek.A) {
			modules := cfg.GoModulesForVulncheck()
			if len(modules) == 0 {
				a.Log("no modules configured for vulncheck")
				return
			}
			for _, mod := range modules {
				cmd, err := govulncheck.Command(a.Context(), "./...")
				if err != nil {
					a.Fatalf("prepare govulncheck: %v", err)
				}
				cmd.Dir = bld.FromGitRoot(mod)
				if err := cmd.Run(); err != nil {
					a.Errorf("govulncheck failed in %s: %v", mod, err)
				}
			}
		},
	})

	t.All = goyek.Define(goyek.Task{
		Name:  "go-all",
		Usage: "run all Go tasks (format, lint, test, vulncheck)",
		Deps:  goyek.Deps{t.Format, t.Lint, t.Test, t.Vulncheck},
	})

	return t
}
