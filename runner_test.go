package pocket

import (
	"context"
	"testing"
)

func TestValidateNoDuplicateFuncs(t *testing.T) {
	noop := func(_ context.Context) error { return nil }

	tests := []struct {
		name     string
		funcs    []*FuncDef
		builtins []*FuncDef
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "no duplicates",
			funcs:    []*FuncDef{Func("a", "task a", noop), Func("b", "task b", noop)},
			builtins: []*FuncDef{Func("c", "task c", noop)},
			wantErr:  false,
		},
		{
			name:     "duplicate user funcs",
			funcs:    []*FuncDef{Func("a", "task a", noop), Func("b", "task b", noop), Func("a", "task a dup", noop)},
			builtins: []*FuncDef{},
			wantErr:  true,
			errMsg:   "duplicate function names: a",
		},
		{
			name:     "user func conflicts with builtin",
			funcs:    []*FuncDef{Func("clean", "my clean", noop)},
			builtins: []*FuncDef{Func("clean", "builtin clean", noop)},
			wantErr:  true,
			errMsg:   "duplicate function names: clean (conflicts with builtin)",
		},
		{
			name: "multiple duplicates",
			funcs: []*FuncDef{
				Func("a", "a1", noop),
				Func("a", "a2", noop),
				Func("b", "b1", noop),
				Func("b", "b2", noop),
			},
			builtins: []*FuncDef{},
			wantErr:  true,
			errMsg:   "duplicate function names: a, b",
		},
		{
			name:     "empty lists",
			funcs:    []*FuncDef{},
			builtins: []*FuncDef{},
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
