// Package skillvalidator provides the skill-validator tool for validating Agent Skill packages.
package skillvalidator

import (
	"github.com/fredrikaverpil/pocket/pk"
	"github.com/fredrikaverpil/pocket/tools/golang"
)

// Name is the binary name for skill-validator.
const Name = "skill-validator"

// Version is the version of skill-validator to install.
// renovate: datasource=go depName=github.com/dacharyc/skill-validator
const Version = "v0.8.0"

// Install is a hidden, global task that installs skill-validator.
// Global ensures it only runs once regardless of path context.
var Install = &pk.Task{
	Name:   "install:skill-validator",
	Usage:  "install skill-validator",
	Body:   golang.Install("github.com/dacharyc/skill-validator", Version),
	Hidden: true,
	Global: true,
}
