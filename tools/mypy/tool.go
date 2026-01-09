// Package mypy provides mypy (Python static type checker) tool integration.
// mypy is installed via uv into a virtual environment.
package mypy

import (
	"github.com/fredrikaverpil/pocket/tool"
	"github.com/fredrikaverpil/pocket/tools/uv"
)

const name = "mypy"

// renovate: datasource=pypi depName=mypy
const version = "1.19.1"

// pythonVersion specifies the Python version for the virtual environment.
const pythonVersion = "3.12"

// Prepare ensures mypy is installed.
var Prepare = tool.PythonToolPreparer(name, version, pythonVersion, uv.CreateVenv, uv.PipInstall)

var t = &tool.Tool{Name: name, Prepare: Prepare}

// Command prepares the tool and returns an exec.Cmd for running mypy.
var Command = t.Command

// Run installs (if needed) and executes mypy.
var Run = t.Run
