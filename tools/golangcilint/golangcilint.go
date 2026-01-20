// Package golangcilint provides the golangci-lint tool for Go linting.
package golangcilint

import "github.com/fredrikaverpil/pocket/pk"

// Name is the binary name for golangci-lint.
const Name = "golangci-lint"

// Version is the version of golangci-lint to install.
// renovate: datasource=go depName=github.com/golangci/golangci-lint/v2
const Version = "v2.1.6"

// Install is a hidden task that installs golangci-lint.
var Install = pk.DefineTask("install:golangci-lint", "install golangci-lint",
	pk.InstallGo("github.com/golangci/golangci-lint/v2/cmd/golangci-lint", Version),
).Hidden()
