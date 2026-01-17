package github

import (
	"bytes"
	"testing"
	"text/template"
)

func TestCITemplate(t *testing.T) {
	// Read the template
	tmplContent, err := workflowTemplates.ReadFile("workflows/pocket-ci.yml.tmpl")
	if err != nil {
		t.Fatalf("failed to read template: %v", err)
	}

	// Parse the template
	tmpl, err := template.New("pocket-ci.yml.tmpl").Parse(string(tmplContent))
	if err != nil {
		t.Fatalf("failed to parse template: %v", err)
	}

	tests := []struct {
		name   string
		config CIConfig
		want   []string // strings that should appear in output
	}{
		{
			name:   "default config",
			config: DefaultCIConfig(),
			want: []string{
				"name: pocket-ci",
				"ubuntu-latest",
				"macos-latest",
				"windows-latest",
				"./pok",
				"./pok.ps1",
			},
		},
		{
			name: "split tasks",
			config: CIConfig{
				Matrix: []MatrixEntry{
					{OS: PlatformUbuntu, Shell: ShellBash, Pok: "./pok"},
				},
				Tasks:      []string{"go-lint", "go-test"},
				SplitTasks: true,
				FailFast:   true,
			},
			want: []string{
				"go-lint:",
				"go-test:",
				"fail-fast: true",
			},
		},
		{
			name: "install shells",
			config: CIConfig{
				Matrix: []MatrixEntry{
					{OS: PlatformUbuntu, Shell: ShellZsh, Pok: "./pok"},
				},
				InstallShells: true,
			},
			want: []string{
				"Install shells (Linux)",
				"apt-get install -y zsh fish",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			if err := tmpl.Execute(&buf, tt.config); err != nil {
				t.Fatalf("failed to execute template: %v", err)
			}

			output := buf.String()
			for _, want := range tt.want {
				if !bytes.Contains([]byte(output), []byte(want)) {
					t.Errorf("output missing %q:\n%s", want, output)
				}
			}
		})
	}
}

func TestBuildMatrix(t *testing.T) {
	tests := []struct {
		name     string
		opts     CIOptions
		wantLen  int
		wantOS   []Platform
		wantPok  []string
	}{
		{
			name:    "default platforms and shells",
			opts:    CIOptions{},
			wantLen: 3,
			wantOS:  []Platform{PlatformUbuntu, PlatformMacOS, PlatformWindows},
			wantPok: []string{"./pok", "./pok", "./pok.ps1"},
		},
		{
			name:    "single platform",
			opts:    CIOptions{Platforms: "ubuntu-latest"},
			wantLen: 1,
			wantOS:  []Platform{PlatformUbuntu},
			wantPok: []string{"./pok"},
		},
		{
			name:    "multiple shells on linux",
			opts:    CIOptions{Platforms: "ubuntu-latest", Shells: "bash,zsh"},
			wantLen: 2,
			wantOS:  []Platform{PlatformUbuntu, PlatformUbuntu},
			wantPok: []string{"./pok", "./pok"},
		},
		{
			name:    "windows with powershell",
			opts:    CIOptions{Platforms: "windows-latest", Shells: "pwsh"},
			wantLen: 1,
			wantOS:  []Platform{PlatformWindows},
			wantPok: []string{"./pok.ps1"},
		},
		{
			name:    "invalid shell for platform filtered out",
			opts:    CIOptions{Platforms: "windows-latest", Shells: "zsh,pwsh"},
			wantLen: 1, // zsh is not valid on Windows
			wantOS:  []Platform{PlatformWindows},
			wantPok: []string{"./pok.ps1"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matrix := buildMatrix(tt.opts)
			if len(matrix) != tt.wantLen {
				t.Errorf("got %d entries, want %d", len(matrix), tt.wantLen)
			}
			for i, entry := range matrix {
				if i < len(tt.wantOS) && entry.OS != tt.wantOS[i] {
					t.Errorf("entry %d: got OS %s, want %s", i, entry.OS, tt.wantOS[i])
				}
				if i < len(tt.wantPok) && entry.Pok != tt.wantPok[i] {
					t.Errorf("entry %d: got Pok %s, want %s", i, entry.Pok, tt.wantPok[i])
				}
			}
		})
	}
}

func TestCreateMatrixEntry(t *testing.T) {
	tests := []struct {
		platform Platform
		shell    Shell
		wantOk   bool
		wantPok  string
	}{
		{PlatformUbuntu, ShellBash, true, "./pok"},
		{PlatformUbuntu, ShellZsh, true, "./pok"},
		{PlatformUbuntu, ShellCmd, false, ""}, // cmd not valid on Linux
		{PlatformMacOS, ShellBash, true, "./pok"},
		{PlatformMacOS, ShellZsh, true, "./pok"},
		{PlatformWindows, ShellPowershell, true, "./pok.ps1"},
		{PlatformWindows, ShellCmd, true, "./pok.cmd"},
		{PlatformWindows, ShellBash, true, "./pok"},
		{PlatformWindows, ShellZsh, false, ""}, // zsh not valid on Windows
	}

	for _, tt := range tests {
		t.Run(string(tt.platform)+"-"+string(tt.shell), func(t *testing.T) {
			entry, ok := createMatrixEntry(tt.platform, tt.shell)
			if ok != tt.wantOk {
				t.Errorf("got ok=%v, want ok=%v", ok, tt.wantOk)
			}
			if ok && entry.Pok != tt.wantPok {
				t.Errorf("got Pok=%s, want Pok=%s", entry.Pok, tt.wantPok)
			}
		})
	}
}

func TestSplitAndTrim(t *testing.T) {
	tests := []struct {
		input string
		want  []string
	}{
		{"a,b,c", []string{"a", "b", "c"}},
		{"a, b, c", []string{"a", "b", "c"}},
		{" a , b , c ", []string{"a", "b", "c"}},
		{"ubuntu-latest, macos-latest", []string{"ubuntu-latest", "macos-latest"}},
		{"", nil},
		{"  ", nil},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := splitAndTrim(tt.input)
			if len(got) != len(tt.want) {
				t.Errorf("got %v, want %v", got, tt.want)
				return
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("got[%d]=%s, want[%d]=%s", i, got[i], i, tt.want[i])
				}
			}
		})
	}
}
