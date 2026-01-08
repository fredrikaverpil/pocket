# pocket

An opinonated build system platform, powered by
[goyek](https://github.com/goyek/goyek).

> [!WARNING]
>
> Under heavy development. Breaking changes will occur.

## Features

- **Cross-platform**: No Makefiles - works on Windows, macOS, and Linux
- **Task management**: Defines tasks like `go-test`, `go-lint`...
- **Tool management**: Downloads and caches tools in `.pocket/`, which are used
  by tasks
- **Simple invocation**: Just `./pok <task>` or `./pok -h` to list all tasks

## Bootstrap a new project

Run the init command in your project root:

```bash
go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
```

This creates:

- `.pocket/` - build module with config and tasks
- `./pok` - wrapper script (or `pok.cmd`/`pok.ps1` on Windows)

### Run tasks

```bash
./pok            # run all tasks (generate, lint, format, test)
./pok update     # update pocket to latest version
./pok generate   # regenerate shim
```

Run `./pok -h` for a list of all possible tasks to run.

### Shell alias (optional)

For even shorter commands, add an alias to your shell profile:

```bash
# ~/.bashrc or ~/.zshrc
alias pok='./pok'
```

Then run tasks with just `pok <task>`.

### Configuration

Edit `.pocket/config.go` to configure task groups.

**Auto-detection (recommended):**

```go
import (
    "github.com/fredrikaverpil/pocket"
    "github.com/fredrikaverpil/pocket/tasks/golang"
    "github.com/fredrikaverpil/pocket/tasks/python"
    "github.com/fredrikaverpil/pocket/tasks/markdown"
)

var Config = pocket.Config{
    TaskGroups: []pocket.TaskGroup{
        golang.Auto(),   // auto-detects go.mod files
        python.Auto(),   // auto-detects pyproject.toml, setup.py, setup.cfg
        markdown.Auto(), // formats markdown from root
    },
}
```

**Auto-detection with options:**

```go
golang.Auto(golang.AutoConfig{
    // Default options for all detected modules
    Options: golang.Options{Skip: []string{"vulncheck"}},
    // Override specific paths
    Overrides: map[string]golang.Options{
        ".pocket": {Only: []string{"format"}},
    },
})
```

**Explicit configuration:**

```go
var Config = pocket.Config{
    TaskGroups: []pocket.TaskGroup{
        golang.New(map[string]golang.Options{
            ".":          {},                           // all tasks enabled
            "subdir/lib": {Skip: []string{"format"}},   // skip format for this module
            "generated":  {Only: []string{"test"}},     // only run test for generated code
        }),
    },
}
```

**Task-specific options:**

```go
golang.New(map[string]golang.Options{
    "proj1": {
        Lint: golang.LintOptions{ConfigFile: "proj1/.golangci.yml"},
    },
    "proj2": {
        Skip: []string{"test"},  // skip tests for this module
    },
})
```

### Custom Tasks

Add your own tasks in `.pocket/mytask.go`:

```go
import (
    "github.com/fredrikaverpil/pocket"
    "github.com/goyek/goyek/v3"
)

var Config = pocket.Config{
    TaskGroups: []pocket.TaskGroup{...},

    // Custom tasks per module path
    Tasks: map[string][]goyek.Task{
        ".": {  // available from root ./pok
            {
                Name:  "deploy",
                Usage: "deploy to production",
                Action: func(a *goyek.A) {
                    a.Log("Deploying...")
                    // your logic here
                },
            },
        },
    },
}
```

Custom tasks appear in `./pok -h` and run as part of `./pok all`.

For multi-module projects, you can define context-specific tasks that only
appear when running the shim from that folder:

```go
Tasks: map[string][]goyek.Task{
    ".":            {rootTask},
    "services/api": {apiTask},  // only visible from ./services/api/
}
```

See [goyek documentation](https://github.com/goyek/goyek) for more task options.

### Multi-Module Projects and Context Awareness

Pocket automatically generates shims in each module directory, making tasks
context-aware. When you run `./pok` from a subdirectory, only tasks relevant to
that module are executed.

**Example project structure:**

```
myproject/
├── .pocket/           # build configuration
├── pok                # root shim
├── go.mod             # root Go module
├── services/
│   └── api/
│       ├── pok        # auto-generated shim for this module
│       └── go.mod     # separate Go module
└── libs/
    └── common/
        ├── pok        # auto-generated shim for this module
        └── go.mod     # separate Go module
```

**How it works:**

1. `./pok generate` creates shims in all detected module directories
1. Each shim knows its "context" (relative path from repo root)
1. Running `./pok` from a subdirectory filters tasks to that module only

```bash
# From repo root - runs tasks on ALL modules
./pok go-test

# From services/api/ - runs tasks only on that module
cd services/api
./pok go-test
```

This enables focused workflows in large monorepos while keeping a single
configuration in `.pocket/config.go`.

### Windows Support

When bootstrapping, pocket automatically detects your platform:

- **Unix/macOS/WSL**: Creates `./pok` (Posix bash script)
- **Windows**: Creates `pok.cmd` and `pok.ps1`

To add additional shim types after bootstrapping, update your
`.pocket/config.go`:

```go
var Config = pocket.Config{
    Shim: &pocket.ShimConfig{
        Name:       "pok",  // base name (default: "pok")
        Posix:      true,   // ./pok (bash) - default
        Windows:    true,   // pok.cmd (requires Go in PATH)
        PowerShell: true,   // pok.ps1 (can auto-download Go)
    },
    // ... rest of config
}
```

After updating the config, run `./pok generate` to create the Windows shims.

**Shim types:**

| Shim          | File      | Go Auto-Download | Notes                        |
| ------------- | --------- | ---------------- | ---------------------------- |
| Posix         | `./pok`   | Yes              | Works with bash, Git Bash    |
| Windows (CMD) | `pok.cmd` | No               | Requires Go in PATH          |
| PowerShell    | `pok.ps1` | Yes              | Full-featured Windows option |

**Using the shims on Windows:**

```batch
rem CMD
pok.cmd go-test

rem PowerShell
.\pok.ps1 go-test
```

## Terminology

### Config (`pocket.Config`)

- Project-level configuration
- Defines which task groups to use, custom tasks, and shim settings
- Lives in `.pocket/config.go`

### Task Groups

- Curated collection of related tasks for a language or purpose (e.g.,
  `golang`, `python`, `markdown`)
- Created with `golang.New(map[string]golang.Options{...})` or `golang.Auto()`
- Implement the `pocket.TaskGroup` interface
- Configured in `pocket.Config.TaskGroups`

### Options (`golang.Options`, `python.Options`, etc.)

- Per-directory configuration for a task group
- Controls which tasks run (`Skip`, `Only`)
- Contains task-specific options (`Lint`, `Test`, `Format`, etc.)

### Tasks

- What users execute: `go-format`, `go-lint`, `py-typecheck`...
- Each task runs on one or more directories
- Can be used standalone: `golang.LintTask(map[string]golang.Options{...})`
- Defined in `tasks/`

### Tools (internal)

- Binaries downloaded to `.pocket/tools/` and symlinked to `.pocket/bin/`
- Examples: golangci-lint, govulncheck, mdformat, ruff
- Used internally by tasks

## Acknowledgements

- [goyek](https://github.com/goyek/goyek) - Powers the task system
- [einride/sage](https://github.com/einride/sage) - Inspiration and code for the
  tool management approach
