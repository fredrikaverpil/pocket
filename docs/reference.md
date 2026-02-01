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

## Composition

The `Runnable` interface is the core abstraction for executable units:

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
| `WithOptions` | Apply path filtering and execution options to a runnable               |

```go
pk.Serial(Format, Lint, Test)
pk.Parallel(Lint, Test, Build)
pk.WithOptions(task, pk.WithIncludePath("services"))
```

---

## Path Options

Options passed to `WithOptions` to control where and how tasks execute.

### Generic Options (pk.With\*)

These options work with any task:

| Option            | Description                                                |
| :---------------- | :--------------------------------------------------------- |
| `WithIncludePath` | Run only in directories matching the regex patterns        |
| `WithExcludePath` | Skip directories matching the regex patterns               |
| `WithDetect`      | Dynamically discover paths using a detection function      |
| `WithName`        | Add suffix to task names (e.g., `py-test` → `py-test:3.9`) |
| `WithForceRun`    | Bypass task deduplication for the wrapped runnable         |
| `WithFlag`        | Set a default flag value for a task in scope               |
| `WithSkipTask`    | Skip specified tasks within this scope                     |
| `WithExcludeTask` | Exclude a task from directories matching patterns          |

```go
pk.WithOptions(
    pk.Parallel(Lint, Test),
    pk.WithIncludePath("services/.*"),
    pk.WithExcludePath("vendor"),
    pk.WithFlag(Test, "race", true),
)
```

### Task-Specific Options

Task configuration is done explicitly via `pk.WithFlag()` and `pk.WithName()`:

```go
pk.WithOptions(
    python.Tasks(),
    pk.WithName("3.9"),                          // Add ":3.9" suffix to task names
    pk.WithFlag(python.Format, "python", "3.9"), // Set Python version
    pk.WithFlag(python.Lint, "python", "3.9"),
    pk.WithFlag(python.Test, "python", "3.9"),
    pk.WithFlag(python.Test, "coverage", true),  // Enable coverage flag
    pk.WithDetect(python.Detect()),
)
```

### Creating Custom Options

Use `CombineOptions`, `WithFlag`, and `WithContextValue` when building task
packages:

| Function           | Description                                      |
| :----------------- | :----------------------------------------------- |
| `CombineOptions`   | Combine multiple PathOptions into one            |
| `WithFlag`         | Set a task flag value (preferred for CLI flags)  |
| `WithContextValue` | Add key-value pair to context (for task authors) |

```go
// Simple example: enable a feature via flag
func EnableFeature() pk.PathOption {
    return pk.WithFlag(MyTask, "feature", true)
}

// Advanced example: combine multiple effects
func WithCustomConfig(value string) pk.PathOption {
    return pk.CombineOptions(
        pk.WithContextValue(configKey{}, value), // Runtime config
        pk.WithName(value),                      // Name suffix
    )
}
```

---

## Detection

Detection functions dynamically discover directories based on marker files.

```go
type DetectFunc func(dirs []string, gitRoot string) []string
```

| Function       | Description                                            |
| :------------- | :----------------------------------------------------- |
| `DetectByFile` | Find directories containing any of the specified files |

```go
pk.WithDetect(pk.DetectByFile("go.mod", "package.json"))
```

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
| `Manual`         | Only run when explicitly invoked by name               |
| `IsManual`       | Returns whether the task is manual-only                |
| `HideHeader`     | Suppress the `:: taskname` header (for machine output) |
| `IsHeaderHidden` | Returns whether the task header is hidden              |
| `Global`         | Deduplicate by name only (ignore path context)         |
| `IsGlobal`       | Returns whether the task deduplicates globally         |
| `Name`           | Returns the task name                                  |
| `Usage`          | Returns the task usage description                     |
| `Flags`          | Returns the task's `*flag.FlagSet`                     |

```go
var Internal = pk.NewTask("internal", "...", nil, body).Hidden()
var Deploy = pk.NewTask("deploy", "...", nil, body).Manual()
var Matrix = pk.NewTask("matrix", "...", nil, body).HideHeader()
var Install = pk.NewTask("install:tool", "...", nil, body).Hidden().Global()
```

### Task Identity and Deduplication

Tasks are deduplicated during execution to prevent running the same work twice.
The deduplication key depends on task type:

| Task Type | Deduplication Key    | Use Case                       |
| :-------- | :------------------- | :----------------------------- |
| Regular   | `effectiveName@path` | Most tasks - run once per path |
| Global    | `baseName@.`         | Install tasks - run once total |

**Effective name** = base name + optional suffix from `WithName`:

- Base name: `py-test` (defined in `NewTask`)
- Effective name: `py-test:3.9` (with `pk.WithName("3.9")`)

This enables multi-version testing where `py-test:3.9` and `py-test:3.10` run
separately, while `install:uv` (a global task) runs only once regardless of
which Python version triggered it.

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

### Cargo (Rust) Tools

```go
func InstallCargo(name string, opts ...CargoOption) Runnable
```

Installs a Rust crate to `.pocket/tools/cargo/<name>/<version>/` and symlinks to
`.pocket/bin/`. Uses **Symlink pattern**.

| Option             | Description                                  |
| :----------------- | :------------------------------------------- |
| `WithCargoVersion` | Specify version for crates.io installs       |
| `WithCargoGit`     | Specify git repository URL                   |
| `WithCargoGitTag`  | Specify git tag/branch/commit for git builds |

```go
// From crates.io
pk.InstallCargo("ripgrep", pk.WithCargoVersion("14.1.0"))

// From git repository
pk.InstallCargo("ts_query_ls",
    pk.WithCargoGit("https://github.com/ribru17/ts_query_ls"),
)

// From git with specific tag
pk.InstallCargo("tool",
    pk.WithCargoGit("https://github.com/org/tool"),
    pk.WithCargoGitTag("v1.0.0"),
)
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

### Download

```go
func Download(url string, opts ...DownloadOpt) Runnable
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
pk.Download(
    "https://example.com/tool-v1.0.0-linux-amd64.tar.gz",
    pk.WithDestDir(pk.FromToolsDir("tool", "v1.0.0")),
    pk.WithFormat("tar.gz"),
    pk.WithExtract(pk.WithExtractFile("tool")),
    pk.WithSymlink(),
    pk.WithSkipIfExists(pk.FromToolsDir("tool", "v1.0.0", "tool")),
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
linkPath, err := pk.CreateSymlinkAs("/path/to/binary", "custom-name")

// CreateSymlinkWithCompanions copies companion files (useful on Windows)
linkPath, err := pk.CreateSymlinkWithCompanions("/path/to/binary", "*.dll")
```

---

## Platform Helpers

### Runtime Detection

| Function               | Description                                  |
| :--------------------- | :------------------------------------------- |
| `HostOS`               | Current OS: `darwin`, `linux`, `windows`     |
| `HostArch`             | Current architecture: `amd64`, `arm64`       |
| `BinaryName`           | Append `.exe` on Windows                     |
| `DefaultArchiveFormat` | Returns `zip` on Windows, `tar.gz` otherwise |

### Architecture Conversion

| Function      | Conversion                              |
| :------------ | :-------------------------------------- |
| `ArchToX8664` | `amd64` → `x86_64`, `arm64` → `aarch64` |
| `ArchToX64`   | `amd64` → `x64`                         |
| `OSToTitle`   | `darwin` → `Darwin`                     |

### Constants

```go
// OS
const Darwin, Linux, Windows = "darwin", "linux", "windows"

// Architecture (Go-style)
const AMD64, ARM64 = "amd64", "arm64"

// Architecture (alternative naming)
const X8664, AARCH64, X64 = "x86_64", "aarch64", "x64"
```

---

## Context

### Accessors

| Function            | Description                                 |
| :------------------ | :------------------------------------------ |
| `PathFromContext`   | Current execution path relative to git root |
| `PlanFromContext`   | The `*Plan` from context (nil if not set)   |
| `Verbose`           | Whether `-v` flag was provided              |
| `OutputFromContext` | The `Output` struct for writing             |

### Setters (Advanced)

| Function      | Description                   |
| :------------ | :---------------------------- |
| `WithPath`    | Set execution path in context |
| `WithPlan`    | Set Plan in context           |
| `WithVerbose` | Set verbose mode in context   |
| `WithOutput`  | Set Output in context         |

### Environment Variables

| Function     | Description                                      |
| :----------- | :----------------------------------------------- |
| `WithEnv`    | Set an environment variable for `Exec` calls     |
| `WithoutEnv` | Filter out environment variables matching prefix |

```go
// Set an environment variable
ctx = pk.WithEnv(ctx, "MY_VAR=value")

// Remove environment variables matching prefix
ctx = pk.WithoutEnv(ctx, "VIRTUAL_ENV")

// Use with Exec
pk.Exec(ctx, "mycmd", "arg1") // runs with modified environment
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

Task names in `TaskInfo` include any suffix from `WithName`. For example, a task
named `py-test` wrapped with `pk.WithName("3.9")` will have
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
