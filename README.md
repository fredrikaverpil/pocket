# Pocket

Pocket is a composable task runner framework, designed for speed, simplicity,
and monorepo workflows.

It replaces complex Makefiles and shell scripts with a type-safe, Go-based
configuration that handles task composition, tool installation, and
cross-platform execution.

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
- **Cross-Platform**: Built for macOS, Linux, and Windows.

> [!NOTE]
>
> You don't need Go installed to use Pocket. The `./pok` shim automatically
> downloads Go to `.pocket/` if needed.

## Quickstart

### 1. Bootstrap your project

Run the following command in your project root (requires Go installed for this
initial step):

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

## Introspection

Pocket provides a built-in `plan` command to visualize your execution tree:

```bash
./pok plan
```

This is particularly useful for debugging complex compositions or generating
CI/CD matrices (via `./pok plan -json`).

## Documentation

- [Tasks & Tools](./docs/tasks-and-tools.md) - Defining work, executing
  commands, and managing dependencies.
- [Composition & Path Filtering](./docs/composition-and-paths.md) - Building
  execution trees and monorepo support.
- [API Reference](./REFERENCE.md) - Technical specification of the `pk` package.
- [Architecture](./ARCHITECTURE.md) - Internal design, shim model, and execution
  internals.
