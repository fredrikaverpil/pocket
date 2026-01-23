# Pocket User Guide

This guide covers everything you need to know to use Pocket effectively, from
defining your first task to building complex CI pipelines.

## Table of Contents

- [Tasks](#tasks)
  - [Defining Tasks](#defining-tasks)
  - [The Do Helper](#the-do-helper)
  - [Hidden Tasks](#hidden-tasks)
  - [Manual Tasks](#manual-tasks)
  - [Task Flags](#task-flags)
  - [Suppressing Headers](#suppressing-headers)
- [Executing Commands](#executing-commands)
  - [The Exec Helper](#the-exec-helper)
  - [Output Functions](#output-functions)
- [Tool Management](#tool-management)
  - [Go Tools](#go-tools)
  - [Custom Tools](#custom-tools)
  - [Download API](#download-api)
  - [Extract API](#extract-api)
  - [Platform Helpers](#platform-helpers)
- [Composition](#composition)
  - [Serial Execution](#serial-execution)
  - [Parallel Execution](#parallel-execution)
  - [Task Deduplication](#task-deduplication)
- [Path Filtering](#path-filtering)
  - [Include and Exclude](#include-and-exclude)
  - [Auto-Detection](#auto-detection)
  - [Task-Specific Scoping](#task-specific-scoping)
  - [Shim Scoping](#shim-scoping)
- [Configuration](#configuration)
  - [Config Struct](#config-struct)
  - [Directory Skipping](#directory-skipping)
  - [Shim Generation](#shim-generation)
- [Plan Introspection](#plan-introspection)
  - [Accessing the Plan](#accessing-the-plan)
  - [Plan Structure](#plan-structure)
- [GitHub Actions Integration](#github-actions-integration)
  - [Simple Workflow](#simple-workflow)
  - [Matrix Workflow](#matrix-workflow)
  - [MatrixConfig Options](#matrixconfig-options)

---

## Tasks

Tasks are the fundamental units of work in Pocket—linting, testing, building,
deploying. Each task has a name, description, optional flags, and a body that
defines what it does.

### Defining Tasks

A task is created using `pk.NewTask`. It requires a name, a usage description,
an optional `*flag.FlagSet`, and a `Runnable` body.

```go
var Hello = pk.NewTask(
    "hello",
    "print a greeting",
    nil, // no flags
    pk.Do(func(ctx context.Context) error {
        fmt.Println("Hello!")
        return nil
    }),
)
```

### The Do Helper

The `pk.Do` function wraps a simple Go function `func(context.Context) error`
into a `Runnable`. This is the most common way to implement task logic.

```go
pk.Do(func(ctx context.Context) error {
    // Your task logic here
    return nil
})
```

### Hidden Tasks

If a task is only intended to be used as a dependency or called
programmatically, hide it from CLI help output:

```go
var InternalTask = pk.NewTask("internal", "...", nil, body).Hidden()
```

Hidden tasks still execute when part of a composition tree—they just don't
appear in `./pok -h`.

### Manual Tasks

By default, tasks in `Config.Auto` run when you execute bare `./pok`. If you
want a task to _only_ run when explicitly named (e.g., `./pok deploy`), use the
`Manual()` method:

```go
var Deploy = pk.NewTask("deploy", "deploy the app", nil, body).Manual()
```

Or add it to `Config.Manual`:

```go
var Config = &pk.Config{
    Auto:   pk.Serial(Lint, Test),
    Manual: []pk.Runnable{Deploy},
}
```

### Task Flags

Pocket uses the standard library's `flag` package. Define flags for a task by
passing a `FlagSet`:

```go
var (
    deployFlags = flag.NewFlagSet("deploy", flag.ContinueOnError)
    env         = deployFlags.String("env", "staging", "target environment")
)

var Deploy = pk.NewTask("deploy", "deploy the app", deployFlags,
    pk.Do(func(ctx context.Context) error {
        fmt.Printf("Deploying to %s...\n", *env)
        return nil
    }),
)
```

Run it with:

```bash
./pok deploy -env prod
```

### Suppressing Headers

By default, tasks print a `:: taskname` header before execution. For tasks that
output machine-readable data (e.g., JSON), suppress the header:

```go
var Matrix = pk.NewTask("gha-matrix", "output CI matrix", nil, body).HideHeader()
```

---

## Executing Commands

### The Exec Helper

`pk.Exec` runs external commands with proper output handling:

```go
pk.Do(func(ctx context.Context) error {
    return pk.Exec(ctx, "go", "test", "./...")
})
```

**Features:**

- Correctly captures and buffers output (important for parallel execution)
- Respects context cancellation (graceful shutdown)
- Adds `.pocket/bin` to the command's `PATH`
- Sends SIGINT on cancellation (Unix), allowing graceful cleanup

### Output Functions

Use these instead of `fmt.Print*` to ensure correct output handling in parallel
contexts:

| Function                          | Description                |
| :-------------------------------- | :------------------------- |
| `pk.Printf(ctx, format, args...)` | Formatted output to stdout |
| `pk.Println(ctx, args...)`        | Line output to stdout      |
| `pk.Errorf(ctx, format, args...)` | Formatted output to stderr |

```go
pk.Do(func(ctx context.Context) error {
    pk.Printf(ctx, "Processing %d items...\n", count)
    return nil
})
```

---

## Tool Management

One of Pocket's strengths is automated tool installation. Tools are downloaded,
versioned, and cached in `.pocket/tools/`, then symlinked to `.pocket/bin/`.

### Go Tools

Use `pk.InstallGo` for Go-based tools:

```go
var installLint = pk.NewTask(
    "install:golangci-lint",
    "install linter",
    nil,
    pk.InstallGo("github.com/golangci/golangci-lint/cmd/golangci-lint", "v1.64.8"),
).Hidden()

var Lint = pk.NewTask("lint", "run golangci-lint", nil,
    pk.Serial(
        installLint,
        pk.Do(func(ctx context.Context) error {
            return pk.Exec(ctx, "golangci-lint", "run")
        }),
    ),
)
```

Because Pocket deduplicates tasks, `installLint` only runs once even if multiple
tasks depend on it.

### Custom Tools

For non-Go tools, use the Download API to fetch binaries from GitHub releases or
other sources.

```go
var installStyLua = pk.NewTask("install:stylua", "install StyLua formatter", nil,
    pk.Download(
        fmt.Sprintf(
            "https://github.com/JohnnyMorganz/StyLua/releases/download/v%s/stylua-%s-%s.zip",
            "2.0.2",
            pk.HostOS(),
            pk.ArchToX8664(pk.HostArch()),
        ),
        pk.WithDestDir(pk.FromToolsDir("stylua", "2.0.2")),
        pk.WithFormat("zip"),
        pk.WithExtract(pk.WithExtractFile(pk.BinaryName("stylua"))),
        pk.WithSymlink(),
        pk.WithSkipIfExists(pk.FromToolsDir("stylua", "2.0.2", pk.BinaryName("stylua"))),
    ),
).Hidden()
```

### Download API

`pk.Download` creates a `Runnable` that fetches a URL and optionally extracts
it.

```go
func Download(url string, opts ...DownloadOpt) Runnable
```

**Download Options:**

| Option                   | Description                                        |
| :----------------------- | :------------------------------------------------- |
| `WithDestDir(dir)`       | Destination directory for extraction               |
| `WithFormat(format)`     | Archive format: `"tar.gz"`, `"tar"`, `"zip"`, `""` |
| `WithExtract(opt)`       | Add extraction options (see Extract API)           |
| `WithSymlink()`          | Create symlink in `.pocket/bin/` after extraction  |
| `WithSkipIfExists(path)` | Skip download if the specified file exists         |

### Extract API

Extraction options control how archives are unpacked:

| Option                      | Description                                    |
| :-------------------------- | :--------------------------------------------- |
| `WithExtractFile(name)`     | Extract only the specified file (by base name) |
| `WithRenameFile(src, dest)` | Extract a file and rename it                   |
| `WithFlatten()`             | Flatten directory structure to destDir root    |

**Standalone extraction functions:**

```go
func ExtractTarGz(src, destDir string, opts ...ExtractOpt) error
func ExtractTar(src, destDir string, opts ...ExtractOpt) error
func ExtractZip(src, destDir string, opts ...ExtractOpt) error
```

### Platform Helpers

These functions help construct platform-specific download URLs:

| Function                 | Description                                      |
| :----------------------- | :----------------------------------------------- |
| `HostOS()`               | Current OS: `"darwin"`, `"linux"`, `"windows"`   |
| `HostArch()`             | Current arch: `"amd64"`, `"arm64"`               |
| `ArchToX8664(arch)`      | Convert `amd64`→`x86_64`, `arm64`→`aarch64`      |
| `ArchToX64(arch)`        | Convert `amd64`→`x64`                            |
| `BinaryName(name)`       | Append `.exe` on Windows                         |
| `OSToTitle(os)`          | Convert `darwin`→`Darwin`                        |
| `DefaultArchiveFormat()` | Returns `"zip"` on Windows, `"tar.gz"` otherwise |

**Platform constants:**

```go
const (
    Darwin  = "darwin"
    Linux   = "linux"
    Windows = "windows"
    AMD64   = "amd64"
    ARM64   = "arm64"
    X8664   = "x86_64"
    AARCH64 = "aarch64"
    X64     = "x64"
)
```

---

## Composition

Tasks are composed using `Serial` and `Parallel` combinators to build execution
trees.

### Serial Execution

`pk.Serial` runs tasks one after another. If any task returns an error,
execution stops immediately.

```go
var Auto = pk.Serial(
    Format,  // runs first
    Lint,    // runs second
    Test,    // runs third
)
```

### Parallel Execution

`pk.Parallel` runs tasks concurrently. Pocket automatically buffers output so
logs don't interleave—each task's output flushes atomically when it completes.

```go
var Auto = pk.Parallel(Lint, Test, Build)
```

**Behavior:**

- Single task in Parallel → runs without buffering (real-time output)
- Multiple tasks → buffered output, first-to-complete flushes first
- If one task fails, context is cancelled and remaining tasks exit early

### Task Deduplication

The same task at the same path only runs once per invocation, even if referenced
multiple times in your composition tree. This makes it safe to compose shared
dependencies without redundant work.

```go
pk.Serial(
    pk.Parallel(Lint, Test),  // Both depend on Install
    Build,                     // Also depends on Install
)
// Install runs once, not three times
```

Use `WithForceRun()` to bypass deduplication when needed:

```go
pk.WithOptions(
    CleanTask,
    pk.WithForceRun(),  // Always run, even if already executed
)
```

---

## Path Filtering

In monorepos or multi-module projects, you often want to run tasks only in
specific directories. All path patterns are **regular expressions**.

### Include and Exclude

Use `pk.WithOptions` to apply path constraints:

```go
pk.WithOptions(
    pk.Parallel(Lint, Test),
    pk.WithIncludePath("services/.*"),     // Only in services/ subdirectories
    pk.WithExcludePath("vendor"),          // Skip vendor/ everywhere
)
```

### Auto-Detection

Auto-detection scans your repository for marker files (like `go.mod` or
`package.json`) and runs tasks in those directories:

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithDetect(pk.DetectByFile("go.mod")),
)
```

**Built-in detection:**

```go
func DetectByFile(filenames ...string) DetectFunc
```

Pocket uses **refining composition**: nested `WithOptions` accumulate
constraints. Inner detection functions only search within directories allowed by
the outer scope.

```go
pk.WithOptions(
    pk.WithOptions(
        golang.Tasks(),
        pk.WithDetect(pk.DetectByFile("go.mod")),
    ),
    pk.WithExcludePath("testdata"),  // Applies to inner scope too
)
```

The filesystem is walked **once** and cached, ensuring detection is fast even in
large repositories.

### Task-Specific Scoping

Apply constraints to specific tasks without refactoring the tree:

| Option                               | Description                            |
| :----------------------------------- | :------------------------------------- |
| `WithExcludePath(patterns...)`       | Exclude paths for ALL tasks in scope   |
| `WithExcludeTask(task, patterns...)` | Exclude paths for a SPECIFIC task only |
| `WithSkipTask(tasks...)`             | Remove tasks entirely from scope       |
| `WithFlag(task, name, value)`        | Set default flag value for a task      |

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithExcludePath("vendor"),              // Global: no tasks run in vendor/
    pk.WithExcludeTask(golang.Test, "foo/.*"), // Only go-test skips foo/
    pk.WithSkipTask(golang.Lint),              // Remove linting entirely
    pk.WithFlag(golang.Test, "race", true),    // Enable race detector
)
```

Tasks can be specified by string name or task object (recommended for type
safety).

### Shim Scoping

Pocket generates `./pok` shims in directories matched by `WithIncludePath` or
`WithDetect`.

- Running `./pok` from **root** shows and executes all tasks
- Running `./pok` from a **subdirectory** only shows tasks scoped to that path

```bash
./pok                       # runs all tasks across all paths
cd services/api && ./pok    # only runs tasks scoped to services/api
```

---

## Configuration

### Config Struct

The main entry point for configuring Pocket:

```go
type Config struct {
    Auto              Runnable      // Tasks executed on bare ./pok
    Manual            []Runnable    // Tasks only run when explicitly invoked
    SkipDirs          []string      // Directories to skip during filesystem walk
    IncludeHiddenDirs bool          // Include hidden directories (default: false)
    Shims             *ShimConfig   // Which shim scripts to generate
}
```

### Directory Skipping

Control which directories are skipped during filesystem walking:

```go
// Default skip list (used when SkipDirs is nil)
var DefaultSkipDirs = []string{
    "vendor",       // Go, PHP, Ruby dependencies
    "node_modules", // Node.js dependencies
    "dist",         // Build output
    "__pycache__",  // Python bytecode cache
    "venv",         // Python virtual environment
}
```

**Usage:**

```go
var Config = &pk.Config{
    Auto: pk.Serial(Lint, Test),

    // Extend defaults
    SkipDirs: append(pk.DefaultSkipDirs, "testdata", "generated"),

    // Or skip nothing
    // SkipDirs: []string{},

    // Include hidden directories (.git, .cache, etc.)
    IncludeHiddenDirs: false,  // default
}
```

### Shim Generation

Control which shim scripts are generated:

```go
type ShimConfig struct {
    Posix      bool  // pok (default)
    Windows    bool  // pok.cmd
    PowerShell bool  // pok.ps1
}
```

**Helpers:**

```go
pk.DefaultShimConfig()  // POSIX only
pk.AllShimsConfig()     // All three shims
```

**Usage:**

```go
var Config = &pk.Config{
    Auto:  pk.Serial(Lint, Test),
    Shims: pk.AllShimsConfig(),  // Generate all platform shims
}
```

---

## Plan Introspection

Pocket builds an execution plan before running tasks. This plan is accessible at
runtime for advanced use cases like CI matrix generation.

### Accessing the Plan

```go
pk.Do(func(ctx context.Context) error {
    plan := pk.PlanFromContext(ctx)
    if plan == nil {
        return errors.New("no plan in context")
    }

    // Use plan for introspection
    for _, task := range plan.Tasks() {
        fmt.Printf("Task: %s\n", task.Name())
    }
    return nil
})
```

### Plan Structure

```go
type Plan struct {
    // Internal: tree, tasks, pathMappings, moduleDirectories, shimConfig
}

// Public methods
func (p *Plan) Tasks() []*Task           // All tasks in the plan
func (p *Plan) ShimConfig() *ShimConfig  // Resolved shim configuration
```

**Context accessors:**

| Function                 | Description                                 |
| :----------------------- | :------------------------------------------ |
| `PlanFromContext(ctx)`   | Get the Plan from context (nil if not set)  |
| `PathFromContext(ctx)`   | Current execution path relative to git root |
| `Verbose(ctx)`           | Whether `-v` flag was provided              |
| `OutputFromContext(ctx)` | Get Output struct for writing               |

**Path helpers:**

| Function                  | Description                                   |
| :------------------------ | :-------------------------------------------- |
| `FromGitRoot(elems...)`   | Absolute path relative to git repository root |
| `FromPocketDir(elems...)` | Absolute path relative to `.pocket/`          |
| `FromBinDir(elems...)`    | Absolute path relative to `.pocket/bin/`      |
| `FromToolsDir(elems...)`  | Absolute path relative to `.pocket/tools/`    |

---

## GitHub Actions Integration

Pocket provides two approaches for GitHub Actions CI/CD integration.

### Simple Workflow

A static workflow that runs all tasks on configured platforms:

```bash
./pok github-workflows
```

This generates `.github/workflows/pocket.yml`:

```yaml
jobs:
  pocket:
    strategy:
      matrix:
        os: [ubuntu-latest, macos-latest, windows-latest]
    runs-on: ${{ matrix.os }}
    steps:
      - uses: actions/checkout@v4
      - run: ./pok -v
```

**Pros:** Simple, predictable, easy to understand.

**Cons:** All tasks run serially; no per-task platform customization.

### Matrix Workflow

A two-phase approach that generates a GitHub Actions matrix from your task
configuration:

**Phase 1 (Plan):** Runs `./pok gha-matrix` to generate JSON matrix.

**Phase 2 (Run):** Uses the matrix to run each task as a separate job.

```go
import "github.com/fredrikaverpil/pocket/tasks/github"

var matrixConfig = github.MatrixConfig{
    DefaultPlatforms: []string{"ubuntu-latest", "macos-latest", "windows-latest"},
    TaskOverrides: map[string]github.TaskOverride{
        "go-lint": {Platforms: []string{"ubuntu-latest"}},  // lint only on Linux
    },
    ExcludeTasks: []string{"github-workflows"},
}

var Config = &pk.Config{
    Auto: pk.Parallel(golang.Tasks()),
    Manual: []pk.Runnable{
        github.Matrix(matrixConfig),
    },
}
```

Running `./pok gha-matrix` outputs:

```json
{
  "include": [
    {
      "task": "go-lint",
      "os": "ubuntu-latest",
      "shell": "bash",
      "shim": "./pok"
    },
    {
      "task": "go-test",
      "os": "ubuntu-latest",
      "shell": "bash",
      "shim": "./pok"
    },
    {
      "task": "go-test",
      "os": "macos-latest",
      "shell": "bash",
      "shim": "./pok"
    }
  ]
}
```

### MatrixConfig Options

```go
type MatrixConfig struct {
    // DefaultPlatforms for all tasks. Default: ["ubuntu-latest"]
    DefaultPlatforms []string

    // TaskOverrides provides per-task platform configuration.
    // Keys are regular expressions matched against task names.
    TaskOverrides map[string]TaskOverride

    // ExcludeTasks removes tasks from the matrix entirely.
    ExcludeTasks []string

    // WindowsShell: "powershell" (default) or "bash"
    WindowsShell string

    // WindowsShim: "ps1" (default) or "cmd"
    WindowsShim string
}

type TaskOverride struct {
    // Platforms overrides DefaultPlatforms for this task.
    Platforms []string

    // SkipGitDiff disables the git-diff check after this task.
    SkipGitDiff bool
}
```

**Benefits comparison:**

| Feature                          | Simple    | Matrix   |
| -------------------------------- | --------- | -------- |
| Per-task visibility in GitHub UI | No        | Yes      |
| Per-task platform configuration  | No        | Yes      |
| Parallel task execution          | No        | Yes      |
| Fail-fast granularity            | All tasks | Per task |
| Configuration complexity         | Low       | Medium   |
