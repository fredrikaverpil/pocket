package workflows

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/fredrikaverpil/bld"
)

// TemplateData is the data passed to workflow templates.
type TemplateData struct {
	GeneratedBy string

	// Go config
	GoModulesFormat    []string
	GoModulesTest      []string
	GoModulesLint      []string
	GoModulesVulncheck []string
	GoVersions         []string
	OSVersions         []string
	SkipGoFormat       bool
	SkipGoTest         bool
	SkipGoLint         bool
	SkipGoVulncheck    bool
}

// FromConfig creates TemplateData from a Config.
func FromConfig(cfg bld.Config) TemplateData {
	cfg = cfg.WithDefaults()
	data := TemplateData{
		GeneratedBy:        "bld",
		GoModulesFormat:    cfg.GoModulesForFormat(),
		GoModulesTest:      cfg.GoModulesForTest(),
		GoModulesLint:      cfg.GoModulesForLint(),
		GoModulesVulncheck: cfg.GoModulesForVulncheck(),
	}

	// Determine skips based on whether any modules are configured
	data.SkipGoFormat = len(data.GoModulesFormat) == 0
	data.SkipGoTest = len(data.GoModulesTest) == 0
	data.SkipGoLint = len(data.GoModulesLint) == 0
	data.SkipGoVulncheck = len(data.GoModulesVulncheck) == 0

	// Extract Go versions from all configured modules
	data.GoVersions = extractGoVersions(cfg)

	if cfg.GitHub != nil {
		data.OSVersions = cfg.GitHub.OSVersions
	}
	return data
}

// extractGoVersions collects Go versions from all module go.mod files
// and merges them with any extra versions from config.
func extractGoVersions(cfg bld.Config) []string {
	seen := make(map[string]bool)
	var versions []string

	// Extract versions from go.mod files
	if cfg.Go != nil {
		for path := range cfg.Go.Modules {
			version, err := bld.ExtractGoVersion(path)
			if err != nil {
				// Skip modules where we can't extract version
				continue
			}
			if !seen[version] {
				seen[version] = true
				versions = append(versions, version)
			}
		}
	}

	// Add extra versions from config
	if cfg.GitHub != nil {
		for _, v := range cfg.GitHub.ExtraGoVersions {
			if !seen[v] {
				seen[v] = true
				versions = append(versions, v)
			}
		}
	}

	return versions
}

// Generate generates GitHub Actions workflows from templates.
func Generate(cfg bld.Config) error {
	if cfg.GitHub == nil {
		return nil // No GitHub config, nothing to generate
	}

	cfg = cfg.WithDefaults()
	data := FromConfig(cfg)
	outDir := bld.FromGitRoot(".github", "workflows")

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return fmt.Errorf("create workflows dir: %w", err)
	}

	funcMap := template.FuncMap{
		"toJSON": func(v any) string {
			b, err := json.Marshal(v)
			if err != nil {
				panic(fmt.Sprintf("toJSON: failed to marshal %T: %v", v, err))
			}
			return string(b)
		},
	}

	// Generate Go workflow if we have Go modules with any enabled tasks
	if cfg.HasGo() && (!data.SkipGoFormat || !data.SkipGoTest || !data.SkipGoLint || !data.SkipGoVulncheck) {
		if err := generateFromTemplate(
			"github/golang/ci.yml.tmpl",
			filepath.Join(outDir, "bld-go.yml"),
			data,
			funcMap,
		); err != nil {
			return err
		}
	}

	// Generate generic workflows
	generics := []struct {
		tmpl string
		out  string
		skip bool
	}{
		{"github/generic/pr.yml.tmpl", "bld-pr.yml", cfg.GitHub.SkipPR},
		{"github/generic/stale.yml.tmpl", "bld-stale.yml", cfg.GitHub.SkipStale},
		{"github/generic/release.yml.tmpl", "bld-release.yml", cfg.GitHub.SkipRelease},
		{"github/generic/sync.yml.tmpl", "bld-sync.yml", cfg.GitHub.SkipSync},
	}

	for _, g := range generics {
		outPath := filepath.Join(outDir, g.out)
		if g.skip {
			// Remove if exists
			_ = os.Remove(outPath)
			continue
		}
		if err := generateFromTemplate(g.tmpl, outPath, data, funcMap); err != nil {
			return err
		}
	}

	return nil
}

func generateFromTemplate(tmplPath, outPath string, data TemplateData, funcMap template.FuncMap) error {
	content, err := FS.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("read template %s: %w", tmplPath, err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Funcs(funcMap).Parse(string(content))
	if err != nil {
		return fmt.Errorf("parse template %s: %w", tmplPath, err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return fmt.Errorf("execute template %s: %w", tmplPath, err)
	}

	if err := os.WriteFile(outPath, buf.Bytes(), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outPath, err)
	}

	return nil
}
