# pocket

A cross-platform build system inspired by
[Sage](https://github.com/einride/sage). Define tasks, compose them with
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
- **Composable**: Define tasks, compose with `Serial()`/`Parallel()` - Pocket
  handles the execution graph
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

var Hello = pocket.Task("hello", "say hello", hello)

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

This is where Pocket shines. Compose tasks in `AutoRun` with `Serial()` and
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

Tasks can depend on other tasks. Dependencies are deduplicated automatically -
each task runs at most once per execution.

```go
var Install = pocket.Task("install:tool", "install tool",
    pocket.InstallGo("github.com/org/tool", "v1.0.0"),
    pocket.AsHidden(),
)

var Lint = pocket.Task("lint", "run linter", pocket.Serial(
    Install,  // runs first (deduplicated across the tree)
    pocket.Run("tool", "lint", "./..."),
))
```

## Concepts

### Tasks

The core abstraction is `Runnable` - anything that can be executed. Tasks are
named Runnables created with `pocket.Task()` that appear in the CLI:

```go
// Simple: static command
var Format = pocket.Task("format", "format code",
    pocket.Run("go", "fmt", "./..."),
)

// Composed: multiple steps
var Build = pocket.Task("build", "build the project", pocket.Serial(
    Install,
    pocket.Run("go", "build", "./..."),
))

// Dynamic: args computed at runtime
var Lint = pocket.Task("lint", "run linter", lintCmd())

func lintCmd() pocket.Runnable {
    return pocket.Do(func(ctx context.Context) error {
        args := []string{"run"}
        if pocket.Verbose(ctx) {
            args = append(args, "-v")
        }
        return pocket.Exec(ctx, "golangci-lint", args...)
    })
}
```

Tasks can be:

- **Visible**: Shown in `./pok -h`, callable from CLI
- **Hidden**: Not shown in help, used as dependencies (`pocket.AsHidden()`)

### Executing Commands

Pocket provides two ways to run external commands:

**`Run(name, args...)`** - Static command with fixed arguments:

```go
pocket.Run("go", "fmt", "./...")
```

**`Do(fn)`** - Dynamic commands or arbitrary Go code:

```go
pocket.Do(func(ctx context.Context) error {
    args := []string{"test"}
    if pocket.Verbose(ctx) {
        args = append(args, "-v")
    }
    return pocket.Exec(ctx, "go", args...)
})
```

Use `Do` for dynamic arguments, complex logic, file I/O, or multiple commands.
Both run with proper output handling and respect the current path context.
They're no-ops in collect mode (plan generation).

### Serial and Parallel

Use `Serial` and `Parallel` to compose Runnables:

```go
pocket.Serial(fn1, fn2, fn3)    // run in sequence
pocket.Parallel(fn1, fn2, fn3)  // run concurrently
```

**With dependencies** - compose install dependencies into your task:

```go
var Lint = pocket.Task("lint", "run linter", pocket.Serial(
    linter.Install,  // runs first (deduplicated)
    lintCmd(),       // then the actual linting
))

func lintCmd() pocket.Runnable {
    return pocket.Do(func(ctx context.Context) error {
        args := []string{"run"}
        if pocket.Verbose(ctx) {
            args = append(args, "-v")
        }
        return pocket.Exec(ctx, linter.Name, args...)
    })
}
```

> [!NOTE]
>
> Be careful when using `pocket.Parallel()`. Only parallelize tasks that don't
> conflict - typically read-only operations like linting or testing. Tasks that
> mutate files (formatters, code generators) should run in serial before other
> tasks read those files.

### Tools vs Tasks

Pocket conceptually distinguishes between **tools** (installers) and **tasks**
(runners). Tools are responsible for downloading and installing binaries; tasks
use those binaries to do work.

#### 1. Tool Package

A tool package ensures a binary is available. It exports:

- `Name` - the binary name (used with `pocket.Run` or `pocket.Exec`)
- `Install` - a hidden task that downloads/installs the binary
- `Config` (optional) - configuration file lookup settings

```go
// tools/golangcilint/golangcilint.go
package golangcilint

const Name = "golangci-lint"
const Version = "v2.0.2"

// For Go tools: use InstallGo directly
var Install = pocket.Task("install:golangci-lint", "install golangci-lint",
    pocket.InstallGo("github.com/golangci/golangci-lint/v2/cmd/golangci-lint", Version),
    pocket.AsHidden(),
)

var Config = pocket.ToolConfig{
    UserFiles:   []string{".golangci.yml", ".golangci.yaml", ".golangci.toml"},
    DefaultFile: ".golangci.yml",
    DefaultData: defaultConfig,
}
```

For tools with complex installation (downloads, extraction):

```go
// tools/stylua/stylua.go
package stylua

const Name = "stylua"
const Version = "v2.0.2"

var Install = pocket.Task("install:stylua", "install stylua",
    pocket.Download(downloadURL(),
        pocket.WithDestDir(destDir()),
        pocket.WithFormat(pocket.DefaultArchiveFormat()),
        pocket.WithExtract(pocket.WithExtractFile(pocket.BinaryName(Name))),
        pocket.WithSymlink(),
        pocket.WithSkipIfExists(binaryPath()),
    ),
    pocket.AsHidden(),
)
```

#### 2. Task Package

A task package provides related tasks that use tools:

```go
// tasks/golang/lint.go
package golang

var Lint = pocket.Task("go-lint", "run golangci-lint", pocket.Serial(
    golangcilint.Install,  // ensure tool is installed first
    lintCmd(),             // then run linting
), pocket.Opts(LintOptions{}))

func lintCmd() pocket.Runnable {
    return pocket.Do(func(ctx context.Context) error {
        opts := pocket.Options[LintOptions](ctx)
        args := []string{"run"}
        if pocket.Verbose(ctx) {
            args = append(args, "-v")
        }
        if !opts.SkipFix {
            args = append(args, "--fix")
        }
        args = append(args, "./...")
        return pocket.Exec(ctx, golangcilint.Name, args...)
    })
}
```

The `Tasks()` function composes tasks, and `Detect()` enables auto-discovery:

```go
// tasks/golang/workflow.go
package golang

func Tasks() pocket.Runnable {
    return pocket.Serial(Format, Lint, Test)
}

func Detect() func() []string {
    return func() []string { return pocket.DetectByFile("go.mod") }
}
```

#### Summary

| Type     | Purpose         | Exports                              | Example            |
| -------- | --------------- | ------------------------------------ | ------------------ |
| **Tool** | Installs binary | `Name`, `Install`, optional `Config` | ruff, golangcilint |
| **Task** | Uses tools      | TaskDefs + `Tasks()` + `Detect()`    | python, golang     |

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
        pocket.RunIn(golang.Tasks(), pocket.Detect(golang.Detect())),
        pocket.RunIn(markdown.Tasks(), pocket.Detect(markdown.Detect())),
    ),

    // ManualRun requires explicit ./pok <name>
    ManualRun: []pocket.Runnable{
        Deploy,
        Release,
    },
}

var Deploy = pocket.Task("deploy", "deploy to production", deploy)
var Release = pocket.Task("release", "create a release", release)
```

Running `./pok` executes AutoRun. Running `./pok deploy` executes the Deploy
task.

## Path Filtering

Use `RunIn()` to control where tasks are visible and run. **`RunIn()` is
optional** - tasks without it only run at the git root:

```go
var Config = pocket.Config{
    AutoRun: pocket.Parallel(
        // These use RunIn - run in each detected Go/Markdown location
        pocket.RunIn(golang.Tasks(), pocket.Detect(golang.Detect())),
        pocket.RunIn(markdown.Tasks(), pocket.Detect(markdown.Detect())),

        // No RunIn - only runs at git root
        github.Workflows,
    ),
}
```

For e.g. monorepos, use `RunIn()` to control where tasks run:

```go
// Run in specific directories
pocket.RunIn(myTask, pocket.Include("services/api", "services/web"))

// Auto-detect directories containing go.mod
pocket.RunIn(golang.Tasks(), pocket.Detect(golang.Detect()))

// Exclude directories
pocket.RunIn(golang.Tasks(), pocket.Detect(golang.Detect()), pocket.Exclude("vendor"))

// Combine detection with filtering
pocket.RunIn(golang.Tasks(),
    pocket.Detect(golang.Detect()),
    pocket.Include("services/.*"),
    pocket.Exclude("testdata"),
)
```

### Skipping Tasks in Specific Paths

While `Exclude()` excludes entire task compositions from directories, use
`Skip()` to skip specific tasks within a composition. This controls which tasks
run in each detected directory - it doesn't modify the task's arguments.

For example, in a monorepo with multiple Go services (each with their own
`go.mod`), you might want to skip slow tests in certain services:

```go
var Config = pocket.Config{
    AutoRun: pocket.RunIn(golang.Tasks(),
        pocket.Detect(golang.Detect()),
        pocket.Skip(golang.Test, "services/api", "services/worker"),
    ),

    // Make skipped tests available under a different name
    ManualRun: []pocket.Runnable{
        pocket.RunIn(pocket.Clone(golang.Test, pocket.Named("integration-test")),
            pocket.Include("services/api", "services/worker"),
        ),
    },
}
```

Here `services/api/` and `services/worker/` are separate Go modules detected by
`golang.Detect()`. All composed tasks (format, lint, vulncheck) run in all
detected modules, but `go-test` is skipped in those two. The skipped tests are
available as `integration-test` when run from those directories.

Note: `Clone(..., Named(...))` creates a copy of the task with a different CLI
name. This avoids duplicate names when the same task appears in both AutoRun and
ManualRun.

## Options

Tasks can accept options:

```go
type DeployOptions struct {
    Env    string `arg:"env" usage:"target environment"`
    DryRun bool   `arg:"dry-run" usage:"print without executing"`
}

var Deploy = pocket.Task("deploy", "deploy to environment", deploy,
    pocket.Opts(DeployOptions{Env: "staging"}),
)

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

### Helpers

```go
// Composition
pocket.Serial(task1, task2, task3)     // run in sequence
pocket.Parallel(task1, task2, task3)   // run concurrently

// Command execution (returns Runnable)
pocket.Run("cmd", "arg1", "arg2")     // static command
pocket.Do(func(ctx) error)            // dynamic commands, arbitrary Go code

// Lower-level execution (inside Do() closures)
pocket.Exec(ctx, "cmd", "arg1", "arg2")       // run command in current path
pocket.ExecIn(ctx, "dir", "cmd", "args"...)   // run command in specific dir
pocket.Command(ctx, "cmd", "args"...)         // create exec.Cmd with .pocket/bin in PATH
pocket.Printf(ctx, "format %s", arg)          // formatted output to stdout
pocket.Println(ctx, "message")                // line output to stdout

// Context
pocket.Options[T](ctx)        // get typed options from context
pocket.Path(ctx)              // current path (for path-filtered tasks)
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

// Installation (returns Runnable)
pocket.InstallGo("github.com/org/tool", "v1.0.0")  // go install

// Installation helpers (inside Do() closures)
pocket.CreateSymlink("path/to/binary")              // symlink to .pocket/bin/
pocket.ConfigPath(ctx, "tool", config)              // find/create config file

// Download & Extract (returns Runnable)
pocket.Download(url,
    pocket.WithDestDir(dir),                              // extraction destination
    pocket.WithFormat("tar.gz"),                          // format: tar.gz, tar, zip, ""
    pocket.WithExtract(pocket.WithExtractFile(name)),     // extract specific file
    pocket.WithExtract(pocket.WithRenameFile(src, dest)), // rename during extract
    pocket.WithExtract(pocket.WithFlatten()),             // flatten directory structure
    pocket.WithSymlink(),                                 // symlink to .pocket/bin/
    pocket.WithSkipIfExists(path),                        // skip if file exists
    pocket.WithHTTPHeader(key, value),                    // add HTTP header
)
pocket.FromLocal(path, opts...)  // process local file with same options

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
    AutoRun: pocket.RunIn(golang.Tasks(),
        pocket.Detect(golang.Detect()),
        pocket.Skip(golang.Test, "services/worker"),
    ),

    // ManualRun: requires ./pok <name>
    ManualRun: []pocket.Runnable{
        pocket.RunIn(
            pocket.Clone(golang.Test, pocket.Named("slow-test")),
            pocket.Include("services/worker"),
        ),
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
