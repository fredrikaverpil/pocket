package pk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"strings"
	"testing"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
)

// execJSONTestCtx returns a context wired with output buffers suitable for
// running JSON-driven tasks within a test.
func execJSONTestCtx(t *testing.T) (context.Context, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	var stdout, stderr bytes.Buffer
	out := &pkrun.Output{Stdout: &stdout, Stderr: &stderr}
	ctx := context.Background()
	ctx = context.WithValue(ctx, ctxkey.Output{}, out)
	return ctx, &stdout, &stderr
}

func TestParseExecJSON_Valid(t *testing.T) {
	doc := `{
		"version": 1,
		"tree": {
			"serial": [
				{"exec": ["echo", "a"], "name": "step-a"},
				{"parallel": [
					{"exec": ["echo", "b"], "name": "step-b"},
					{"exec": ["echo", "c"], "name": "step-c"}
				]}
			]
		}
	}`
	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if root.Version != 1 {
		t.Errorf("version = %d, want 1", root.Version)
	}
	if root.Tree == nil || len(root.Tree.Serial) != 2 {
		t.Errorf("expected serial with 2 children, got %+v", root.Tree)
	}
}

func TestParseExecJSON_Errors(t *testing.T) {
	tests := []struct {
		name string
		doc  string
		want string
	}{
		{
			name: "unknown top-level field",
			doc:  `{"version":1,"tree":{"exec":["x"],"name":"x"},"extra":1}`,
			want: `unknown field "extra"`,
		},
		{
			name: "unknown nested field",
			doc:  `{"version":1,"tree":{"exec":["x"],"name":"x","bogus":1}}`,
			want: `unknown field "bogus"`,
		},
		{
			name: "missing tree",
			doc:  `{"version":1}`,
			want: "tree: required",
		},
		{
			name: "unsupported version",
			doc:  `{"version":2,"tree":{"exec":["x"],"name":"x"}}`,
			want: "version: unsupported",
		},
		{
			name: "version zero (absent)",
			doc:  `{"tree":{"exec":["x"],"name":"x"}}`,
			want: "version: unsupported value 0",
		},
		{
			name: "no kind key",
			doc:  `{"version":1,"tree":{"name":"x"}}`,
			want: "expected one of: exec, serial, parallel",
		},
		{
			name: "two kind keys",
			doc:  `{"version":1,"tree":{"exec":["x"],"serial":[]}}`,
			want: "expected exactly one of",
		},
		{
			name: "exec empty",
			doc:  `{"version":1,"tree":{"exec":[],"name":"x"}}`,
			want: "exec: empty array",
		},
		{
			name: "exec missing name",
			doc:  `{"version":1,"tree":{"exec":["x"]}}`,
			want: "name: required",
		},
		{
			name: "serial empty",
			doc:  `{"version":1,"tree":{"serial":[]}}`,
			want: "serial: empty array",
		},
		{
			name: "parallel empty",
			doc:  `{"version":1,"tree":{"parallel":[]}}`,
			want: "parallel: empty array",
		},
		{
			name: "name on composition",
			doc:  `{"version":1,"tree":{"serial":[{"exec":["x"],"name":"x"}],"name":"oops"}}`,
			want: "name: not allowed on serial composition",
		},
		{
			name: "paths on composition",
			doc:  `{"version":1,"tree":{"parallel":[{"exec":["x"],"name":"x"}],"paths":["."]}}`,
			want: "paths: not allowed on parallel composition",
		},
		{
			name: "paths empty array",
			doc:  `{"version":1,"tree":{"exec":["x"],"name":"x","paths":[]}}`,
			want: "paths: empty array",
		},
		{
			name: "paths with empty entry",
			doc:  `{"version":1,"tree":{"exec":["x"],"name":"x","paths":[""]}}`,
			want: "paths[0]: empty path",
		},
		{
			name: "nested unknown kind not allowed",
			doc:  `{"version":1,"tree":{"serial":[{"exec":["x"],"name":"x","serial":[]}]}}`,
			want: "expected exactly one of",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			_, err := parseExecJSON(strings.NewReader(tc.doc))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.want)
			}
			if !strings.Contains(err.Error(), tc.want) {
				t.Errorf("error %q does not contain %q", err.Error(), tc.want)
			}
		})
	}
}

// markerScript returns a shell command that appends a token to a file.
// Portable across platforms used in CI (linux, darwin, windows).
func markerScript(t *testing.T, file, token string) []string {
	t.Helper()
	if runtime.GOOS == "windows" {
		return []string{"cmd", "/C", fmt.Sprintf(`echo %s >> %s`, token, file)}
	}
	return []string{"sh", "-c", fmt.Sprintf("printf '%%s\\n' %q >> %q", token, file)}
}

// readMarkers returns the lines from a marker file (or nil if it does not exist).
func readMarkers(t *testing.T, file string) []string {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatal(err)
	}
	var out []string
	for line := range strings.SplitSeq(strings.TrimRight(string(data), "\n\r"), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			out = append(out, line)
		}
	}
	return out
}

func TestRunExecJSON_SerialOrder(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "out.txt")

	doc := fmt.Sprintf(`{
		"version": 1,
		"tree": {
			"serial": [
				{"exec": %s, "name": "a"},
				{"exec": %s, "name": "b"},
				{"exec": %s, "name": "c"}
			]
		}
	}`,
		mustJSON(t, markerScript(t, marker, "a")),
		mustJSON(t, markerScript(t, marker, "b")),
		mustJSON(t, markerScript(t, marker, "c")),
	)

	ctx, stdout, _ := execJSONTestCtx(t)
	if err := runExecJSON(ctx, strings.NewReader(doc)); err != nil {
		t.Fatalf("runExecJSON: %v\nstdout: %s", err, stdout.String())
	}

	got := readMarkers(t, marker)
	want := []string{"a", "b", "c"}
	if !slices.Equal(got, want) {
		t.Errorf("markers = %v, want %v\nstdout: %s", got, want, stdout.String())
	}

	// Headers should appear for each task in stdout.
	for _, name := range want {
		if !strings.Contains(stdout.String(), ":: "+name) {
			t.Errorf("expected stdout to contain task header for %q, got:\n%s", name, stdout.String())
		}
	}
}

func TestRunExecJSON_ParallelRunsAll(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "out.txt")

	doc := fmt.Sprintf(`{
		"version": 1,
		"tree": {
			"parallel": [
				{"exec": %s, "name": "a"},
				{"exec": %s, "name": "b"},
				{"exec": %s, "name": "c"}
			]
		}
	}`,
		mustJSON(t, markerScript(t, marker, "a")),
		mustJSON(t, markerScript(t, marker, "b")),
		mustJSON(t, markerScript(t, marker, "c")),
	)

	ctx, _, _ := execJSONTestCtx(t)
	if err := runExecJSON(ctx, strings.NewReader(doc)); err != nil {
		t.Fatalf("runExecJSON: %v", err)
	}

	got := readMarkers(t, marker)
	sort.Strings(got)
	want := []string{"a", "b", "c"}
	if !slices.Equal(got, want) {
		t.Errorf("markers = %v, want all of %v", got, want)
	}
}

func TestRunExecJSON_TaskFailureReturnsError(t *testing.T) {
	doc := `{
		"version": 1,
		"tree": {
			"serial": [
				{"exec": ["sh", "-c", "exit 1"], "name": "fail"}
			]
		}
	}`
	if runtime.GOOS == "windows" {
		doc = `{
			"version": 1,
			"tree": {
				"serial": [
					{"exec": ["cmd", "/C", "exit 1"], "name": "fail"}
				]
			}
		}`
	}
	ctx, _, _ := execJSONTestCtx(t)
	err := runExecJSON(ctx, strings.NewReader(doc))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
}

func TestRunExecJSON_ValidationErrorEmitsJSONOnStderr(t *testing.T) {
	doc := `{"version":1,"tree":{}}`
	ctx, _, stderr := execJSONTestCtx(t)
	err := runExecJSON(ctx, strings.NewReader(doc))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	var obj map[string]string
	if jerr := json.Unmarshal(bytes.TrimSpace(stderr.Bytes()), &obj); jerr != nil {
		t.Fatalf("stderr is not JSON: %v\n%s", jerr, stderr.String())
	}
	if obj["error"] == "" {
		t.Errorf("expected error field, got %v", obj)
	}
}

func TestRunExecJSON_MultiPathHeaders(t *testing.T) {
	// Use a no-op task that records the path it ran at via the context.
	// We replace the closure post-build to avoid invoking real shells.
	doc := `{"version":1,"tree":{"exec":["echo","x"],"name":"multi","paths":["a","b"]}}`

	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []taskNodeInfo
	tree, err := buildRunnable(root.Tree, &nodes)
	if err != nil {
		t.Fatal(err)
	}
	if len(nodes) != 1 {
		t.Fatalf("expected 1 task node, got %d", len(nodes))
	}
	var paths []string
	nodes[0].task.Do = func(ctx context.Context) error {
		paths = append(paths, pkrun.PathFromContext(ctx))
		return nil
	}
	plan := buildPlanFromJSON(tree, nodes)

	ctx, stdout, _ := execJSONTestCtx(t)
	ctx = context.WithValue(ctx, ctxkey.Plan{}, plan)
	ctx = withExecutionTracker(ctx, newExecutionTracker())
	if err := tree.run(ctx); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(paths, []string{"a", "b"}) {
		t.Errorf("paths visited = %v, want [a b]", paths)
	}
	// Per-path headers should appear in stdout.
	for _, p := range []string{"a", "b"} {
		want := fmt.Sprintf(":: multi [%s]", p)
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("expected header %q, got:\n%s", want, stdout.String())
		}
	}
}

func TestBuildRunnable_SingleTaskAtRootIsBareTask(t *testing.T) {
	doc := `{"version":1,"tree":{"exec":["echo","x"],"name":"x"}}`
	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []taskNodeInfo
	r, err := buildRunnable(root.Tree, &nodes)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.(*Task); !ok {
		t.Errorf("expected *Task for single-task root, got %T", r)
	}
	if len(nodes) != 1 || nodes[0].name != "x" {
		t.Errorf("unexpected taskNodes: %+v", nodes)
	}
}

func TestBuildRunnable_MultiPathWrapsInPathFilter(t *testing.T) {
	doc := `{"version":1,"tree":{"exec":["echo","x"],"name":"x","paths":["a","b"]}}`
	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []taskNodeInfo
	r, err := buildRunnable(root.Tree, &nodes)
	if err != nil {
		t.Fatal(err)
	}
	pf, ok := r.(*pathFilter)
	if !ok {
		t.Fatalf("expected *pathFilter, got %T", r)
	}
	if !slices.Equal(pf.resolvedPaths, []string{"a", "b"}) {
		t.Errorf("resolvedPaths = %v, want [a b]", pf.resolvedPaths)
	}
}

func TestBuildPlanFromJSON_PopulatesPathMappings(t *testing.T) {
	doc := `{"version":1,"tree":{"serial":[
		{"exec":["echo","a"],"name":"a","paths":["x"]},
		{"exec":["echo","b"],"name":"b"}
	]}}`
	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []taskNodeInfo
	tree, err := buildRunnable(root.Tree, &nodes)
	if err != nil {
		t.Fatal(err)
	}
	plan := buildPlanFromJSON(tree, nodes)
	if got := plan.pathMappings["a"].resolvedPaths; !slices.Equal(got, []string{"x"}) {
		t.Errorf("a paths = %v, want [x]", got)
	}
	if got := plan.pathMappings["b"].resolvedPaths; !slices.Equal(got, []string{"."}) {
		t.Errorf("b paths = %v, want [.]", got)
	}
	if inst := plan.taskInstanceByName("a"); inst == nil {
		t.Error("taskInstanceByName(a) returned nil")
	}
}

func TestEmitInvocationJSON_FullTree(t *testing.T) {
	task := &Task{Name: "lint", Usage: "lint", Do: func(_ context.Context) error { return nil }}
	cfg := &Config{Auto: Serial(task)}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := emitInvocationJSON(plan, "", &buf); err != nil {
		t.Fatal(err)
	}

	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatalf("emitted JSON does not parse: %v\n%s", err, buf.String())
	}
	if doc["version"].(float64) != 1 {
		t.Errorf("version = %v, want 1", doc["version"])
	}
	tree, _ := doc["tree"].(map[string]any)
	if tree == nil {
		t.Fatalf("missing tree: %s", buf.String())
	}
	if _, ok := tree["serial"]; !ok {
		t.Errorf("expected serial kind key in tree, got %v", tree)
	}
}

func TestEmitInvocationJSON_SingleTask(t *testing.T) {
	task := &Task{Name: "lint", Usage: "lint", Do: func(_ context.Context) error { return nil }}
	cfg := &Config{Auto: Serial(task)}
	plan, err := newPlan(cfg, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}

	var buf bytes.Buffer
	if err := emitInvocationJSON(plan, "lint", &buf); err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
		t.Fatal(err)
	}
	tree, _ := doc["tree"].(map[string]any)
	if tree["name"] != "lint" {
		t.Errorf("name = %v, want lint", tree["name"])
	}
	paths, _ := tree["paths"].([]any)
	if len(paths) != 1 || paths[0] != "." {
		t.Errorf("paths = %v, want [.]", paths)
	}
}

func TestEmitInvocationJSON_UnknownTask(t *testing.T) {
	plan, err := newPlan(&Config{Auto: nil}, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := emitInvocationJSON(plan, "missing", &buf); err == nil {
		t.Error("expected error for unknown task")
	}
}

func TestExecJSONSchema_IsValidJSON(t *testing.T) {
	var doc map[string]any
	if err := json.Unmarshal([]byte(execJSONSchema), &doc); err != nil {
		t.Fatalf("execJSONSchema is not valid JSON: %v", err)
	}
	if _, ok := doc["definitions"]; !ok {
		t.Error("schema missing definitions block")
	}
}

func TestRunExecJSON_DuplicateNameDeduplicatesAtSamePath(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, "out.txt")

	doc := fmt.Sprintf(`{
		"version": 1,
		"tree": {
			"serial": [
				{"exec": %s, "name": "same"},
				{"exec": %s, "name": "same"}
			]
		}
	}`,
		mustJSON(t, markerScript(t, marker, "first")),
		mustJSON(t, markerScript(t, marker, "second")),
	)

	ctx, _, _ := execJSONTestCtx(t)
	if err := runExecJSON(ctx, strings.NewReader(doc)); err != nil {
		t.Fatalf("runExecJSON: %v", err)
	}

	got := readMarkers(t, marker)
	if len(got) != 1 || got[0] != "first" {
		t.Errorf("expected only first marker (dedup), got %v", got)
	}
}

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	data, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}
