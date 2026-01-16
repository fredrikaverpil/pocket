package pocket

import (
	"context"
	"testing"
)

func TestValidateNoDuplicateFuncs(t *testing.T) {
	noop := func(_ context.Context) error { return nil }

	tests := []struct {
		name     string
		funcs    []*TaskDef
		builtins []*TaskDef
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "no duplicates",
			funcs:    []*TaskDef{Task("a", "task a", noop), Task("b", "task b", noop)},
			builtins: []*TaskDef{Task("c", "task c", noop)},
			wantErr:  false,
		},
		{
			name:     "duplicate user funcs",
			funcs:    []*TaskDef{Task("a", "task a", noop), Task("b", "task b", noop), Task("a", "task a dup", noop)},
			builtins: []*TaskDef{},
			wantErr:  true,
			errMsg:   "duplicate function names: a",
		},
		{
			name:     "user func conflicts with builtin",
			funcs:    []*TaskDef{Task("clean", "my clean", noop)},
			builtins: []*TaskDef{Task("clean", "builtin clean", noop)},
			wantErr:  true,
			errMsg:   "duplicate function names: clean (conflicts with builtin)",
		},
		{
			name: "multiple duplicates",
			funcs: []*TaskDef{
				Task("a", "a1", noop),
				Task("a", "a2", noop),
				Task("b", "b1", noop),
				Task("b", "b2", noop),
			},
			builtins: []*TaskDef{},
			wantErr:  true,
			errMsg:   "duplicate function names: a, b",
		},
		{
			name:     "empty lists",
			funcs:    []*TaskDef{},
			builtins: []*TaskDef{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNoDuplicateFuncs(tt.funcs, tt.builtins)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got nil")
					return
				}
				if err.Error() != tt.errMsg {
					t.Errorf("expected error %q, got %q", tt.errMsg, err.Error())
				}
			} else if err != nil {
				t.Errorf("expected no error, got %v", err)
			}
		})
	}
}
