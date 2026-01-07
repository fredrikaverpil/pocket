package pocket

import (
	"reflect"
	"testing"

	"github.com/goyek/goyek/v3"
)

func TestConfig_UniqueModulePaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
		want   []string
	}{
		{
			name:   "empty config always includes root",
			config: Config{},
			want:   []string{"."},
		},
		{
			name: "go modules only",
			config: Config{
				Go: &GoConfig{
					Modules: map[string]GoModuleOptions{
						".":     {},
						"tests": {},
					},
				},
			},
			want: []string{".", "tests"},
		},
		{
			name: "lua modules only",
			config: Config{
				Lua: &LuaConfig{
					Modules: map[string]LuaModuleOptions{
						".":       {},
						"scripts": {},
					},
				},
			},
			want: []string{".", "scripts"},
		},
		{
			name: "markdown modules only",
			config: Config{
				Markdown: &MarkdownConfig{
					Modules: map[string]MarkdownModuleOptions{
						".":    {},
						"docs": {},
					},
				},
			},
			want: []string{".", "docs"},
		},
		{
			name: "custom tasks only",
			config: Config{
				Custom: map[string][]goyek.Task{
					".":      nil,
					"deploy": nil,
				},
			},
			want: []string{".", "deploy"},
		},
		{
			name: "multiple language configs with duplicates",
			config: Config{
				Go: &GoConfig{
					Modules: map[string]GoModuleOptions{
						".":     {},
						"tests": {},
					},
				},
				Lua: &LuaConfig{
					Modules: map[string]LuaModuleOptions{
						".":       {},
						"scripts": {},
						"tests":   {}, // Duplicate with Go.
					},
				},
				Markdown: &MarkdownConfig{
					Modules: map[string]MarkdownModuleOptions{
						".":    {},
						"docs": {},
					},
				},
			},
			want: []string{".", "docs", "scripts", "tests"},
		},
		{
			name: "all config types combined",
			config: Config{
				Go: &GoConfig{
					Modules: map[string]GoModuleOptions{
						".": {},
					},
				},
				Lua: &LuaConfig{
					Modules: map[string]LuaModuleOptions{
						"lua": {},
					},
				},
				Markdown: &MarkdownConfig{
					Modules: map[string]MarkdownModuleOptions{
						"docs": {},
					},
				},
				Custom: map[string][]goyek.Task{
					"deploy": nil,
				},
			},
			want: []string{".", "deploy", "docs", "lua"},
		},
		{
			name: "nested paths",
			config: Config{
				Go: &GoConfig{
					Modules: map[string]GoModuleOptions{
						".":     {},
						"a/b":   {},
						"a/b/c": {},
						"x":     {},
					},
				},
			},
			want: []string{".", "a/b", "a/b/c", "x"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.UniqueModulePaths()
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("UniqueModulePaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_ForContext(t *testing.T) {
	t.Parallel()

	baseConfig := Config{
		Shim: &ShimConfig{Name: "mybld", Posix: true},
		Go: &GoConfig{
			Modules: map[string]GoModuleOptions{
				".":     {SkipFormat: false},
				"tests": {SkipLint: true},
				"cmd":   {SkipTest: true},
			},
		},
		Lua: &LuaConfig{
			Modules: map[string]LuaModuleOptions{
				".":       {},
				"scripts": {SkipFormat: true},
			},
		},
		Markdown: &MarkdownConfig{
			Modules: map[string]MarkdownModuleOptions{
				".":    {},
				"docs": {},
			},
		},
		Custom: map[string][]goyek.Task{
			".":      nil,
			"deploy": nil,
		},
	}

	tests := []struct {
		name       string
		context    string
		wantHasGo  bool
		wantHasLua bool
		wantHasMd  bool
		wantCustom bool
	}{
		{
			name:       "root context returns full config",
			context:    ".",
			wantHasGo:  true,
			wantHasLua: true,
			wantHasMd:  true,
			wantCustom: true,
		},
		{
			name:       "tests context has only Go",
			context:    "tests",
			wantHasGo:  true,
			wantHasLua: false,
			wantHasMd:  false,
			wantCustom: false,
		},
		{
			name:       "scripts context has only Lua",
			context:    "scripts",
			wantHasGo:  false,
			wantHasLua: true,
			wantHasMd:  false,
			wantCustom: false,
		},
		{
			name:       "docs context has only Markdown",
			context:    "docs",
			wantHasGo:  false,
			wantHasLua: false,
			wantHasMd:  true,
			wantCustom: false,
		},
		{
			name:       "deploy context has only Custom",
			context:    "deploy",
			wantHasGo:  false,
			wantHasLua: false,
			wantHasMd:  false,
			wantCustom: true,
		},
		{
			name:       "non-existent context",
			context:    "nonexistent",
			wantHasGo:  false,
			wantHasLua: false,
			wantHasMd:  false,
			wantCustom: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := baseConfig.ForContext(tt.context)

			// Verify Shim config is preserved.
			if got.Shim == nil || got.Shim.Name != baseConfig.Shim.Name {
				gotName := ""
				if got.Shim != nil {
					gotName = got.Shim.Name
				}
				t.Errorf("ForContext(%q).Shim.Name = %q, want %q", tt.context, gotName, baseConfig.Shim.Name)
			}

			// Verify Go config.
			if got.HasGo() != tt.wantHasGo {
				t.Errorf("ForContext(%q).HasGo() = %v, want %v", tt.context, got.HasGo(), tt.wantHasGo)
			}

			// Verify Lua config.
			if got.HasLua() != tt.wantHasLua {
				t.Errorf("ForContext(%q).HasLua() = %v, want %v", tt.context, got.HasLua(), tt.wantHasLua)
			}

			// Verify Markdown config.
			if got.HasMarkdown() != tt.wantHasMd {
				t.Errorf("ForContext(%q).HasMarkdown() = %v, want %v", tt.context, got.HasMarkdown(), tt.wantHasMd)
			}

			// Verify Custom config.
			hasCustom := len(got.Custom) > 0
			if hasCustom != tt.wantCustom {
				t.Errorf("ForContext(%q) has custom = %v, want %v", tt.context, hasCustom, tt.wantCustom)
			}
		})
	}
}

func TestConfig_ForContext_PreservesModuleOptions(t *testing.T) {
	t.Parallel()

	config := Config{
		Go: &GoConfig{
			Modules: map[string]GoModuleOptions{
				"tests": {
					SkipFormat:    true,
					SkipTest:      false,
					SkipLint:      true,
					SkipVulncheck: false,
				},
			},
		},
	}

	filtered := config.ForContext("tests")

	if !filtered.HasGo() {
		t.Fatal("ForContext(tests).HasGo() = false, want true")
	}

	opts, ok := filtered.Go.Modules["tests"]
	if !ok {
		t.Fatal("ForContext(tests).Go.Modules[tests] not found")
	}

	if !opts.SkipFormat {
		t.Error("SkipFormat = false, want true")
	}
	if opts.SkipTest {
		t.Error("SkipTest = true, want false")
	}
	if !opts.SkipLint {
		t.Error("SkipLint = false, want true")
	}
	if opts.SkipVulncheck {
		t.Error("SkipVulncheck = true, want false")
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       Config
		wantShimName string
	}{
		{
			name:         "empty config gets default shim name",
			config:       Config{},
			wantShimName: "pok",
		},
		{
			name: "custom shim name is preserved",
			config: Config{
				Shim: &ShimConfig{Name: "build", Posix: true},
			},
			wantShimName: "build",
		},
		{
			name: "all custom values are preserved",
			config: Config{
				Shim: &ShimConfig{Name: "mybld", Posix: true},
			},
			wantShimName: "mybld",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.WithDefaults()

			if got.Shim == nil || got.Shim.Name != tt.wantShimName {
				gotName := ""
				if got.Shim != nil {
					gotName = got.Shim.Name
				}
				t.Errorf("WithDefaults().Shim.Name = %q, want %q", gotName, tt.wantShimName)
			}
		})
	}
}

func TestConfig_HasGo(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{
			name:   "nil Go config",
			config: Config{},
			want:   false,
		},
		{
			name: "empty Go modules",
			config: Config{
				Go: &GoConfig{},
			},
			want: false,
		},
		{
			name: "has Go modules",
			config: Config{
				Go: &GoConfig{
					Modules: map[string]GoModuleOptions{
						".": {},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.config.HasGo(); got != tt.want {
				t.Errorf("HasGo() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_HasLua(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{
			name:   "nil Lua config",
			config: Config{},
			want:   false,
		},
		{
			name: "empty Lua modules",
			config: Config{
				Lua: &LuaConfig{},
			},
			want: false,
		},
		{
			name: "has Lua modules",
			config: Config{
				Lua: &LuaConfig{
					Modules: map[string]LuaModuleOptions{
						".": {},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.config.HasLua(); got != tt.want {
				t.Errorf("HasLua() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_HasMarkdown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
		want   bool
	}{
		{
			name:   "nil Markdown config",
			config: Config{},
			want:   false,
		},
		{
			name: "empty Markdown modules",
			config: Config{
				Markdown: &MarkdownConfig{},
			},
			want: false,
		},
		{
			name: "has Markdown modules",
			config: Config{
				Markdown: &MarkdownConfig{
					Modules: map[string]MarkdownModuleOptions{
						".": {},
					},
				},
			},
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := tt.config.HasMarkdown(); got != tt.want {
				t.Errorf("HasMarkdown() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GoModulesForFormat(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
		want   []string
	}{
		{
			name:   "nil Go config",
			config: Config{},
			want:   nil,
		},
		{
			name: "all modules included",
			config: Config{
				Go: &GoConfig{
					Modules: map[string]GoModuleOptions{
						".":     {},
						"tests": {},
					},
				},
			},
			want: []string{".", "tests"},
		},
		{
			name: "some modules skipped",
			config: Config{
				Go: &GoConfig{
					Modules: map[string]GoModuleOptions{
						".":     {},
						"tests": {SkipFormat: true},
						"cmd":   {},
					},
				},
			},
			want: []string{".", "cmd"},
		},
		{
			name: "all modules skipped",
			config: Config{
				Go: &GoConfig{
					Modules: map[string]GoModuleOptions{
						".":     {SkipFormat: true},
						"tests": {SkipFormat: true},
					},
				},
			},
			want: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.GoModulesForFormat()
			// Sort for comparison since map iteration order is not guaranteed.
			if !equalStringSlicesUnordered(got, tt.want) {
				t.Errorf("GoModulesForFormat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_GoModulesForTest(t *testing.T) {
	t.Parallel()

	config := Config{
		Go: &GoConfig{
			Modules: map[string]GoModuleOptions{
				".":     {},
				"tests": {SkipTest: true},
				"cmd":   {},
			},
		},
	}

	got := config.GoModulesForTest()
	want := []string{".", "cmd"}

	if !equalStringSlicesUnordered(got, want) {
		t.Errorf("GoModulesForTest() = %v, want %v", got, want)
	}
}

func TestConfig_GoModulesForLint(t *testing.T) {
	t.Parallel()

	config := Config{
		Go: &GoConfig{
			Modules: map[string]GoModuleOptions{
				".":     {},
				"tests": {SkipLint: true},
				"cmd":   {},
			},
		},
	}

	got := config.GoModulesForLint()
	want := []string{".", "cmd"}

	if !equalStringSlicesUnordered(got, want) {
		t.Errorf("GoModulesForLint() = %v, want %v", got, want)
	}
}

func TestConfig_GoModulesForVulncheck(t *testing.T) {
	t.Parallel()

	config := Config{
		Go: &GoConfig{
			Modules: map[string]GoModuleOptions{
				".":     {},
				"tests": {SkipVulncheck: true},
				"cmd":   {},
			},
		},
	}

	got := config.GoModulesForVulncheck()
	want := []string{".", "cmd"}

	if !equalStringSlicesUnordered(got, want) {
		t.Errorf("GoModulesForVulncheck() = %v, want %v", got, want)
	}
}

// equalStringSlicesUnordered compares two string slices ignoring order.
func equalStringSlicesUnordered(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	if len(a) == 0 && len(b) == 0 {
		return true
	}
	counts := make(map[string]int)
	for _, s := range a {
		counts[s]++
	}
	for _, s := range b {
		counts[s]--
		if counts[s] < 0 {
			return false
		}
	}
	return true
}
