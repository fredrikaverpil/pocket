package shim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	pocket "github.com/fredrikaverpil/pocket"
)

func TestGenerate_PosixShim(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory with go.mod.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket dir: %v", err)
	}
	gomod := "module pocket\n\ngo 1.24.4\n"
	if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	cfg := pocket.Config{
		Shim: &pocket.ShimConfig{
			Name:  "pok",
			Posix: true,
		},
	}

	if err := GenerateWithRoot(cfg, tmpDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Check shim was created.
	shimPath := filepath.Join(tmpDir, "pok")
	if _, err := os.Stat(shimPath); os.IsNotExist(err) {
		t.Fatal("shim was not created")
	}

	// Check shim content.
	content, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatalf("reading shim: %v", err)
	}

	if !strings.Contains(string(content), "#!/") {
		t.Error("shim missing shebang")
	}
	if !strings.Contains(string(content), ".pocket") {
		t.Error("shim missing .pocket reference")
	}
}

func TestGenerate_WindowsShim(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory with go.mod.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket dir: %v", err)
	}
	gomod := "module pocket\n\ngo 1.24.4\n"
	if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	cfg := pocket.Config{
		Shim: &pocket.ShimConfig{
			Name:    "pok",
			Windows: true,
		},
	}

	if err := GenerateWithRoot(cfg, tmpDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Check shim was created.
	shimPath := filepath.Join(tmpDir, "pok.cmd")
	if _, err := os.Stat(shimPath); os.IsNotExist(err) {
		t.Fatal("shim was not created")
	}
}

func TestGenerate_PowerShellShim(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory with go.mod.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket dir: %v", err)
	}
	gomod := "module pocket\n\ngo 1.24.4\n"
	if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	cfg := pocket.Config{
		Shim: &pocket.ShimConfig{
			Name:       "pok",
			PowerShell: true,
		},
	}

	if err := GenerateWithRoot(cfg, tmpDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Check shim was created.
	shimPath := filepath.Join(tmpDir, "pok.ps1")
	if _, err := os.Stat(shimPath); os.IsNotExist(err) {
		t.Fatal("shim was not created")
	}
}

func TestGenerate_AllShimTypes(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory with go.mod.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket dir: %v", err)
	}
	gomod := "module pocket\n\ngo 1.24.4\n"
	if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	cfg := pocket.Config{
		Shim: &pocket.ShimConfig{
			Name:       "build",
			Posix:      true,
			Windows:    true,
			PowerShell: true,
		},
	}

	if err := GenerateWithRoot(cfg, tmpDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Check all shims were created.
	for _, shimFile := range []string{"build", "build.cmd", "build.ps1"} {
		shimPath := filepath.Join(tmpDir, shimFile)
		if _, err := os.Stat(shimPath); os.IsNotExist(err) {
			t.Errorf("shim %q was not created", shimFile)
		}
	}
}

func TestExtractGoVersionFromDir(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		goModContent string
		wantVersion  string
		wantErr      bool
	}{
		{
			name:         "standard go.mod",
			goModContent: "module example\n\ngo 1.24.4\n",
			wantVersion:  "1.24.4",
			wantErr:      false,
		},
		{
			name:         "go directive with extra whitespace",
			goModContent: "module example\n\ngo   1.24.4  \n",
			wantVersion:  "1.24.4",
			wantErr:      false,
		},
		{
			name:         "go directive first",
			goModContent: "go 1.24.4\nmodule example\n",
			wantVersion:  "1.24.4",
			wantErr:      false,
		},
		{
			name:         "missing go directive",
			goModContent: "module example\n\nrequire foo v1.0.0\n",
			wantVersion:  "",
			wantErr:      true,
		},
		{
			name:         "go directive with toolchain version",
			goModContent: "module example\n\ngo 1.24.4\n\ntoolchain go1.24.4\n",
			wantVersion:  "1.24.4",
			wantErr:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			tmpDir := t.TempDir()
			if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(tt.goModContent), 0o644); err != nil {
				t.Fatalf("writing go.mod: %v", err)
			}

			got, err := extractGoVersionFromDir(tmpDir)
			if (err != nil) != tt.wantErr {
				t.Errorf("extractGoVersionFromDir() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.wantVersion {
				t.Errorf("extractGoVersionFromDir() = %q, want %q", got, tt.wantVersion)
			}
		})
	}
}

func TestGenerateWithRoot_MissingGoMod(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory without go.mod.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket directory: %v", err)
	}

	cfg := pocket.Config{
		Shim: &pocket.ShimConfig{
			Name:  "pok",
			Posix: true,
		},
	}

	err := GenerateWithRoot(cfg, tmpDir)
	if err == nil {
		t.Error("expected error for missing go.mod, got nil")
	}
	if !strings.Contains(err.Error(), "go.mod") {
		t.Errorf("expected error to mention go.mod, got: %v", err)
	}
}

func TestGenerateWithRoot_MissingGoDirective(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory with go.mod that has no go directive.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket directory: %v", err)
	}

	goMod := "module pocket\n\nrequire example.com/foo v1.0.0\n"
	if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(goMod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	cfg := pocket.Config{
		Shim: &pocket.ShimConfig{
			Name:  "pok",
			Posix: true,
		},
	}

	err := GenerateWithRoot(cfg, tmpDir)
	if err == nil {
		t.Error("expected error for missing go directive, got nil")
	}
	if !strings.Contains(err.Error(), "go directive") {
		t.Errorf("expected error to mention go directive, got: %v", err)
	}
}

func TestGenerateWithRoot_ShimContent(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory with go.mod.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket dir: %v", err)
	}
	gomod := "module pocket\n\ngo 1.24.4\n"
	if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	cfg := pocket.Config{
		Shim: &pocket.ShimConfig{
			Name:       "pok",
			Posix:      true,
			Windows:    true,
			PowerShell: true,
		},
	}

	if err := GenerateWithRoot(cfg, tmpDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Verify Posix shim content.
	t.Run("posix content", func(t *testing.T) {
		t.Parallel()
		content, err := os.ReadFile(filepath.Join(tmpDir, "pok"))
		if err != nil {
			t.Fatalf("reading posix shim: %v", err)
		}
		contentStr := string(content)

		if !strings.Contains(contentStr, `POK_DIR=".pocket"`) {
			t.Error("posix shim missing POK_DIR")
		}
		if !strings.Contains(contentStr, `POK_CONTEXT="."`) {
			t.Error("posix shim missing POK_CONTEXT")
		}
		if !strings.Contains(contentStr, `GO_VERSION="1.24.4"`) {
			t.Error("posix shim missing GO_VERSION")
		}
	})

	// Verify Windows CMD shim content.
	t.Run("windows content", func(t *testing.T) {
		t.Parallel()
		content, err := os.ReadFile(filepath.Join(tmpDir, "pok.cmd"))
		if err != nil {
			t.Fatalf("reading windows shim: %v", err)
		}
		contentStr := string(content)

		if !strings.Contains(contentStr, `set "POK_DIR=.pocket"`) {
			t.Error("windows shim missing POK_DIR")
		}
		if !strings.Contains(contentStr, `set "POK_CONTEXT=."`) {
			t.Error("windows shim missing POK_CONTEXT")
		}
	})

	// Verify PowerShell shim content.
	t.Run("powershell content", func(t *testing.T) {
		t.Parallel()
		content, err := os.ReadFile(filepath.Join(tmpDir, "pok.ps1"))
		if err != nil {
			t.Fatalf("reading powershell shim: %v", err)
		}
		contentStr := string(content)

		if !strings.Contains(contentStr, `$PocketDir = ".pocket"`) {
			t.Error("powershell shim missing PocketDir")
		}
		if !strings.Contains(contentStr, `$PocketContext = "."`) {
			t.Error("powershell shim missing PocketContext")
		}
		if !strings.Contains(contentStr, `$GoVersion = "1.24.4"`) {
			t.Error("powershell shim missing GoVersion")
		}
	})
}

func TestGenerateWithRoot_MultiModule(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory with go.mod.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket dir: %v", err)
	}
	gomod := "module pocket\n\ngo 1.24.4\n"
	if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// Create module directories.
	moduleDirs := []string{"services/api", "services/web", "libs/common"}
	for _, dir := range moduleDirs {
		if err := os.MkdirAll(filepath.Join(tmpDir, dir), 0o755); err != nil {
			t.Fatalf("creating directory %s: %v", dir, err)
		}
	}

	// Create a PathFilter that includes the module directories.
	// This simulates what AutoDetect would produce.
	mockTask := &pocket.Task{Name: "mock", Usage: "mock task"}
	pathFilter := pocket.Paths(mockTask).In(moduleDirs...)

	cfg := pocket.Config{
		Run: pathFilter,
		Shim: &pocket.ShimConfig{
			Name:       "pok",
			Posix:      true,
			Windows:    true,
			PowerShell: true,
		},
	}

	if err := GenerateWithRoot(cfg, tmpDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Verify shims at each location with correct content.
	tests := []struct {
		shimPath      string
		wantPocketDir string
		wantContext   string
		isPosix       bool
		isWindows     bool
		isPowerShell  bool
	}{
		// Root shims
		{"pok", ".pocket", ".", true, false, false},
		{"pok.cmd", ".pocket", ".", false, true, false},
		{"pok.ps1", ".pocket", ".", false, false, true},
		// services/api shims
		{"services/api/pok", "../../.pocket", "services/api", true, false, false},
		{"services/api/pok.cmd", "../../.pocket", "services/api", false, true, false},
		{"services/api/pok.ps1", "../../.pocket", "services/api", false, false, true},
		// services/web shims
		{"services/web/pok", "../../.pocket", "services/web", true, false, false},
		{"services/web/pok.cmd", "../../.pocket", "services/web", false, true, false},
		{"services/web/pok.ps1", "../../.pocket", "services/web", false, false, true},
		// libs/common shims
		{"libs/common/pok", "../../.pocket", "libs/common", true, false, false},
		{"libs/common/pok.cmd", "../../.pocket", "libs/common", false, true, false},
		{"libs/common/pok.ps1", "../../.pocket", "libs/common", false, false, true},
	}

	for _, tt := range tests {
		t.Run(tt.shimPath, func(t *testing.T) {
			t.Parallel()
			fullPath := filepath.Join(tmpDir, tt.shimPath)
			content, err := os.ReadFile(fullPath)
			if err != nil {
				t.Fatalf("reading shim %q: %v", tt.shimPath, err)
			}
			contentStr := string(content)

			switch {
			case tt.isPosix:
				if !strings.Contains(contentStr, `POK_DIR="`+tt.wantPocketDir+`"`) {
					t.Errorf("posix shim %q: expected POK_DIR=%q", tt.shimPath, tt.wantPocketDir)
				}
				if !strings.Contains(contentStr, `POK_CONTEXT="`+tt.wantContext+`"`) {
					t.Errorf("posix shim %q: expected POK_CONTEXT=%q", tt.shimPath, tt.wantContext)
				}
			case tt.isWindows:
				if !strings.Contains(contentStr, `set "POK_DIR=`+tt.wantPocketDir+`"`) {
					t.Errorf("windows shim %q: expected POK_DIR=%q", tt.shimPath, tt.wantPocketDir)
				}
				if !strings.Contains(contentStr, `set "POK_CONTEXT=`+tt.wantContext+`"`) {
					t.Errorf("windows shim %q: expected POK_CONTEXT=%q", tt.shimPath, tt.wantContext)
				}
			case tt.isPowerShell:
				if !strings.Contains(contentStr, `$PocketDir = "`+tt.wantPocketDir+`"`) {
					t.Errorf("powershell shim %q: expected PocketDir=%q", tt.shimPath, tt.wantPocketDir)
				}
				if !strings.Contains(contentStr, `$PocketContext = "`+tt.wantContext+`"`) {
					t.Errorf("powershell shim %q: expected PocketContext=%q", tt.shimPath, tt.wantContext)
				}
			}
		})
	}
}

func TestGenerateWithRoot_DeeplyNested(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()

	// Create .pocket directory with go.mod.
	pocketDir := filepath.Join(tmpDir, ".pocket")
	if err := os.MkdirAll(pocketDir, 0o755); err != nil {
		t.Fatalf("creating .pocket dir: %v", err)
	}
	gomod := "module pocket\n\ngo 1.24.4\n"
	if err := os.WriteFile(filepath.Join(pocketDir, "go.mod"), []byte(gomod), 0o644); err != nil {
		t.Fatalf("writing go.mod: %v", err)
	}

	// Create deeply nested directory.
	deepDir := "a/b/c/d/e"
	if err := os.MkdirAll(filepath.Join(tmpDir, deepDir), 0o755); err != nil {
		t.Fatalf("creating deep directory: %v", err)
	}

	// Create a PathFilter for the deep directory.
	mockTask := &pocket.Task{Name: "mock", Usage: "mock task"}
	pathFilter := pocket.Paths(mockTask).In(deepDir)

	cfg := pocket.Config{
		Run: pathFilter,
		Shim: &pocket.ShimConfig{
			Name:  "pok",
			Posix: true,
		},
	}

	if err := GenerateWithRoot(cfg, tmpDir); err != nil {
		t.Fatalf("Generate: %v", err)
	}

	// Verify shim at deep location.
	shimPath := filepath.Join(tmpDir, deepDir, "pok")
	content, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatalf("reading shim: %v", err)
	}

	// Should have 5 levels of "../" to get back to root.
	wantPocketDir := "../../../../../.pocket"
	if !strings.Contains(string(content), `POK_DIR="`+wantPocketDir+`"`) {
		t.Errorf("deeply nested shim: expected POK_DIR=%q, got content:\n%s", wantPocketDir, content)
	}
	if !strings.Contains(string(content), `POK_CONTEXT="`+deepDir+`"`) {
		t.Errorf("deeply nested shim: expected POK_CONTEXT=%q", deepDir)
	}
}
