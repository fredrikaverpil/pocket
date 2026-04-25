# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with
code in this repository.

## What is Pocket

Pocket is a Go-based composable task runner framework. It replaces Makefiles and
shell scripts with type-safe Go configuration that handles task composition,
tool installation, and cross-platform execution. The `./pok` shim bootstraps
everything—no pre-installed dependencies beyond a shell.

## Commands

```bash
./pok                    # run all auto tasks (lint, test, format, etc.)
./pok -v                 # verbose mode (streams output in real-time)
./pok go-test            # run a specific task
./pok go-test -race      # run a specific task with flags
./pok go-lint            # run golangci-lint
./pok plan               # show execution plan without running
./pok -h                 # list all available tasks
```

Run Go tests directly (useful during development):

```bash
go test ./pk/...                      # test the core engine
go test ./pk/download/...             # test download/extract
go test ./pk/conventionalcommits/...  # test commit validation
go test -run TestName ./pk/...        # run a single test
```

## Architecture

### Core packages

- **`pk/`** — The public API and engine. Contains `Config`, `Task`, `Runnable`,
  composition (`Serial`, `Parallel`, `WithOptions`), plan building, CLI, output
  buffering, and task execution. This is the main package users import.
- **`pk/run/`** — Runtime helpers for task authors: `Exec` (run external
  commands), `GetFlags` (retrieve typed flags from context), context accessors
  (`PathFromContext`, `Verbose`, `PlanFromContext`), and output functions
  (`Printf`, `Println`, `Errorf`).
- **`pk/download/`** — Download, extract (tar.gz, zip, gz), and symlink API for
  tool installation.
- **`pk/platform/`** — OS/arch detection and naming helpers (`HostOS`,
  `HostArch`, `ArchToX8664`, etc.).
- **`pk/repopath/`** — Git root detection.
- **`pk/conventionalcommits/`** — Commit message validation.

### Tool packages (`tools/`)

Each tool package owns its complete lifecycle: installation, versioning, and
making itself available. Three patterns exist:

1. **Symlink** (native binaries) — download, extract, symlink to `.pocket/bin/`,
   invoke via `run.Exec(ctx, "tool", ...)`
2. **Tool Exec** (runtime-dependent) — package exposes `Exec()` function, no
   symlink
3. **Runtime Run** (project-managed) — use runtime's `Run()` directly (e.g.,
   `uv.Run`)

### Task packages (`tasks/`)

Pre-built opinionated tasks that wrap tools: `golang`, `python`, `markdown`,
`github`, `lua`, `treesitter`, `docs`, `claude`. Each exposes a `Tasks()`
function returning composed runnables, plus individual task variables.

### Execution model

1. User's `.pocket/config.go` defines a `Config` with `Auto` (composition tree)
   and `Manual` tasks
2. Plan builder walks the composition tree, resolves paths (via
   `WithDetect`/`WithPath`), and creates task instances
3. Executor runs the plan: `Serial` = sequential (stop on error), `Parallel` =
   concurrent with buffered output
4. Generated subdirectory shims set `TASK_SCOPE`, causing bare `./pok` and
   direct task execution to run only tasks relevant to that shim path
5. Same task at same path is deduplicated; `Global: true` tasks deduplicate
   across all paths

### Key files in this repo's own config

- `.pocket/config.go` — This project's Pocket configuration (uses
  `golang.Tasks()`, `markdown.Format`, `github.Tasks()`)
- `.pocket/main.go` — Entry point that calls `pk.RunMain(Config)`
- `cmd/pocket/main.go` — The `pocket init` bootstrapper CLI

## Conventions

- The `pk` package re-exports platform helpers from `pk/platform/` for
  convenience (e.g., `pk.HostOS()` delegates to `platform.HostOS()`)
- Install tasks use `Hidden: true` (internal detail) and `Global: true` (run
  once regardless of path)
- Task flags are structs with `flag` and `usage` tags, accessed via
  `run.GetFlags[T](ctx)`
- Pointer flag fields (`*bool`, `*string`) mean "not set" when nil — used with
  `pk.WithFlags` for selective overrides
