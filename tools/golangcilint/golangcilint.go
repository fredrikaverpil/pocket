// Package golangcilint provides golangci-lint integration.
package golangcilint

import "github.com/fredrikaverpil/pocket"

// Name is the binary name for golangci-lint.
const Name = "golangci-lint"

// renovate: datasource=go depName=github.com/golangci/golangci-lint/v2
const Version = "v2.0.2"

// Install ensures golangci-lint is available.
var Install = pocket.Task("install:golangci-lint", "install golangci-lint",
	pocket.InstallGo("github.com/golangci/golangci-lint/v2/cmd/golangci-lint", Version),
).Hidden()

// Config for golangci-lint configuration file lookup.
var Config = pocket.ToolConfig{
	UserFiles: []string{
		".golangci.yml",
		".golangci.yaml",
		".golangci.toml",
		".golangci.json",
	},
	DefaultFile: "", // No default - use golangci-lint defaults
}
