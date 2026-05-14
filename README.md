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
- **Introspectable Execution Plans**: Preview exactly what will run with
  `./pok plan` before executing, or access the plan programmatically to generate
  CI matrices, documentation, or custom tooling. Your tasks become the single
  source of truth.
- **LLM-Friendly**: Emit executable task trees and global execution options as
  JSON, e.g. `./pok -json -g go-test | ./pok exec` or `./pok exec < tree.json`.
- **Cross-Platform**: Built for macOS, Linux, and Windows.

<details>
<summary>Example: <code>./pok -h</code></summary>

```sh
pocket dev

Usage:
  pok [global-flags]
  pok [global-flags] <task> [task-flags]

Global flags:
  -c, --commits     validate conventional commits after execution
  -g, --gitdiff     run git diff check after execution
  -h, --help        show help
  --json            emit task plan as JSON instead of executing
  -s, --serial      force serial execution (disables parallelism and output buffering)
  -v, --verbose     verbose mode
  --version         show version

Auto tasks:
  github-workflows  bootstrap GitHub workflow files
  go-fix            update code for newer Go versions
  go-format         format Go code
  go-lint           run golangci-lint
  go-test           run go tests
  go-vulncheck      run govulncheck
  md-format         format Markdown files

Manual tasks:
  go-pprof          launch pprof web UI for profile analysis

Builtin tasks:
  shims             regenerate shims in all directories
  plan              show execution plan without running tasks
  exec              execute a JSON task tree read from stdin
  self-update       update Pocket and regenerate scaffolded files
  purge             remove .pocket/tools, .pocket/bin, and .pocket/venvs

Run 'pok <task> -h' for task-specific flags.
```

</details>

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
    "github.com/fredrikaverpil/pocket/pk/run"
)

type HelloFlags struct {
    Name string `flag:"name" usage:"name to greet"`
}

var Hello = &pk.Task{
    Name:  "hello",
    Usage: "say hello",
    Flags: HelloFlags{Name: "World"},
    Do: func(ctx context.Context) error {
        f := run.GetFlags[HelloFlags](ctx)
        fmt.Printf("Hello, %s, from Pocket!\n", f.Name)
        return nil
    },
}

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

**Tasks** are units of work, like linting, testing, building, deploying. They're
composed into execution trees using `Serial` and `Parallel`.

**Tools** are the dependencies tasks need to run. Instead of assuming
`golangci-lint` is installed, you declare it as a tool. Pocket downloads,
versions, and caches tools in `.pocket/tools/`, ensuring reproducible builds
across machines and CI environments. Everyone gets the exact same version.

> [!TIP]
>
> Pocket has both tools and tasks hosted in this repo. They are used throughout
> my personal projects and are quite opinionated. Examples:
>
> - [claudeline](https://github.com/fredrikaverpil/claudeline/blob/main/.pocket/config.go)
> - [creosote](https://github.com/fredrikaverpil/creosote/blob/main/.pocket/config.go)
> - [dependabot-generate](https://github.com/fredrikaverpil/dependabot-generate/blob/main/.pocket/config.go)
> - [fredrikaverpil.github.io](https://github.com/fredrikaverpil/fredrikaverpil.github.io/blob/main/.pocket/config.go)
> - [go-playground](https://github.com/fredrikaverpil/go-playground/blob/main/.pocket/config.go)
> - [multipr](https://github.com/fredrikaverpil/multipr/blob/main/.pocket/config.go)
> - [neotest-golang](https://github.com/fredrikaverpil/neotest-golang/blob/main/.pocket/config.go)
> - [pr.nvim](https://github.com/fredrikaverpil/pr.nvim/blob/main/.pocket/config.go)
>
> Feel free to use my tools/tasks or build your own!

## Composition

The `Auto` field in your config defines what runs when you execute `./pok`
without arguments. Use `Serial` and `Parallel` combinators to build complex
workflows:

```go
var Config = &pk.Config{
    Auto: pk.Serial(
        Format,                   // first format
        pk.Parallel(Lint, Test),  // then run concurrently
        Build,                    // finally build
    ),
}
```

- **Serial**: Tasks run one after another, stopping on first failure
- **Parallel**: Tasks run concurrently with buffered output (no interleaving)

> [!NOTE]
>
> The intent is you run all of your CI testing, linting etc on `./pok`. Special
> tasks like `./pok deploy -prod` can be added into `pk.Config{Manual: ...}`.

### Options

Wrap tasks with `WithOptions` to customize behavior:

- **Auto-detection**: `WithDetect` scans for marker files (e.g. `go.mod`,
  `pyproject.toml`) to run tasks only in matching directories
- **Path filtering**: `WithPath("services/*")` runs only in matching paths
- **Path exclusion**: `WithSkipPath("vendor")` skips specific directories
- **Flag overrides**: `WithFlags(FlagsStruct{Field: value})` sets task-specific
  flags
- [and more...](./docs/reference.md)

```go
pk.WithOptions(
    pk.Parallel(Lint, Test),
    pk.WithDetect(golang.Detect()),      // run in each Go module
    pk.WithSkipPath("testdata"),          // skip test fixtures
    pk.WithFlags(golang.TestFlags{Race: false}),  // disable race detector for Test
)
```

> [!NOTE]
>
> Note that flags have to be explicitly defined in the task. The flags are not
> automatically appending anything to the tool's execution arguments.

### Task Deduplication

The same task at the same path only runs once per invocation, even if referenced
multiple times in your composition tree. This makes it safe to compose shared
dependencies without worrying about redundant work. Use `WithForceRun()` to
bypass deduplication when needed.

### Shim Scoping

Pocket generates shims in each detected module directory (`pk.WithDetect`) and
each path defined with `pk.WithPath`. The root shim runs everything, while
subfolder shims set `TASK_SCOPE` so both bare `./pok` and direct task invocation
only run work relevant to that path:

```bash
./pok                       # runs all tasks across all paths
cd services/api && ./pok    # only runs tasks scoped to services/api
./pok lint                  # run a specific task, scoped to services/api
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

It can also visualize executable JSON trees without running them:

```bash
./pok -json go-test | ./pok plan
./pok plan tree.json
```

## JSON Execution (for agents)

Pocket can also be driven from a JSON document, primarily for LLMs and agents
that compose task trees on-the-fly without writing Go code:

```bash
echo '{"version":1,"tree":{"type":"serial","children":[
  {"type":"command","name":"lint","argv":["golangci-lint","run","./..."]},
  {"type":"command","name":"test","argv":["go","test","./..."]}
]}}' | ./pok exec
```

The same engine drives both paths — composition, deduplication, and output
buffering behave identically. Inspect an existing project's executable task tree
as JSON with `./pok -json [task]`, and print the v1 schema with
`./pok exec --schema`. See the
[JSON Execution](./docs/reference.md#json-execution) reference for full schema
and rules.

### Programmatic Plan Access

Tasks can access the full execution plan at runtime via
`run.PlanFromContext(ctx)`. This enables powerful workflows like **automatic CI
workflow generation** -- instead of manually syncing your CI configuration with
your tasks, let Pocket generate it.

Here's an example using the
[GitHub Actions per-task](./docs/guide.md#per-task-workflow) integration
([see it in action](https://github.com/fredrikaverpil/pocket/actions/workflows/pocket-pertask.yml)):

```go
import "github.com/fredrikaverpil/pocket/tasks/github"

var Config = &pk.Config{
    Auto: pk.Parallel(
        Lint, Test, Build,
        pk.WithOptions(
            github.Tasks(),
            pk.WithFlags(github.WorkflowFlags{
                PerPocketTaskJob: new(true),
                Platforms:        []github.Platform{github.Ubuntu, github.MacOS},
                PerPocketTaskJobOptions: map[string]github.PerPocketTaskJobOption{
                    Lint.Name: {Platforms: []github.Platform{github.Ubuntu}},
                },
            }),
        ),
    ),
}
```

## Documentation

### Using Pocket

- [User Guide](./docs/guide.md) - Tasks, tools, composition, path filtering, and
  CI integration.
- [API Reference](./docs/reference.md) - Technical specification of the `pk` and
  `pk/download` packages.

### Extending Pocket

- [Adding Tools](./.claude/skills/adding-tools/SKILL.md) - How to add new tools
  (binaries, linters, formatters) with cross-platform support and Renovate
  integration. [Patterns](./.claude/skills/adding-tools/PATTERNS.md).
- [Adding Tasks](./.claude/skills/adding-tasks/SKILL.md) - How to write tasks
  that wrap tools with opinionated defaults, flags, and verbose handling.
  [Patterns](./.claude/skills/adding-tasks/PATTERNS.md).
- [Engine Architecture](./.claude/skills/pocket-engine/SKILL.md) - Internal
  design, plan building, composition, and execution pipeline.
  [Internals](./.claude/skills/pocket-engine/INTERNALS.md).
