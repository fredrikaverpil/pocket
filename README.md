# pocket

A cross-platform build system inspired by
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

- **Zero-install**: The `./pok` shim bootstraps Go and all dependencies
  automatically
- **Cross-platform**: Works on Windows, macOS, and Linux (no Makefiles)
- **Composable**: Define functions, compose with `Serial()`/`Parallel()` -
  Pocket handles the execution graph
- **Monorepo-ready**: Auto-detects directories (by go.mod, pyproject.toml, etc.)
  with per-directory task visibility
- **Tool management**: Downloads and caches tools in `.pocket/`

## Quickstart

### Bootstrap

Run in your project root (requires Go for this step only):

```bash
go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
```

This creates `.pocket/` and `./pok` (the wrapper script).

### Your first task

Edit `.pocket/config.go` and add a task to your config's `ManualRun`:

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
./pok -h        # list tasks
./pok hello     # run task
./pok hello -h  # show help for task (options, usage)
./pok -v hello  # run with verbose output
```

### Composition

This is where Pocket shines. Compose functions in `AutoRun` with `Serial()` and
`Parallel()` for controlled execution order. Combined with path filtering,
Pocket generates "pok" shims in each detected directory with fine-grained
control over which tasks are available at each location.

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

```bash
./pok       # run entire AutoRun tree
./pok plan  # show execution tree (useful for debugging composition)
```

### Dependencies

Functions can depend on other functions. Dependencies are deduplicated
automatically - each function runs at most once per execution.

```go
var Install = pocket.Func("install:tool", "install tool", install).Hidden()
var Lint = pocket.Func("lint", "run linter", lint)

func lint(ctx context.Context) error {
    // Ensure tool is installed (runs once, even if called multiple times)
    pocket.Serial(Install)
    return pocket.Exec(ctx, "tool", "lint", "./...")
}
```

## Concepts

### Functions

Everything in Pocket is a function created with `pocket.Func()`. Functions are
the building block - they become **tasks** when exposed via CLI, or **tools**
when they install binaries:

```go
var MyFunc = pocket.Func("name", "description", implementation)

func implementation(ctx context.Context) error {
    // do work
    return nil
}
```

Functions can be:

- **Visible**: Shown in `./pok -h` as tasks, callable from CLI
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

Use `Serial` and `Parallel` to compose functions:

```go
pocket.Serial(fn1, fn2, fn3)    // run in sequence
pocket.Parallel(fn1, fn2, fn3)  // run concurrently
```

**With dependencies** - compose install dependencies into your function:

```go
var Lint = pocket.Func("lint", "run linter", pocket.Serial(
    linter.Install,  // runs first
    lint,            // then the actual linting
))

func lint(ctx context.Context) error {
    return pocket.Exec(ctx, linter.Name, "run", "./...")
}
```

> [!NOTE]
>
> Be careful when using `pocket.Parallel()`. Only parallelize functions that
> don't conflict - typically read-only operations like linting or testing.
> Functions that mutate files (formatters, code generators) should run in serial
> before other functions read those files.

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

var Lint = pocket.Func("py-lint", "lint Python files", pocket.Serial(
    ruff.Install,  // ensure tool is installed first
    lint,
))

func lint(ctx context.Context) error {
    return pocket.Exec(ctx, ruff.Name, "check", ".")  // run via Name constant
}
```

The `Tasks()` function composes tasks, and `Detect()` enables auto-discovery:

```go
// tasks/python/workflow.go
package python

func Tasks() pocket.Runnable {
    return pocket.Serial(Format, Lint)
}

func Detect() func() []string {
    return func() []string { return pocket.DetectByFile("pyproject.toml") }
}
```

#### Summary

| Type     | Purpose         | Exports                              | Example            |
| -------- | --------------- | ------------------------------------ | ------------------ |
| **Tool** | Installs binary | `Name`, `Install`, optional `Config` | ruff, golangcilint |
| **Task** | Uses tools      | FuncDefs + `Tasks()` + `Detect()`    | python, golang     |

#### 3. Tool Configuration

Tools that use config files can export a `ToolConfig`. Tasks then use
`pocket.ConfigPath()` to find an existing config or create a default one:

```go
// In tool package: define where to look for config
var Config = pocket.ToolConfig{
    UserFiles: []string{
        "ruff.toml",                      // relative: check in task CWD
        pocket.FromGitRoot("ruff.toml"),  // absolute: check in repo root
    },
    DefaultFile: "ruff.toml",    // fallback filename
    DefaultData: defaultConfig,  // embedded default config
}

// In task: find or create config
func lint(ctx context.Context) error {
    configPath, _ := pocket.ConfigPath(ctx, "ruff", ruff.Config)
    return pocket.Exec(ctx, ruff.Name, "check", "--config", configPath, ".")
}
```

`ConfigPath` checks each path in `UserFiles`:

- **Relative paths** are resolved from the task's current directory
  (`Path(ctx)`)
- **Absolute paths** are used as-is (use `FromGitRoot()` for repo-root configs)

If no user config is found, it writes `DefaultData` to
`.pocket/tools/<tool>/<DefaultFile>` and returns that path. This lets each
directory in a monorepo have its own config, while providing sensible defaults.

### Config Usage

The config ties everything together:

```go
var Config = pocket.Config{
    // AutoRun executes on ./pok (no arguments)
    AutoRun: pocket.Serial(
        // Use task collections with auto-detection
        pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()),
        pocket.Paths(markdown.Tasks()).DetectBy(markdown.Detect()),
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
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect())

// Exclude directories
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()).Except("vendor")

// Combine detection with filtering
pocket.Paths(golang.Tasks()).DetectBy(golang.Detect()).In("services/.*").Except("testdata")
```

### Skipping Tasks in Specific Paths

While `Except()` excludes entire task compositions from directories, use
`SkipTask()` to skip specific tasks within a composition. This controls which
tasks run in each detected directory - it doesn't modify the task's arguments.

For example, in a monorepo with multiple Go services (each with their own
`go.mod`), you might want to skip slow tests in certain services:

```go
var Config = pocket.Config{
    AutoRun: pocket.Paths(golang.Tasks()).
        DetectBy(golang.Detect()).
        SkipTask(golang.Test, "services/api", "services/worker"),

    // Make skipped tests available under a different name
    ManualRun: []pocket.Runnable{
        pocket.Paths(golang.Test.WithName("integration-test")).In("services/api", "services/worker"),
    },
}
```

Here `services/api/` and `services/worker/` are separate Go modules detected by
`golang.Detect()`. All composed tasks (format, lint, vulncheck) run in all
detected modules, but `go-test` is skipped in those two. The skipped tests are
available as `integration-test` when run from those directories.

Note: `WithName()` creates a copy of the task with a different CLI name. This
avoids duplicate names when the same task appears in both AutoRun and ManualRun.

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
// Composition
pocket.Serial(fn1, fn2, fn3)     // run in sequence
pocket.Parallel(fn1, fn2, fn3)   // run concurrently

// Execution
pocket.Exec(ctx, "cmd", "arg1", "arg2")       // run command in current path
pocket.ExecIn(ctx, "dir", "cmd", "args"...)   // run command in specific dir
pocket.Command(ctx, "cmd", "args"...)         // create exec.Cmd with .pocket/bin in PATH
pocket.Printf(ctx, "format %s", arg)          // formatted output to stdout
pocket.Println(ctx, "message")                // line output to stdout

// Context
pocket.Options[T](ctx)        // get typed options from context
pocket.Path(ctx)              // current path (for path-filtered functions)
pocket.Verbose(ctx)           // whether -v flag is set
pocket.CWD(ctx)               // where CLI was invoked (relative to git root)

// Paths
pocket.GitRoot()              // git repository root
pocket.FromGitRoot("subdir")  // path relative to git root
pocket.FromPocketDir("file")  // path relative to .pocket/
pocket.FromToolsDir("tool")   // path relative to .pocket/tools/
pocket.FromBinDir("tool")     // path relative to .pocket/bin/
pocket.BinaryName("tool")     // append .exe on Windows

// Detection
pocket.DetectByFile("go.mod")       // find dirs containing file
pocket.DetectByExtension(".lua")    // find dirs with file extension

// Installation
pocket.InstallGo(ctx, "github.com/org/tool", "v1.0.0")  // go install
pocket.CreateSymlink("path/to/binary")                  // symlink to .pocket/bin/
pocket.ConfigPath(ctx, "tool", config)                   // find/create config file

// Download & Extract
pocket.Download(ctx, url,
    pocket.WithDestDir(dir),                              // extraction destination
    pocket.WithFormat("tar.gz"),                          // format: tar.gz, tar, zip, ""
    pocket.WithExtract(pocket.WithExtractFile(name)),     // extract specific file
    pocket.WithExtract(pocket.WithRenameFile(src, dest)), // rename during extract
    pocket.WithExtract(pocket.WithFlatten()),             // flatten directory structure
    pocket.WithSymlink(),                                 // symlink to .pocket/bin/
    pocket.WithSkipIfExists(path),                        // skip if file exists
    pocket.WithHTTPHeader(key, value),                    // add HTTP header
)
pocket.FromLocal(ctx, path, opts...)  // process local file with same options

// Platform
pocket.HostOS()                     // runtime.GOOS ("darwin", "linux", "windows")
pocket.HostArch()                   // runtime.GOARCH ("amd64", "arm64")
pocket.DefaultArchiveFormat()       // "zip" on Windows, "tar.gz" otherwise
pocket.DefaultArchiveFormatFor(os)  // "zip" for Windows, "tar.gz" otherwise
pocket.ArchToX8664(arch)            // convert "amd64" → "x86_64"
pocket.ArchToAMD64(arch)            // convert "x86_64" → "amd64"
pocket.ArchToX64(arch)              // convert "amd64" → "x64"
pocket.OSToTitle(os)                // convert "darwin" → "Darwin"
pocket.OSToUpper(os)                // convert "darwin" → "DARWIN"

// Platform constants
pocket.Darwin, pocket.Linux, pocket.Windows          // OS names
pocket.AMD64, pocket.ARM64                           // Go-style arch
pocket.X8664, pocket.AARCH64, pocket.X64             // alternative arch names

// Module
pocket.GoVersionFromDir("dir")    // read Go version from go.mod
```

### Config Structure

```go
var Config = pocket.Config{
    // AutoRun: runs on ./pok (no arguments)
    AutoRun: pocket.Paths(golang.Tasks()).
        DetectBy(golang.Detect()).
        SkipTask(golang.Test, "services/worker"),

    // ManualRun: requires ./pok <name>
    ManualRun: []pocket.Runnable{
        pocket.Paths(golang.Test.WithName("slow-test")).In("services/worker"),
    },

    // Shim: configure wrapper scripts
    Shim: &pocket.ShimConfig{
        Name:       "pok",   // base name
        Posix:      true,    // ./pok
        Windows:    true,    // pok.cmd
        PowerShell: true,    // pok.ps1
    },

    // SkipGenerate: don't run "generate" before tasks (default: false)
    SkipGenerate: false,

    // SkipGitDiff: don't fail on uncommitted changes after tasks (default: false)
    SkipGitDiff: false,
}
```

## Documentation

- [Architecture](architecture.md) - Internal design: execution model, shim
  generation, path resolution
