package pipeline

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pipeline.yaml")

	configContent := `
book: mcdm.heroes.v1
input: ./input/heroes/Draw Steel Heroes.md
locale: en

classification:
  registry: ./classification.json
  freeze: false

output:
  base_dir: ../data/data-rules
  formats:
    - md
    - json
    - yaml
  variants:
    linked: true
    dse: true
    dse_linked: true
  stripped:
    enabled: true
    output_dir: ../data/data-rules-clean
  aggregate:
    enabled: true
    output_dir: ../data/data-unified
  scc_map:
    enabled: true
    output_file: ./output/scc-to-path.json
`

	if err := os.WriteFile(configPath, []byte(configContent), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Book != "mcdm.heroes.v1" {
		t.Errorf("Book = %q, want mcdm.heroes.v1", cfg.Book)
	}
	if cfg.Locale != "en" {
		t.Errorf("Locale = %q, want en", cfg.Locale)
	}
	if len(cfg.Output.Formats) != 3 {
		t.Errorf("Formats count = %d, want 3", len(cfg.Output.Formats))
	}
	if !cfg.Output.Variants.Linked {
		t.Error("expected Variants.Linked = true")
	}
	if !cfg.Output.Variants.DSE {
		t.Error("expected Variants.DSE = true")
	}
	if !cfg.Output.Variants.DSELinked {
		t.Error("expected Variants.DSELinked = true")
	}
	if !cfg.Output.Stripped.Enabled {
		t.Error("expected Stripped.Enabled = true")
	}
	if !cfg.Output.Aggregate.Enabled {
		t.Error("expected Aggregate.Enabled = true")
	}
	if !cfg.Output.SCCMap.Enabled {
		t.Error("expected SCCMap.Enabled = true")
	}
}

func TestLoadConfigDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "pipeline.yaml")

	// Minimal config
	if err := os.WriteFile(configPath, []byte("book: test\ninput: test.md\n"), 0644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if cfg.Locale != "en" {
		t.Errorf("default locale = %q, want en", cfg.Locale)
	}
	if len(cfg.Output.Formats) != 1 || cfg.Output.Formats[0] != "md" {
		t.Errorf("default formats = %v, want [md]", cfg.Output.Formats)
	}
}

func TestConfigHasFormat(t *testing.T) {
	cfg := &Config{
		Output: OutputConfig{
			Formats: []string{"md", "json"},
		},
	}

	if !cfg.HasFormat("md") {
		t.Error("expected HasFormat(md) = true")
	}
	if !cfg.HasFormat("json") {
		t.Error("expected HasFormat(json) = true")
	}
	if cfg.HasFormat("yaml") {
		t.Error("expected HasFormat(yaml) = false")
	}
}

func TestConfigResolvePath(t *testing.T) {
	cfg := &Config{ConfigDir: "/home/user/project"}

	tests := []struct {
		input string
		want  string
	}{
		{"./input/test.md", "/home/user/project/input/test.md"},
		{"/absolute/path.md", "/absolute/path.md"},
		{"../data/output", "/home/user/data/output"},
	}

	for _, tt := range tests {
		got := cfg.ResolvePath(tt.input)
		if got != tt.want {
			t.Errorf("ResolvePath(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}
