// Package gotestsum provides gotestsum CLI tool integration.
package gotestsum

import (
	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golang"
)

// Name is the binary name for gotestsum.
const Name = "gotestsum"

// Version is the version of gotestsum to install.
// renovate: datasource=github-releases depName=gotestyourself/gotestsum
const Version = "v1.13.0"

// Install ensures gotestsum CLI is available.
var Install = pk.NewTask("install:gotestsum", "install gotestsum CLI", nil,
	golang.Install("gotest.tools/gotestsum", Version),
).Hidden().Global()
