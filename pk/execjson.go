package pk

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"slices"
	"strings"

	"github.com/fredrikaverpil/pocket/pk/internal/ctxkey"
	pkrun "github.com/fredrikaverpil/pocket/pk/run"
)

// execJSONVersion is the schema version supported by the exec builtin.
const execJSONVersion = 1

const (
	jsonNodeTypeTask     = "task"
	jsonNodeTypeCommand  = "command"
	jsonNodeTypeSerial   = "serial"
	jsonNodeTypeParallel = "parallel"
)

// jsonRoot is the root document for JSON-driven task execution.
type jsonRoot struct {
	Version int          `json:"version"`
	Options *jsonOptions `json:"options,omitempty"`
	Tree    *jsonNode    `json:"tree"`
}

// jsonOptions are global Pocket execution options serialized with JSON trees.
type jsonOptions struct {
	Verbose bool `json:"verbose,omitempty"`
	Serial  bool `json:"serial,omitempty"`
	GitDiff bool `json:"gitdiff,omitempty"`
	Commits bool `json:"commits,omitempty"`
}

// jsonNode is a single node in the JSON execution tree.
type jsonNode struct {
	Type     string      `json:"type"`
	Name     string      `json:"name,omitempty"`
	Argv     []string    `json:"argv,omitempty"`
	Paths    []string    `json:"paths,omitempty"`
	Children []*jsonNode `json:"children,omitempty"`
}

// parseExecJSON reads and validates a JSON execution document from r.
// Unknown fields and malformed shapes return an error.
func parseExecJSON(r io.Reader) (*jsonRoot, error) {
	dec := json.NewDecoder(r)
	dec.DisallowUnknownFields()
	var root jsonRoot
	if err := dec.Decode(&root); err != nil {
		return nil, friendlyDecodeError(err)
	}
	var trailing struct{}
	if err := dec.Decode(&trailing); err != io.EOF {
		if err != nil {
			return nil, friendlyDecodeError(err)
		}
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

// friendlyDecodeError rewrites Go's encoding/json errors into agent-friendly
// messages: field paths instead of Go types, schema-oriented expected types,
// and a stripped "json: " prefix.
func friendlyDecodeError(err error) error {
	if typeErr, ok := errors.AsType[*json.UnmarshalTypeError](err); ok {
		field := typeErr.Field
		if field == "" {
			field = "<root>"
		}
		return fmt.Errorf("%s: expected %s, got %s", field, expectedSchemaType(field), typeErr.Value)
	}
	if synErr, ok := errors.AsType[*json.SyntaxError](err); ok {
		return fmt.Errorf("invalid JSON at byte offset %d: %s", synErr.Offset, synErr.Error())
	}
	msg := strings.TrimPrefix(err.Error(), "json: ")
	return errors.New(msg)
}

// expectedSchemaType returns the JSON-schema type expected for a field path
// like "tree.children.argv" or "version". Trailing segment determines the type.
func expectedSchemaType(field string) string {
	last := field
	if idx := strings.LastIndex(field, "."); idx >= 0 {
		last = field[idx+1:]
	}
	switch last {
	case "version":
		return "integer"
	case "tree", "options":
		return "object"
	case "verbose", "serial", "gitdiff", "commits":
		return "boolean"
	case "type", "name":
		return "string"
	case "argv", "paths":
		return "array of strings"
	case "children":
		return "array of nodes"
	default:
		return "different type"
	}
}

// validateNode enforces the v1 schema rules on a single node and its children.
func validateNode(n *jsonNode, path string) error {
	if n == nil {
		return fmt.Errorf("%s: node is null", path)
	}
	if n.Type == "" {
		return fmt.Errorf("%s.type: required", path)
	}

	switch n.Type {
	case jsonNodeTypeTask:
		if n.Name == "" {
			return fmt.Errorf("%s.name: required for task nodes", path)
		}
		if n.Argv != nil {
			return fmt.Errorf("%s.argv: not allowed on task nodes", path)
		}
		if n.Children != nil {
			return fmt.Errorf("%s.children: not allowed on task nodes", path)
		}
		return validatePaths(n.Paths, path)

	case jsonNodeTypeCommand:
		if n.Name == "" {
			return fmt.Errorf("%s.name: required for command nodes", path)
		}
		if len(n.Argv) == 0 {
			return fmt.Errorf("%s.argv: empty array", path)
		}
		if n.Children != nil {
			return fmt.Errorf("%s.children: not allowed on command nodes", path)
		}
		return validatePaths(n.Paths, path)

	case jsonNodeTypeSerial, jsonNodeTypeParallel:
		if n.Name != "" {
			return fmt.Errorf("%s.name: not allowed on %s nodes", path, n.Type)
		}
		if n.Argv != nil {
			return fmt.Errorf("%s.argv: not allowed on %s nodes", path, n.Type)
		}
		if n.Paths != nil {
			return fmt.Errorf("%s.paths: not allowed on %s nodes", path, n.Type)
		}
		if len(n.Children) == 0 {
			return fmt.Errorf("%s.children: empty array", path)
		}
		for i, child := range n.Children {
			if err := validateNode(child, fmt.Sprintf("%s.children[%d]", path, i)); err != nil {
				return err
			}
		}
		return nil

	default:
		return fmt.Errorf(
			"%s.type: unsupported value %q (expected task, command, serial, parallel)",
			path,
			n.Type,
		)
	}
}

// validatePaths validates optional task or command paths.
func validatePaths(paths []string, path string) error {
	if paths == nil {
		return nil
	}
	if len(paths) == 0 {
		return fmt.Errorf("%s.paths: empty array (omit the field to default to the task's paths)", path)
	}
	for i, p := range paths {
		if p == "" {
			return fmt.Errorf("%s.paths[%d]: empty path", path, i)
		}
	}
	return nil
}

// taskNodeInfo carries data needed to populate the Plan for one JSON leaf node.
type taskNodeInfo struct {
	task          *Task
	name          string
	resolvedPaths []string
	flags         map[string]any
	isManual      bool
	verbose       bool
}

// jsonTaskRef executes an existing Pocket task reference from JSON.
type jsonTaskRef struct {
	task  *Task
	name  string
	paths []string
}

// run implements Runnable for task references in JSON documents.
func (r *jsonTaskRef) run(ctx context.Context) error {
	baseName := r.task.Name
	if len(r.name) > len(baseName) && r.name[:len(baseName)] == baseName && r.name[len(baseName)] == ':' {
		ctx = contextWithNameSuffix(ctx, r.name[len(baseName)+1:])
	}
	for _, path := range r.paths {
		pathCtx := pkrun.ContextWithPath(ctx, path)
		if err := r.task.run(pathCtx); err != nil {
			return fmt.Errorf("task %s in %s: %w", r.name, path, err)
		}
	}
	return nil
}

// buildRunnable converts a validated jsonNode tree to a Runnable.
// Encountered leaf nodes are appended to taskNodes for later Plan construction.
func buildRunnable(n *jsonNode, taskNodes *[]taskNodeInfo, basePlan *Plan) (Runnable, error) {
	switch n.Type {
	case jsonNodeTypeCommand:
		if basePlan != nil && basePlan.taskInstanceByName(n.Name) != nil {
			return nil, fmt.Errorf("command %q conflicts with existing Pocket task", n.Name)
		}
		argv := slices.Clone(n.Argv)
		paths := resolvedJSONPaths(n.Paths, nil)
		t := &Task{
			Name:  n.Name,
			Usage: "JSON command task",
			Do: func(ctx context.Context) error {
				return pkrun.Exec(ctx, argv[0], argv[1:]...)
			},
		}
		if err := t.buildFlagSet(); err != nil {
			return nil, fmt.Errorf("command %q: %w", n.Name, err)
		}
		*taskNodes = append(*taskNodes, taskNodeInfo{task: t, name: n.Name, resolvedPaths: paths})
		if len(paths) == 1 && paths[0] == "." {
			return t, nil
		}
		return &pathFilter{inner: t, resolvedPaths: paths}, nil

	case jsonNodeTypeTask:
		if basePlan == nil {
			return nil, fmt.Errorf("task %q: no Pocket plan available", n.Name)
		}
		inst := basePlan.taskInstanceByName(n.Name)
		if inst == nil {
			return nil, fmt.Errorf("task %q: not found in Pocket plan", n.Name)
		}
		paths := resolvedJSONPaths(n.Paths, inst.resolvedPaths)
		*taskNodes = append(*taskNodes, taskNodeInfo{
			task:          inst.task,
			name:          inst.name,
			resolvedPaths: paths,
			flags:         inst.flags,
			isManual:      inst.isManual,
			verbose:       inst.verbose,
		})
		return &jsonTaskRef{task: inst.task, name: inst.name, paths: paths}, nil

	case jsonNodeTypeSerial:
		children := make([]Runnable, len(n.Children))
		for i, child := range n.Children {
			runnable, err := buildRunnable(child, taskNodes, basePlan)
			if err != nil {
				return nil, err
			}
			children[i] = runnable
		}
		return Serial(children...), nil

	case jsonNodeTypeParallel:
		children := make([]Runnable, len(n.Children))
		for i, child := range n.Children {
			runnable, err := buildRunnable(child, taskNodes, basePlan)
			if err != nil {
				return nil, err
			}
			children[i] = runnable
		}
		return Parallel(children...), nil
	}
	return nil, fmt.Errorf("invalid node type %q", n.Type)
}

// resolvedJSONPaths returns explicit JSON paths, fallback paths, or root.
func resolvedJSONPaths(paths, fallback []string) []string {
	if len(paths) > 0 {
		return slices.Clone(paths)
	}
	if len(fallback) > 0 {
		return slices.Clone(fallback)
	}
	return []string{"."}
}

// buildPlanFromJSON constructs a Plan from a JSON-derived tree and collected
// leaf nodes. It preserves the base Pocket plan so referenced tasks keep their
// flags, path mappings, and composed subtasks.
func buildPlanFromJSON(tree Runnable, taskNodes []taskNodeInfo, basePlan *Plan) *Plan {
	var taskInstances []taskInstance
	pathMappings := make(map[string]pathInfo)
	moduleDirectories := []string{"."}

	if basePlan != nil {
		taskInstances = slices.Clone(basePlan.taskInstances)
		pathMappings = make(map[string]pathInfo, len(basePlan.pathMappings)+len(taskNodes))
		for name, info := range basePlan.pathMappings {
			pathMappings[name] = info
		}
		moduleDirectories = slices.Clone(basePlan.moduleDirectories)
	}

	seen := make(map[string]int, len(taskInstances))
	for i := range taskInstances {
		seen[taskInstances[i].name] = i
	}
	seenJSON := make(map[string]bool, len(taskNodes))

	for _, info := range taskNodes {
		if idx, ok := seen[info.name]; ok {
			paths := slices.Clone(info.resolvedPaths)
			if seenJSON[info.name] {
				existing := pathMappings[info.name]
				paths = slices.Clone(existing.resolvedPaths)
				for _, p := range info.resolvedPaths {
					if !slices.Contains(paths, p) {
						paths = append(paths, p)
					}
				}
			}
			pathMappings[info.name] = pathInfo{resolvedPaths: paths, includePaths: paths}
			taskInstances[idx].resolvedPaths = paths
			seenJSON[info.name] = true
			continue
		}
		paths := slices.Clone(info.resolvedPaths)
		pathMappings[info.name] = pathInfo{resolvedPaths: paths, includePaths: paths}
		seen[info.name] = len(taskInstances)
		seenJSON[info.name] = true
		taskInstances = append(taskInstances, taskInstance{
			task:          info.task,
			name:          info.name,
			resolvedPaths: paths,
			flags:         info.flags,
			isManual:      info.isManual,
			verbose:       info.verbose,
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
		moduleDirectories: moduleDirectories,
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
	ctx = contextWithJSONOptions(ctx, root.Options)
	basePlan := planFromContext(ctx)
	var taskNodes []taskNodeInfo
	tree, err := buildRunnable(root.Tree, &taskNodes, basePlan)
	if err != nil {
		emitJSONError(ctx, err)
		return err
	}
	plan := buildPlanFromJSON(tree, taskNodes, basePlan)

	ctx = context.WithValue(ctx, ctxkey.Plan{}, plan)
	ctx = withExecutionTracker(ctx, newExecutionTracker())
	if err := tree.run(ctx); err != nil {
		return err
	}
	return runPostActions(ctx)
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

// contextWithJSONOptions applies JSON options to ctx. Existing true values win,
// so CLI flags on the receiving process override or add to serialized options.
func contextWithJSONOptions(ctx context.Context, opts *jsonOptions) context.Context {
	if opts == nil {
		return ctx
	}
	if opts.Verbose && !pkrun.Verbose(ctx) {
		ctx = context.WithValue(ctx, ctxkey.Verbose{}, true)
	}
	if opts.Serial && !serialFromContext(ctx) {
		ctx = context.WithValue(ctx, ctxkey.Serial{}, true)
	}
	if opts.GitDiff && !gitDiffEnabled(ctx) {
		ctx = context.WithValue(ctx, ctxkey.GitDiff{}, true)
	}
	if opts.Commits && !commitsCheckEnabled(ctx) {
		ctx = context.WithValue(ctx, ctxkey.CommitsCheck{}, true)
	}
	return ctx
}

// jsonOptionsFromContext returns the JSON options represented by ctx.
func jsonOptionsFromContext(ctx context.Context) *jsonOptions {
	opts := &jsonOptions{
		Verbose: pkrun.Verbose(ctx),
		Serial:  serialFromContext(ctx),
		GitDiff: gitDiffEnabled(ctx),
		Commits: commitsCheckEnabled(ctx),
	}
	if !opts.Verbose && !opts.Serial && !opts.GitDiff && !opts.Commits {
		return nil
	}
	return opts
}

// emitInvocationJSON writes the JSON representation of a Plan invocation.
// If taskName is empty the whole Auto tree is emitted; otherwise only the
// named task slice. Returns an error if the named task does not exist.
func emitInvocationJSON(ctx context.Context, p *Plan, taskName string, w io.Writer) error {
	if p == nil {
		return fmt.Errorf("plan is nil")
	}
	var tree map[string]any
	if taskName == "" {
		if p.tree == nil {
			tree = map[string]any{"type": jsonNodeTypeSerial, "children": []map[string]any{}}
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
			"type":  jsonNodeTypeTask,
			"name":  inst.name,
			"paths": paths,
		}
	}
	doc := map[string]any{
		"version": execJSONVersion,
		"tree":    tree,
	}
	if opts := jsonOptionsFromContext(ctx); opts != nil {
		doc["options"] = opts
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(doc)
}

// emitJSONNode converts a Runnable to its JSON representation.
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
			"type":  jsonNodeTypeTask,
			"name":  effectiveName,
			"paths": paths,
		}
	case *serial:
		children := make([]map[string]any, 0, len(v.runnables))
		for _, child := range v.runnables {
			children = append(children, emitJSONNode(child, nameSuffix, p))
		}
		return map[string]any{"type": jsonNodeTypeSerial, "children": children}
	case *parallel:
		children := make([]map[string]any, 0, len(v.runnables))
		for _, child := range v.runnables {
			children = append(children, emitJSONNode(child, nameSuffix, p))
		}
		return map[string]any{"type": jsonNodeTypeParallel, "children": children}
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
    "options": {"$ref": "#/definitions/options"},
    "tree": {"$ref": "#/definitions/node"}
  },
  "definitions": {
    "options": {
      "type": "object",
      "additionalProperties": false,
      "properties": {
        "verbose": {"type": "boolean"},
        "serial": {"type": "boolean"},
        "gitdiff": {"type": "boolean"},
        "commits": {"type": "boolean"}
      }
    },
    "node": {
      "type": "object",
      "additionalProperties": false,
      "required": ["type"],
      "properties": {
        "type": {"type": "string", "enum": ["task", "command", "serial", "parallel"]},
        "name": {"type": "string", "minLength": 1},
        "argv": {
          "type": "array",
          "items": {"type": "string"},
          "minItems": 1
        },
        "paths": {
          "type": "array",
          "items": {"type": "string", "minLength": 1},
          "minItems": 1
        },
        "children": {
          "type": "array",
          "items": {"$ref": "#/definitions/node"},
          "minItems": 1
        }
      },
      "oneOf": [
        {
          "properties": {"type": {"const": "task"}},
          "required": ["type", "name"],
          "not": {"anyOf": [{"required": ["argv"]}, {"required": ["children"]}]}
        },
        {
          "properties": {"type": {"const": "command"}},
          "required": ["type", "name", "argv"],
          "not": {"required": ["children"]}
        },
        {
          "properties": {"type": {"const": "serial"}},
          "required": ["type", "children"],
          "not": {"anyOf": [{"required": ["name"]}, {"required": ["argv"]}, {"required": ["paths"]}]}
        },
        {
          "properties": {"type": {"const": "parallel"}},
          "required": ["type", "children"],
          "not": {"anyOf": [{"required": ["name"]}, {"required": ["argv"]}, {"required": ["paths"]}]}
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
