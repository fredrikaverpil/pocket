// Package golangcilint provides golangci-lint integration.
package golangcilint

import (
	"context"

	"github.com/fredrikaverpil/pocket"
)

// Name is the binary name for golangci-lint.
const Name = "golangci-lint"

// renovate: datasource=go depName=github.com/golangci/golangci-lint/v2
const Version = "v2.0.2"

// Install ensures golangci-lint is available.
var Install = pocket.Func("install:golangci-lint", "install golangci-lint", install).Hidden()

func install(ctx context.Context) error {
	pocket.Printf(ctx, "Installing golangci-lint %s...\n", Version)
	return pocket.InstallGo(ctx, "github.com/golangci/golangci-lint/v2/cmd/golangci-lint", Version)
}

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
