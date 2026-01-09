# pocket

A cross-platform build system. Write tasks, control execution order, and let
pocket handle tool installation.

> [!WARNING]
>
> Under heavy development. Breaking changes will occur until the initial
> release.

## Quickstart

### Bootstrap

Run in your project root (requires Go):

```bash
go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
```

This creates `.pocket/` (build module) and `./pok` (wrapper script).

### Your first task

Edit `.pocket/config.go`:

```go
package main

import (
    "context"
    "fmt"

    "github.com/fredrikaverpil/pocket"
)

var Config = pocket.Config{
    Run: helloTask,
}

var helloTask = &pocket.Task{
    Name:  "hello",
    Usage: "say hello",
    Action: func(ctx context.Context, opts *pocket.RunContext) error {
        fmt.Println("Hello from pocket!")
        return nil
    },
}
```

```bash
./pok -h      # list tasks
./pok hello   # run specific task
./pok         # run all tasks
```

### Execution order

Use `Serial()` and `Parallel()` to control how tasks run:

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        formatTask,          // first
        pocket.Parallel(     // then these in parallel
            lintTask,
            testTask,
        ),
        buildTask,           // last
    ),
}
```

### Tool management

Pocket can download and cache tools in `.pocket/tools/`, so you don't need to
rely on CI to install them. Here's a task that uses golangci-lint:

```go
import "github.com/fredrikaverpil/pocket/tools/golangcilint"

var lintTask = &pocket.Task{
    Name:  "lint",
    Usage: "run linter",
    Deps:  pocket.Deps(golangcilint.Prepare),  // download tool first
    Action: func(ctx context.Context, opts *pocket.RunContext) error {
        return golangcilint.Run(ctx, "run", "./...")
    },
}
```

The first run downloads the tool; subsequent runs use the cached version.

## Configuration

### Tasks with options

For reusable tasks, create functions that accept options:

```go
type LintOptions struct {
    ConfigFile string
    Fix        bool
}

func LintTask(opts LintOptions) *pocket.Task {
    return &pocket.Task{
        Name:  "lint",
        Usage: "run linter",
        Action: func(ctx context.Context, _ *pocket.RunContext) error {
            args := []string{"run"}
            if opts.ConfigFile != "" {
                args = append(args, "-c", opts.ConfigFile)
            }
            if opts.Fix {
                args = append(args, "--fix")
            }
            args = append(args, "./...")
            return pocket.Command(ctx, "golangci-lint", args...).Run()
        },
    }
}

var Config = pocket.Config{
    Run: LintTask(LintOptions{ConfigFile: ".golangci.yml", Fix: true}),
}
```

### Tasks with arguments

Tasks can accept runtime arguments via `key=value` syntax:

```go
var deployTask = &pocket.Task{
    Name:  "deploy",
    Usage: "deploy to environment",
    Args: []pocket.ArgDef{
        {Name: "env", Usage: "target environment", Default: "staging"},
    },
    Action: func(ctx context.Context, opts *pocket.RunContext) error {
        fmt.Printf("Deploying to %s...\n", opts.Args["env"])
        return nil
    },
}
```

```bash
./pok deploy              # uses default: staging
./pok deploy env=prod     # override: prod
```

### Path filtering

For monorepos, use `Paths()` to control where `pok` shims are generated:

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        rootTask,                                  // visible at git root only
        pocket.Paths(apiTask).In("services/api"),  // visible in services/api/
        pocket.Paths(webTask).In("services/web"),  // visible in services/web/
    ),
}
```

```go
// Multiple directories
pocket.Paths(myTask).In("proj1", "proj2")

// Regex patterns
pocket.Paths(myTask).In("services/.*")

// Exclude directories
pocket.Paths(myTask).In("services/.*").Except("services/legacy")
```

Running `./pok` from a subdirectory shows only tasks relevant to that directory.

You can also run multiple tasks in the same path:

```go
pocket.Paths(myTask1, myTask2, myTask3).In(pocket.FromGitRoot("docs"))
```

### Bundled task packages

Pocket includes ready-made task packages for common languages:

```go
import (
    "github.com/fredrikaverpil/pocket"
    "github.com/fredrikaverpil/pocket/tasks/golang"
    "github.com/fredrikaverpil/pocket/tasks/python"
    "github.com/fredrikaverpil/pocket/tasks/markdown"
)

var Config = pocket.Config{
    Run: pocket.Serial(
        golang.Tasks(),
        python.Tasks(),
        markdown.Tasks(),
    ),
}
```

Configure with options:

```go
golang.Tasks(golang.Options{LintConfig: ".golangci.yml"})
python.Tasks(python.Options{RuffConfig: "pyproject.toml"})
```

### Auto-detection

For projects with multiple modules, use `AutoDetect()` to automatically find
directories containing relevant files:

```go
var Config = pocket.Config{
    Run: pocket.Serial(
        pocket.AutoDetect(golang.Tasks()),    // finds all go.mod directories
        pocket.AutoDetect(python.Tasks()),    // finds all pyproject.toml directories
    ),
}
```

This is opinionated: it runs the same tasks across all detected directories.
Combine with path filtering for more control:

```go
pocket.AutoDetect(golang.Tasks()).Except("vendor", "testdata")
```

### Skipping tasks

Use `Skip()` to exclude specific tasks from a task group. Pass the task
constructor function directly - pocket uses reflection to extract the task name:

```go
import "github.com/fredrikaverpil/pocket/tasks/golang"

var Config = pocket.Config{
    Run: pocket.AutoDetect(golang.Tasks()).Skip(golang.TestTask),
}
```

Skip multiple tasks:

```go
pocket.AutoDetect(golang.Tasks()).Skip(golang.TestTask, golang.VulncheckTask)
```

This works with any task constructor that returns `*pocket.Task`:

```go
// Custom task constructors work too
func MySlowTask() *pocket.Task {
    return &pocket.Task{Name: "my-slow-task", ...}
}

pocket.Paths(myTasks).In(".").Skip(MySlowTask)
```

Skipped tasks are excluded from both execution and CLI help output.

## Reference

### Convenience functions

```go
// Path helpers
pocket.GitRoot()              // git repository root
pocket.FromGitRoot("subdir")  // path relative to git root
pocket.FromPocketDir("file")  // path relative to .pocket/
pocket.FromBinDir("tool")     // path relative to .pocket/bin/
pocket.BinaryName("mytool")   // appends .exe on Windows

// Execution
cmd := pocket.Command(ctx, "go", "build", "./...")  // PATH includes .pocket/bin/
cmd.Run()

// Detection (for Detectable interface)
pocket.DetectByFile("go.mod")       // dirs containing file
pocket.DetectByExtension(".lua")    // dirs containing extension
```

### Windows support

Pocket auto-detects your platform during bootstrap:

- **Unix/macOS/WSL**: Creates `./pok` (bash script)
- **Windows**: Creates `pok.cmd` and `pok.ps1`

Add additional shim types in `.pocket/config.go`:

```go
var Config = pocket.Config{
    Shim: &pocket.ShimConfig{
        Posix:      true,   // ./pok
        Windows:    true,   // pok.cmd
        PowerShell: true,   // pok.ps1
    },
}
```

### Shim customization

```go
Shim: &pocket.ShimConfig{Name: "build"}  // creates ./build instead of ./pok
```

Or use a shell alias: `alias pok='./pok'`

## Acknowledgements

- [einride/sage](https://github.com/einride/sage) - Inspiration for the task
  system and tool management approach
