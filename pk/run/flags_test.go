package run

import (
	"testing"
	"time"

	"gotest.tools/v3/assert"
)

type mapToStructFlags struct {
	Name    string        `flag:"name"`
	Count   int           `flag:"count"`
	Enabled bool          `flag:"enabled"`
	Timeout time.Duration `flag:"timeout"`
	OptBool *bool         `flag:"opt-bool"`
	OptStr  *string       `flag:"opt-str"`
}

func TestMapToStruct(t *testing.T) {
	tests := []struct {
		name    string
		input   map[string]any
		want    mapToStructFlags
		wantErr string // substring of the expected error; empty means no error
	}{
		{
			name:  "non-pointer convertible values are set",
			input: map[string]any{"name": "x", "count": 3, "enabled": true},
			want:  mapToStructFlags{Name: "x", Count: 3, Enabled: true},
		},
		{
			name:  "pointer convertible values yield non-nil pointers",
			input: map[string]any{"opt-bool": true, "opt-str": "y"},
			want:  mapToStructFlags{OptBool: new(true), OptStr: new("y")},
		},
		{
			name:  "absent pointer keys stay nil",
			input: map[string]any{"name": "x"},
			want:  mapToStructFlags{Name: "x"},
		},
		{
			name:  "duration converts from int64",
			input: map[string]any{"timeout": int64(5 * time.Second)},
			want:  mapToStructFlags{Timeout: 5 * time.Second},
		},
		{
			name:    "non-pointer non-convertible value errors",
			input:   map[string]any{"enabled": 5}, // int is not convertible to bool
			wantErr: `flag "enabled": cannot convert int to bool`,
		},
		{
			name:    "pointer non-convertible value errors",
			input:   map[string]any{"opt-bool": "nope"}, // string is not convertible to bool
			wantErr: `flag "opt-bool": cannot convert string to bool`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var got mapToStructFlags
			err := mapToStruct(tt.input, &got)
			if tt.wantErr != "" {
				assert.ErrorContains(t, err, tt.wantErr)
				return
			}
			assert.NilError(t, err)
			assert.DeepEqual(t, got, tt.want)
		})
	}
}

func TestMapToStruct_RejectsNonStructDst(t *testing.T) {
	var notStruct int
	err := mapToStruct(map[string]any{}, &notStruct)
	assert.ErrorContains(t, err, "pointer to struct")
}
