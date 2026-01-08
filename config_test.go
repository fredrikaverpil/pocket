package pocket

import (
	"reflect"
	"testing"
)

func TestConfig_TaskPaths(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		config Config
		want   []string
	}{
		{
			name:   "empty config returns empty",
			config: Config{},
			want:   []string{},
		},
		{
			name: "tasks paths",
			config: Config{
				Tasks: map[string][]*Task{
					".":      nil,
					"deploy": nil,
				},
			},
			want: []string{".", "deploy"},
		},
		{
			name: "nested tasks paths",
			config: Config{
				Tasks: map[string][]*Task{
					".":     nil,
					"a/b":   nil,
					"a/b/c": nil,
				},
			},
			want: []string{".", "a/b", "a/b/c"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.TaskPaths()
			if !equalStringSlicesUnordered(got, tt.want) {
				t.Errorf("TaskPaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConfig_ForContext(t *testing.T) {
	t.Parallel()

	baseConfig := Config{
		Shim:        &ShimConfig{Name: "mypok", Posix: true},
		SkipGitDiff: true,
		Tasks: map[string][]*Task{
			".":      nil,
			"deploy": nil,
			"tests":  nil,
		},
	}

	tests := []struct {
		name      string
		context   string
		wantTasks bool
	}{
		{
			name:      "root context returns full config",
			context:   ".",
			wantTasks: true,
		},
		{
			name:      "deploy context has tasks",
			context:   "deploy",
			wantTasks: true,
		},
		{
			name:      "tests context has tasks",
			context:   "tests",
			wantTasks: true,
		},
		{
			name:      "non-existent context",
			context:   "nonexistent",
			wantTasks: false,
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

			// Verify SkipGitDiff is preserved.
			if got.SkipGitDiff != baseConfig.SkipGitDiff {
				t.Errorf(
					"ForContext(%q).SkipGitDiff = %v, want %v",
					tt.context,
					got.SkipGitDiff,
					baseConfig.SkipGitDiff,
				)
			}

			// Verify Tasks config.
			hasTasks := len(got.Tasks) > 0
			if hasTasks != tt.wantTasks {
				t.Errorf("ForContext(%q) has tasks = %v, want %v", tt.context, hasTasks, tt.wantTasks)
			}

			// For root context, verify all tasks are included.
			if tt.context == "." {
				if len(got.Tasks) != len(baseConfig.Tasks) {
					t.Errorf("ForContext(.) tasks count = %d, want %d", len(got.Tasks), len(baseConfig.Tasks))
				}
			}
		})
	}
}

func TestConfig_WithDefaults(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name         string
		config       Config
		wantShimName string
		wantPosix    bool
	}{
		{
			name:         "empty config gets default shim name and posix",
			config:       Config{},
			wantShimName: "pok",
			wantPosix:    true,
		},
		{
			name: "custom shim name is preserved",
			config: Config{
				Shim: &ShimConfig{Name: "build", Posix: true},
			},
			wantShimName: "build",
			wantPosix:    true,
		},
		{
			name: "empty shim name gets default",
			config: Config{
				Shim: &ShimConfig{Posix: true},
			},
			wantShimName: "pok",
			wantPosix:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.config.WithDefaults()

			if got.Shim == nil {
				t.Fatal("WithDefaults().Shim is nil")
			}
			if got.Shim.Name != tt.wantShimName {
				t.Errorf("WithDefaults().Shim.Name = %q, want %q", got.Shim.Name, tt.wantShimName)
			}
		})
	}
}

func TestBaseModuleConfig_ShouldRun(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		opts BaseModuleConfig
		task string
		want bool
	}{
		{
			name: "empty options runs all",
			opts: BaseModuleConfig{},
			task: "format",
			want: true,
		},
		{
			name: "skip excludes task",
			opts: BaseModuleConfig{Skip: []string{"format"}},
			task: "format",
			want: false,
		},
		{
			name: "skip allows other tasks",
			opts: BaseModuleConfig{Skip: []string{"format"}},
			task: "test",
			want: true,
		},
		{
			name: "only includes specified task",
			opts: BaseModuleConfig{Only: []string{"format", "test"}},
			task: "format",
			want: true,
		},
		{
			name: "only excludes unspecified task",
			opts: BaseModuleConfig{Only: []string{"format", "test"}},
			task: "lint",
			want: false,
		},
		{
			name: "only takes precedence over skip",
			opts: BaseModuleConfig{Only: []string{"format"}, Skip: []string{"format"}},
			task: "format",
			want: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := tt.opts.ShouldRun(tt.task)
			if got != tt.want {
				t.Errorf("ShouldRun(%q) = %v, want %v", tt.task, got, tt.want)
			}
		})
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

func TestAllModulePaths(t *testing.T) {
	t.Parallel()

	// Create a mock task group for testing.
	mockTG := &mockTaskGroup{
		name: "test",
		modules: map[string]BaseModuleConfig{
			".":     {},
			"tests": {},
		},
	}

	tests := []struct {
		name string
		cfg  Config
		want []string
	}{
		{
			name: "empty config returns root",
			cfg:  Config{},
			want: []string{"."},
		},
		{
			name: "task group paths included",
			cfg: Config{
				TaskGroups: []TaskGroup{mockTG},
			},
			want: []string{".", "tests"},
		},
		{
			name: "tasks paths included",
			cfg: Config{
				Tasks: map[string][]*Task{
					"deploy": nil,
				},
			},
			want: []string{".", "deploy"},
		},
		{
			name: "task group and tasks paths combined",
			cfg: Config{
				TaskGroups: []TaskGroup{mockTG},
				Tasks: map[string][]*Task{
					"deploy": nil,
				},
			},
			want: []string{".", "deploy", "tests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := AllModulePaths(tt.cfg)
			if !equalStringSlicesUnordered(got, tt.want) {
				t.Errorf("AllModulePaths() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestModulesFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		modules map[string]BaseModuleConfig
		task    string
		want    []string
	}{
		{
			name:    "nil modules",
			modules: nil,
			task:    "format",
			want:    nil,
		},
		{
			name: "all modules included",
			modules: map[string]BaseModuleConfig{
				".":     {},
				"tests": {},
			},
			task: "format",
			want: []string{".", "tests"},
		},
		{
			name: "some modules skipped",
			modules: map[string]BaseModuleConfig{
				".":     {},
				"tests": {Skip: []string{"format"}},
				"cmd":   {},
			},
			task: "format",
			want: []string{".", "cmd"},
		},
		{
			name: "all modules skipped",
			modules: map[string]BaseModuleConfig{
				".":     {Skip: []string{"format"}},
				"tests": {Skip: []string{"format"}},
			},
			task: "format",
			want: nil,
		},
		{
			name: "only filter",
			modules: map[string]BaseModuleConfig{
				".":     {Only: []string{"test"}},
				"tests": {Only: []string{"format", "test"}},
			},
			task: "format",
			want: []string{"tests"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			tg := &mockTaskGroup{name: "test", modules: tt.modules}
			got := ModulesFor(tg, tt.task)
			if !equalStringSlicesUnordered(got, tt.want) {
				t.Errorf("ModulesFor(%q) = %v, want %v", tt.task, got, tt.want)
			}
		})
	}
}

func TestFilterTaskGroupsForContext(t *testing.T) {
	t.Parallel()

	tg1 := &mockTaskGroup{
		name: "tg1",
		modules: map[string]BaseModuleConfig{
			".":     {},
			"tests": {},
		},
	}
	tg2 := &mockTaskGroup{
		name: "tg2",
		modules: map[string]BaseModuleConfig{
			".":       {},
			"scripts": {},
		},
	}

	tests := []struct {
		name        string
		taskGroups  []TaskGroup
		context     string
		wantTGNames []string
	}{
		{
			name:        "root returns all",
			taskGroups:  []TaskGroup{tg1, tg2},
			context:     ".",
			wantTGNames: []string{"tg1", "tg2"},
		},
		{
			name:        "tests returns only tg1",
			taskGroups:  []TaskGroup{tg1, tg2},
			context:     "tests",
			wantTGNames: []string{"tg1"},
		},
		{
			name:        "scripts returns only tg2",
			taskGroups:  []TaskGroup{tg1, tg2},
			context:     "scripts",
			wantTGNames: []string{"tg2"},
		},
		{
			name:        "nonexistent returns none",
			taskGroups:  []TaskGroup{tg1, tg2},
			context:     "nonexistent",
			wantTGNames: []string{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := FilterTaskGroupsForContext(tt.taskGroups, tt.context)
			gotNames := make([]string, len(got))
			for i, tg := range got {
				gotNames[i] = tg.Name()
			}
			if !reflect.DeepEqual(gotNames, tt.wantTGNames) {
				t.Errorf("FilterTaskGroupsForContext() names = %v, want %v", gotNames, tt.wantTGNames)
			}
		})
	}
}

// mockTaskGroup is a simple TaskGroup implementation for testing.
type mockTaskGroup struct {
	name    string
	modules map[string]BaseModuleConfig
}

func (tg *mockTaskGroup) Name() string { return tg.name }

func (tg *mockTaskGroup) DetectModules() []string {
	paths := make([]string, 0, len(tg.modules))
	for k := range tg.modules {
		paths = append(paths, k)
	}
	return paths
}

func (tg *mockTaskGroup) ModuleConfigs() map[string]ModuleConfig {
	result := make(map[string]ModuleConfig, len(tg.modules))
	for k, v := range tg.modules {
		result[k] = v
	}
	return result
}

func (tg *mockTaskGroup) Tasks(_ Config) []*Task { return nil }

func (tg *mockTaskGroup) ForContext(context string) TaskGroup {
	if context == "." {
		return tg
	}
	if opts, ok := tg.modules[context]; ok {
		return &mockTaskGroup{
			name:    tg.name,
			modules: map[string]BaseModuleConfig{context: opts},
		}
	}
	return nil
}
