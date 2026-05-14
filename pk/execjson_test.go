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
			"type": "serial",
			"children": [
				{"type": "command", "argv": ["echo", "a"], "name": "step-a"},
				{"type": "parallel", "children": [
					{"type": "command", "argv": ["echo", "b"], "name": "step-b"},
					{"type": "task", "name": "go-test"}
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
	if root.Tree == nil || len(root.Tree.Children) != 2 {
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
			doc:  `{"version":1,"tree":{"type":"command","argv":["x"],"name":"x"},"extra":1}`,
			want: `unknown field "extra"`,
		},
		{
			name: "unknown nested field",
			doc:  `{"version":1,"tree":{"type":"command","argv":["x"],"name":"x","bogus":1}}`,
			want: `unknown field "bogus"`,
		},
		{
			name: "missing tree",
			doc:  `{"version":1}`,
			want: "tree: required",
		},
		{
			name: "unsupported version",
			doc:  `{"version":2,"tree":{"type":"command","argv":["x"],"name":"x"}}`,
			want: "version: unsupported",
		},
		{
			name: "version zero absent",
			doc:  `{"tree":{"type":"command","argv":["x"],"name":"x"}}`,
			want: "version: unsupported value 0",
		},
		{
			name: "missing type",
			doc:  `{"version":1,"tree":{"name":"x"}}`,
			want: "tree.type: required",
		},
		{
			name: "unknown type",
			doc:  `{"version":1,"tree":{"type":"shell","name":"x"}}`,
			want: `tree.type: unsupported value "shell"`,
		},
		{
			name: "command argv empty",
			doc:  `{"version":1,"tree":{"type":"command","argv":[],"name":"x"}}`,
			want: "argv: empty array",
		},
		{
			name: "command missing name",
			doc:  `{"version":1,"tree":{"type":"command","argv":["x"]}}`,
			want: "name: required for command nodes",
		},
		{
			name: "task missing name",
			doc:  `{"version":1,"tree":{"type":"task"}}`,
			want: "name: required for task nodes",
		},
		{
			name: "serial empty",
			doc:  `{"version":1,"tree":{"type":"serial","children":[]}}`,
			want: "children: empty array",
		},
		{
			name: "parallel empty",
			doc:  `{"version":1,"tree":{"type":"parallel","children":[]}}`,
			want: "children: empty array",
		},
		{
			name: "name on composition",
			doc:  `{"version":1,"tree":{"type":"serial","children":[{"type":"command","argv":["x"],"name":"x"}],"name":"oops"}}`,
			want: "name: not allowed on serial nodes",
		},
		{
			name: "paths on composition",
			doc:  `{"version":1,"tree":{"type":"parallel","children":[{"type":"command","argv":["x"],"name":"x"}],"paths":["."]}}`,
			want: "paths: not allowed on parallel nodes",
		},
		{
			name: "argv on task",
			doc:  `{"version":1,"tree":{"type":"task","name":"x","argv":["x"]}}`,
			want: "argv: not allowed on task nodes",
		},
		{
			name: "children on command",
			doc:  `{"version":1,"tree":{"type":"command","name":"x","argv":["x"],"children":[]}}`,
			want: "children: not allowed on command nodes",
		},
		{
			name: "paths empty array",
			doc:  `{"version":1,"tree":{"type":"command","argv":["x"],"name":"x","paths":[]}}`,
			want: "paths: empty array",
		},
		{
			name: "paths with empty entry",
			doc:  `{"version":1,"tree":{"type":"command","argv":["x"],"name":"x","paths":[""]}}`,
			want: "paths[0]: empty path",
		},
		{
			name: "argv as string instead of array",
			doc:  `{"version":1,"tree":{"type":"command","argv":"echo hi","name":"x"}}`,
			want: "tree.argv: expected array of strings, got string",
		},
		{
			name: "paths as string instead of array",
			doc:  `{"version":1,"tree":{"type":"command","argv":["x"],"name":"x","paths":"."}}`,
			want: "tree.paths: expected array of strings, got string",
		},
		{
			name: "version as string instead of integer",
			doc:  `{"version":"1","tree":{"type":"command","argv":["x"],"name":"x"}}`,
			want: "version: expected integer, got string",
		},
		{
			name: "tree as string instead of object",
			doc:  `{"version":1,"tree":"oops"}`,
			want: "tree: expected object, got string",
		},
		{
			name: "children as object instead of array",
			doc:  `{"version":1,"tree":{"type":"serial","children":{}}}`,
			want: "tree.children: expected array of nodes, got object",
		},
		{
			name: "syntax error",
			doc:  `{"version":1,"tree":{"type":"command","argv":["x"]`,
			want: "unexpected EOF",
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
	return []string{"sh", "-c", fmt.Sprintf("printf '%%s\n' %q >> %q", token, file)}
}

// readMarkers returns the lines from a marker file, or nil if it does not exist.
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
			"type": "serial",
			"children": [
				{"type": "command", "argv": %s, "name": "a"},
				{"type": "command", "argv": %s, "name": "b"},
				{"type": "command", "argv": %s, "name": "c"}
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
			"type": "parallel",
			"children": [
				{"type": "command", "argv": %s, "name": "a"},
				{"type": "command", "argv": %s, "name": "b"},
				{"type": "command", "argv": %s, "name": "c"}
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
	doc := `{"version":1,"tree":{"type":"command","argv":["sh","-c","exit 1"],"name":"fail"}}`
	if runtime.GOOS == "windows" {
		doc = `{"version":1,"tree":{"type":"command","argv":["cmd","/C","exit 1"],"name":"fail"}}`
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
	doc := `{"version":1,"tree":{"type":"command","argv":["echo","x"],"name":"multi","paths":["a","b"]}}`

	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []taskNodeInfo
	tree, err := buildRunnable(root.Tree, &nodes, nil)
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
	plan := buildPlanFromJSON(tree, nodes, nil)

	ctx, stdout, _ := execJSONTestCtx(t)
	ctx = context.WithValue(ctx, ctxkey.Plan{}, plan)
	ctx = withExecutionTracker(ctx, newExecutionTracker())
	if err := tree.run(ctx); err != nil {
		t.Fatal(err)
	}

	if !slices.Equal(paths, []string{"a", "b"}) {
		t.Errorf("paths visited = %v, want [a b]", paths)
	}
	for _, p := range []string{"a", "b"} {
		want := fmt.Sprintf(":: multi [%s]", p)
		if !strings.Contains(stdout.String(), want) {
			t.Errorf("expected header %q, got:\n%s", want, stdout.String())
		}
	}
}

func TestBuildRunnable_SingleCommandAtRootIsBareTask(t *testing.T) {
	doc := `{"version":1,"tree":{"type":"command","argv":["echo","x"],"name":"x"}}`
	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []taskNodeInfo
	r, err := buildRunnable(root.Tree, &nodes, nil)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := r.(*Task); !ok {
		t.Errorf("expected *Task for single command at root, got %T", r)
	}
	if len(nodes) != 1 || nodes[0].name != "x" {
		t.Errorf("unexpected taskNodes: %+v", nodes)
	}
}

func TestBuildRunnable_MultiPathWrapsCommandInPathFilter(t *testing.T) {
	doc := `{"version":1,"tree":{"type":"command","argv":["echo","x"],"name":"x","paths":["a","b"]}}`
	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []taskNodeInfo
	r, err := buildRunnable(root.Tree, &nodes, nil)
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

func TestRunExecJSON_TaskReference(t *testing.T) {
	var ran []string
	task := &Task{Name: "lint", Usage: "lint", Do: func(ctx context.Context) error {
		ran = append(ran, pkrun.PathFromContext(ctx))
		return nil
	}}
	basePlan, err := newPlan(&Config{Auto: Serial(task)}, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	doc := `{"version":1,"tree":{"type":"task","name":"lint","paths":["api"]}}`
	ctx, stdout, _ := execJSONTestCtx(t)
	ctx = context.WithValue(ctx, ctxkey.Plan{}, basePlan)
	if err := runExecJSON(ctx, strings.NewReader(doc)); err != nil {
		t.Fatalf("runExecJSON: %v", err)
	}
	if !slices.Equal(ran, []string{"api"}) {
		t.Errorf("paths visited = %v, want [api]", ran)
	}
	if !strings.Contains(stdout.String(), ":: lint [api]") {
		t.Errorf("expected task header for path, got:\n%s", stdout.String())
	}
}

func TestBuildPlanFromJSON_TaskReferencePathsOverrideBasePlan(t *testing.T) {
	task := &Task{Name: "lint", Usage: "lint", Do: func(_ context.Context) error { return nil }}
	basePlan, err := newPlan(&Config{Auto: Serial(task)}, "/tmp", []string{"."})
	if err != nil {
		t.Fatal(err)
	}
	doc := `{"version":1,"tree":{"type":"task","name":"lint","paths":["api"]}}`
	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []taskNodeInfo
	tree, err := buildRunnable(root.Tree, &nodes, basePlan)
	if err != nil {
		t.Fatal(err)
	}
	plan := buildPlanFromJSON(tree, nodes, basePlan)
	if got := plan.pathMappings["lint"].resolvedPaths; !slices.Equal(got, []string{"api"}) {
		t.Errorf("lint paths = %v, want [api]", got)
	}
}

func TestBuildPlanFromJSON_PopulatesPathMappings(t *testing.T) {
	doc := `{"version":1,"tree":{"type":"serial","children":[
		{"type":"command","argv":["echo","a"],"name":"a","paths":["x"]},
		{"type":"command","argv":["echo","b"],"name":"b"}
	]}}`
	root, err := parseExecJSON(strings.NewReader(doc))
	if err != nil {
		t.Fatal(err)
	}
	var nodes []taskNodeInfo
	tree, err := buildRunnable(root.Tree, &nodes, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan := buildPlanFromJSON(tree, nodes, nil)
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
	if tree["type"] != "serial" {
		t.Errorf("expected serial type in tree, got %v", tree)
	}
	if _, ok := tree["children"]; !ok {
		t.Errorf("expected children in tree, got %v", tree)
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
	if tree["type"] != "task" {
		t.Errorf("type = %v, want task", tree["type"])
	}
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
			"type": "serial",
			"children": [
				{"type": "command", "argv": %s, "name": "same"},
				{"type": "command", "argv": %s, "name": "same"}
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
