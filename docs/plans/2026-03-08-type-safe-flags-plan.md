# Type-Safe Flags Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Replace the untyped `map[string]FlagDef` flag system with type-safe structs using struct tags and reflection.

**Architecture:** The `Task.Flags` field changes from `map[string]FlagDef` to `any` (validated as a struct at plan-build time). A new `flagsFromStruct()` reflect helper converts structs to/from `flag.FlagSet` and `map[string]any`. All existing task packages, tests, and config are migrated.

**Tech Stack:** Go stdlib `reflect`, `flag`, struct tags

---

### Task 1: Add reflect helper — `flagsFromStruct`

This is the core engine piece. A new file `pk/flags.go` with reflect-based helpers that convert between flag structs and the internal representations.

**Files:**
- Create: `pk/flags.go`
- Test: `pk/flags_test.go`

**Step 1: Write the failing tests**

Create `pk/flags_test.go`:

```go
package pk

import (
	"flag"
	"testing"
	"time"
)

type testFlags struct {
	Name    string        `flag:"name"    usage:"a name"`
	Verbose bool          `flag:"verbose" usage:"verbose mode"`
	Count   int           `flag:"count"   usage:"a count"`
	Big     int64         `flag:"big"     usage:"a big number"`
	Small   uint          `flag:"small"   usage:"a small number"`
	Large   uint64        `flag:"large"   usage:"a large number"`
	Rate    float64       `flag:"rate"    usage:"a rate"`
	Dur     time.Duration `flag:"dur"     usage:"a duration"`
}

func TestBuildFlagSetFromStruct(t *testing.T) {
	t.Run("AllTypes", func(t *testing.T) {
		flags := testFlags{
			Name:    "hello",
			Verbose: true,
			Count:   42,
			Big:     100,
			Small:   5,
			Large:   999,
			Rate:    1.5,
			Dur:     5 * time.Second,
		}

		fs, err := buildFlagSetFromStruct("test", flags)
		if err != nil {
			t.Fatal(err)
		}

		// Verify all flags are registered with correct defaults.
		for _, want := range []string{"name", "verbose", "count", "big", "small", "large", "rate", "dur"} {
			if f := fs.Lookup(want); f == nil {
				t.Errorf("expected flag %q to be registered", want)
			}
		}
	})

	t.Run("NotAStruct", func(t *testing.T) {
		_, err := buildFlagSetFromStruct("test", "not a struct")
		if err == nil {
			t.Error("expected error for non-struct")
		}
	})

	t.Run("MissingFlagTag", func(t *testing.T) {
		type bad struct {
			Name string `usage:"a name"` // no flag tag
		}
		_, err := buildFlagSetFromStruct("test", bad{})
		if err == nil {
			t.Error("expected error for missing flag tag")
		}
	})

	t.Run("UnsupportedFieldType", func(t *testing.T) {
		type bad struct {
			Names []string `flag:"names" usage:"list of names"`
		}
		_, err := buildFlagSetFromStruct("test", bad{})
		if err == nil {
			t.Error("expected error for unsupported field type")
		}
	})

	t.Run("NilFlags", func(t *testing.T) {
		fs, err := buildFlagSetFromStruct("test", nil)
		if err != nil {
			t.Fatal(err)
		}
		// Should return empty flagset.
		hasFlags := false
		fs.VisitAll(func(*flag.Flag) { hasFlags = true })
		if hasFlags {
			t.Error("expected no flags for nil input")
		}
	})
}

func TestStructToMap(t *testing.T) {
	flags := testFlags{
		Name:    "hello",
		Verbose: true,
		Count:   42,
		Rate:    1.5,
	}

	m, err := structToMap(flags)
	if err != nil {
		t.Fatal(err)
	}

	if m["name"] != "hello" {
		t.Errorf("expected name=hello, got %v", m["name"])
	}
	if m["verbose"] != true {
		t.Errorf("expected verbose=true, got %v", m["verbose"])
	}
	if m["count"] != 42 {
		t.Errorf("expected count=42, got %v", m["count"])
	}
	if m["rate"] != 1.5 {
		t.Errorf("expected rate=1.5, got %v", m["rate"])
	}
}

func TestMapToStruct(t *testing.T) {
	m := map[string]any{
		"name":    "world",
		"verbose": false,
		"count":   99,
		"big":     int64(200),
		"small":   uint(10),
		"large":   uint64(500),
		"rate":    2.5,
		"dur":     5 * time.Second,
	}

	var result testFlags
	if err := mapToStruct(m, &result); err != nil {
		t.Fatal(err)
	}

	if result.Name != "world" {
		t.Errorf("expected name=world, got %q", result.Name)
	}
	if result.Verbose {
		t.Errorf("expected verbose=false, got %v", result.Verbose)
	}
	if result.Count != 99 {
		t.Errorf("expected count=99, got %d", result.Count)
	}
	if result.Rate != 2.5 {
		t.Errorf("expected rate=2.5, got %f", result.Rate)
	}
}

func TestDiffStructs(t *testing.T) {
	defaults := testFlags{
		Name:    "default",
		Verbose: true,
		Count:   10,
		Rate:    1.0,
	}

	overrides := testFlags{
		Name:    "custom",  // differs
		Verbose: true,     // same as default
		Count:   10,       // same as default
		Rate:    2.0,      // differs
	}

	diff, err := diffStructs(defaults, overrides)
	if err != nil {
		t.Fatal(err)
	}

	// Only changed fields should be in diff.
	if diff["name"] != "custom" {
		t.Errorf("expected name=custom, got %v", diff["name"])
	}
	if diff["rate"] != 2.0 {
		t.Errorf("expected rate=2.0, got %v", diff["rate"])
	}
	if _, ok := diff["verbose"]; ok {
		t.Error("verbose should not be in diff (unchanged)")
	}
	if _, ok := diff["count"]; ok {
		t.Error("count should not be in diff (unchanged)")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -run 'TestBuildFlagSetFromStruct|TestStructToMap|TestMapToStruct|TestDiffStructs' -v`
Expected: FAIL (functions not defined)

**Step 3: Write the implementation**

Create `pk/flags.go`:

```go
package pk

import (
	"flag"
	"fmt"
	"io"
	"reflect"
	"sort"
	"time"
)

// buildFlagSetFromStruct creates a *flag.FlagSet from a flags struct.
// The struct's field values are used as defaults. The "flag" struct tag
// provides the CLI flag name, and the "usage" tag provides help text.
//
// Supported field types: string, bool, int, int64, uint, uint64, float64,
// time.Duration — matching the flag standard library.
//
// Flags are declared as data on the Task struct (not inside Do) because
// the engine needs them at plan-build time for CLI parsing, help text,
// plan-level overrides, and introspection.
//
// Returns an empty FlagSet if flags is nil.
func buildFlagSetFromStruct(taskName string, flags any) (*flag.FlagSet, error) {
	fs := flag.NewFlagSet(taskName, flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if flags == nil {
		return fs, nil
	}

	v := reflect.ValueOf(flags)
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("task %q: Flags must be a struct, got %T", taskName, flags)
	}

	// Collect and sort field names for deterministic output.
	type fieldInfo struct {
		flagName string
		usage    string
		value    reflect.Value
		kind     reflect.Kind
	}
	var fields []fieldInfo

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}

		flagName := f.Tag.Get("flag")
		if flagName == "" {
			return nil, fmt.Errorf("task %q: field %q is missing the \"flag\" struct tag", taskName, f.Name)
		}

		usage := f.Tag.Get("usage")
		fields = append(fields, fieldInfo{
			flagName: flagName,
			usage:    usage,
			value:    v.Field(i),
			kind:     f.Type.Kind(),
		})
	}

	sort.Slice(fields, func(i, j int) bool {
		return fields[i].flagName < fields[j].flagName
	})

	for _, fi := range fields {
		switch fi.kind {
		case reflect.String:
			fs.String(fi.flagName, fi.value.String(), fi.usage)
		case reflect.Bool:
			fs.Bool(fi.flagName, fi.value.Bool(), fi.usage)
		case reflect.Int:
			fs.Int(fi.flagName, int(fi.value.Int()), fi.usage)
		case reflect.Int64:
			if fi.value.Type() == reflect.TypeOf(time.Duration(0)) {
				fs.Duration(fi.flagName, time.Duration(fi.value.Int()), fi.usage)
			} else {
				fs.Int64(fi.flagName, fi.value.Int(), fi.usage)
			}
		case reflect.Uint:
			fs.Uint(fi.flagName, uint(fi.value.Uint()), fi.usage)
		case reflect.Uint64:
			fs.Uint64(fi.flagName, fi.value.Uint(), fi.usage)
		case reflect.Float64:
			fs.Float64(fi.flagName, fi.value.Float(), fi.usage)
		default:
			return nil, fmt.Errorf("task %q: flag %q has unsupported type %v", taskName, fi.flagName, fi.kind)
		}
	}

	return fs, nil
}

// structToMap converts a flags struct to map[string]any keyed by flag names.
func structToMap(flags any) (map[string]any, error) {
	v := reflect.ValueOf(flags)
	t := v.Type()

	if t.Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", flags)
	}

	m := make(map[string]any, t.NumField())
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		flagName := f.Tag.Get("flag")
		if flagName == "" {
			continue
		}
		m[flagName] = v.Field(i).Interface()
	}
	return m, nil
}

// mapToStruct populates a flags struct pointer from a map[string]any.
func mapToStruct(m map[string]any, dst any) error {
	v := reflect.ValueOf(dst)
	if v.Kind() != reflect.Pointer || v.Elem().Kind() != reflect.Struct {
		return fmt.Errorf("dst must be a pointer to struct, got %T", dst)
	}
	v = v.Elem()
	t := v.Type()

	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		flagName := f.Tag.Get("flag")
		if flagName == "" {
			continue
		}
		val, ok := m[flagName]
		if !ok {
			continue
		}
		field := v.Field(i)
		rv := reflect.ValueOf(val)
		if rv.Type().ConvertibleTo(field.Type()) {
			field.Set(rv.Convert(field.Type()))
		}
	}
	return nil
}

// diffStructs compares two structs of the same type and returns a map of
// flag names to values for fields that differ. Used by WithFlags to
// detect which fields the user explicitly overrode vs left at defaults.
func diffStructs(defaults, overrides any) (map[string]any, error) {
	dv := reflect.ValueOf(defaults)
	ov := reflect.ValueOf(overrides)

	if dv.Type() != ov.Type() {
		return nil, fmt.Errorf("type mismatch: %T vs %T", defaults, overrides)
	}
	if dv.Type().Kind() != reflect.Struct {
		return nil, fmt.Errorf("expected struct, got %T", defaults)
	}

	t := dv.Type()
	diff := make(map[string]any)
	for i := range t.NumField() {
		f := t.Field(i)
		if !f.IsExported() {
			continue
		}
		flagName := f.Tag.Get("flag")
		if flagName == "" {
			continue
		}
		if !reflect.DeepEqual(dv.Field(i).Interface(), ov.Field(i).Interface()) {
			diff[flagName] = ov.Field(i).Interface()
		}
	}
	return diff, nil
}
```

**Step 4: Run tests to verify they pass**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -run 'TestBuildFlagSetFromStruct|TestStructToMap|TestMapToStruct|TestDiffStructs' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pk/flags.go pk/flags_test.go
git commit -m "feat(pk): add reflect helpers for type-safe flags"
```

---

### Task 2: Add `GetFlags[T]` function

The new generic function that retrieves the resolved flags struct from context.

**Files:**
- Modify: `pk/flags.go`
- Modify: `pk/flags_test.go`

**Step 1: Write the failing test**

Append to `pk/flags_test.go`:

```go
func TestGetFlags(t *testing.T) {
	t.Run("FromContext", func(t *testing.T) {
		m := map[string]any{
			"name":    "hello",
			"verbose": true,
			"count":   42,
			"big":     int64(0),
			"small":   uint(0),
			"large":   uint64(0),
			"rate":    0.0,
			"dur":     time.Duration(0),
		}
		ctx := withTaskFlags(context.Background(), m)

		flags := GetFlags[testFlags](ctx)
		if flags.Name != "hello" {
			t.Errorf("expected name=hello, got %q", flags.Name)
		}
		if !flags.Verbose {
			t.Error("expected verbose=true")
		}
		if flags.Count != 42 {
			t.Errorf("expected count=42, got %d", flags.Count)
		}
	})

	t.Run("NoFlagsInContextPanics", func(t *testing.T) {
		ctx := context.Background()
		assertFlagPanic(t, func() {
			GetFlags[testFlags](ctx)
		}, "no flags in context")
	})
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -run TestGetFlags -v`
Expected: FAIL

**Step 3: Write the implementation**

Add to `pk/flags.go`:

```go
// GetFlags retrieves the resolved flags struct from context.
// Panics with a flagError if no flags are in context.
// The panic is recovered by task.run() and surfaced as a returned error.
func GetFlags[T any](ctx context.Context) T {
	var zero T
	m := taskFlagsFromContext(ctx)
	if m == nil {
		panic(flagError{fmt.Errorf("no flags in context")})
	}
	if err := mapToStruct(m, &zero); err != nil {
		panic(flagError{err})
	}
	return zero
}
```

**Step 4: Run test to verify it passes**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -run TestGetFlags -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pk/flags.go pk/flags_test.go
git commit -m "feat(pk): add GetFlags[T] for type-safe flag retrieval"
```

---

### Task 3: Wire struct-based `Flags` into `Task` and `buildFlagSet`

Change `Task.Flags` from `map[string]FlagDef` to `any` and update `buildFlagSet` to use the new reflect helpers.

**Files:**
- Modify: `pk/task.go:33-66` (Task struct and FlagDef)
- Modify: `pk/task.go:126-162` (buildFlagSet)
- Modify: `pk/task.go:185-204` (flag resolution in run)

**Step 1: Write the failing test**

Add to `pk/flags_test.go`:

```go
func TestBuildFlagSet_Struct(t *testing.T) {
	type flags struct {
		Fix  bool   `flag:"fix"  usage:"apply fixes"`
		Name string `flag:"name" usage:"a name"`
	}

	task := &Task{
		Name:  "test",
		Flags: flags{Fix: true, Name: "default"},
		Do:    func(_ context.Context) error { return nil },
	}

	if err := task.buildFlagSet(); err != nil {
		t.Fatal(err)
	}

	if f := task.flagSet.Lookup("fix"); f == nil {
		t.Error("expected 'fix' flag")
	}
	if f := task.flagSet.Lookup("name"); f == nil {
		t.Error("expected 'name' flag")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -run TestBuildFlagSet_Struct -v`
Expected: FAIL (Flags is still map type)

**Step 3: Modify `pk/task.go`**

3a. Change the `Task.Flags` field type and remove `FlagDef`:

```go
// In Task struct, change:
//     Flags map[string]FlagDef
// To:
//     Flags any

// Remove the FlagDef type entirely.
```

3b. Update `buildFlagSet()` to delegate to `buildFlagSetFromStruct`:

```go
func (t *Task) buildFlagSet() error {
	fs, err := buildFlagSetFromStruct(t.Name, t.Flags)
	if err != nil {
		return err
	}
	t.flagSet = fs
	return nil
}
```

3c. Update flag resolution in `task.run()` (lines 185-204). Replace:
```go
if len(t.Flags) > 0 {
    resolved := make(map[string]any, len(t.Flags))
    for name, def := range t.Flags {
        resolved[name] = def.Default
    }
```
With:
```go
if t.Flags != nil {
    resolved, err := structToMap(t.Flags)
    if err != nil {
        return fmt.Errorf("task %q: %w", t.Name, err)
    }
```

Keep the rest of the resolution logic (plan overrides + CLI overrides) unchanged — it already works with `map[string]any`.

3d. Update `GetFlag` to also handle struct-based context values (it will be removed later, but needs to compile during migration). No change needed — `GetFlag` reads from `map[string]any` context which is still the internal representation.

**Step 4: Run all flag tests**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -run 'TestBuildFlagSet|TestGetFlag|TestGetFlags' -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pk/task.go pk/flags.go pk/flags_test.go
git commit -m "refactor(pk): change Task.Flags from map[string]FlagDef to any (struct)"
```

---

### Task 4: Add `WithFlags` PathOption and update `flagOverride`

Replace `WithFlag(task, name, value)` with `WithFlags(task, flagsStruct)`.

**Files:**
- Modify: `pk/options.go:66-76` (WithFlag → WithFlags)
- Modify: `pk/options.go:186-214` (pathFilter and flagOverride types)
- Modify: `pk/plan.go:373-382` (flag merging in walk)

**Step 1: Write the failing test**

Add to `pk/flags_test.go`:

```go
func TestWithFlags(t *testing.T) {
	type flags struct {
		Mode  string `flag:"mode"  usage:"mode"`
		Count int    `flag:"count" usage:"count"`
	}

	task := &Task{
		Name:  "test",
		Flags: flags{Mode: "default", Count: 10},
		Do:    func(_ context.Context) error { return nil },
	}

	// WithFlags should produce a PathOption.
	opt := WithFlags(task, flags{Mode: "custom", Count: 10})
	pf := &pathFilter{}
	opt(pf)

	// Should have one flag override for "mode" (Count unchanged = not in diff).
	if len(pf.flags) != 1 {
		t.Fatalf("expected 1 flag override, got %d", len(pf.flags))
	}
	if pf.flags[0].flagName != "mode" {
		t.Errorf("expected flagName=mode, got %q", pf.flags[0].flagName)
	}
	if pf.flags[0].value != "custom" {
		t.Errorf("expected value=custom, got %v", pf.flags[0].value)
	}
}
```

**Step 2: Run test to verify it fails**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -run TestWithFlags -v`
Expected: FAIL

**Step 3: Write the implementation**

Replace `WithFlag` in `pk/options.go`:

```go
// WithFlags sets flag overrides for a specific task in the current scope.
// The flags struct must be the same type as the task's Flags field.
// Only fields that differ from the task's defaults are applied as overrides.
// The task can be specified by its string name or by the task object itself.
func WithFlags(task any, flags any) PathOption {
	return func(pf *pathFilter) {
		taskName := toTaskName(task)

		// Get defaults from the task to compute diff.
		var defaults any
		if t, ok := task.(*Task); ok && t.Flags != nil {
			defaults = t.Flags
		}

		if defaults != nil {
			diff, err := diffStructs(defaults, flags)
			if err != nil {
				panic(fmt.Sprintf("pk.WithFlags: %v", err))
			}
			for name, value := range diff {
				pf.flags = append(pf.flags, flagOverride{
					taskName: taskName,
					flagName: name,
					value:    value,
				})
			}
		} else {
			// No defaults available (task passed as string) — use all fields.
			m, err := structToMap(flags)
			if err != nil {
				panic(fmt.Sprintf("pk.WithFlags: %v", err))
			}
			for name, value := range m {
				pf.flags = append(pf.flags, flagOverride{
					taskName: taskName,
					flagName: name,
					value:    value,
				})
			}
		}
	}
}
```

The `flagOverride` struct and the plan merging logic in `plan.go` (`walk` method, lines 373-382) remain unchanged — they already work with individual `flagName`/`value` pairs.

**Step 4: Run tests**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -run TestWithFlags -v`
Expected: PASS

**Step 5: Commit**

```bash
git add pk/options.go pk/flags.go pk/flags_test.go
git commit -m "feat(pk): add WithFlags for type-safe plan-level flag overrides"
```

---

### Task 5: Update `printTaskHelp` for nil check

The help printer checks `task.flagSet` and `len(t.Flags)`. With `any` type, `len()` won't work.

**Files:**
- Modify: `pk/cli.go:182-208` (printTaskHelp — already uses flagSet, likely works)
- Modify: `pk/task.go` (anywhere `len(t.Flags)` appears)

**Step 1: Check for compilation errors**

Run: `cd /Users/fredrik/code/public/pocket && go build ./pk/`

Fix any remaining `len(t.Flags)` references → `t.Flags != nil`.

**Step 2: Run all pk tests**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -v`
Expected: Some tests FAIL because they still use `map[string]FlagDef` syntax.

**Step 3: Commit**

```bash
git add pk/
git commit -m "fix(pk): update internal nil checks for struct-based Flags"
```

---

### Task 6: Migrate pk/ tests to struct-based flags

Update all tests in `pk/` that create tasks with `map[string]FlagDef` to use struct-based flags.

**Files:**
- Modify: `pk/task_test.go`
- Modify: `pk/e2e_test.go`
- Modify: `pk/integration_test.go`
- Modify: `pk/cli_test.go`
- Modify: `pk/builtins_test.go`
- Modify: `pk/builtins.go` (planTask, selfUpdateTask use `map[string]FlagDef`)

**Step 1: Define test flag structs and update tests**

Create inline test flag structs and replace all `map[string]FlagDef{...}` with struct instances, all `GetFlag[T](ctx, name)` with `GetFlags[T](ctx).Field`, and all `WithFlag(task, name, value)` with `WithFlags(task, Struct{...})`.

Key conversions:

`pk/builtins.go` — planTask:
```go
type planFlags struct {
	JSON bool `flag:"json" usage:"output as JSON"`
}

var planTask = &Task{
	// ...
	Flags: planFlags{},
	Do: func(ctx context.Context) error {
		if GetFlags[planFlags](ctx).JSON {
			// ...
```

`pk/builtins.go` — selfUpdateTask:
```go
type selfUpdateFlags struct {
	Force bool `flag:"force" usage:"bypass Go proxy cache (slower, but guarantees latest)"`
}

var selfUpdateTask = &Task{
	// ...
	Flags: selfUpdateFlags{},
	Do: func(ctx context.Context) error {
		if GetFlags[selfUpdateFlags](ctx).Force {
			// ...
```

`pk/e2e_test.go` — recorder.taskWithFlags needs updating. Change its signature to accept `any` instead of `map[string]FlagDef`:
```go
func (r *recorder) taskWithFlags(name string, flags any) *Task {
```

All test callsites that create `map[string]FlagDef{...}` need corresponding struct types and `WithFlags(...)` calls.

**Step 2: Run all pk tests**

Run: `cd /Users/fredrik/code/public/pocket && go test ./pk/ -v`
Expected: PASS

**Step 3: Commit**

```bash
git add pk/
git commit -m "refactor(pk): migrate all pk/ tests and builtins to struct-based flags"
```

---

### Task 7: Migrate `tasks/golang/` to struct-based flags

**Files:**
- Modify: `tasks/golang/test.go`
- Modify: `tasks/golang/lint.go`
- Modify: `tasks/golang/format.go`
- Modify: `tasks/golang/pprof.go`
- Modify: `tasks/golang/release.go`

**Step 1: Migrate each file**

For each task file, replace the flag constants + `map[string]FlagDef` + `GetFlag[T](ctx, name)` with a flags struct + `GetFlags[T](ctx).Field`.

Example — `tasks/golang/test.go`:

```go
package golang

import (
	"context"
	"strings"

	"github.com/fredrikaverpil/pocket/pk"
)

// TestFlags defines flags for the Test task.
type TestFlags struct {
	Race         bool   `flag:"race"         usage:"enable race detector"`
	Run          string `flag:"run"          usage:"run only tests matching regexp"`
	Timeout      string `flag:"timeout"      usage:"test timeout (e.g., 5m, 30s)"`
	Coverage     bool   `flag:"coverage"     usage:"enable coverage and write to coverage.out"`
	CoverageHTML bool   `flag:"coverage-html" usage:"enable coverage and generate coverage.html"`
	CPUProfile   string `flag:"cpuprofile"   usage:"write CPU profile to file (e.g., cpu.prof)"`
	MemProfile   string `flag:"memprofile"   usage:"write memory profile to file (e.g., mem.prof)"`
	BlockProfile string `flag:"blockprofile" usage:"write block profile to file (e.g., block.prof)"`
	MutexProfile string `flag:"mutexprofile" usage:"write mutex profile to file (e.g., mutex.prof)"`
	Pkg          string `flag:"pkg"          usage:"package pattern to test (e.g., ./pk)"`
	Args         string `flag:"args"         usage:"additional arguments to pass to go test"`
}

var Test = &pk.Task{
	Name:  "go-test",
	Usage: "run go tests",
	Flags: TestFlags{Race: true, Pkg: "./..."},
	Do: func(ctx context.Context) error {
		f := pk.GetFlags[TestFlags](ctx)
		args := []string{"test"}
		if f.Race {
			args = append(args, "-race")
		}
		if f.Coverage || f.CoverageHTML {
			args = append(args, "-coverprofile=coverage.out", "-covermode=atomic")
		}
		if f.Run != "" {
			args = append(args, "-run", f.Run)
		}
		if f.Timeout != "" {
			args = append(args, "-timeout", f.Timeout)
		}
		if f.CPUProfile != "" {
			args = append(args, "-cpuprofile="+f.CPUProfile)
		}
		if f.MemProfile != "" {
			args = append(args, "-memprofile="+f.MemProfile)
		}
		if f.BlockProfile != "" {
			args = append(args, "-blockprofile="+f.BlockProfile)
		}
		if f.MutexProfile != "" {
			args = append(args, "-mutexprofile="+f.MutexProfile)
		}
		if f.Args != "" {
			args = append(args, strings.Fields(f.Args)...)
		}
		args = append(args, f.Pkg)
		if err := pk.Exec(ctx, "go", args...); err != nil {
			return err
		}
		if f.CoverageHTML {
			return pk.Exec(ctx, "go", "tool", "cover", "-html=coverage.out", "-o", "coverage.html")
		}
		return nil
	},
}
```

Apply the same pattern to `lint.go`, `format.go`, `pprof.go`, `release.go`.

Remove all `const FlagXxx = "xxx"` constants from each file.

**Step 2: Verify compilation**

Run: `cd /Users/fredrik/code/public/pocket && go build ./tasks/golang/`
Expected: PASS

**Step 3: Commit**

```bash
git add tasks/golang/
git commit -m "refactor(golang): migrate flags to type-safe structs"
```

---

### Task 8: Migrate `tasks/python/` to struct-based flags

**Files:**
- Modify: `tasks/python/tasks.go` (remove `FlagPython` constant)
- Modify: `tasks/python/test.go`
- Modify: `tasks/python/lint.go`
- Modify: `tasks/python/format.go`
- Modify: `tasks/python/typecheck.go`

**Step 1: Migrate**

The Python tasks share a `FlagPython` constant. With the struct approach, each task has its own flags struct that includes a `Python` field. This is a feature — each task's flags are self-contained and discoverable.

Example — shared pattern for `Python` field:

```go
// tasks/python/test.go
type TestFlags struct {
	Coverage bool   `flag:"coverage" usage:"enable coverage reporting"`
	Python   string `flag:"python"   usage:"Python version to use (e.g., 3.9)"`
}

var Test = &pk.Task{
	Name:  "py-test",
	Usage: "run Python tests",
	Flags: TestFlags{},
	Body:  pk.Serial(uv.Install, testSyncCmd(), testCmd()),
}
```

Note: The `Body` tasks (`testSyncCmd()`, `testCmd()`) use `GetFlag[string](ctx, FlagPython)` internally. These are `pk.Do()` wrappers that run within the task's context, so they have access to the resolved flags. Update these to `pk.GetFlags[TestFlags](ctx).Python`.

**Step 2: Verify compilation**

Run: `cd /Users/fredrik/code/public/pocket && go build ./tasks/python/`
Expected: PASS

**Step 3: Commit**

```bash
git add tasks/python/
git commit -m "refactor(python): migrate flags to type-safe structs"
```

---

### Task 9: Migrate remaining task packages

**Files:**
- Modify: `tasks/github/workflows.go`
- Modify: `tasks/markdown/format.go`
- Modify: `tasks/lua/format.go`
- Modify: `tasks/treesitter/query.go`
- Modify: `tasks/claude/validate.go`
- Modify: `tasks/docs/zensical.go`

**Step 1: Migrate each file**

Same pattern as Tasks 7-8. Create a `XxxFlags` struct for each task, replace constants and `GetFlag` calls.

**Step 2: Verify compilation**

Run: `cd /Users/fredrik/code/public/pocket && go build ./tasks/...`
Expected: PASS

**Step 3: Commit**

```bash
git add tasks/
git commit -m "refactor(tasks): migrate remaining task packages to type-safe flags"
```

---

### Task 10: Migrate `.pocket/config.go` to `WithFlags`

**Files:**
- Modify: `.pocket/config.go`

**Step 1: Update config**

Replace:
```go
pk.WithFlag(github.Workflows, github.FlagSkipPocket, true),
pk.WithFlag(github.Workflows, github.FlagIncludePocketPerjob, true),
```

With:
```go
pk.WithFlags(github.Workflows, github.WorkflowFlags{SkipPocket: true, IncludePocketPerjob: true}),
```

**Step 2: Verify compilation and run**

Run: `cd /Users/fredrik/code/public/pocket && go build ./.pocket/ && ./pok -h`
Expected: PASS, help output shows tasks as before

**Step 3: Commit**

```bash
git add .pocket/config.go
git commit -m "refactor(config): migrate to WithFlags for type-safe overrides"
```

---

### Task 11: Remove old API — `FlagDef`, `GetFlag`, `WithFlag`

**Files:**
- Modify: `pk/task.go` (remove `FlagDef` type, remove `GetFlag` function)
- Modify: `pk/options.go` (remove `WithFlag` function)

**Step 1: Remove**

Delete:
- `FlagDef` struct (task.go:59-66)
- `GetFlag[T]` function (task.go:106-124)
- `WithFlag` function (options.go:66-76)

**Step 2: Verify nothing references the old API**

Run: `cd /Users/fredrik/code/public/pocket && go build ./...`
Expected: PASS (all callers already migrated in previous tasks)

**Step 3: Run full test suite**

Run: `cd /Users/fredrik/code/public/pocket && go test ./...`
Expected: PASS

**Step 4: Commit**

```bash
git add pk/task.go pk/options.go
git commit -m "refactor(pk): remove FlagDef, GetFlag, and WithFlag (replaced by struct API)"
```

---

### Task 12: Update documentation

**Files:**
- Modify: `README.md` (flag examples)
- Modify: `docs/guide.md` (flag examples)
- Modify: `docs/reference.md` (API reference)
- Modify: `.claude/skills/adding-tasks/PATTERNS.md`
- Modify: `.claude/skills/adding-tasks/SKILL.md`
- Modify: `.claude/skills/pocket-engine/INTERNALS.md`

**Step 1: Update all docs**

Replace all `map[string]FlagDef` examples with struct-based examples. Replace `GetFlag[T](ctx, name)` with `GetFlags[T](ctx).Field`. Replace `WithFlag(task, name, value)` with `WithFlags(task, FlagsStruct{...})`.

**Step 2: Commit**

```bash
git add README.md docs/ .claude/skills/
git commit -m "docs: update flag documentation for type-safe struct API"
```

---

### Task 13: Run full validation

**Step 1: Run `./pok`**

Run: `cd /Users/fredrik/code/public/pocket && ./pok`
Expected: All tasks pass (formatting, linting, tests)

**Step 2: Run with verbose**

Run: `cd /Users/fredrik/code/public/pocket && ./pok -v`
Expected: PASS with verbose output

**Step 3: Test CLI flag override**

Run: `cd /Users/fredrik/code/public/pocket && ./pok go-test -race=false -run TestGetFlags`
Expected: PASS, race detector disabled, only TestGetFlags tests run

**Step 4: Test help**

Run: `cd /Users/fredrik/code/public/pocket && ./pok go-test -h`
Expected: Shows flag names, types, and usage from struct tags

**Step 5: Test plan output**

Run: `cd /Users/fredrik/code/public/pocket && ./pok plan -json`
Expected: JSON output includes flag info

**Step 6: Commit if any fixes needed**

```bash
git add -A
git commit -m "fix: address issues found during validation"
```
