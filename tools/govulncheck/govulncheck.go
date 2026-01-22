// Package govulncheck provides govulncheck integration.
package govulncheck

import "github.com/fredrikaverpil/pocket/pk"

// Name is the binary name for govulncheck.
const Name = "govulncheck"

// renovate: datasource=go depName=golang.org/x/vuln
const Version = "v1.1.4"

// Install ensures govulncheck is available.
var Install = pk.NewTask("install:govulncheck", "install govulncheck", nil,
	pk.InstallGo("golang.org/x/vuln/cmd/govulncheck", Version),
).Hidden()
