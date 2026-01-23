# Pocket

Pocket is a [Sage](https://github.com/einride/sage)-inspired composable task
runner framework, designed for speed, simplicity, and monorepo workflows.

It replaces quirky Makefiles and shell scripts with a type-safe, Go-based
configuration that handles task composition, tool installation, and
cross-platform execution.

Pocket brings CI to your local machine. Instead of complex provider-specific
workflows, define your tasks once and run them anywhere—locally, in GitHub
Actions, GitLab CI, Codeberg's Woodpecker, Tangled Spindles, or any other
provider. Your CI becomes portable.

> [!WARNING]
>
> Under heavy development. Breaking changes will occur until the initial
> release.

## Key Features

- **Zero-Install Bootstrapping**: The `./pok` shim automatically manages the
  correct Go version and tools for your project. No pre-installed dependencies
  required other than a shell.
- **Composable by Design**: Build complex workflows using `Serial` and
  `Parallel` combinators with automatic output buffering.
- **First-Class Monorepo Support**: Auto-detect modules (e.g., `go.mod`,
  `package.json`) and run tasks in the correct directory context automatically.
- **Automated Tool Management**: Define tool dependencies (like `golangci-lint`)
  directly in Go. Pocket handles downloading, versioning, and caching in
  `.pocket/`.
- **CI/CD Integration**: Access the execution plan programmatically to generate
  CI matrices, documentation, or custom tooling. Your tasks become the single
  source of truth.
- **Cross-Platform**: Built for macOS, Linux, and Windows.

## Quickstart

### 1. Bootstrap your project

Run the following command in your project root (requires Go installed for this
initial step only):

```bash
go run github.com/fredrikaverpil/pocket/cmd/pocket@latest init
```

This creates the `.pocket/` directory and the `./pok` wrapper script.

### 2. Define your first task

Edit `.pocket/config.go` to define your task tree:

```go
package main

import (
    "context"
    "fmt"
    "github.com/fredrikaverpil/pocket/pk"
)

var Hello = pk.NewTask("hello", "say hello", nil, pk.Do(func(ctx context.Context) error {
    fmt.Println("Hello from Pocket!")
    return nil
}))

var Config = &pk.Config{
    Auto: pk.Serial(Hello),
}
```

### 3. Run it

```bash
./pok hello
```

## Concepts

Pocket is built around two primitives: **tasks** and **tools**.

**Tasks** are units of work—linting, testing, building, deploying. They're
composed into execution trees using `Serial` and `Parallel`.

**Tools** are the dependencies tasks need to run. Instead of assuming
`golangci-lint` is installed, you declare it as a tool:

```go
pk.InstallGo("github.com/golangci/golangci-lint/cmd/golangci-lint", "v1.64.8")
```

Pocket downloads, versions, and caches tools in `.pocket/tools/`, ensuring
reproducible builds across machines and CI environments. Everyone gets the exact
same version.

> [!TIP]
>
> You can use Pocket as a task runner, or fork it entirely to build your own CI
> platform tailored to your organization's needs.

## Composition

The `Auto` field in your config defines what runs when you execute `./pok`
without arguments. Use `Serial` and `Parallel` combinators to build complex
workflows:

```go
var Config = &pk.Config{
    Auto: pk.Serial(
        pk.Parallel(Fix, Format),  // run concurrently
        Test,                      // then run tests
        Build,                     // finally build
    ),
}
```

- **Serial**: Tasks run one after another, stopping on first failure
- **Parallel**: Tasks run concurrently with buffered output (no interleaving)

### Options

Wrap tasks with `WithOptions` to customize behavior:

- **Auto-detection**: `WithDetect(golang.Detect())` finds all `go.mod`
  directories
- **Path filtering**: `WithIncludePath("services/*")` runs only in matching
  paths
- **Path exclusion**: `WithExcludePath("vendor")` skips specific directories
- **Flag overrides**: `WithFlag(Task, "name", "value")` sets task-specific flags
- [and more...](./REFERENCE.md)

```go
pk.WithOptions(
    pk.Parallel(Lint, Test),
    pk.WithDetect(golang.Detect()),      // run in each Go module
    pk.WithExcludePath("testdata"),      // skip test fixtures
    pk.WithFlag(Test, "race", true),     // enable race detector for Test
    pk.WithFlag(Test, "timeout", "5m"),  // set test timeout
)
```

### Task Deduplication

The same task at the same path only runs once per invocation, even if referenced
multiple times in your composition tree. This makes it safe to compose shared
dependencies without worrying about redundant work. Use `WithForceRun()` to
bypass deduplication when needed.

### Shim Scoping

Pocket generates shims in each detected module directory and each path defined
with `pk.WithIncludePath`. The root shim runs everything, while subfolder shims
only run tasks scoped to that path:

```bash
./pok                       # runs all tasks across all paths
cd services/api && ./pok    # only runs tasks scoped to services/api
./pok lint                  # run a specific task
```

By default, only the POSIX `pok` shim is generated. For Windows support, enable
`pok.cmd` (batch) and `pok.ps1` (PowerShell) via configuration:

```go
var Config = &pk.Config{
    Auto: pk.Serial(Build, Test),
    Plan: &pk.PlanConfig{
        Shims: pk.AllShimsConfig(), // generates pok, pok.cmd, and pok.ps1
    },
}
```

## Introspection

Pocket provides a built-in `plan` command to visualize your execution tree:

```bash
./pok plan
```

### Programmatic Plan Access

Tasks can access the full execution plan at runtime via
`pk.PlanFromContext(ctx)`. This enables powerful workflows like **automatic CI
matrix generation**—instead of manually syncing your CI configuration with your
tasks, let Pocket generate it.

Here's an example using the [GitHub Actions matrix](./docs/github-actions.md)
integration
([see it in action](https://github.com/fredrikaverpil/pocket/actions/workflows/pocket-matrix.yml)):

```go
import "github.com/fredrikaverpil/pocket/tasks/github"

var Config = &pk.Config{
    Auto: pk.Parallel(Lint, Test, Build),
    Manual: []pk.Runnable{
        github.Matrix(github.MatrixConfig{
            DefaultPlatforms: []string{"ubuntu-latest", "macos-latest"},
            TaskOverrides: map[string]github.TaskOverride{
                "lint": {Platforms: []string{"ubuntu-latest"}},
            },
        }),
    },
}
```

Running `./pok gha-matrix` outputs JSON for GitHub Actions' `fromJson()`:

```json
{"include":[{"task":"lint","os":"ubuntu-latest",...},{"task":"test","os":"ubuntu-latest",...}]}
```

Your task definitions become the single source of truth. Add a new task to
`Auto`, and CI automatically runs it in parallel across all configured
platforms—no YAML editing required.

## Documentation

- [User Guide](./docs/guide.md) - Tasks, tools, composition, path filtering, and
  CI integration.
- [API Reference](./docs/reference.md) - Technical specification of the `pk`
  package.
- [Architecture](./docs/architecture.md) - Internal design, shim model, and
  execution internals.
