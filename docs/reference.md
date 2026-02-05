# API Reference

Technical reference for the `github.com/fredrikaverpil/pocket/pk` package.

## Table of Contents

- [Configuration](#configuration)
- [Composition](#composition)
- [Path Options](#path-options)
- [Detection](#detection)
- [Tasks](#tasks)
- [Execution](#execution)
- [Tool Installation](#tool-installation)
- [Download and Extract](#download-and-extract)
- [Platform Helpers](#platform-helpers)
- [Context](#context)
- [Output](#output)
- [Path Helpers](#path-helpers)
- [Plan Introspection](#plan-introspection)
- [Errors](#errors)
- [CLI](#cli)

---

## Configuration

The `Config` struct is the main entry point for configuring Pocket.

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

```go
// Default directories skipped (used when SkipDirs is nil)
var DefaultSkipDirs = []string{"vendor", "node_modules", "dist", "__pycache__", "venv"}
```

### Shim Configuration

```go
type ShimConfig struct {
    Posix      bool  // pok (default when Shims is nil)
    Windows    bool  // pok.cmd
    PowerShell bool  // pok.ps1
}
```

| Function            | Description                              |
| :------------------ | :--------------------------------------- |
| `DefaultShimConfig` | Returns config with POSIX only (default) |
| `AllShimsConfig`    | Returns config with all shims enabled    |

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

## Tasks

Tasks are the fundamental units of work.

### Creating Tasks

```go
func NewTask(name, usage string, flags *flag.FlagSet, body Runnable) *Task
```

```go
var Lint = pk.NewTask("lint", "run linters", nil, pk.Do(func(ctx context.Context) error {
    return pk.Exec(ctx, "golangci-lint", "run")
}))
```

### Task Methods

| Method           | Description                                            |
| :--------------- | :----------------------------------------------------- |
| `Hidden`         | Exclude from CLI help output                           |
| `IsHidden`       | Returns whether the task is hidden                     |
| `HideHeader`     | Suppress the `:: taskname` header (for machine output) |
| `IsHeaderHidden` | Returns whether the task header is hidden              |
| `Global`         | Deduplicate by name only (ignore path context)         |
| `IsGlobal`       | Returns whether the task deduplicates globally         |
| `Name`           | Returns the task name                                  |
| `Usage`          | Returns the task usage description                     |
| `Flags`          | Returns the task's `*flag.FlagSet`                     |

```go
var Internal = pk.NewTask("internal", "...", nil, body).Hidden()
var Matrix = pk.NewTask("matrix", "...", nil, body).HideHeader()
var Install = pk.NewTask("install:tool", "...", nil, body).Hidden().Global()
```

---

## Composition

The `Runnable` interface is the core abstraction for executable units
(conceptually referred to as "tasks"):

```go
type Runnable interface {
    run(ctx context.Context) error
}
```

### Combinators

| Function      | Description                                                            |
| :------------ | :--------------------------------------------------------------------- |
| `Serial`      | Execute runnables sequentially; stops on first error                   |
| `Parallel`    | Execute runnables concurrently; buffers output to prevent interleaving |
| `WithOptions` | Wrap a runnable with configuration to create task instances            |

```go
pk.Serial(Format, Lint, Test)
pk.Parallel(Lint, Test, Build)
pk.WithOptions(Test, pk.WithIncludePath("services"))
```

---

## Task Options

Options passed to `WithOptions` to control where and how tasks execute.

### Generic Options (pk.With\*)

These options work with any task:

| Option             | Description                                                 |
| :----------------- | :---------------------------------------------------------- |
| `WithIncludePath`  | Run only in directories matching the regex patterns         |
| `WithExcludePath`  | Skip directories matching the regex patterns                |
| `WithDetect`       | Dynamically discover paths using a detection function       |
| `WithNameSuffix`   | Create a named variant (e.g., `py-test` → `py-test:3.9`)    |
| `WithForceRun`     | Bypass task deduplication for the wrapped runnable          |
| `WithFlag`         | Set a default flag value for a task in scope                |
| `WithSkipTask`     | Skip specified tasks within this scope                      |
| `WithExcludeTask`  | Exclude a task from directories matching patterns           |
| `WithContextValue`        | Pass structured config (structs, maps) to tasks via context |

```go
pk.WithOptions(
    pk.Parallel(Lint, Test),
    pk.WithIncludePath("services/.*"),
    pk.WithFlag(Test, "race", true),
)
```

### Task Instances and Variants

A **task** is a reusable definition (like `python.Test`). During planning, the
composition tree is walked and every task becomes a **task instance** with its
resolved configuration (paths, flags, context values).

Use `WithOptions` to configure instances. Use `WithNameSuffix` to create
distinct **variants** of the same task:

```go
// Same task definition, two distinct variants
pk.WithOptions(python.Test, pk.WithNameSuffix("3.9"), pk.WithFlag(python.Test, "python", "3.9"))
pk.WithOptions(python.Test, pk.WithNameSuffix("3.10"), pk.WithFlag(python.Test, "python", "3.10"))
```

Each variant has an **effective name** (base name + suffix). Variants are
deduplicated separately, so `py-test:3.9` and `py-test:3.10` both run.

### Default Execution Path

Tasks run at the repository root (`.`) by default. Use `WithIncludePath` or
`WithDetect` to run tasks in specific directories.

---

## Detection

Detection functions dynamically discover directories based on marker files. The
tasks will run in the detected paths.

```go
type DetectFunc func(dirs []string, gitRoot string) []string
```

| Function       | Description                                            |
| :------------- | :----------------------------------------------------- |
| `DetectByFile` | Find directories containing any of the specified files |

```go
pk.WithOptions(
    pk.Parallel(Lint, Test),
    pk.WithDetect(pk.DetectByFile("go.mod", "package.json")),
)
```

---

## Deduplication

Tasks are deduplicated during execution to prevent running the same work twice.
The deduplication key is `effectiveName@path`:

| Component       | Description                                          |
| :-------------- | :--------------------------------------------------- |
| `effectiveName` | Base name + optional suffix (e.g., `py-test:3.9`)    |
| `path`          | Execution directory relative to git root (e.g., `.`) |

This means:

- Same task at the same path runs only once
- Same task at different paths runs once per path
- Different variants (via `WithNameSuffix`) run separately

**Global tasks** deduplicate by `baseName@.` only, ignoring the execution path.
Use `.Global()` for tool installation tasks that should run once regardless of
how many paths trigger them:

```go
var InstallUV = pk.NewTask("install:uv", "install uv", nil, body).Hidden().Global()
```

**Force execution** with `WithForceRun()` to bypass deduplication entirely:

```go
pk.WithOptions(CleanupTask, pk.WithForceRun())
```

---

## Execution

### Running Code

| Function       | Description                                          |
| :------------- | :--------------------------------------------------- |
| `Do`           | Wrap a `func(context.Context) error` as a `Runnable` |
| `Exec`         | Execute external command with proper output handling |
| `RegisterPATH` | Register a directory to be added to PATH for Exec    |

```go
pk.Do(func(ctx context.Context) error {
    return pk.Exec(ctx, "go", "test", "./...")
})
```

`Exec` behavior:

- **With `-v`:** Output streams to stdout/stderr in real-time
- **Without `-v`:** Output captured, shown on error or if warnings detected
- Detects: `warn`, `deprecat`, `notice`, `caution`, `error` (case-insensitive)
- Adds `.pocket/bin` to PATH
- Sends SIGINT for graceful shutdown (Unix)

`RegisterPATH` adds directories to PATH for all subsequent `Exec` calls. Use
this for tools that can't be symlinked (e.g., neovim on Windows needs its
runtime files):

```go
pk.RegisterPATH("/path/to/nvim/bin")
```

### Output Functions

| Function  | Description                        |
| :-------- | :--------------------------------- |
| `Printf`  | Formatted output to context stdout |
| `Println` | Line output to context stdout      |
| `Errorf`  | Formatted output to context stderr |

```go
pk.Printf(ctx, "Processing %d items...\n", count)
```

---

## Tool Installation

Each tool package owns its complete lifecycle: installation, versioning, and
making itself available for execution.

### Tool Availability Patterns

| Pattern         | When to use                        | How tasks invoke                 |
| :-------------- | :--------------------------------- | :------------------------------- |
| **Symlink**     | Native binaries (Go, Rust, C)      | `pk.Exec(ctx, "tool", ...)`      |
| **Tool Exec**   | Standalone runtime-dependent tools | `tool.Exec(ctx, ...)`            |
| **Runtime Run** | Project-managed tools              | `uv.Run(ctx, opts, "tool", ...)` |

**Symlink:** Binary symlinked to `.pocket/bin/`, tasks invoke by name via
`pk.Exec`.

**Tool Exec:** Tool package exposes `Exec()` function that handles runtime
invocation internally. No symlink (shebangs fail without runtime on PATH).

**Runtime Run:** Project controls versions via pyproject.toml or package.json.
Use runtime's `Run()` function directly.

### Go Tools

```go
func InstallGo(pkg, version string) Runnable
```

Installs a Go package to `.pocket/tools/go/<pkg>/<version>/bin/` and symlinks to
`.pocket/bin/`. Uses **Symlink pattern**.

```go
pk.InstallGo("github.com/golangci/golangci-lint/cmd/golangci-lint", "v1.64.8")
```

### Runtime-Dependent Tools

For Python/Node tools, see `tools/prettier/` and `tools/mdformat/` for examples
of the **Tool Exec pattern**. Each exposes:

- `Install` - Task ensuring the tool is available
- `Exec(ctx, args...)` - Function to invoke the tool

```go
// Usage in tasks
prettier.Exec(ctx, "--write", "**/*.md")
mdformat.Exec(ctx, "--wrap", "80", ".")
```

---

## Download and Extract

Import: `"github.com/fredrikaverpil/pocket/pk/download"`

### Download

```go
func Download(url string, opts ...Opt) pk.Runnable
```

| Option             | Description                                              |
| :----------------- | :------------------------------------------------------- |
| `WithDestDir`      | Destination directory for extraction                     |
| `WithFormat`       | Archive format: `"tar.gz"`, `"tar"`, `"zip"`, `"gz"`, `` |
| `WithExtract`      | Add extraction options                                   |
| `WithSymlink`      | Create symlink in `.pocket/bin/`                         |
| `WithSkipIfExists` | Skip download if file exists                             |
| `WithOutputName`   | Output filename for `"gz"` format (required for gz)      |

```go
import "github.com/fredrikaverpil/pocket/pk/download"

download.Download(
    "https://example.com/tool-v1.0.0-linux-amd64.tar.gz",
    download.WithDestDir(pk.FromToolsDir("tool", "v1.0.0")),
    download.WithFormat("tar.gz"),
    download.WithExtract(download.WithExtractFile("tool")),
    download.WithSymlink(),
    download.WithSkipIfExists(pk.FromToolsDir("tool", "v1.0.0", "tool")),
)
```

### Extract

| Function       | Description                                |
| :------------- | :----------------------------------------- |
| `ExtractTarGz` | Extract .tar.gz archive                    |
| `ExtractTar`   | Extract .tar archive                       |
| `ExtractZip`   | Extract .zip archive                       |
| `ExtractGz`    | Extract a single gzipped file (not tar.gz) |

```go
// ExtractGz extracts a single gzipped file to destDir with the given name
func ExtractGz(src, destDir, destName string) error
```

| Option            | Description                        |
| :---------------- | :--------------------------------- |
| `WithExtractFile` | Extract only the specified file    |
| `WithRenameFile`  | Extract and rename a specific file |
| `WithFlatten`     | Flatten directory structure        |

### Symlink

| Function                      | Description                                          |
| :---------------------------- | :--------------------------------------------------- |
| `CreateSymlink`               | Create symlink in `.pocket/bin/` to given binary     |
| `CreateSymlinkAs`             | Create symlink with custom name in `.pocket/bin/`    |
| `CreateSymlinkWithCompanions` | Create symlink and copy companion files (e.g., DLLs) |
| `CopyFile`                    | Copy a file from src to dst                          |

```go
// CreateSymlinkAs creates a symlink with a custom name
linkPath, err := download.CreateSymlinkAs("/path/to/binary", "custom-name")

// CreateSymlinkWithCompanions copies companion files (useful on Windows)
linkPath, err := download.CreateSymlinkWithCompanions("/path/to/binary", "*.dll")
```

---

## Platform Helpers

Platform detection and helpers are available directly from the `pk` package.

### Runtime Detection

| Function                  | Description                                  |
| :------------------------ | :------------------------------------------- |
| `pk.HostOS`               | Current OS: `darwin`, `linux`, `windows`     |
| `pk.HostArch`             | Current architecture: `amd64`, `arm64`       |
| `pk.BinaryName`           | Append `.exe` on Windows                     |
| `pk.DefaultArchiveFormat` | Returns `zip` on Windows, `tar.gz` otherwise |

### Architecture Conversion

| Function         | Conversion                              |
| :--------------- | :-------------------------------------- |
| `pk.ArchToX8664` | `amd64` → `x86_64`, `arm64` → `aarch64` |
| `pk.ArchToX64`   | `amd64` → `x64`                         |
| `pk.OSToTitle`   | `darwin` → `Darwin`                     |

### Constants

```go
// OS constants - access via pk.Darwin, pk.Linux, pk.Windows
pk.Darwin  // "darwin"
pk.Linux   // "linux"
pk.Windows // "windows"

// Architecture constants - access via pk.AMD64, pk.ARM64
pk.AMD64 // "amd64"
pk.ARM64 // "arm64"

// Alternative naming - access via pk.X8664, pk.AARCH64, pk.X64
pk.X8664   // "x86_64"
pk.AARCH64 // "aarch64"
pk.X64     // "x64"
```

---

## Context

Context accessors and modifiers are available from the `pk` package.

### Accessors (Getters)

| Function             | Description                                 |
| :------------------- | :------------------------------------------ |
| `pk.PathFromContext` | Current execution path relative to git root |
| `pk.PlanFromContext` | The `*Plan` from context (nil if not set)   |
| `pk.Verbose`         | Whether `-v` flag was provided              |

### Modifiers (Setters)

Context modifiers use the `ContextWith*` naming convention to distinguish them
from `PathOption` functions (which use `With*`).

| Function               | Description                                      |
| :--------------------- | :----------------------------------------------- |
| `pk.ContextWithEnv`    | Set an environment variable for `Exec` calls     |
| `pk.ContextWithoutEnv` | Filter out environment variables matching prefix |
| `pk.ContextWithPath`   | Set the execution path for `Exec` calls          |

```go
// Set an environment variable
ctx = pk.ContextWithEnv(ctx, "MY_VAR=value")

// Remove environment variables matching prefix
ctx = pk.ContextWithoutEnv(ctx, "VIRTUAL_ENV")

// Change execution directory
ctx = pk.ContextWithPath(ctx, "services/api")

// Use with Exec
pk.Exec(ctx, "mycmd", "arg1") // runs with modified environment/path
```

---

## Output

```go
type Output struct {
    Stdout io.Writer
    Stderr io.Writer
}
```

| Function    | Description                                       |
| :---------- | :------------------------------------------------ |
| `StdOutput` | Returns Output writing to os.Stdout and os.Stderr |

---

## Path Helpers

| Function        | Description                                   |
| :-------------- | :-------------------------------------------- |
| `FromGitRoot`   | Absolute path relative to git repository root |
| `FromPocketDir` | Absolute path relative to `.pocket/`          |
| `FromBinDir`    | Absolute path relative to `.pocket/bin/`      |
| `FromToolsDir`  | Absolute path relative to `.pocket/tools/`    |

```go
pk.FromToolsDir("golangci-lint", "v1.64.8", "bin", "golangci-lint")
// → /path/to/repo/.pocket/tools/golangci-lint/v1.64.8/bin/golangci-lint
```

---

## Plan Introspection

The `Plan` represents the execution plan created from a Config.

```go
type Plan struct {
    // Internal: tree, taskInstances, pathMappings, moduleDirectories, shimConfig
}

type TaskInfo struct {
    Name   string         `json:"name"`            // Effective name (e.g., "py-test:3.9")
    Usage  string         `json:"usage,omitempty"` // Description/help text
    Paths  []string       `json:"paths"`           // Directories this task runs in
    Flags  map[string]any `json:"flags,omitempty"` // Flag overrides from pk.WithFlag()
    Hidden bool           `json:"hidden"`          // Whether task is hidden from help
    Manual bool           `json:"manual"`          // Whether task is manual-only
}
```

| Function/Method   | Description                                     |
| :---------------- | :---------------------------------------------- |
| `NewPlan`         | Create plan from Config (walks filesystem once) |
| `Plan.Tasks`      | Returns `[]TaskInfo` with effective names       |
| `Plan.ShimConfig` | Returns resolved `*ShimConfig`                  |

```go
plan := pk.PlanFromContext(ctx)
for _, info := range plan.Tasks() {
    fmt.Printf("Task: %s - %s (paths: %v)\n", info.Name, info.Usage, info.Paths)
}
```

Task names in `TaskInfo` include any suffix from `WithNameSuffix`. For example,
a task named `py-test` wrapped with `pk.WithNameSuffix("3.9")` will have
`Name: "py-test:3.9"`.

---

## Errors

Sentinel errors for error handling:

| Error                   | Description                                         |
| :---------------------- | :-------------------------------------------------- |
| `ErrGitDiffUncommitted` | Returned when `-g` flag detects uncommitted changes |

```go
if errors.Is(err, pk.ErrGitDiffUncommitted) {
    // Handle uncommitted changes
}
```

---

## CLI

### Flags

| Flag        | Description                        |
| :---------- | :--------------------------------- |
| `-g`        | Run git diff check after execution |
| `-h`        | Show help                          |
| `-v`        | Verbose mode                       |
| `--version` | Show version                       |

### Functions

| Function      | Description                                          |
| :------------ | :--------------------------------------------------- |
| `RunMain`     | Main entry point; handles args, help, task execution |
| `ExecuteTask` | Execute a single task by name with plan context      |

```go
// In .pocket/main.go
func main() {
    pk.RunMain(Config)
}

// ExecuteTask signature
func ExecuteTask(ctx context.Context, name string, p *Plan) error
```
