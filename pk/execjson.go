package pk

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"slices"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
)

// execJSONVersion is the schema version supported by the exec builtin.
const execJSONVersion = 1

// jsonRoot is the root document for JSON-driven task execution.
type jsonRoot struct {
	Version int       `json:"version"`
	Tree    *jsonNode `json:"tree"`
}

// jsonNode is a single node in the JSON execution tree. Each node must have
// exactly one kind key set: exec, serial, or parallel.
type jsonNode struct {
	Exec     []string    `json:"exec,omitempty"`
	Serial   []*jsonNode `json:"serial,omitempty"`
	Parallel []*jsonNode `json:"parallel,omitempty"`
	Name     string      `json:"name,omitempty"`
	Paths    []string    `json:"paths,omitempty"`
}

// parseExecJSON reads and validates a JSON execution document from r.
// Unknown fields and malformed shapes return an error.
func parseExecJSON(r io.Reader) (*jsonRoot, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var root jsonRoot
	if err := dec.Decode(&root); err != nil {
		return nil, fmt.Errorf("decoding JSON: %w", err)
	}
	if dec.More() {
		return nil, fmt.Errorf("unexpected trailing content after root document")
	}
	if root.Version != execJSONVersion {
		return nil, fmt.Errorf("version: unsupported value %d (expected %d)", root.Version, execJSONVersion)
	}
	if root.Tree == nil {
		return nil, fmt.Errorf("tree: required")
	}
	if err := validateNode(root.Tree, "tree"); err != nil {
		return nil, err
	}
	return &root, nil
}

// validateNode enforces the v1 schema rules on a single node and its children.
func validateNode(n *jsonNode, path string) error {
	if n == nil {
		return fmt.Errorf("%s: node is null", path)
	}
	var kinds []string
	if n.Exec != nil {
		kinds = append(kinds, "exec")
	}
	if n.Serial != nil {
		kinds = append(kinds, "serial")
	}
	if n.Parallel != nil {
		kinds = append(kinds, "parallel")
	}
	switch len(kinds) {
	case 0:
		return fmt.Errorf("%s: expected one of: exec, serial, parallel", path)
	case 1:
		// valid
	default:
		return fmt.Errorf("%s: expected exactly one of exec/serial/parallel, got %v", path, kinds)
	}

	switch kinds[0] {
	case "exec":
		if len(n.Exec) == 0 {
			return fmt.Errorf("%s.exec: empty array", path)
		}
		if n.Name == "" {
			return fmt.Errorf("%s.name: required for exec nodes", path)
		}
		if n.Paths != nil && len(n.Paths) == 0 {
			return fmt.Errorf("%s.paths: empty array (omit the field to default to root)", path)
		}
		for i, p := range n.Paths {
			if p == "" {
				return fmt.Errorf("%s.paths[%d]: empty path", path, i)
			}
		}
	case "serial", "parallel":
		if n.Name != "" {
			return fmt.Errorf("%s.name: not allowed on %s composition", path, kinds[0])
		}
		if n.Paths != nil {
			return fmt.Errorf("%s.paths: not allowed on %s composition", path, kinds[0])
		}
		children := n.Serial
		if kinds[0] == "parallel" {
			children = n.Parallel
		}
		if len(children) == 0 {
			return fmt.Errorf("%s.%s: empty array", path, kinds[0])
		}
		for i, c := range children {
			if err := validateNode(c, fmt.Sprintf("%s.%s[%d]", path, kinds[0], i)); err != nil {
				return err
			}
		}
	}
	return nil
}

// taskNodeInfo carries data needed to populate the Plan for one JSON task node.
type taskNodeInfo struct {
	task          *Task
	name          string
	resolvedPaths []string
}

// buildRunnable converts a validated jsonNode tree to a Runnable.
// Encountered task nodes are appended to taskNodes for later Plan construction.
func buildRunnable(n *jsonNode, taskNodes *[]taskNodeInfo) (Runnable, error) {
	switch {
	case n.Exec != nil:
		argv := slices.Clone(n.Exec)
		paths := slices.Clone(n.Paths)
		if len(paths) == 0 {
			paths = []string{"."}
		}
		t := &Task{
			Name:  n.Name,
			Usage: "JSON exec task",
			Do: func(ctx context.Context) error {
				return pkrun.Exec(ctx, argv[0], argv[1:]...)
			},
		}
		if err := t.buildFlagSet(); err != nil {
			return nil, fmt.Errorf("task %q: %w", n.Name, err)
		}
		*taskNodes = append(*taskNodes, taskNodeInfo{
			task:          t,
			name:          n.Name,
			resolvedPaths: paths,
		})
		// At root only: task.run prints the simple ":: name" header, no wrap needed.
		if len(paths) == 1 && paths[0] == "." {
			return t, nil
		}
		return &pathFilter{inner: t, resolvedPaths: paths}, nil

	case n.Serial != nil:
		children := make([]Runnable, len(n.Serial))
		for i, c := range n.Serial {
			r, err := buildRunnable(c, taskNodes)
			if err != nil {
				return nil, err
			}
			children[i] = r
		}
		return Serial(children...), nil

	case n.Parallel != nil:
		children := make([]Runnable, len(n.Parallel))
		for i, c := range n.Parallel {
			r, err := buildRunnable(c, taskNodes)
			if err != nil {
				return nil, err
			}
			children[i] = r
		}
		return Parallel(children...), nil
	}
	return nil, fmt.Errorf("invalid node: no kind key set")
}

// buildPlanFromJSON constructs a Plan from a JSON-derived tree and collected
// task nodes. Path mappings merge resolvedPaths for any task name appearing
// multiple times.
func buildPlanFromJSON(tree Runnable, taskNodes []taskNodeInfo) *Plan {
	taskInstances := make([]taskInstance, 0, len(taskNodes))
	pathMappings := make(map[string]pathInfo, len(taskNodes))
	seen := make(map[string]int) // name -> index into taskInstances

	for _, info := range taskNodes {
		if idx, ok := seen[info.name]; ok {
			existing := pathMappings[info.name]
			merged := slices.Clone(existing.resolvedPaths)
			for _, p := range info.resolvedPaths {
				if !slices.Contains(merged, p) {
					merged = append(merged, p)
				}
			}
			pathMappings[info.name] = pathInfo{resolvedPaths: merged, includePaths: merged}
			taskInstances[idx].resolvedPaths = merged
			continue
		}
		paths := slices.Clone(info.resolvedPaths)
		pathMappings[info.name] = pathInfo{resolvedPaths: paths, includePaths: paths}
		seen[info.name] = len(taskInstances)
		taskInstances = append(taskInstances, taskInstance{
			task:          info.task,
			name:          info.name,
			resolvedPaths: paths,
		})
	}

	taskIndex := make(map[string]*taskInstance, len(taskInstances))
	for i := range taskInstances {
		taskIndex[taskInstances[i].name] = &taskInstances[i]
	}

	return &Plan{
		tree:              tree,
		taskInstances:     taskInstances,
		taskIndex:         taskIndex,
		pathMappings:      pathMappings,
		moduleDirectories: []string{"."},
	}
}

// runExecJSON parses a JSON document from r and executes the resulting tree.
// Parse and validation errors are emitted as JSON to stderr.
func runExecJSON(ctx context.Context, r io.Reader) error {
	root, err := parseExecJSON(r)
	if err != nil {
		emitJSONError(ctx, err)
		return err
	}
	var taskNodes []taskNodeInfo
	tree, err := buildRunnable(root.Tree, &taskNodes)
	if err != nil {
		emitJSONError(ctx, err)
		return err
	}
	plan := buildPlanFromJSON(tree, taskNodes)

	ctx = context.WithValue(ctx, ctxkey.Plan{}, plan)
	ctx = withExecutionTracker(ctx, newExecutionTracker())
	return tree.run(ctx)
}

// emitJSONError writes a JSON error object to stderr.
func emitJSONError(ctx context.Context, err error) {
	w := stderrFromContext(ctx)
	obj := map[string]string{"error": err.Error()}
	data, mErr := json.Marshal(obj)
	if mErr != nil {
		fmt.Fprintf(w, "{\"error\":%q}\n", err.Error())
		return
	}
	fmt.Fprintln(w, string(data))
}

// stderrFromContext returns the stderr writer from context, defaulting to
// os.Stderr if none is set.
func stderrFromContext(ctx context.Context) io.Writer {
	if out := pkrun.OutputFromContext(ctx); out != nil {
		return out.Stderr
	}
	return os.Stderr
}

// stdoutFromContext returns the stdout writer from context, defaulting to
// os.Stdout if none is set.
func stdoutFromContext(ctx context.Context) io.Writer {
	if out := pkrun.OutputFromContext(ctx); out != nil {
		return out.Stdout
	}
	return os.Stdout
}

// emitInvocationJSON writes the JSON representation of a Plan invocation.
// If taskName is empty the whole Auto tree is emitted; otherwise only the
// named task slice. Returns an error if the named task does not exist.
func emitInvocationJSON(p *Plan, taskName string, w io.Writer) error {
	if p == nil {
		return fmt.Errorf("plan is nil")
	}
	var tree map[string]any
	if taskName == "" {
		if p.tree == nil {
			tree = map[string]any{"serial": []map[string]any{}}
		} else {
			tree = emitJSONNode(p.tree, "", p)
		}
	} else {
		inst := p.taskInstanceByName(taskName)
		if inst == nil {
			return fmt.Errorf("unknown task %q", taskName)
		}
		paths := inst.resolvedPaths
		if len(paths) == 0 {
			paths = []string{"."}
		}
		tree = map[string]any{
			"name":  inst.name,
			"paths": paths,
		}
	}
	doc := map[string]any{
		"version": execJSONVersion,
		"tree":    tree,
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// emitJSONNode converts a Runnable to its JSON representation, mirroring the
// schema used by the exec builtin (minus the exec field, which is not
// derivable from Go-defined task bodies).
func emitJSONNode(r Runnable, nameSuffix string, p *Plan) map[string]any {
	switch v := r.(type) {
	case *Task:
		effectiveName := v.Name
		if nameSuffix != "" {
			effectiveName = v.Name + ":" + nameSuffix
		}
		paths := []string{"."}
		if info, ok := p.pathMappings[effectiveName]; ok && len(info.resolvedPaths) > 0 {
			paths = info.resolvedPaths
		}
		return map[string]any{
			"name":  effectiveName,
			"paths": paths,
		}
	case *serial:
		children := make([]map[string]any, 0, len(v.runnables))
		for _, child := range v.runnables {
			children = append(children, emitJSONNode(child, nameSuffix, p))
		}
		return map[string]any{"serial": children}
	case *parallel:
		children := make([]map[string]any, 0, len(v.runnables))
		for _, child := range v.runnables {
			children = append(children, emitJSONNode(child, nameSuffix, p))
		}
		return map[string]any{"parallel": children}
	case *pathFilter:
		childSuffix := nameSuffix
		if v.nameSuffix != "" {
			if nameSuffix != "" {
				childSuffix = nameSuffix + ":" + v.nameSuffix
			} else {
				childSuffix = v.nameSuffix
			}
		}
		return emitJSONNode(v.inner, childSuffix, p)
	}
	return map[string]any{}
}

// execJSONSchema is the JSON Schema document for the v1 exec format.
const execJSONSchema = `{
  "$schema": "http://json-schema.org/draft-07/schema#",
  "title": "Pocket exec v1",
  "type": "object",
  "additionalProperties": false,
  "required": ["version", "tree"],
  "properties": {
    "version": {"type": "integer", "const": 1},
    "tree": {"$ref": "#/definitions/node"}
  },
  "definitions": {
    "node": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "exec": {
          "type": "array",
          "items": {"type": "string"},
          "minItems": 1
        },
        "serial": {
          "type": "array",
          "items": {"$ref": "#/definitions/node"},
          "minItems": 1
        },
        "parallel": {
          "type": "array",
          "items": {"$ref": "#/definitions/node"},
          "minItems": 1
        },
        "name": {"type": "string", "minLength": 1},
        "paths": {
          "type": "array",
          "items": {"type": "string", "minLength": 1},
          "minItems": 1
        }
      },
      "oneOf": [
        {
          "required": ["exec", "name"],
          "not": {
            "anyOf": [
              {"required": ["serial"]},
              {"required": ["parallel"]}
            ]
          }
        },
        {
          "required": ["serial"],
          "not": {
            "anyOf": [
              {"required": ["exec"]},
              {"required": ["parallel"]},
              {"required": ["name"]},
              {"required": ["paths"]}
            ]
          }
        },
        {
          "required": ["parallel"],
          "not": {
            "anyOf": [
              {"required": ["exec"]},
              {"required": ["serial"]},
              {"required": ["name"]},
              {"required": ["paths"]}
            ]
          }
        }
      ]
    }
  }
}`

// printExecSchema writes the JSON Schema document to stdout.
func printExecSchema(ctx context.Context) error {
	w := stdoutFromContext(ctx)
	_, err := fmt.Fprintln(w, execJSONSchema)
	return err
}
