# Composition and Path Filtering

Pocket allows you to build complex execution trees by composing tasks and controlling where they execute.

## Composition

Tasks are composed using `Serial` and `Parallel` combinators.

### Serial Execution
`pk.Serial` runs tasks one after another. If any task returns an error, execution stops immediately.

```go
// Run format, then lint
var Auto = pk.Serial(Format, Lint)
```

### Parallel Execution
`pk.Parallel` runs tasks concurrently. Pocket automatically buffers the output of parallel tasks so that logs don't interleave, flushing them to the console as each task finishes.

```go
// Run lint and test at the same time
var Auto = pk.Parallel(Lint, Test)
```

## Path Filtering

In monorepos or multi-module projects, you often want to run tasks only in specific directories. Pocket provides `pk.WithOptions` to apply path-based constraints. All path patterns are interpreted as **regular expressions**.

### Include and Exclude
You can manually specify which paths to include or exclude. Exclusions can be global to the scope or targeted at specific tasks using their task objects or string names.

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithIncludePath("services/api"),
    pk.WithExcludePath("vendor"),              // Global: no tasks run in vendor/
    pk.WithExcludeTask(golang.Test, "foo/.*"), // Targeted: only 'golang.Test' skips paths matching foo/.*
)
```

### Task Skipping and Flags
You can completely remove tasks from a scope or configure their flags without changing the task definitions.

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithSkipTask(golang.Lint),         // Remove linting from this scope
    pk.WithFlag(golang.Test, "race", true), // Ensure race detector is enabled
)
```


### Task Skipping and Flags
You can completely remove tasks from a scope or configure their flags without changing the task definitions.

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithSkipTask("go-lint"),           // Remove linting from this scope
    pk.WithFlag("go-test", "race", true), // Ensure race detector is enabled
)
```

### Auto-Detection
The most powerful way to handle paths is through auto-detection. Pocket can scan your repository for marker files (like `go.mod` or `package.json`) and run tasks in those directories.

```go
pk.WithOptions(
    golang.Tasks(),
    pk.WithDetect(pk.DetectByFile("go.mod")),
)
```

Pocket uses **refining composition**: if you nest `WithOptions`, inner detection functions only search within the directories allowed by the outer scope.

Pocket walks the filesystem once and caches the results, ensuring that detection is extremely fast even in large repositories.

## How Paths Affect Execution

When a task is run with a path constraint:
1. **Working Directory**: The task's body (and any `pk.Exec` calls) will execute with the working directory set to the matched path.
2. **Context**: `pk.PathFromContext(ctx)` will return the path relative to the git root.
3. **Deduplication**: Tasks are deduplicated by the tuple `(task_name, path)`. The same task can run in multiple directories, but it will never run twice in the *same* directory during a single invocation.

## Shim Scoping

Pocket generates `./pok` shims in directories matched by `WithIncludePath` or `WithDetect`.

- Running `./pok` from the **root** shows and executes all tasks across all paths.
- Running `./pok` from a **subdirectory** (e.g., `services/api/pok`) only shows and executes tasks scoped to that directory.

This provides a localized development experience while maintaining a centralized configuration.
