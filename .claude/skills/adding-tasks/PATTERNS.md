# Task patterns — complete examples

## Simple task with tool installation

The most common pattern: install a tool, then run it.

**Example: go-vulncheck** (`tasks/golang/vulncheck.go`)

```go
var Vulncheck = &pk.Task{
    Name:  "go-vulncheck",
    Usage: "run govulncheck",
    Body:  pk.Serial(govulncheck.Install, vulncheckCmd()),
}

func vulncheckCmd() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        args := []string{}
        if pk.Verbose(ctx) {
            args = append(args, "-show", "verbose")
        }
        args = append(args, "./...")
        return pk.Exec(ctx, govulncheck.Name, args...)
    })
}
```

**Key points:**
- `pk.Serial(tool.Install, cmd())` chains install then execute
- `pk.Verbose(ctx)` maps to the tool's own verbose flag
- `pk.Exec(ctx, name, args...)` runs the tool with `.pocket/bin` on PATH

---

## Task with flags

Flags let users and config authors customize task behavior.

**Example: go-lint** (`tasks/golang/lint.go`)

```go
const (
    FlagLintConfig = "config"
    FlagLintFix    = "fix"
)

var Lint = &pk.Task{
    Name:  "go-lint",
    Usage: "run golangci-lint",
    Flags: map[string]pk.FlagDef{
        FlagLintConfig: {Default: "", Usage: "path to golangci-lint config file"},
        FlagLintFix:    {Default: true, Usage: "apply fixes"},
    },
    Body: pk.Serial(golangcilint.Install, lintCmd()),
}

func lintCmd() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        args := []string{"run"}
        if pk.Verbose(ctx) {
            args = append(args, "-v")
        }
        if config := pk.GetFlag[string](ctx, FlagLintConfig); config != "" {
            args = append(args, "-c", config)
        }
        if pk.GetFlag[bool](ctx, FlagLintFix) {
            args = append(args, "--fix")
        }
        args = append(args, "./...")
        return pk.Exec(ctx, golangcilint.Name, args...)
    })
}
```

**Flag types:**
- `pk.GetFlag[bool](ctx, FlagLintFix)` — boolean
- `pk.GetFlag[string](ctx, FlagLintConfig)` — string

---

## Task with Do (inline function)

Use `Do` instead of `Body` when the task is a single function that doesn't
compose other runnables.

**Example: go-fix** (`tasks/golang/fix.go`)

```go
var Fix = &pk.Task{
    Name:  "go-fix",
    Usage: "update code for newer Go versions",
    Body: pk.Do(func(ctx context.Context) error {
        args := []string{"fix"}
        if pk.Verbose(ctx) {
            args = append(args, "-v")
        }
        args = append(args, "./...")
        return pk.Exec(ctx, "go", args...)
    }),
}
```

---

## Task using tool.Exec (Python/Node tools)

Python and Node tools don't symlink into `.pocket/bin/`. Instead, they expose
an `Exec()` function. Use that instead of `pk.Exec`.

**Example: md-format** (`tasks/markdown/format.go`)

```go
const (
    FlagCheck  = "check"
    FlagConfig = "config"
)

var Format = &pk.Task{
    Name:  "md-format",
    Usage: "format Markdown files",
    Flags: map[string]pk.FlagDef{
        FlagCheck:  {Default: false, Usage: "check only, don't write"},
        FlagConfig: {Default: "", Usage: "path to prettier config file"},
    },
    Body: pk.Serial(prettier.Install, formatCmd()),
}

func formatCmd() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        configPath := pk.GetFlag[string](ctx, FlagConfig)
        if configPath == "" {
            configPath = prettier.EnsureDefaultConfig()
        }

        args := []string{}
        if pk.GetFlag[bool](ctx, FlagCheck) {
            args = append(args, "--check")
        } else {
            args = append(args, "--write")
        }
        args = append(args, "--config", configPath)

        if ignorePath, err := prettier.EnsureIgnoreFile(); err == nil {
            args = append(args, "--ignore-path", ignorePath)
        }

        pattern := pk.FromGitRoot("**/*.md")
        args = append(args, pattern)

        return prettier.Exec(ctx, args...)
    })
}
```

**Key difference:** `prettier.Exec(ctx, args...)` instead of `pk.Exec(ctx, ...)`.

---

## Python tasks — flag-passing pattern

When a detail like Python version cannot be abstracted away in the task, expose
it as a flag and pass it through to the underlying tool.

**Example: py-lint** (`tasks/python/lint.go`)

```go
// FlagPython is shared across Lint, Format, Test, and Typecheck tasks.
const FlagPython = "python"

// FlagLintSkipFix is specific to the Lint task.
const FlagLintSkipFix = "skip-fix"

var Lint = &pk.Task{
    Name:  "py-lint",
    Usage: "lint Python files",
    Flags: map[string]pk.FlagDef{
        FlagPython:      {Default: "", Usage: "Python version (for target-version inference)"},
        FlagLintSkipFix: {Default: false, Usage: "don't auto-fix issues"},
    },
    Body: pk.Serial(uv.Install, lintSyncCmd(), lintCmd()),
}

func lintSyncCmd() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
        return uv.Sync(ctx, uv.SyncOptions{
            PythonVersion: version,
            AllGroups:     true,
        })
    })
}

func lintCmd() pk.Runnable {
    return pk.Do(func(ctx context.Context) error {
        version := resolveVersion(ctx, pk.GetFlag[string](ctx, FlagPython))
        return runLint(ctx, version, pk.GetFlag[bool](ctx, FlagLintSkipFix))
    })
}

func runLint(ctx context.Context, pythonVersion string, skipFix bool) error {
    args := []string{"check", "--exclude", ".pocket"}
    if pk.Verbose(ctx) {
        args = append(args, "--verbose")
    }
    if !skipFix {
        args = append(args, "--fix")
    }
    if pythonVersion != "" {
        args = append(args, "--target-version", pythonVersionToRuff(pythonVersion))
    }
    args = append(args, pk.PathFromContext(ctx))
    return uv.Run(ctx, uv.RunOptions{PythonVersion: pythonVersion}, "ruff", args...)
}
```

**Multi-stage pattern:**
1. Sync dependencies (`uv.Sync`)
2. Run tool (`uv.Run`)
3. Implementation function handles args and verbose

---

## Package-level Tasks() factory

Each task package exposes a `Tasks()` function that composes its tasks into a
single runnable for easy wiring into config.

**Example: golang** (`tasks/golang/tasks.go`)

```go
func Tasks(tasks ...pk.Runnable) pk.Runnable {
    if len(tasks) == 0 {
        return pk.Serial(
            Fix,
            Format,
            Lint,
            pk.Parallel(Test, Vulncheck),
        )
    }
    return pk.Serial(tasks...)
}
```

**Example: python** (`tasks/python/tasks.go`)

```go
func Tasks() pk.Runnable {
    return pk.Serial(uv.Install, Format, Lint, pk.Parallel(Typecheck, Test))
}
```

Use `pk.Serial` for tasks that must run in order, `pk.Parallel` for independent
tasks that can run concurrently.

---

## Detect function

When tasks should only run in directories matching certain files, provide a
`Detect()` function:

```go
func Detect() pk.DetectFunc {
    return pk.DetectByFile("go.mod")
}
```

Used in config with `pk.WithDetect`:

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithDetect(golang.Detect()),
)
```

For tasks that always run from the repo root (e.g. markdown formatting):

```go
func Detect() pk.DetectFunc {
    return func(_ []string, _ string) []string {
        return []string{"."}
    }
}
```

---

## Wiring tasks into config

### Direct reference

```go
Auto: pk.Parallel(
    markdown.Format,
),
```

### Factory function

```go
Auto: pk.Parallel(
    golang.Tasks(),
),
```

### With options (flags, detection, context values)

```go
Auto: pk.Parallel(
    pk.WithOptions(
        python.Tasks(),
        pk.WithDetect(python.Detect()),
        pk.WithFlag(python.Lint, python.FlagPython, "3.12"),
        pk.WithFlag(python.Test, python.FlagTestCoverage, true),
    ),
),
```

### Manual tasks

Tasks in `Manual` are only run when explicitly invoked via `./pok <task>`:

```go
Manual: []pk.Runnable{
    Hello,
},
```

---

## Project-managed dependencies

When a project controls its own tool versions via `pyproject.toml`/`uv.lock` or
`package.json`/`bun.lock`, tasks use the runtime's Sync/Run functions directly.
No tool package under `tools/` is needed.

### Python (uv.Sync + uv.Run)

When `ProjectDir` is left empty, `uv.Sync` and `uv.Run` default to
`PathFromContext(ctx)` — the project's own directory containing
`pyproject.toml`.

```go
var Docs = &pk.Task{
    Name:  "docs",
    Usage: "build documentation",
    Body: pk.Serial(
        uv.Install,
        pk.Do(func(ctx context.Context) error {
            // Sync dependencies from the project's pyproject.toml.
            if err := uv.Sync(ctx, uv.SyncOptions{
                AllGroups: true,
            }); err != nil {
                return err
            }

            // Run the tool from the synced environment.
            return uv.Run(ctx, uv.RunOptions{}, "zensical", "build")
        }),
    ),
}
```

The existing `tasks/python/` package follows this pattern — `py-lint`,
`py-format`, `py-test`, `py-typecheck` all use `uv.Sync` + `uv.Run` against
the project's own `pyproject.toml`.

### Node (bun.InstallFromLockfile + bun.Run)

For bun, pass the project directory explicitly:

```go
var Build = &pk.Task{
    Name:  "build",
    Usage: "build frontend",
    Body: pk.Serial(
        bun.Install,
        pk.Do(func(ctx context.Context) error {
            projectDir := pk.FromGitRoot(pk.PathFromContext(ctx))

            if err := bun.InstallFromLockfile(ctx, projectDir); err != nil {
                return err
            }

            return bun.Run(ctx, projectDir, "vite", "build")
        }),
    ),
}
```

### Standalone vs project-managed

| Aspect              | Standalone                     | Project-managed                    |
|---------------------|--------------------------------|------------------------------------|
| Version control     | Pocket (embedded in tool pkg)  | Project's lockfile                 |
| Tool package needed | Yes (`tools/<name>/`)          | No — use `uv`/`bun` directly      |
| Installation        | `tool.Install` task            | `uv.Sync` / `bun.InstallFromLockfile` |
| Invocation          | `tool.Exec(ctx, ...)`         | `uv.Run(ctx, ...)` / `bun.Run(ctx, ...)` |

Some tools (like zensical) can be used either way. Use **standalone** when
Pocket should control the version. Use **project-managed** when the project
needs to pin its own version.

---

## Cross-platform path handling

Use Pocket helpers instead of hardcoded paths:

```go
// Absolute path from git root
pattern := pk.FromGitRoot("**/*.md")
absDir := pk.FromGitRoot(pk.PathFromContext(ctx))

// Context-relative path (set by detection/path filtering)
dir := pk.PathFromContext(ctx)
```

Avoid shell-specific constructs (`&&`, pipes, backticks). Use Go code for any
logic that would otherwise need shell features.

---

## Verbose handling reference

Map `pk.Verbose(ctx)` to the underlying tool's verbose flag. Each tool uses
different flags:

| Tool           | Verbose flag              |
|----------------|---------------------------|
| go             | `-v`                      |
| golangci-lint  | `-v`                      |
| govulncheck    | `-show verbose`           |
| ruff           | `--verbose`               |
| pytest         | `-vv`                     |
| mypy           | `-v`                      |
| stylua         | `--verbose`               |

When no tool flag exists, use `pk.Printf` for informational output:

```go
if pk.Verbose(ctx) {
    pk.Printf(ctx, "  no query directories found\n")
}
```
