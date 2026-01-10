package pocket

import (
	"context"
	"testing"
)

func TestTask_SetArgs_Defaults(t *testing.T) {
	task := &Task{
		Name: "test-task",
		Args: []ArgDef{
			{Name: "name", Default: "world"},
			{Name: "count", Default: "10"},
		},
	}

	task.SetArgs(nil)

	if task.args["name"] != "world" {
		t.Errorf("expected default name='world', got %q", task.args["name"])
	}
	if task.args["count"] != "10" {
		t.Errorf("expected default count='10', got %q", task.args["count"])
	}
}

func TestTask_SetArgs_Override(t *testing.T) {
	task := &Task{
		Name: "test-task",
		Args: []ArgDef{
			{Name: "name", Default: "world"},
			{Name: "count", Default: "10"},
		},
	}

	task.SetArgs(map[string]string{
		"name": "Claude",
	})

	if task.args["name"] != "Claude" {
		t.Errorf("expected name='Claude', got %q", task.args["name"])
	}
	if task.args["count"] != "10" {
		t.Errorf("expected default count='10', got %q", task.args["count"])
	}
}

func TestTask_SetArgs_NoDefaults(t *testing.T) {
	task := &Task{
		Name: "test-task",
		Args: []ArgDef{
			{Name: "name", Usage: "no default"},
		},
	}

	task.SetArgs(map[string]string{
		"name": "test",
	})

	if task.args["name"] != "test" {
		t.Errorf("expected name='test', got %q", task.args["name"])
	}
}

func TestTask_SetArgs_ExtraArgs(t *testing.T) {
	task := &Task{
		Name: "test-task",
		Args: []ArgDef{
			{Name: "name", Default: "world"},
		},
	}

	// Pass an extra arg not in Args definition
	task.SetArgs(map[string]string{
		"name":  "test",
		"extra": "value",
	})

	if task.args["name"] != "test" {
		t.Errorf("expected name='test', got %q", task.args["name"])
	}
	if task.args["extra"] != "value" {
		t.Errorf("expected extra='value', got %q", task.args["extra"])
	}
}

func TestTask_ActionReceivesArgs(t *testing.T) {
	var receivedArgs map[string]string

	task := &Task{
		Name: "test-task",
		Args: []ArgDef{
			{Name: "name", Default: "world"},
		},
		Action: func(_ context.Context, opts *RunContext) error {
			receivedArgs = opts.Args
			return nil
		},
	}

	task.SetArgs(map[string]string{"name": "test"})
	err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedArgs["name"] != "test" {
		t.Errorf("expected action to receive name='test', got %q", receivedArgs["name"])
	}
}

func TestTask_RunInitializesArgs(t *testing.T) {
	var receivedArgs map[string]string

	task := &Task{
		Name: "test-task",
		Args: []ArgDef{
			{Name: "name", Default: "default-value"},
		},
		Action: func(_ context.Context, opts *RunContext) error {
			receivedArgs = opts.Args
			return nil
		},
	}

	// Don't call SetArgs, let Run() initialize it
	err := task.Run(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if receivedArgs["name"] != "default-value" {
		t.Errorf("expected action to receive name='default-value', got %q", receivedArgs["name"])
	}
}

func TestTask_ActionReceivesVerbose(t *testing.T) {
	var receivedVerbose bool

	task := &Task{
		Name: "test-task",
		Action: func(_ context.Context, rc *RunContext) error {
			receivedVerbose = rc.Verbose
			return nil
		},
	}

	// Run without verbose.
	ctx := context.Background()
	if err := task.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if receivedVerbose {
		t.Error("expected Verbose=false without WithVerbose")
	}

	// Run with verbose (new task since sync.Once).
	task2 := &Task{
		Name: "test-task-verbose",
		Action: func(_ context.Context, rc *RunContext) error {
			receivedVerbose = rc.Verbose
			return nil
		},
	}
	ctx = WithVerbose(ctx, true)
	if err := task2.Run(ctx); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !receivedVerbose {
		t.Error("expected Verbose=true with WithVerbose")
	}
}
