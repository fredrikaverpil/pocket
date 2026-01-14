// Package tasks provides the entry point for running pocket tasks.
// Import this package in .pocket/main.go to use pocket.
package tasks

import (
	"github.com/fredrikaverpil/pocket"
	// Import scaffold to trigger init() which registers GenerateAll.
	_ "github.com/fredrikaverpil/pocket/internal/scaffold"
)

// Run is the main entry point for running pocket tasks.
// It parses CLI flags, discovers functions from Config, and runs them.
//
// Example usage in .pocket/main.go:
//
//	package main
//
//	import "github.com/fredrikaverpil/pocket/tasks"
//
//	func main() {
//	    tasks.Run(Config)
//	}
func Run(cfg pocket.Config) {
	pocket.RunConfig(cfg)
}
