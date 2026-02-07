---
name: adding-tools
description: >-
  Guide for adding new tools (binaries, linters, formatters) to Pocket. Covers
  Go packages, GitHub release binaries, Python/uv tools, and Node/bun tools.
  Use when creating a new tool package under tools/ or modifying an existing
  tool's installation, versioning, or cross-platform support.
---

# Adding tools to Pocket

A Pocket "tool" is a reusable binary or package that Pocket downloads, installs,
and makes available. Each tool owns its complete lifecycle: version definition,
installation, and execution.

## Directory structure

```
tools/<toolname>/
├── <toolname>.go        # Package with Name, Version, Install task
├── requirements.txt     # (Python tools, pip-based)
├── pyproject.toml       # (Python tools, uv sync-based)
├── uv.lock              # (Python tools, uv sync-based)
├── package.json         # (Node tools)
└── bun.lock             # (Node tools)
```

## Required exports

Every tool package must export:

- `Name` constant (the binary name)
- `Version` constant or `Version()` function
- `Install` variable (`*pk.Task`, hidden + global)

## Version specification

Use inline constants with Renovate comments for automatic updates:

```go
// renovate: datasource=github-releases depName=owner/repo
const Version = "1.2.3"
```

For tools with lockfiles (Python/Node), compute a hash-based version:

```go
//go:embed requirements.txt
var requirements []byte

func Version() string {
    h := sha256.New()
    h.Write(requirements)
    return hex.EncodeToString(h.Sum(nil))[:12]
}
```

## Install task pattern

```go
var Install = &pk.Task{
    Name:   "install:<tool-name>",
    Usage:  "install <tool-name>",
    Body:   installFunc(),  // or golang.Install(...) for Go tools
    Hidden: true,           // internal, not shown in CLI help
    Global: true,           // run once total, not per-path
}
```

## Installation patterns

Choose the pattern that matches your tool. See [PATTERNS.md](PATTERNS.md) for
complete examples of each pattern.

### Pattern 1: Go package (`go install`)

For tools written in Go. Simplest pattern.

```go
Body: golang.Install("github.com/org/tool/cmd/tool", Version),
```

### Pattern 2: GitHub release binary

For pre-compiled binaries. The `pk/download` package provides helper functions
for downloading, extracting archives, and making the binary available in Pocket
(symlink into `.pocket/bin/`). Use `download.Download` with platform-aware URLs.
Must handle platform/arch mapping (see cross-platform section in PATTERNS.md).

### Pattern 3: Python tool (via uv)

For Python tools. Depends on `uv.Install`. Prefer `pyproject.toml`/`uv.lock`
with `uv.Sync` (reproducible, locked). `requirements.txt` with
`uv.PipInstallRequirements` exists in older tools but should not be used for new
ones. Exposes an `Exec()` function instead of a symlink.

### Pattern 4: Node tool (via bun)

For Node.js tools. Depends on `bun.Install`. Embeds `package.json` and
`bun.lock` via `//go:embed`. Exposes an `Exec()` function instead of a symlink.

## Ecosystem tools (uv, bun)

The `uv` and `bun` packages are themselves tools (with their own `Install`
tasks) but also serve as ecosystems for other tools. When your tool depends on
one of these, chain installation with `pk.Serial`:

```go
Body: pk.Serial(uv.Install, installMyTool()),
// or
Body: pk.Serial(bun.Install, installMyTool()),
```

These ecosystem tools provide helper functions for creating venvs, installing
packages, and running commands. See [PATTERNS.md](PATTERNS.md) for details.

## Cross-platform support

All tools must support Linux, macOS, and Windows. Key considerations:

- Use `pk.HostOS()` and `pk.HostArch()` for platform detection
- Use `pk.BinaryName(name)` to append `.exe` on Windows
- Use `pk.DefaultArchiveFormat()` (`"zip"` on Windows, `"tar.gz"` otherwise)
- Release URL naming varies per project (see PATTERNS.md for mapping examples)
- `download.WithSymlink()` automatically copies instead of symlinking on Windows
- bun on Windows requires extra care (see PATTERNS.md)

## Making the tool available

Two approaches depending on tool type:

**Symlinked binaries** (native/Go tools): Use `download.WithSymlink()` or
`download.CreateSymlink()`. The binary is symlinked into `.pocket/bin/` which
is on PATH during task execution. Invoke with `pk.Exec(ctx, Name, args...)`.

**Exec function** (Python/Node tools): No symlink. Expose a public
`Exec(ctx, args...)` function that invokes the tool through its runtime.

## Wiring the tool into a task

```go
var Lint = &pk.Task{
    Name:  "lint",
    Usage: "run linter",
    Body: pk.Serial(
        mytool.Install,
        pk.Do(func(ctx context.Context) error {
            return pk.Exec(ctx, mytool.Name, "run", "./...")
        }),
    ),
}
```

## Idempotency

Always skip installation if the binary already exists:

```go
// For download-based tools:
download.WithSkipIfExists(binaryPath)

// For custom install functions:
if _, err := os.Stat(binary); err == nil {
    return nil
}
```

## Checklist

1. Create `tools/<name>/<name>.go` with `Name`, `Version`, `Install`
2. Add Renovate comment on version constant/variable
3. Handle all three platforms (Linux, macOS, Windows)
4. Ensure idempotent installation (skip if exists)
5. Set `Hidden: true` and `Global: true` on the Install task
6. Wire into a task via `pk.Serial(tool.Install, ...)`
7. Run `go mod tidy` in `.pocket/`
