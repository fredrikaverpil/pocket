package pocket

import (
	"context"
	"os"
	"strings"
	"testing"
)

// CLITestOptions is a typed options struct for CLI tests.
type CLITestOptions struct {
	Name  string `arg:"name"  usage:"who to greet"`
	Count int    `arg:"count" usage:"how many times"`
}

func TestPrintFuncHelp_NoArgs(t *testing.T) {
	fn := Func("test-func", "a test func", func(_ context.Context) error { return nil })

	// Verify func with no args is set up correctly.
	info, err := inspectArgs(fn.opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info != nil {
		t.Error("expected nil argsInfo for nil opts")
	}
}

func TestPrintFuncHelp_WithArgs(t *testing.T) {
	fn := Func("greet", "print a greeting", func(_ context.Context) error { return nil }).
		With(CLITestOptions{Name: "world", Count: 5})

	info, err := inspectArgs(fn.opts)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if info == nil {
		t.Fatal("expected non-nil argsInfo")
	}
	if len(info.Fields) != 2 {
		t.Errorf("expected 2 fields, got %d", len(info.Fields))
	}
	if info.Fields[0].Name != "name" {
		t.Errorf("expected first field name='name', got %q", info.Fields[0].Name)
	}
	if info.Fields[0].Default != "world" {
		t.Errorf("expected first field default='world', got %v", info.Fields[0].Default)
	}
	if info.Fields[1].Default != 5 {
		t.Errorf("expected second field default=5, got %v", info.Fields[1].Default)
	}
}

func TestParseTaskArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantArgs    map[string]string
		wantHelp    bool
		wantErr     bool
		errContains string
	}{
		{
			name:     "empty args",
			args:     []string{},
			wantArgs: map[string]string{},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "-key=value format",
			args:     []string{"-name=world"},
			wantArgs: map[string]string{"name": "world"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "-key value format",
			args:     []string{"-name", "world"},
			wantArgs: map[string]string{"name": "world"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "multiple args with mixed formats",
			args:     []string{"-name=Freddy", "-count", "10"},
			wantArgs: map[string]string{"name": "Freddy", "count": "10"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "value with equals sign",
			args:     []string{"-filter=key=value"},
			wantArgs: map[string]string{"filter": "key=value"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "empty value",
			args:     []string{"-empty="},
			wantArgs: map[string]string{"empty": ""},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "value with spaces",
			args:     []string{"-msg=hello world"},
			wantArgs: map[string]string{"msg": "hello world"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "value with spaces using space separator",
			args:     []string{"-msg", "hello world"},
			wantArgs: map[string]string{"msg": "hello world"},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "help flag",
			args:     []string{"-h"},
			wantArgs: nil,
			wantHelp: true,
			wantErr:  false,
		},
		{
			name:     "help flag after args",
			args:     []string{"-name=world", "-h"},
			wantArgs: nil,
			wantHelp: true,
			wantErr:  false,
		},
		{
			name:        "missing dash prefix",
			args:        []string{"name=world"},
			wantArgs:    nil,
			wantHelp:    false,
			wantErr:     true,
			errContains: "expected -key=value or -key value",
		},
		{
			name:     "flag alone treated as boolean",
			args:     []string{"-name"},
			wantArgs: map[string]string{"name": ""},
			wantHelp: false,
			wantErr:  false,
		},
		{
			name:     "multiple boolean flags",
			args:     []string{"-skip-race", "-skip-coverage"},
			wantArgs: map[string]string{"skip-race": "", "skip-coverage": ""},
			wantHelp: false,
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotArgs, gotHelp, err := parseTaskArgs(tt.args)

			if tt.wantErr {
				if err == nil {
					t.Errorf("parseTaskArgs(%v): expected error containing %q, got nil", tt.args, tt.errContains)
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("parseTaskArgs(%v): error %q does not contain %q", tt.args, err.Error(), tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("parseTaskArgs(%v): unexpected error: %v", tt.args, err)
				return
			}

			if gotHelp != tt.wantHelp {
				t.Errorf("parseTaskArgs(%v): got help=%v, want %v", tt.args, gotHelp, tt.wantHelp)
			}

			if tt.wantHelp {
				return
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("parseTaskArgs(%v): got %d args, want %d", tt.args, len(gotArgs), len(tt.wantArgs))
			}

			for k, wantV := range tt.wantArgs {
				if gotV, ok := gotArgs[k]; !ok {
					t.Errorf("parseTaskArgs(%v): missing key %q", tt.args, k)
				} else if gotV != wantV {
					t.Errorf("parseTaskArgs(%v): key %q = %q, want %q", tt.args, k, gotV, wantV)
				}
			}
		})
	}
}

func TestDetectCwd_WithEnvVar(t *testing.T) {
	// Set the environment variable.
	os.Setenv("POK_CONTEXT", "proj1")
	defer os.Unsetenv("POK_CONTEXT")

	cwd := detectCwd()
	if cwd != "proj1" {
		t.Errorf("expected cwd to be 'proj1', got %q", cwd)
	}
}

func TestDetectCwd_WithoutEnvVar(t *testing.T) {
	// Ensure the environment variable is not set.
	os.Unsetenv("POK_CONTEXT")

	cwd := detectCwd()
	// Should fall back to detecting from actual cwd.
	// Since we're running in the repo, it should return "." or a valid path.
	if cwd == "" {
		t.Error("expected cwd to be non-empty")
	}
}

func TestFilterFuncsByCwd(t *testing.T) {
	fn1 := Func("fn1", "func 1", func(_ context.Context) error { return nil })
	fn2 := Func("fn2", "func 2", func(_ context.Context) error { return nil })
	fn3 := Func("fn3", "func 3", func(_ context.Context) error { return nil }) // no path mapping

	// Create path mappings.
	// fn1 runs in proj1, fn2 runs in root.
	mappings := map[string]*PathFilter{
		"fn1": Paths(fn1).In("proj1"),
		"fn2": Paths(fn2).In("."),
	}

	funcs := []*FuncDef{fn1, fn2, fn3}

	// Test filtering from root.
	rootFuncs := filterFuncsByCwd(funcs, ".", mappings)
	if len(rootFuncs) != 2 {
		t.Errorf("expected 2 funcs at root, got %d", len(rootFuncs))
	}
	// fn2 and fn3 should be visible (fn2 has ".", fn3 has no mapping but root-only).

	// Test filtering from proj1.
	proj1Funcs := filterFuncsByCwd(funcs, "proj1", mappings)
	if len(proj1Funcs) != 1 {
		t.Errorf("expected 1 func in proj1, got %d", len(proj1Funcs))
	}
	if proj1Funcs[0].name != "fn1" {
		t.Errorf("expected fn1 in proj1, got %s", proj1Funcs[0].name)
	}

	// Test filtering from unknown directory.
	otherFuncs := filterFuncsByCwd(funcs, "other", mappings)
	if len(otherFuncs) != 0 {
		t.Errorf("expected 0 funcs in other, got %d", len(otherFuncs))
	}
}

func TestIsFuncVisibleIn(t *testing.T) {
	fn := Func("dummy", "dummy", func(_ context.Context) error { return nil })

	mappings := map[string]*PathFilter{
		"fn1": Paths(fn).In("proj1", "proj2"),
		"fn2": Paths(fn).In("."),
	}

	tests := []struct {
		funcName string
		cwd      string
		visible  bool
	}{
		{"fn1", "proj1", true},
		{"fn1", "proj2", true},
		{"fn1", ".", false},
		{"fn1", "other", false},
		{"fn2", ".", true},
		{"fn2", "proj1", false},
		{"fn3", ".", true}, // no mapping = root only
		{"fn3", "proj1", false},
	}

	for _, tt := range tests {
		result := isFuncVisibleIn(tt.funcName, tt.cwd, mappings)
		if result != tt.visible {
			t.Errorf("isFuncVisibleIn(%q, %q) = %v, want %v",
				tt.funcName, tt.cwd, result, tt.visible)
		}
	}
}
