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
    pocket.Println(ctx, "Hello from pocket!")
    return nil
}
```

```bash
./pok -h      # list functions
./pok hello   # run function
```

### Composition

Use `Serial()` and `Parallel()` to control execution order:

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

Pocket conceptually distinguishes between **tools** (provide capabilities) and
**tasks** (do work). Here's how they build on each other:

#### 1. Runtime Tool

A runtime tool ensures a dependency (often a binary) is available. It only
exports a hidden `Install` function:

```go
// tools/bun/bun.go
package bun

var Install = pocket.Func("install:bun", "ensure bun available", install).Hidden()

func install(ctx context.Context) error {
    // Install bun (skipped for this example)

    // Check if bun is in PATH
    if _, err := exec.LookPath("bun"); err != nil {
        return fmt.Errorf("bun not found - install from https://bun.sh")
    }
    return nil
}
```

#### 2. Action Tool

An action tool does something useful. It depends on a runtime tool and exports
`Install` (hidden), an action function (visible), and an `Exec()` helper:

```go
// tools/prettier/prettier.go
package prettier

const Version = "3.4.2"

// Hidden - ensures prettier is available
var Install = pocket.Func("install:prettier", "install prettier", install).Hidden()

func install(ctx context.Context) error {
    pocket.Serial(ctx, bun.Install)  // Depend on bun
    return nil
}

// Visible - can be used directly in config
var Format = pocket.Func("prettier", "format with prettier", format)

func format(ctx context.Context) error {
    return Exec(ctx, "--write", ".")
}

// Exec runs prettier with any arguments - for programmatic use
func Exec(ctx context.Context, args ...string) error {
    pocket.Serial(ctx, Install)
    return pocket.Exec(ctx, "bunx", "prettier@"+Version, args...)
}
```

#### 3. Task Package

A task package provides related functions. Individual functions defined with
`pocket.Func()` are **tasks**. The `Workflow()` function composes these tasks
into a **workflow** using `Serial`/`Parallel`, and `Detect()` enables
auto-discovery:

```go
// tasks/golang/golang.go
package golang

var Format = pocket.Func("go-format", "format Go code", format)
var Lint = pocket.Func("go-lint", "lint Go code", lint)
var Test = pocket.Func("go-test", "run tests", test)
var Vulncheck = pocket.Func("go-vulncheck", "check vulnerabilities", vulncheck)

func format(ctx context.Context) error {
    return pocket.Exec(ctx, "go", "fmt", "./...")
}

func lint(ctx context.Context) error {
    return golangcilint.Exec(ctx, "run", "--fix", "./...")
}

func test(ctx context.Context) error {
    return pocket.Exec(ctx, "go", "test", "./...")
}

func vulncheck(ctx context.Context) error {
    return govulncheck.Exec(ctx, "./...")
}

// Workflow returns all Go tasks composed - no ctx means composition mode
func Workflow() pocket.Runnable {
    return pocket.Serial(
        Format,                           // mutates files
        Lint,                             // mutates files (--fix)
        pocket.Parallel(Test, Vulncheck), // read-only - safe to parallelize
    )
}

// Detect finds directories containing go.mod
func Detect() func() []string {
    return func() []string { return pocket.DetectByFile("go.mod") }
}
```

> [!NOTE]
>
> Be careful when using `pocket.Parallel()`. Only parallelize functions that
> don't conflict - typically read-only operations like linting or testing.
> Functions that mutate files (formatters, code generators) should run in serial
> before other functions read those files.

#### Customization

Pocket ships with pre-configured tools and tasks that are opinionated - they
include default configurations, version pinning, and sensible defaults. However,
Pocket is designed for customization:

- **Project-level**: Define your own tools and tasks in `.pocket/*.go`
- **Organization-level**: Create a "Pocket platform" (like this repo, but
  without the Pocket internals) - a Go module that uses Pocket's APIs to define
  your organization's standard tools, tasks, and configurations

A platform module depends on Pocket and exports its own tools and tasks:

```go
// github.com/myorg/platform/tasks/golang/golang.go
package golang

import "github.com/fredrikaverpil/pocket"

var Format = pocket.Func("go-format", "format Go code", format)
// ... your organization's opinionated Go tasks
```

Then projects import from your platform instead of Pocket's bundled tasks:

```go
import "github.com/myorg/platform/tasks/golang"

var Config = pocket.Config{
    AutoRun: pocket.Paths(golang.Workflow()).DetectBy(golang.Detect()),
}
```

#### Summary

| Type             | Purpose            | Exports                               | Example          |
| ---------------- | ------------------ | ------------------------------------- | ---------------- |
| **Runtime Tool** | Provides a runtime | `Install` (hidden)                    | bun, uv          |
| **Action Tool**  | Does something     | `Install` + action func + `Exec()`    | prettier, ruff   |
| **Task Package** | Orchestrates tools | Tasks + `Workflow()` + `Detect()` | markdown, golang |

### Config Usage

The config ties everything together:

```go
var Config = pocket.Config{
    // AutoRun executes on ./pok (no arguments)
    AutoRun: pocket.Serial(
        // Use task collections with auto-detection
        pocket.Paths(golang.Workflow()).DetectBy(golang.Detect()),
        pocket.Paths(markdown.Workflow()).DetectBy(markdown.Detect()),

        // Or use action tools directly
        pocket.Paths(prettier.Format).In("docs"),
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

For monorepos, use `Paths()` to control where functions run:

```go
// Run in specific directories
pocket.Paths(myFunc).In("services/api", "services/web")

// Auto-detect directories containing go.mod
pocket.Paths(golang.Workflow()).DetectBy(golang.Detect())

// Exclude directories
pocket.Paths(golang.Workflow()).DetectBy(golang.Detect()).Except("vendor")

// Skip specific functions in specific paths
pocket.Paths(golang.Workflow()).DetectBy(golang.Detect()).Skip(golang.GoTest, "docs")
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
