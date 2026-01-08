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

Edit `.pocket/config.go` to configure task groups:

```go
import (
    "github.com/fredrikaverpil/pocket"
    "github.com/fredrikaverpil/pocket/tasks/golang"
)

var Config = pocket.Config{
    TaskGroups: []pocket.TaskGroup{
        golang.New(golang.Config{
            Modules: map[string]golang.Options{
                ".":          {},                           // all tasks enabled
                "subdir/lib": {Skip: []string{"format"}},   // skip format for this module
                "generated":  {Only: []string{"test"}},     // only run test for generated code
            },
        }),
    },
}
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

### Tools

- Binaries downloaded to `.pocket/tools/` and symlinked to `.pocket/bin/`
- Examples: golangci-lint, govulncheck, mdformat, uv
- Have versions, download URLs, Renovate comments
- Expose `Prepare()`, `Command()`, `Run()` functions

### Tasks (goyek tasks)

- What users execute: `go-format`, `go-lint`, `lua-format`...
- Use tools via their Go API
- Defined in `tasks/`

### Task Groups

- Group related tasks for a language or purpose (e.g., `golang`, `lua`, `markdown`)
- Each task group owns its own `Config` and `Options` types
- Implement the `pocket.TaskGroup` interface
- Configured in `pocket.Config.TaskGroups`

## Acknowledgements

- [goyek](https://github.com/goyek/goyek) - Powers the task system
- [einride/sage](https://github.com/einride/sage) - Inspiration and code for the
  tool management approach
