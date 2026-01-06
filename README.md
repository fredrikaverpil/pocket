# bld

> [!WARNING]
>
> Under heavy development. Breaking changes will occur.

A cross-platform build system for Go projects, powered by
[goyek](https://github.com/goyek/goyek).

## Features

- **Cross-platform**: No Makefiles - works on Windows, macOS, and Linux
- **Tool management**: Downloads and caches tools in `.bld/`
- **CI workflow generation**: Generates GitHub Actions workflows from templates
- **Simple invocation**: Just `go run ./.bld <task>`

## Quick Start

### Bootstrap a new project

Run the init command in your project root (must have a `go.mod`):

```bash
go run github.com/fredrikaverpil/bld/cmd/bld@latest init
```

This creates:

- `.bld/` - build module with config and tasks
- `./bld` - wrapper script

### Run tasks

```bash
./bld            # run all tasks (lint, format, test)
./bld go-fmt     # format Go code (gofumpt, goimports, gci, golines)
./bld go-lint    # run golangci-lint
./bld go-test    # run tests
./bld update     # generate CI workflows
```

### Shell alias (optional)

For even shorter commands, add an alias to your shell profile:

```bash
# ~/.bashrc or ~/.zshrc
alias bld='./bld'
```

Then run tasks with just `bld go-fmt`.

## Configuration

```go
bld.Config{
    // Go configuration (nil = no Go tasks)
    Go: &bld.GoConfig{
        Modules: map[string]bld.GoModuleOptions{
            ".":          {},                         // all tasks enabled
            "subdir/lib": {SkipFormat: true},         // skip format for this module
            "generated":  {SkipLint: true},           // skip lint for generated code
        },
    },

    // GitHub Actions configuration (nil = no GitHub workflows)
    GitHub: &bld.GitHubConfig{
        // Go versions are automatically extracted from each module's go.mod.
        // Add extra versions to test against (e.g., "stable", "oldstable", pre-releases).
        ExtraGoVersions: []string{"stable"},
        OSVersions:      []string{"ubuntu-latest"},  // default

        // Skip generic workflows
        SkipPR:      false,  // semantic PR validation
        SkipStale:   false,  // stale issue/PR management
        SkipRelease: false,  // release-please
        SkipSync:    false,  // auto-sync bld updates
    },
}
```

Task skips in `GoModuleOptions` affect both local execution and CI:

- If all modules skip format → no format job in CI workflow
- `go-fmt` task only runs on modules where `SkipFormat: false`

## Generated Workflows

Running `go run ./.bld update` generates:

- `bld-go.yml` - Go CI (format check, tests)
- `bld-pr.yml` - Semantic PR title validation
- `bld-stale.yml` - Stale issue/PR management
- `bld-release.yml` - Release-please automation
- `bld-sync.yml` - Monthly bld update PRs

## Project Structure

```
your-project/
├── .bld/
│   ├── main.go      # task definitions
│   ├── config.go    # project config
│   └── go.mod
├── .github/workflows/
│   └── bld-*.yml    # generated
└── ...
```

## Terminology

### Tools

- Binaries we download in `.bld/tools/` and install to `.bld/bin/`
- Examples: golangci-lint, buf, mdformat, uv, stylua
- Have versions, download URLs, Renovate comments
- Expose a (hidden) `Prepare` task and helper functions

### Tasks (goyek tasks)

- What projects execute: `go-fmt`, `go-lint`, `go-test`
- May use tools: `go-lint` → uses golangci-lint tool
- May use system binaries: `go-fmt` → uses system go
- Defined in `tasks/`

## Adding Custom Tasks

Add tasks directly in `.bld/main.go`. Custom tasks are preserved when running
`./bld update`.

```go
// .bld/main.go
package main

import (
    "os/exec"

    "github.com/fredrikaverpil/bld"
    "github.com/fredrikaverpil/bld/tasks/golang"
    "github.com/fredrikaverpil/bld/workflows"
    "github.com/goyek/goyek/v3"
    "github.com/goyek/x/boot"
)

var tasks = golang.NewTasks(Config)

// Custom task example
var generate = goyek.Define(goyek.Task{
    Name:  "generate",
    Usage: "run go generate",
    Action: func(a *goyek.A) {
        cmd := exec.CommandContext(a.Context(), "go", "generate", "./...")
        cmd.Dir = bld.FromGitRoot()
        if err := cmd.Run(); err != nil {
            a.Fatalf("go generate: %v", err)
        }
    },
})

var update = goyek.Define(goyek.Task{
    Name:  "update",
    Usage: "update bld and generate CI workflows",
    Action: func(a *goyek.A) {
        // ... existing update task
    },
})

func main() {
    goyek.SetDefault(tasks.All)
    boot.Main()
}
```

Run custom tasks with `./bld generate`.

## Adding Custom Tools

Create a tool in your `.bld/tools/` directory:

```go
// .bld/tools/mytool/tool.go
package mytool

import (
    "context"
    "github.com/fredrikaverpil/bld"
    "github.com/fredrikaverpil/bld/tool"
    "github.com/goyek/goyek/v3"
)

const name = "mytool"

// renovate: datasource=github-releases depName=owner/mytool
const version = "1.0.0"

var Prepare = goyek.Define(goyek.Task{
    Name: "mytool:prepare",
    Action: func(a *goyek.A) {
        binDir := bld.FromToolsDir(name, version, "bin")
        binary := binDir + "/" + name

        tool.FromRemote(
            a.Context(),
            "https://github.com/owner/mytool/releases/...",
            tool.WithDestinationDir(binDir),
            tool.WithUntarGz(),
            tool.WithExtractFiles(name),
            tool.WithSkipIfFileExists(binary),
            tool.WithSymlink(binary),
        )
    },
})

func Run(ctx context.Context, args ...string) error {
    return bld.Command(ctx, bld.FromBinDir(name), args...).Run()
}
```

## Windows

The `./bld` wrapper script requires a bash-compatible shell. On Windows, use one
of:

- Git Bash (included with Git for Windows)
- WSL (Windows Subsystem for Linux)
- GitHub Actions (uses bash by default)

Alternatively, create a `bld.cmd` wrapper:

```batch
@echo off
go run -C .bld . %*
```

## License

MIT
