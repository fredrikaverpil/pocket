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

Run the init command in your project root (must have a `go.mod`):

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

```go
pocket.Config{
    // Go configuration (nil = no Go tasks)
    Go: &pocket.GoConfig{
        Modules: map[string]pocket.GoModuleOptions{
            ".":          {},                         // all tasks enabled
            "subdir/lib": {SkipFormat: true},         // skip format for this module
            "generated":  {SkipLint: true},           // skip lint for generated code
        },
    },
}
```

Task skips in `GoModuleOptions` control which tasks run on each module:

- `go-fmt` task only runs on modules where `SkipFormat: false`

### Project Structure

```
your-project/
├── .pocket/
│   ├── main.go      # generated (do not edit)
│   ├── config.go    # project config (edit this)
│   └── go.mod
├── pok              # wrapper script (platform-specific)
└── ...
```

### Custom Tasks

Add your own tasks in `.pocket/config.go`:

```go
import (
    "github.com/fredrikaverpil/pocket"
    "github.com/goyek/goyek/v3"
)

var Config = pocket.Config{
    Go: &pocket.GoConfig{...},

    // Custom tasks per folder
    Custom: map[string][]goyek.Task{
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
Custom: map[string][]goyek.Task{
    ".":            {rootTask},
    "services/api": {apiTask},  // only in ./services/api/pok
}
```

See [goyek documentation](https://github.com/goyek/goyek) for more task options
like dependencies, parallel execution, and error handling.

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

**Alternative approaches:**

- Git Bash (included with Git for Windows) - use `./pok` directly
- WSL (Windows Subsystem for Linux) - use `./pok` directly

## Adding a New Ecosystem

To add support for a new language/ecosystem (e.g., Python, Lua):

1. **Create tools** in `tools/<toolname>/tool.go`
   - Export `Prepare(ctx) error`, `Command(ctx, args) (*Cmd, error)`,
     `Run(ctx, args) error`
   - Both `Command()` and `Run()` auto-prepare the tool
   - Add Renovate comment for version updates
1. **Create tasks** in `tasks/<ecosystem>/tasks.go`
   - Define goyek tasks that use the tools
   - Use `tool.Command()` when you need to customize the command (e.g., set
     `cmd.Dir`), or `tool.Run()` for simple execution
1. **Add tool tests** in `tools/tools_test.go`
   - Add one line to the `tools` table
1. **Wire up in config** - add config options in `config.go` if needed

## Terminology

### Tools

- Binaries downloaded to `.pocket/tools/` and symlinked to `.pocket/bin/`
- Examples: golangci-lint, govulncheck, mdformat, uv
- Have versions, download URLs, Renovate comments
- Expose `Prepare()`, `Command()`, `Run()` functions

### Tasks (goyek tasks)

- What users execute: `go-fmt`, `go-lint`, `go-test`
- Use tools via their Go API
- Defined in `tasks/`

## Acknowledgements

- [goyek](https://github.com/goyek/goyek) - Powers the task system
- [einride/sage](https://github.com/einride/sage) - Inspiration for the tool
  management approach
