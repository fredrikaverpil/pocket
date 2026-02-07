## Project Overview

Pocket is a composable task runner framework for Go, designed for monorepo
workflows. Tasks are composed using `Serial`/`Parallel` combinators and can be
filtered by directory paths.

## Build and Test Commands

```bash
# Run pocket (executes full task tree)
cd .pocket && go run .

# Run tests
go test ./pk/...

# Run a single test
go test ./pk/... -run TestTask_Run_Deduplication

# View plan (introspection)
cd .pocket && go run . plan
cd .pocket && go run . plan -json
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
