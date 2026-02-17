// Package golangcilint provides the golangci-lint tool for Go linting.
package golangcilint

import (
	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golang"
)

// Name is the binary name for golangci-lint.
const Name = "golangci-lint"

// Version is the version of golangci-lint to install.
// renovate: datasource=go depName=github.com/golangci/golangci-lint/v2
const Version = "v2.10.1"

// Install is a hidden, global task that installs golangci-lint.
// Global ensures it only runs once regardless of path context.
var Install = &pk.Task{
	Name:   "install:golangci-lint",
	Usage:  "install golangci-lint",
	Body:   golang.Install("github.com/golangci/golangci-lint/v2/cmd/golangci-lint", Version),
	Hidden: true,
	Global: true,
}
