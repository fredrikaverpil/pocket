# pocket

A cross-platform build system inspired by [Mage](https://magefile.org/) and
[Sage](https://github.com/einride/sage). Define functions, compose them with
`Serial`/`Parallel`, and let pocket handle tool installation.

> [!NOTE]
>
> You don't need Go installed to use Pocket. The `./pok` shim automatically
> downloads Go to `.pocket/` if needed.

> [!WARNING]
>
> Under heavy development. Breaking changes will occur until the initial
> release.

## Features

- **Cross-platform**: Works on Windows, macOS, and Linux (no Makefiles)
- **Function-based**: Define functions with `pocket.Func()`, compose with
  `Serial()`/`Parallel()`
- **Dependency management**: Functions can depend on other functions with
  automatic deduplication
- **Tool management**: Downloads and caches tools in `.pocket/`
- **Path filtering**: Run different functions in different directories

## Quickstart

### Bootstrap

Run in your project root (requires Go for this step only):

```bash
go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
```

This creates `.pocket/` and `./pok` (the wrapper script).

### Your first function

Edit `.pocket/config.go`:

```go
package main

import (
    "context"
    "github.com/fredrikaverpil/pocket"
)

var Config = pocket.Config{
    ManualRun: []pocket.Runnable{Hello},
}

var Hello = pocket.Func("hello", "say hello", hello)

func hello(ctx context.Context) error {
    if pocket.Verbose(ctx) {
        pocket.Println(ctx, "Running hello function...")
    }
    pocket.Println(ctx, "Hello from pocket!")
    return nil
}
```

```bash
./pok -h        # list functions
./pok hello     # run function
./pok hello -h  # show help for function (options, usage)
./pok -v hello  # run with verbose output
```

### Composition

Create multiple functions and compose them in `AutoRun` with `Serial()` and
`Parallel()` for controlled execution order:

```go
var Config = pocket.Config{
    AutoRun: pocket.Serial(
        Format,              // first
        pocket.Parallel(     // then these in parallel
            Lint,
            Test,
        ),
        Build,               // last
    ),
}
```

Running `./pok` without arguments executes the entire `AutoRun` tree.

### Dependencies

Functions can depend on other functions. Dependencies are deduplicated
automatically - each function runs at most once per execution.

```go
var Install = pocket.Func("install:tool", "install tool", install).Hidden()
var Lint = pocket.Func("lint", "run linter", lint)

func lint(ctx context.Context) error {
    // Ensure tool is installed (runs once, even if called multiple times)
    pocket.Serial(ctx, Install)
    return pocket.Exec(ctx, "tool", "lint", "./...")
}
```

## Concepts

### Functions

Everything in Pocket is a function created with `pocket.Func()`:

```go
var MyFunc = pocket.Func("name", "description", implementation)

func implementation(ctx context.Context) error {
    // do work
    return nil
}
```

Functions can be:

- **Visible**: Shown in `./pok -h` and callable from CLI
- **Hidden**: Not shown in help, used as dependencies (`.Hidden()`)

### Executing Commands

Use `pocket.Exec()` to run system commands in your `pocket.Func` functions:

```go
func format(ctx context.Context) error {
    return pocket.Exec(ctx, "go", "fmt", "./...")
}
```

Commands run with proper output handling and respect the current path context.

### Serial and Parallel

These have two modes based on the first argument:

**Composition mode** (no context) - returns a Runnable. Used in your
`.pocket/config.go`:

```go
pocket.Serial(fn1, fn2, fn3)    // run in sequence
pocket.Parallel(fn1, fn2, fn3)  // run concurrently
```

**Execution mode** (with context) - runs immediately, used in tools and tasks:

```go
pocket.Serial(ctx, fn1, fn2)    // run dependencies in sequence
pocket.Parallel(ctx, fn1, fn2)  // run dependencies concurrently
```

### Tools vs Tasks

Pocket conceptually distinguishes between **tools** (installers) and **tasks**
(runners). Tools are responsible for downloading and installing binaries; tasks
use those binaries to do work.

#### 1. Tool Package

A tool package ensures a binary is available. It exports:

- `Name` - the binary name (used with `pocket.Exec`)
- `Install` - a hidden function that downloads/installs the binary
- `Config` (optional) - configuration file lookup settings

```go
// tools/ruff/ruff.go
package ruff

const Name = "ruff"
const Version = "0.14.0"

var Install = pocket.Func("install:ruff", "install ruff", install).Hidden()

var Config = pocket.ToolConfig{
    UserFiles:   []string{"ruff.toml", ".ruff.toml", "pyproject.toml"},
    DefaultFile: "ruff.toml",
    DefaultData: defaultConfig,
}

func install(ctx context.Context) error {
    // Download and install ruff to .pocket/bin/
    // ...
}
```

#### 2. Task Package

A task package provides related functions that use tools:

```go
// tasks/python/lint.go
package python

var Lint = pocket.Func("py-lint", "lint Python files", lint)

func lint(ctx context.Context) error {
    pocket.Serial(ctx, ruff.Install)  // ensure tool is installed
    return pocket.Exec(ctx, ruff.Name, "check", ".")  // run via Name constant
}
```

The `Workflow()` function composes tasks, and `Detect()` enables auto-discovery:

```go
// tasks/python/workflow.go
package python

func Workflow() pocket.Runnable {
    return pocket.Serial(Format, Lint)
}

func Detect() func() []string {
    return func() []string { return pocket.DetectByFile("pyproject.toml") }
}
```

> [!NOTE]
>
> Be careful when using `pocket.Parallel()`. Only parallelize functions that
> don't conflict - typically read-only operations like linting or testing.
> Functions that mutate files (formatters, code generators) should run in serial
> before other functions read those files.

#### Summary

| Type     | Purpose         | Exports                              | Example            |
| -------- | --------------- | ------------------------------------ | ------------------ |
| **Tool** | Installs binary | `Name`, `Install`, optional `Config` | ruff, golangcilint |
| **Task** | Uses tools      | FuncDefs + `Workflow()` + `Detect()` | python, golang     |

### Config Usage

The config ties everything together:

```go
var Config = pocket.Config{
    // AutoRun executes on ./pok (no arguments)
    AutoRun: pocket.Serial(
        // Use task collections with auto-detection
        pocket.Paths(golang.Workflow()).DetectBy(golang.Detect()),
        pocket.Paths(markdown.Workflow()).DetectBy(markdown.Detect()),
    ),

    // ManualRun requires explicit ./pok <name>
    ManualRun: []pocket.Runnable{
        Deploy,
        Release,
    },
}

var Deploy = pocket.Func("deploy", "deploy to production", deploy)
var Release = pocket.Func("release", "create a release", release)
```

Running `./pok` executes AutoRun. Running `./pok deploy` executes the Deploy
function.

## Path Filtering

For e.g. monorepos, use `Paths()` to control where functions run:

```go
// Run in specific directories
pocket.Paths(myFunc).In("services/api", "services/web")

// Auto-detect directories containing go.mod
pocket.Paths(golang.Workflow()).DetectBy(golang.Detect())

// Exclude directories
pocket.Paths(golang.Workflow()).DetectBy(golang.Detect()).Except("vendor")

// Skip specific functions in specific paths
pocket.Paths(golang.Workflow()).DetectBy(golang.Detect()).Skip(golang.Test, "docs")
```

## Options

Functions can accept options:

```go
type DeployOptions struct {
    Env    string `arg:"env" usage:"target environment"`
    DryRun bool   `arg:"dry-run" usage:"print without executing"`
}

var Deploy = pocket.Func("deploy", "deploy to environment", deploy).
    With(DeployOptions{Env: "staging"})

func deploy(ctx context.Context) error {
    opts := pocket.Options[DeployOptions](ctx)
    if opts.DryRun {
        pocket.Printf(ctx, "Would deploy to %s\n", opts.Env)
        return nil
    }
    // deploy...
    return nil
}
```

```bash
./pok deploy                     # uses default (staging)
./pok deploy -env=prod -dry-run  # override at runtime
```

## Reference

### Helper Functions

```go
// Execution
pocket.Exec(ctx, "command", "arg1", "arg2")  // run command
pocket.Printf(ctx, "format %s", arg)          // formatted output
pocket.Println(ctx, "message")                // line output

// Paths
pocket.GitRoot()              // git repository root
pocket.FromGitRoot("subdir")  // path relative to git root
pocket.FromPocketDir("file")  // path relative to .pocket/
pocket.FromBinDir("tool")     // path relative to .pocket/bin/

// Context
pocket.Options[T](ctx)        // get typed options
pocket.Path(ctx)              // current path (for path-filtered functions)

// Detection
pocket.DetectByFile("go.mod")       // find dirs with file
pocket.DetectByExtension(".lua")    // find dirs with extension

// Installation
pocket.InstallGo(ctx, "pkg/path", "version")  // go install
pocket.ConfigPath("tool", config)              // find/create config file
```

### Config Structure

```go
var Config = pocket.Config{
    // AutoRun: runs on ./pok (no arguments)
    AutoRun: pocket.Serial(...),

    // ManualRun: requires ./pok <name>
    ManualRun: []pocket.Runnable{...},

    // Shim: configure wrapper scripts
    Shim: &pocket.ShimConfig{
        Name:       "pok",   // base name
        Posix:      true,    // ./pok
        Windows:    true,    // pok.cmd
        PowerShell: true,    // pok.ps1
    },
}
```

## Acknowledgements

- [einride/sage](https://github.com/einride/sage) - Inspiration for the
  function-based architecture and dependency pattern
- [magefile/mage](https://github.com/magefile/mage) - Inspiration for the
  Go-based build system approach
