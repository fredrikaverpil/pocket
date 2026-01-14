// Package govulncheck provides govulncheck integration.
package govulncheck

import (
	"context"

	"github.com/fredrikaverpil/pocket"
)

// Name is the binary name for govulncheck.
const Name = "govulncheck"

// renovate: datasource=go depName=golang.org/x/vuln
const Version = "v1.1.4"

// Install ensures govulncheck is available.
var Install = pocket.Func("install:govulncheck", "install govulncheck", install).Hidden()

func install(ctx context.Context) error {
	return pocket.InstallGo(ctx, "golang.org/x/vuln/cmd/govulncheck", Version)
}
