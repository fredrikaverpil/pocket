package pk

import (
	"context"
	"flag"
	"strings"
	"testing"
	"time"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
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

	t.Run("ProgrammaticOnlyField", func(t *testing.T) {
		type mixed struct {
			Name  string   `flag:"name" usage:"a name"`
			Items []string // no flag tag — programmatic-only
		}
		fs, err := buildFlagSetFromStruct("test", mixed{Name: "default"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		// Should only register 1 CLI flag, skipping the programmatic-only field.
		count := 0
		fs.VisitAll(func(_ *flag.Flag) { count++ })
		if count != 1 {
			t.Errorf("expected 1 CLI flag, got %d", count)
		}
	})

	t.Run("PointerBoolField", func(t *testing.T) {
		type flags struct {
			Enable *bool `flag:"enable" usage:"enable feature"`
		}
		fs, err := buildFlagSetFromStruct("test", flags{Enable: new(true)})
		if err != nil {
			t.Fatal(err)
		}
		f := fs.Lookup("enable")
		if f == nil {
			t.Fatal("expected flag 'enable' to be registered")
		}
		if f.DefValue != "true" {
			t.Errorf("expected default 'true', got %q", f.DefValue)
		}
	})

	t.Run("NilPointerBoolField", func(t *testing.T) {
		type flags struct {
			Enable *bool `flag:"enable" usage:"enable feature"`
		}
		fs, err := buildFlagSetFromStruct("test", flags{Enable: nil})
		if err != nil {
			t.Fatal(err)
		}
		f := fs.Lookup("enable")
		if f == nil {
			t.Fatal("expected flag 'enable' to be registered")
		}
		if f.DefValue != "false" {
			t.Errorf("expected default 'false' for nil pointer, got %q", f.DefValue)
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

func TestGetFlags_AllTypes(t *testing.T) {
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
	ctx := context.WithValue(context.Background(), ctxkey.TaskFlags{}, m)

	result := pkrun.GetFlags[testFlags](ctx)

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
		Name:    "custom", // differs
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
		ctx := context.WithValue(context.Background(), ctxkey.TaskFlags{}, m)

		flags := pkrun.GetFlags[testFlags](ctx)
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
			pkrun.GetFlags[testFlags](ctx)
		}, "no flags in context")
	})
}

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

	// WithFlags should produce an Option that stores deferred resolution.
	opt := WithFlags(flags{Mode: "custom", Count: 10})
	pf := &pathFilter{inner: task}
	opt(pf)

	// Should have one deferred flag override.
	if len(pf.flags) != 1 {
		t.Fatalf("expected 1 flag override, got %d", len(pf.flags))
	}
	if pf.flags[0].flagsType == nil {
		t.Error("flagsType should be set")
	}

	// Resolve and verify diff.
	resolved, err := resolveTypedFlags(pf.flags, task)
	if err != nil {
		t.Fatal(err)
	}
	if len(resolved) != 1 {
		t.Fatalf("expected 1 resolved flag override, got %d", len(resolved))
	}
	if resolved[0].flagName != "mode" {
		t.Errorf("expected flagName=mode, got %q", resolved[0].flagName)
	}
	if resolved[0].value != "custom" {
		t.Errorf("expected value=custom, got %v", resolved[0].value)
	}
}

func TestWithFlags_InferTask(t *testing.T) {
	type inferFlags struct {
		Mode string `flag:"mode" usage:"mode"`
	}
	task := &Task{
		Name:  "inferred",
		Usage: "test task",
		Flags: inferFlags{Mode: "default"},
		Do:    func(_ context.Context) error { return nil },
	}

	// WithFlags without task argument — infer from flags type.
	opt := WithFlags(inferFlags{Mode: "custom"})

	pf := &pathFilter{
		inner: task,
		flags: []flagOverride{},
	}
	opt(pf)

	// Should store the flags type for deferred resolution.
	if len(pf.flags) != 1 {
		t.Fatalf("expected 1 flag override, got %d", len(pf.flags))
	}
	if pf.flags[0].taskName != "" {
		t.Errorf("taskName should be empty for deferred resolution, got %q", pf.flags[0].taskName)
	}
	if pf.flags[0].flagsType == nil {
		t.Error("flagsType should be set")
	}
}

func TestWithFlags_AmbiguousFlagsType(t *testing.T) {
	type sharedFlags struct {
		Mode string `flag:"mode" usage:"mode"`
	}
	task1 := &Task{Name: "task1", Flags: sharedFlags{}, Do: func(_ context.Context) error { return nil }}
	task2 := &Task{Name: "task2", Flags: sharedFlags{}, Do: func(_ context.Context) error { return nil }}

	cfg := &Config{
		Auto: WithOptions(
			Parallel(task1, task2),
			WithFlags(sharedFlags{Mode: "x"}),
		),
	}

	_, err := newPlan(cfg, t.TempDir(), []string{"."})
	if err == nil {
		t.Fatal("expected error for ambiguous flags type")
	}
	if !strings.Contains(err.Error(), "ambiguous flags type") {
		t.Errorf("expected 'ambiguous flags type' in error, got: %v", err)
	}
}

func TestWithFlags_NoMatchingTask(t *testing.T) {
	type taskFlags struct {
		Mode string `flag:"mode" usage:"mode"`
	}
	type wrongFlags struct {
		Other string `flag:"other" usage:"other"`
	}
	task := &Task{Name: "task", Flags: taskFlags{}, Do: func(_ context.Context) error { return nil }}

	cfg := &Config{
		Auto: WithOptions(
			task,
			WithFlags(wrongFlags{Other: "x"}),
		),
	}

	_, err := newPlan(cfg, t.TempDir(), []string{"."})
	if err == nil {
		t.Fatal("expected error for no matching task")
	}
	if !strings.Contains(err.Error(), "no task found with flags type") {
		t.Errorf("expected 'no task found with flags type' in error, got: %v", err)
	}
}

func TestDiffStructs_PointerFields(t *testing.T) {
	type flags struct {
		Enable  *bool  `flag:"enable"`
		Verbose *bool  `flag:"verbose"`
		Name    string `flag:"name"`
	}

	t.Run("NilPointerSkipped", func(t *testing.T) {
		defaults := flags{Enable: new(true), Verbose: new(false), Name: "default"}
		overrides := flags{Enable: nil, Verbose: nil, Name: "custom"}

		diff, err := diffStructs(defaults, overrides)
		if err != nil {
			t.Fatal(err)
		}
		// nil pointers should be skipped — only Name differs.
		if _, ok := diff["enable"]; ok {
			t.Error("nil pointer 'enable' should not be in diff")
		}
		if _, ok := diff["verbose"]; ok {
			t.Error("nil pointer 'verbose' should not be in diff")
		}
		if diff["name"] != "custom" {
			t.Errorf("expected name=custom, got %v", diff["name"])
		}
	})

	t.Run("NonNilPointerIncluded", func(t *testing.T) {
		defaults := flags{Enable: new(true), Verbose: new(false)}
		overrides := flags{Enable: new(false), Verbose: nil}

		diff, err := diffStructs(defaults, overrides)
		if err != nil {
			t.Fatal(err)
		}
		// Enable explicitly set to false (differs from default true).
		if diff["enable"] != false {
			t.Errorf("expected enable=false, got %v", diff["enable"])
		}
		// Verbose nil = not set, should be absent.
		if _, ok := diff["verbose"]; ok {
			t.Error("nil pointer 'verbose' should not be in diff")
		}
	})

	t.Run("SameValueNotInDiff", func(t *testing.T) {
		defaults := flags{Enable: new(true)}
		overrides := flags{Enable: new(true)}

		diff, err := diffStructs(defaults, overrides)
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := diff["enable"]; ok {
			t.Error("same value should not be in diff")
		}
	})
}

func TestStructToMap_PointerFields(t *testing.T) {
	type flags struct {
		Enable *bool   `flag:"enable"`
		Name   *string `flag:"name"`
		Count  int     `flag:"count"`
	}

	t.Run("NonNilDereferenced", func(t *testing.T) {
		name := "hello"
		m, err := structToMap(flags{Enable: new(true), Name: &name, Count: 5})
		if err != nil {
			t.Fatal(err)
		}
		if m["enable"] != true {
			t.Errorf("expected enable=true, got %v", m["enable"])
		}
		if m["name"] != "hello" {
			t.Errorf("expected name=hello, got %v", m["name"])
		}
		if m["count"] != 5 {
			t.Errorf("expected count=5, got %v", m["count"])
		}
	})

	t.Run("NilSkipped", func(t *testing.T) {
		m, err := structToMap(flags{Enable: nil, Name: nil, Count: 5})
		if err != nil {
			t.Fatal(err)
		}
		if _, ok := m["enable"]; ok {
			t.Error("nil pointer 'enable' should not be in map")
		}
		if _, ok := m["name"]; ok {
			t.Error("nil pointer 'name' should not be in map")
		}
		if m["count"] != 5 {
			t.Errorf("expected count=5, got %v", m["count"])
		}
	})
}

func TestGetFlags_PointerFields(t *testing.T) {
	type flags struct {
		Enable *bool   `flag:"enable"`
		Name   *string `flag:"name"`
		Count  int     `flag:"count"`
	}

	m := map[string]any{
		"enable": true,
		"name":   "hello",
		"count":  42,
	}
	ctx := context.WithValue(context.Background(), ctxkey.TaskFlags{}, m)

	result := pkrun.GetFlags[flags](ctx)

	if result.Enable == nil || *result.Enable != true {
		t.Errorf("expected enable=true, got %v", result.Enable)
	}
	if result.Name == nil || *result.Name != "hello" {
		t.Errorf("expected name=hello, got %v", result.Name)
	}
	if result.Count != 42 {
		t.Errorf("expected count=42, got %d", result.Count)
	}
}
