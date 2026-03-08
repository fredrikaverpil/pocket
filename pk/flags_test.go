package pk

import (
	"context"
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
		Name:    "custom", // differs
		Verbose: true,    // same as default
		Count:   10,      // same as default
		Rate:    2.0,     // differs
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
