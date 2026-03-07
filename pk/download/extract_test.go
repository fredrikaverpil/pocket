package download

import (
	"testing"
)

func TestResolveOutputName(t *testing.T) {
	tests := []struct {
		name        string
		fullPath    string
		baseName    string
		cfg         *extractConfig
		wantName    string
		wantExtract bool
	}{
		{
			name:        "no config extracts with full path",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{}},
			wantName:    "dir/file.txt",
			wantExtract: true,
		},
		{
			name:        "flatten returns base name",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{}, flatten: true},
			wantName:    "file.txt",
			wantExtract: true,
		},
		{
			name:        "rename map matches full path",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{"dir/file.txt": "renamed.txt"}},
			wantName:    "renamed.txt",
			wantExtract: true,
		},
		{
			name:        "rename map matches base name",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{"file.txt": "renamed.txt"}},
			wantName:    "renamed.txt",
			wantExtract: true,
		},
		{
			name:        "rename map skips unmatched file",
			fullPath:    "dir/other.txt",
			baseName:    "other.txt",
			cfg:         &extractConfig{renameMap: map[string]string{"file.txt": "renamed.txt"}},
			wantName:    "",
			wantExtract: false,
		},
		{
			name:        "full path match takes precedence over base name",
			fullPath:    "dir/file.txt",
			baseName:    "file.txt",
			cfg:         &extractConfig{renameMap: map[string]string{"dir/file.txt": "full.txt", "file.txt": "base.txt"}},
			wantName:    "full.txt",
			wantExtract: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotName, gotExtract := resolveOutputName(tt.fullPath, tt.baseName, tt.cfg)
			if gotName != tt.wantName {
				t.Errorf("name: got %q, want %q", gotName, tt.wantName)
			}
			if gotExtract != tt.wantExtract {
				t.Errorf("extract: got %v, want %v", gotExtract, tt.wantExtract)
			}
		})
	}
}
