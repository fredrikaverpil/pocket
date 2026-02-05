// Package govulncheck provides govulncheck integration.
package govulncheck

import (
	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golang"
)

// Name is the binary name for govulncheck.
const Name = "govulncheck"

// renovate: datasource=go depName=golang.org/x/vuln
const Version = "v1.1.4"

// Install ensures govulncheck is available.
var Install = pk.NewTask(pk.TaskConfig{
	Name:   "install:govulncheck",
	Usage:  "install govulncheck",
	Body:   golang.Install("golang.org/x/vuln/cmd/govulncheck", Version),
	Hidden: true,
	Global: true,
})
