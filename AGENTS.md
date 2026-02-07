## Project Overview

Pocket is a composable task runner framework for Go, designed for monorepo
workflows. Tasks are composed using `Serial`/`Parallel` combinators and can be
filtered by directory paths.

## Commands

```bash
./pok              # Run all auto tasks
./pok -v           # Run with verbose output
./pok -h           # Show help and available tasks
./pok <task>       # Run a specific task
./pok plan         # View execution plan
./pok plan -json   # View execution plan as JSON
```

To run a specific test:

```bash
./pok go-test -run TestName
```

## Project Structure

- `pk/` - Core engine (composition, planning, execution, context)
- `tools/` - Tool packages (installation, versioning, platform support)
- `tasks/` - Task packages (opinionated wrappers around tools)
- `.pocket/` - User configuration (`config.go`, `main.go`) and runtime data
- `docs/` - User guide and API reference

## Entry Points

- `main.go` - Temporary shim that delegates to `.pocket/main.go` via `go run`
- `.pocket/main.go` - Auto-generated, calls `pk.RunMain(Config)`
- `.pocket/config.go` - User configuration defining the task composition tree

## Go Version

Requires Go (version defined in `.pocket/go.mod`). Pocket downloads Go
automatically if not already installed.

## Documentation

See `README.md` for usage documentation, user guide, and API reference links.
See `PLAN.md` for implementation phases and roadmap.
