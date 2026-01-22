# Tasks and Tools

Tasks are the fundamental units of work in Pocket. This guide explains how to define tasks, execute commands, handle CLI flags, and manage tool dependencies.

## Defining Tasks

A task is created using `pk.NewTask`. It requires a name, a usage description, an optional `*flag.FlagSet`, and a `Runnable` body.

```go
var Hello = pk.NewTask(
    "hello",
    "print a greeting",
    nil, // no flags
    pk.Do(func(ctx context.Context) error {
        fmt.Println("Hello!")
        return nil
    }),
)
```

### The `Do` Helper
The `pk.Do` function wraps a simple Go function `func(context.Context) error` into a `Runnable`. This is the most flexible way to implement task logic.

### Hidden Tasks
If a task is only intended to be used as a dependency or called programmatically, you can hide it from the CLI help output:

```go
var InternalTask = pk.NewTask("internal", "...", nil, body).Hidden()
```

### Manual Tasks
By default, tasks in the `Auto` configuration run automatically. If you want a task to *only* run when explicitly named (e.g., `./pok deploy`), use the `Manual()` method:

```go
var Deploy = pk.NewTask("deploy", "...", nil, body).Manual()
```

## Executing Commands

Pocket provides the `pk.Exec` helper to run external commands. It ensures that:
1. The command's output is correctly captured and buffered (important for parallel execution).
2. The command respects context cancellation (graceful shutdown).
3. The `.pocket/bin` directory is added to the command's `PATH`.

```go
pk.Do(func(ctx context.Context) error {
    return pk.Exec(ctx, "go", "test", "./...")
})
```

## Task Flags

Pocket uses the standard library's `flag` package. You can define flags for a task by passing a `FlagSet`.

```go
var (
    deployFlags = flag.NewFlagSet("deploy", flag.ContinueOnError)
    env         = deployFlags.String("env", "staging", "target environment")
)

var Deploy = pk.NewTask("deploy", "deploy the app", deployFlags, pk.Do(func(ctx context.Context) error {
    fmt.Printf("Deploying to %s...\n", *env)
    return nil
}))
```

Run it with:
```bash
./pok deploy -env prod
```

## Tool Management

One of Pocket's strengths is automated tool installation. The `pk.InstallGo` function creates a `Runnable` that installs a Go-based tool and symlinks it into `.pocket/bin`.

### Example: golangci-lint

It is a common pattern to define a hidden task for tool installation and then make the main task depend on it using `pk.Serial`.

```go
var installLint = pk.NewTask(
    "install:golangci-lint",
    "install linter",
    nil,
    pk.InstallGo("github.com/golangci/golangci-lint/cmd/golangci-lint", "v1.60.0"),
).Hidden()

var Lint = pk.NewTask(
    "lint",
    "run golangci-lint",
    nil,
    pk.Serial(
        installLint,
        pk.Do(func(ctx context.Context) error {
            return pk.Exec(ctx, "golangci-lint", "run")
        }),
    ),
)
```

Because Pocket automatically deduplicates tasks, `installLint` will only run once even if multiple tasks depend on it.
