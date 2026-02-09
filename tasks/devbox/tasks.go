// Package devbox provides task bundles for devbox-managed projects.
//
// Devbox creates portable, isolated dev environments using Nix. Pocket
// downloads the devbox binary automatically, and devbox handles Nix
// installation and package management.
//
// # Usage Modes
//
// ## Run devbox scripts (defined in devbox.json)
//
//	devbox.Run("test")     // runs "devbox run test"
//	devbox.Run("lint")     // runs "devbox run lint"
//
// ## Run arbitrary commands in the devbox environment
//
//	// In a task's Do function:
//	devboxtool.Exec(ctx, "python", "--version")
//
// ## Detect and compose
//
//	pk.WithOptions(
//	    devbox.Tasks("test", "lint"),
//	    pk.WithDetect(devbox.Detect()),
//	)
//
// # Where packages live
//
// The devbox binary itself is installed into .pocket/tools/devbox/{version}/bin/.
// Packages defined in devbox.json are managed by devbox and live in /nix/store
// with symlinks in .devbox/nix/profile/default/bin/. Pocket does not manage
// the devbox packages — only the devbox binary.
package devbox

import (
	"context"

	"github.com/fredrikaverpil/pocket/pk"
	devboxtool "github.com/fredrikaverpil/pocket/tools/devbox"
)

// FlagScript is the flag name for specifying the devbox script to run.
const FlagScript = "script"

// Packages installs devbox packages from devbox.json.
// This runs "devbox install" which downloads Nix packages without
// entering an interactive shell.
var Packages = &pk.Task{
	Name:  "devbox-install",
	Usage: "install devbox packages from devbox.json",
	Body:  pk.Serial(devboxtool.Install, packagesCmd()),
}

func packagesCmd() pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return pk.Exec(ctx, devboxtool.Name, "install")
	})
}

// Run returns a task that runs a named script from devbox.json.
//
// Scripts are defined in devbox.json under shell.scripts:
//
//	{
//	  "shell": {
//	    "scripts": {
//	      "test": "pytest",
//	      "lint": "ruff check ."
//	    }
//	  }
//	}
//
// Usage:
//
//	devbox.Run("test")  // creates task "devbox-run:test"
//	devbox.Run("lint")  // creates task "devbox-run:lint"
func Run(script string) *pk.Task {
	return &pk.Task{
		Name:  "devbox-run:" + script,
		Usage: "run devbox script: " + script,
		Body:  pk.Serial(devboxtool.Install, runScriptCmd(script)),
	}
}

func runScriptCmd(script string) pk.Runnable {
	return pk.Do(func(ctx context.Context) error {
		return pk.Exec(ctx, devboxtool.Name, "run", script)
	})
}

// Detect returns a DetectFunc that finds devbox projects
// (directories containing devbox.json).
func Detect() pk.DetectFunc {
	return pk.DetectByFile("devbox.json")
}

// Tasks returns devbox tasks composed together.
// Pass script names to run, or omit to just install packages.
//
// With scripts:
//
//	devbox.Tasks("test", "lint")
//	// → Serial(Install, Packages, Parallel(Run("test"), Run("lint")))
//
// Without scripts:
//
//	devbox.Tasks()
//	// → Serial(Install, Packages)
func Tasks(scripts ...string) pk.Runnable {
	if len(scripts) == 0 {
		return pk.Serial(devboxtool.Install, Packages)
	}

	runners := make([]pk.Runnable, len(scripts))
	for i, s := range scripts {
		runners[i] = Run(s)
	}

	return pk.Serial(
		Packages,
		pk.Parallel(runners...),
	)
}
