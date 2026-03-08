# Type-Safe Flags

## Problem

The current flag system uses `map[string]FlagDef` with `any`-typed defaults and
string-keyed retrieval via `GetFlag[T](ctx, name)`. This has several issues:

- **Runtime type mismatches**: `GetFlag[bool](ctx, "race")` panics if the
  default was registered as a different type. `WithFlag(task, "race", "oops")`
  compiles fine but fails at runtime.
- **String constants everywhere**: every task defines `const FlagTestRace =
  "race"` etc., which are passed around as arguments.
- **Poor discoverability**: you must read task source code to find available flag
  names and their types.
- **Split source of truth**: flag metadata (name, default, usage) lives in the
  `FlagDef` map while the type lives at the `GetFlag[T]` call site.

## Design

Replace the untyped flag map with a typed struct. Struct field values define
defaults, struct tags provide flag names and usage text.

### Task definition

```go
type TestFlags struct {
    Race    bool   `flag:"race"    usage:"enable race detector"`
    Run     string `flag:"run"     usage:"run only tests matching regexp"`
    Timeout string `flag:"timeout" usage:"test timeout (e.g., 5m, 30s)"`
}

var Test = &pk.Task{
    Name:  "go-test",
    Flags: TestFlags{Race: true},  // struct values are defaults
    Do: func(ctx context.Context) error {
        flags := pk.GetFlags[TestFlags](ctx)
        if flags.Race { ... }
    },
}
```

### Config-level overrides

```go
pk.WithFlags(golang.Test, golang.TestFlags{Race: false})
```

The engine reflect-compares the override struct against the task's default
`Flags` struct. Only fields that differ from defaults are applied as overrides.

### CLI usage

```
./pok go-test -race=false -run TestFoo
```

Unchanged from the user's perspective.

### Why flags are declared as data, not inside Do

Flags must be known at plan-build time (before task execution) for:

1. CLI parsing — the engine must know `-race` is a bool flag
2. Help text — `pok go-test -h` prints available flags without running the task
3. Plan-level overrides — `WithFlags` is resolved during planning
4. Plan introspection — `pok plan -json` includes flag info

This is why flags are declared on the Task struct rather than imperatively inside
the function body.

## API changes

### Task struct

`Flags` field type changes from `map[string]FlagDef` to `any`. Validated at
plan-build time via reflect: must be a struct with supported field types and
`flag`/`usage` struct tags.

### Supported field types

All types supported by the `flag` standard library: `string`, `bool`, `int`,
`int64`, `uint`, `uint64`, `float64`, `time.Duration`.

### New functions

- `pk.GetFlags[T](ctx) T` — returns the resolved flags struct from context
- `pk.WithFlags(task, flagsStruct) PathOption` — plan-level flag overrides

### Removed

- `FlagDef` type
- `GetFlag[T](ctx, name)` function
- `WithFlag(task, name, value)` function
- All `const FlagXxx = "xxx"` string constants in task packages

### Resolution priority (unchanged)

1. CLI flags (highest)
2. Plan-level overrides (`WithFlags`)
3. Task defaults (struct field values)

### Plan introspection

`TaskInfo.Flags` remains `map[string]any` for JSON serialization, built from the
resolved struct via reflect.

### Validation

Invalid flag struct types, unsupported field types, and mismatched task/flags
combinations are caught at plan-build time with clear error messages.

## Naming convention

Flag struct types are named `<Task>Flags` (e.g., `TestFlags`, `LintFlags`,
`WorkflowFlags`) and live in the same package as their task. This makes IDE
autocomplete show the task and its flags type together.
