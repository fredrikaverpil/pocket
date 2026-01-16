// Package govulncheck provides govulncheck integration.
package govulncheck

import "github.com/fredrikaverpil/pocket"

// Name is the binary name for govulncheck.
const Name = "govulncheck"

// renovate: datasource=go depName=golang.org/x/vuln
const Version = "v1.1.4"

// Install ensures govulncheck is available.
var Install = pocket.Task("install:govulncheck", "install govulncheck",
	pocket.InstallGo("golang.org/x/vuln/cmd/govulncheck", Version),
	pocket.AsHidden(),
)
