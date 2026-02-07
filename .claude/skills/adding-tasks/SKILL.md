---
name: adding-tasks
description: >-
  Guide for adding new tasks to Pocket. Covers task structure, naming, flags,
  verbose handling, cross-platform support, composition, and wiring into config.
  Use when creating or modifying task files under tasks/ or wiring tasks in
  .pocket/config.go.
---

# Adding tasks to Pocket

A Pocket task is a wrapper around a tool that provides opinionated defaults and
settings. Tasks make `./pok` execute operations like formatting, linting, and
testing consistently across projects.

## Task structure

Tasks live under `tasks/<domain>/` and follow this layout:

```
tasks/<domain>/
├── tasks.go         # Package-level Tasks() factory + Detect() if applicable
├── format.go        # One file per task action
├── lint.go
└── test.go
```

## Task definition

```go
var Lint = &pk.Task{
    Name:  "go-lint",
    Usage: "run golangci-lint",
    Flags: map[string]pk.FlagDef{
        "fix": {Default: true, Usage: "apply fixes"},
    },
    Body: pk.Serial(golangcilint.Install, lintCmd()),
}
```

Use `Do` for single inline functions, `Body` for composed runnables
(`pk.Serial`/`pk.Parallel`). See [PATTERNS.md](PATTERNS.md) for details.

## Task naming

Name tasks by domain and action, not by the underlying tool:

```
<domain>-<action>    e.g. go-lint, py-format, md-format
```

This abstraction lets you swap the underlying tool without changing the task
name. For example, `go-lint` uses golangci-lint today but could switch to
another linter without breaking user workflows.

If a detail like Python version cannot be abstracted away, expose it as a flag
instead (see the Python tasks pattern in [PATTERNS.md](PATTERNS.md)).

Naming conventions used in Pocket:

| Domain     | Prefix       | Examples                              |
|------------|--------------|---------------------------------------|
| Go         | `go-`        | `go-lint`, `go-test`, `go-format`     |
| Python     | `py-`        | `py-lint`, `py-test`, `py-typecheck`  |
| Lua        | `lua-`       | `lua-format`                          |
| Markdown   | `md-`        | `md-format`                           |
| GitHub     | `github-`    | `github-workflows`                    |
| Commits    | `commits-`   | `commits-validate`                    |

## Verbose handling

Every task **must** handle `pk.Verbose(ctx)`. This controls whether progress
output is shown when running `./pok -v` or `./pok -v <task>`.

Pass the tool's own verbose flag when available:

```go
if pk.Verbose(ctx) {
    args = append(args, "-v")        // or "--verbose", "-vv", etc.
}
```

When no tool flag exists, use `pk.Printf` for conditional output:

```go
if pk.Verbose(ctx) {
    pk.Printf(ctx, "  skipping, no files found\n")
}
```

## Flags

Define only flags you need (YAGNI). Use `pk.GetFlag[T](ctx, "name")` with
type-safe generics:

```go
Flags: map[string]pk.FlagDef{
    "fix":    {Default: true, Usage: "apply fixes"},
    "config": {Default: "", Usage: "path to config file"},
},
```

Users override flags via CLI: `./pok go-lint -fix=false`

Config authors override defaults: `pk.WithFlag(golang.Lint, "fix", false)`

## Cross-platform

Tasks must work on Linux, macOS, and Windows unless explicitly documented
otherwise. Use `pk.FromGitRoot()` for absolute paths and `pk.PathFromContext()`
for context-relative paths. Avoid shell-specific constructs.

## Wiring into config

Tasks are wired into `.pocket/config.go`:

```go
var Config = &pk.Config{
    Auto: pk.Parallel(
        golang.Tasks(),           // factory function
        markdown.Format,          // direct task reference
    ),
    Manual: []pk.Runnable{Hello}, // only via ./pok hello
}
```

See [PATTERNS.md](PATTERNS.md) for composition and detection patterns.
