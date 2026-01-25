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
  - [Python Tools](#python-tools)
    - [Standalone Python Tools](#standalone-python-tools)
    - [Project Python Tools](#project-python-tools)
  - [Node Tools](#node-tools)
    - [Standalone Node Tools](#standalone-node-tools)
    - [Project Node Tools](#project-node-tools)
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
  - [Git Diff Check](#git-diff-check)
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

**Output behavior:**

- **With `-v` flag:** Output streams to stdout/stderr in real-time
- **Without `-v` flag:** Output is captured and shown if:
  - The command fails, OR
  - The output contains warnings (detects: `warn`, `deprecat`, `notice`,
    `caution`, `error`)

This keeps CI logs clean while ensuring warnings and deprecation notices are
never silently hidden.

**Other features:**

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

One of Pocket's strengths is automated tool installation. Each tool package owns
its complete lifecycle: installation, versioning, and making itself available
for execution.

### Tool Availability Patterns

Tools make themselves available through one of three patterns:

| Pattern         | When to use                                          | How tasks invoke                 |
| :-------------- | :--------------------------------------------------- | :------------------------------- |
| **Symlink**     | Native binaries (Go, Rust, C)                        | `pk.Exec(ctx, "tool", ...)`      |
| **Tool Exec**   | Standalone runtime-dependent tools                   | `tool.Exec(ctx, ...)`            |
| **Runtime Run** | Project-managed tools (pyproject.toml, package.json) | `uv.Run(ctx, opts, "tool", ...)` |

**Symlink pattern:** The tool downloads a self-contained binary to
`.pocket/tools/<tool>/<version>/` and symlinks it to `.pocket/bin/`. Since
`pk.Exec` adds `.pocket/bin/` to PATH, tasks can invoke it by name.

**Tool Exec pattern:** For tools that require a runtime (Node.js, Python), the
tool package exposes an `Exec()` function that handles runtime invocation
internally. No symlink is created because the tool's shebang
(`#!/usr/bin/env node`) would fail without the runtime on system PATH.

**Runtime Run pattern:** For tools defined in the project's dependency file
(pyproject.toml, package.json), use the runtime's `Run()` function directly. The
project controls tool versions, not Pocket.

### Go Tools

Use `pk.InstallGo` for Go-based tools:

```go
var installLint = pk.NewTask(
    "install:golangci-lint",
    "install linter",
    nil,
    pk.InstallGo("github.com/golangci/golangci-lint/cmd/golangci-lint", "v1.64.8"),
).Hidden().Global()

var Lint = pk.NewTask("lint", "run golangci-lint", nil,
    pk.Serial(
        installLint,
        pk.Do(func(ctx context.Context) error {
            return pk.Exec(ctx, "golangci-lint", "run")
        }),
    ),
)
```

Install tasks use `.Hidden().Global()`:

- **Hidden**: Excludes from CLI help (internal implementation detail)
- **Global**: Deduplicates by name only, ignoring path context

Without `.Global()`, if `Lint` runs in multiple paths (via `WithDetect`), the
install would run once per path. With `.Global()`, it runs once total.

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
).Hidden().Global()
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

### Python Tools

Pocket provides Python tooling via the `tools/uv` and `tasks/python` packages.
Tools are installed from the project's `pyproject.toml` into version-specific
venvs under `.pocket/venvs/`.

#### Using the Python Task Bundle

The `tasks/python` package provides ready-to-use tasks for common Python
workflows:

```go
import (
    "github.com/fredrikaverpil/pocket/pk"
    "github.com/fredrikaverpil/pocket/tasks/python"
)

var Config = &pk.Config{
    Auto: pk.Serial(
        // Format, lint, typecheck, and test with Python 3.9 (with coverage)
        pk.WithOptions(
            python.Tasks(),
            python.WithVersion("3.9"),
            python.WithTestCoverage(),
            pk.WithDetect(python.Detect()),
        ),
        // Test against remaining Python versions (without coverage)
        pk.WithOptions(
            pk.Parallel(
                pk.WithOptions(python.Test, python.WithVersion("3.10")),
                pk.WithOptions(python.Test, python.WithVersion("3.11")),
                pk.WithOptions(python.Test, python.WithVersion("3.12")),
                pk.WithOptions(python.Test, python.WithVersion("3.13")),
            ),
            pk.WithDetect(python.Detect()),
        ),
    ),
}
```

**Python-specific options:**

| Option                     | Description                                            |
| :------------------------- | :----------------------------------------------------- |
| `python.WithVersion("X")` | Set Python version AND task name suffix (e.g., `:3.9`) |
| `python.WithTestCoverage()`    | Enable coverage for the Test task                      |

**Available tasks:**

| Task               | Description                          |
| :----------------- | :----------------------------------- |
| `python.Tasks()`   | All tasks: Format, Lint, Typecheck, Test |
| `python.Format`    | Format with ruff                     |
| `python.Lint`      | Lint with ruff (auto-fix by default) |
| `python.Typecheck` | Type-check with mypy                 |
| `python.Test`      | Run pytest                           |
| `python.Detect()`  | DetectFunc for pyproject.toml        |

When `python.WithVersion("3.9")` is used, tasks are automatically named with a
suffix (e.g., `py-test:3.9`) for CLI invocation and GitHub Actions matrix
generation.

> [!NOTE]
>
> Tests run without coverage by default to avoid conflicts when testing multiple
> Python versions. Use `python.WithTestCoverage()` to enable coverage for one
> version.

#### Venv Location

Venvs are created at `.pocket/venvs/<project-path>/venv-<version>/`:

- Root project: `.pocket/venvs/venv-3.9/`
- Subdirectory project: `.pocket/venvs/services/api/venv-3.9/`

This path-scoped approach prevents collisions in monorepos with multiple
`pyproject.toml` files.

#### Low-Level uv API

For custom Python tasks, use the `tools/uv` package directly:

```go
import "github.com/fredrikaverpil/pocket/tools/uv"

var Docs = pk.NewTask("docs", "build documentation", nil,
    pk.Serial(
        uv.Install,
        pk.Do(func(ctx context.Context) error {
            // Sync dependencies from pyproject.toml
            if err := uv.Sync(ctx, uv.SyncOptions{
                PythonVersion: "3.12",
                AllGroups:     true,
            }); err != nil {
                return err
            }
            // Run the tool from the synced environment
            return uv.Run(ctx, uv.RunOptions{
                PythonVersion: "3.12",
            }, "mkdocs", "build")
        }),
    ),
)
```

**Key types and functions:**

| Type/Function                             | Description                                  |
| :---------------------------------------- | :------------------------------------------- |
| `uv.Install`                              | Task that ensures uv is available            |
| `uv.SyncOptions`                          | Config for `uv sync` (version, venv, groups) |
| `uv.RunOptions`                           | Config for `uv run` (version, venv)          |
| `uv.Sync(ctx, opts)`                      | Install deps from pyproject.toml             |
| `uv.Run(ctx, opts, cmd, args...)`         | Run command from synced environment          |
| `uv.VenvPath(projectPath, pythonVersion)` | Compute venv path for a project              |

**SyncOptions fields:**

| Field           | Description                                             |
| :-------------- | :------------------------------------------------------ |
| `PythonVersion` | Python version (default: `uv.DefaultPythonVersion`)     |
| `VenvPath`      | Explicit venv path (default: auto-computed)             |
| `ProjectDir`    | Where pyproject.toml lives (default: `PathFromContext`) |
| `AllGroups`     | Install all dependency groups                           |

#### Standalone Python Tools

For tools managed entirely by Pocket (not from pyproject.toml), create a tool
package in `tools/<toolname>/` that provides an `Exec()` function. This follows
the **Tool Exec pattern**—no symlink is created because Python scripts have
shebangs that require the runtime on PATH.

See `tools/mdformat/` for a complete example:

```go
// tools/mdformat/mdformat.go
package mdformat

// Install ensures mdformat is available.
var Install = pk.NewTask("install:mdformat", "install mdformat", nil,
    pk.Serial(uv.Install, installMdformat()),
).Hidden().Global()

func installMdformat() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        venvDir := pk.FromToolsDir("mdformat", Version())
        binary := uv.BinaryPath(venvDir, "mdformat")

        // Skip if already installed.
        if _, err := os.Stat(binary); err == nil {
            return nil
        }

        // Create venv and install.
        if err := uv.CreateVenv(ctx, venvDir, pythonVersion); err != nil {
            return err
        }
        reqPath := filepath.Join(venvDir, "requirements.txt")
        if err := os.WriteFile(reqPath, requirements, 0o644); err != nil {
            return err
        }
        // No symlink - Exec() handles invocation.
        return uv.PipInstallRequirements(ctx, venvDir, reqPath)
    })
}

// Exec runs mdformat with the given arguments.
func Exec(ctx context.Context, args ...string) error {
    venvDir := pk.FromToolsDir("mdformat", Version())
    python := uv.BinaryPath(venvDir, "python")
    // Run as module to avoid shebang path issues.
    execArgs := append([]string{"-m", "mdformat"}, args...)
    return pk.Exec(ctx, python, execArgs...)
}
```

Tasks use the tool via its `Exec()` function:

```go
// tasks/markdown/format.go
var Format = pk.NewTask("md-format", "format Markdown files", nil,
    pk.Serial(mdformat.Install, pk.Do(func(ctx context.Context) error {
        return mdformat.Exec(ctx, "--wrap", "80", ".")
    })),
)
```

**When to use which pattern:**

| Aspect          | Standalone (`.pocket/tools/`) | Project-managed (`.pocket/venvs/`) |
| :-------------- | :---------------------------- | :--------------------------------- |
| Version control | Pocket controls version       | Project's pyproject.toml           |
| Use case        | Shared tools across projects  | Project-specific tooling           |
| Example         | mdformat, prettier            | ruff, mypy, pytest                 |
| Invocation      | `tool.Exec(ctx, ...)`         | `uv.Run(ctx, opts, "tool", ...)`   |

### Node Tools

Pocket provides bun for JavaScript/TypeScript tools via the `tools/bun` package.

#### Standalone Node Tools

For tools managed entirely by Pocket, create a tool package in
`tools/<toolname>/` that provides an `Exec()` function. Embed `package.json` and
`bun.lock` to control versions. This follows the **Tool Exec pattern**—no
symlink is created because Node scripts have shebangs that require Node on PATH.

See `tools/prettier/` for a complete example:

```go
// tools/prettier/prettier.go
package prettier

//go:embed package.json
var packageJSON []byte

//go:embed bun.lock
var lockfile []byte

// Install ensures prettier is available.
var Install = pk.NewTask("install:prettier", "install prettier", nil,
    pk.Serial(bun.Install, installPrettier()),
).Hidden().Global()

func installPrettier() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        installDir := pk.FromToolsDir(Name, Version())
        binary := bun.BinaryPath(installDir, Name)

        // Skip if already installed.
        if _, err := os.Stat(binary); err == nil {
            return nil
        }

        // Write embedded lockfiles.
        os.MkdirAll(installDir, 0o755)
        os.WriteFile(filepath.Join(installDir, "package.json"), packageJSON, 0o644)
        os.WriteFile(filepath.Join(installDir, "bun.lock"), lockfile, 0o644)

        // No symlink - Exec() handles invocation.
        return bun.InstallFromLockfile(ctx, installDir)
    })
}

// Exec runs prettier with the given arguments.
func Exec(ctx context.Context, args ...string) error {
    installDir := pk.FromToolsDir(Name, Version())
    // Run via bun since prettier is a Node.js script.
    return bun.Run(ctx, installDir, Name, args...)
}
```

Tasks use the tool via its `Exec()` function:

```go
// tasks/markdown/format.go
var Format = pk.NewTask("md-format", "format Markdown files", nil,
    pk.Serial(prettier.Install, pk.Do(func(ctx context.Context) error {
        return prettier.Exec(ctx, "--write", "**/*.md")
    })),
)
```

**Key functions:**

| Function                                 | Description                          |
| :--------------------------------------- | :----------------------------------- |
| `bun.Install`                            | Task that ensures bun is available   |
| `bun.InstallFromLockfile(ctx, dir)`      | Install from package.json + bun.lock |
| `bun.BinaryPath(installDir, name)`       | Path to binary in node_modules/.bin  |
| `bun.Run(ctx, installDir, pkg, args...)` | Run a package via bun                |

#### Project Node Tools

For tools where the _project_ controls versions via `package.json`—useful for
build tools, test runners, or tools that should match the project's Node
environment:

```go
import "github.com/fredrikaverpil/pocket/tools/bun"

var Build = pk.NewTask("build", "build frontend", nil,
    pk.Serial(
        bun.Install,
        pk.Do(func(ctx context.Context) error {
            projectDir := pk.FromGitRoot("frontend")

            // Install dependencies from project's package.json + bun.lock.
            if err := bun.InstallFromLockfile(ctx, projectDir); err != nil {
                return err
            }

            // Run the build script.
            return bun.Run(ctx, projectDir, "build")
        }),
    ),
)
```

**When to use which pattern:**

| Aspect           | Standalone                    | Project-managed                   |
| :--------------- | :---------------------------- | :-------------------------------- |
| Version control  | Pocket (embedded lockfile)    | Project's package.json + bun.lock |
| Storage location | `.pocket/tools/<tool>/<ver>/` | Project's node_modules/           |
| Use case         | Shared tools across projects  | Project-specific tooling          |
| Example          | prettier                      | Build tools, test runners         |
| Invocation       | `tool.Exec(ctx, ...)`         | `bun.Run(ctx, dir, "tool", ...)`  |

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

### Option Types

`pk.WithOptions` accepts two types of options:

**Generic options (`pk.With*`)** work with any task:

| Option                               | Description                          |
| :----------------------------------- | :----------------------------------- |
| `pk.WithIncludePath(patterns...)`    | Only run in matching directories     |
| `pk.WithExcludePath(patterns...)`    | Skip matching directories            |
| `pk.WithDetect(fn)`                  | Auto-detect directories              |
| `pk.WithName(suffix)`                | Add suffix to task names (e.g., `:v2`) |
| `pk.WithFlag(task, name, value)`     | Override a task's flag               |
| `pk.WithSkipTask(tasks...)`          | Remove tasks from scope              |
| `pk.WithForceRun()`                  | Disable deduplication                |

**Task-specific options** are provided by task packages:

| Option                   | Package  | Description                              |
| :----------------------- | :------- | :--------------------------------------- |
| `python.WithVersion(v)`  | `python` | Set Python version AND name suffix       |
| `python.WithTestCoverage()`  | `python` | Enable coverage for test task            |
| `golang.WithRace()`      | `golang` | Enable race detector (if available)      |

Task-specific options may combine multiple effects. For example,
`python.WithVersion("3.9")` sets both the Python version for runtime AND adds
`:3.9` to task names for CLI/matrix generation.

```go
pk.WithOptions(
    python.Tasks(),
    python.WithVersion("3.9"),  // Task-specific: sets version + name suffix
    python.WithTestCoverage(),      // Task-specific: enables coverage
    pk.WithDetect(python.Detect()), // Generic: auto-detect Python projects
)
```

**Creating task-specific options:** When building your own task packages, use
`pk.CombineOptions` to compose multiple effects:

```go
func WithVersion(version string) pk.PathOption {
    return pk.CombineOptions(
        pk.WithContextValue(versionKey{}, version), // Runtime config
        pk.WithName(version),                        // Name suffix
    )
}
```

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

> [!NOTE]
>
> `WithFlag` overrides only apply when the task runs as part of the composition
> tree (e.g., via bare `./pok`). When you invoke a task directly (e.g.,
> `./pok my-task`), it runs with its default flag values, bypassing
> composition-level overrides. If you need the flag applied for direct
> invocation, pass it explicitly: `./pok my-task -flag-name value`.

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
    Auto   Runnable     // Tasks executed on bare ./pok
    Manual []Runnable   // Tasks only run when explicitly invoked
    Plan   *PlanConfig  // Plan building, shims, and CI configuration
}

type PlanConfig struct {
    SkipDirs          []string    // Directories to skip during filesystem walk
    IncludeHiddenDirs bool        // Include hidden directories (default: false)
    Shims             *ShimConfig // Which shim scripts to generate
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

    Plan: &pk.PlanConfig{
        // Extend defaults
        SkipDirs: append(pk.DefaultSkipDirs, "testdata", "generated"),

        // Or skip nothing
        // SkipDirs: []string{},

        // Include hidden directories (.git, .cache, etc.)
        IncludeHiddenDirs: false, // default
    },
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
    Auto: pk.Serial(Lint, Test),

    Plan: &pk.PlanConfig{
        Shims: pk.AllShimsConfig(), // Generate all platform shims
    },
}
```

### Git Diff Check

Pocket can run `git diff --exit-code` after task execution to catch unintended
file modifications. This is enabled with the `-g` flag:

```bash
./pok -g          # Run all auto tasks, then git diff
./pok lint -g     # Run lint task, then git diff
```

The `-g` flag causes Pocket to fail if there are uncommitted changes after tasks
complete. This is useful in CI to ensure generated files are up to date.

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

> [!NOTE]
>
> The `github-workflows` task has several flags (e.g., `-include-pocket-matrix`,
> `-skip-pr`). If you configure these via `WithFlag` in your composition,
> remember that flag overrides only apply when running through the composition
> tree (bare `./pok`). When invoking `./pok github-workflows` directly, pass
> flags explicitly: `./pok github-workflows -include-pocket-matrix`.

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
