// Package basedpyright provides basedpyright (Python static type checker) tool integration.
// basedpyright is installed via uv into a virtual environment.
package basedpyright

import (
	"github.com/fredrikaverpil/pocket/tool"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

const name = "basedpyright"

// renovate: datasource=pypi depName=basedpyright
const version = "1.37.0"

// pythonVersion specifies the Python version for the virtual environment.
const pythonVersion = "3.12"

// Prepare ensures basedpyright is installed.
var Prepare = tool.PythonToolPreparer(name, version, pythonVersion, uv.CreateVenv, uv.PipInstall)

var t = &tool.Tool{Name: name, Prepare: Prepare}

// Command prepares the tool and returns an exec.Cmd for running basedpyright.
var Command = t.Command

// Run installs (if needed) and executes basedpyright.
var Run = t.Run
