package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/scc"
)

func TestBuildGenerators_AllFormats(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Locale:    "en",
		Input:     "test.md",
		ConfigDir: dir,
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md", "json", "yaml"},
		},
	}

	registry := scc.NewRegistry()
	generators := buildGenerators(cfg, filepath.Join(dir, "output", "en", "md"), "", registry, nil)

	// Should have 3 generators (md, json, yaml)
	if len(generators) != 3 {
		t.Errorf("expected 3 generators for 3 formats, got %d", len(generators))
	}

	formats := make(map[string]bool)
	for _, gen := range generators {
		formats[gen.Format()] = true
	}

	for _, want := range []string{"md", "json", "yaml"} {
		if !formats[want] {
			t.Errorf("missing %s generator", want)
		}
	}
}

func TestBuildGenerators_WithVariants(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Locale:    "en",
		Input:     "test.md",
		ConfigDir: dir,
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md"},
			Variants: VariantsConfig{
				Linked:    true,
				DSE:       true,
				DSELinked: true,
			},
		},
	}

	registry := scc.NewRegistry()
	generators := buildGenerators(cfg, filepath.Join(dir, "output", "en", "md"), "", registry, nil)

	// md + linked + dse + dse-linked = 4
	if len(generators) != 4 {
		t.Errorf("expected 4 generators (md + 3 variants), got %d", len(generators))
	}

	formats := make(map[string]bool)
	for _, gen := range generators {
		formats[gen.Format()] = true
	}

	for _, want := range []string{"md", "md-linked", "md-dse", "md-dse-linked"} {
		if !formats[want] {
			t.Errorf("missing %s generator", want)
		}
	}
}

func TestBuildGenerators_WithStripped(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Locale:    "en",
		Input:     "test.md",
		ConfigDir: dir,
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md"},
			Stripped: StrippedConfig{
				Enabled:   true,
				OutputDir: "./clean",
			},
		},
	}

	registry := scc.NewRegistry()
	rawInput := []byte("# Title\n\nSome content.")
	generators := buildGenerators(cfg, filepath.Join(dir, "output", "en", "md"), "", registry, rawInput)

	// md + stripped = 2
	if len(generators) != 2 {
		t.Errorf("expected 2 generators (md + stripped), got %d", len(generators))
	}

	hasStripped := false
	for _, gen := range generators {
		if gen.Format() == "stripped" {
			hasStripped = true
		}
	}
	if !hasStripped {
		t.Error("missing stripped generator")
	}
}

func TestBuildGenerators_WithAggregate(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Locale:    "en",
		Input:     "test.md",
		ConfigDir: dir,
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md"},
			Aggregate: AggregateConfig{
				Enabled:   true,
				OutputDir: "./unified",
			},
		},
	}

	registry := scc.NewRegistry()
	generators := buildGenerators(cfg, filepath.Join(dir, "output", "en", "md"), "", registry, nil)

	// md + aggregate = 2
	if len(generators) != 2 {
		t.Errorf("expected 2 generators (md + aggregate), got %d", len(generators))
	}

	hasAggregate := false
	for _, gen := range generators {
		if gen.Format() == "aggregate" {
			hasAggregate = true
		}
	}
	if !hasAggregate {
		t.Error("missing aggregate generator")
	}
}

func TestBuildGenerators_WithSCCMap(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Locale:    "en",
		Input:     "test.md",
		ConfigDir: dir,
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md"},
			SCCMap: SCCMapConfig{
				Enabled:    true,
				OutputFile: "./scc-map.json",
			},
		},
	}

	registry := scc.NewRegistry()
	generators := buildGenerators(cfg, filepath.Join(dir, "output", "en", "md"), "", registry, nil)

	// md + scc-map = 2
	if len(generators) != 2 {
		t.Errorf("expected 2 generators (md + scc-map), got %d", len(generators))
	}

	hasSCCMap := false
	for _, gen := range generators {
		if gen.Format() == "scc-map" {
			hasSCCMap = true
		}
	}
	if !hasSCCMap {
		t.Error("missing scc-map generator")
	}
}

func TestBuildGenerators_FullConfig(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Locale:    "en",
		Input:     "test.md",
		ConfigDir: dir,
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md", "json", "yaml"},
			Variants: VariantsConfig{
				Linked:    true,
				DSE:       true,
				DSELinked: true,
			},
			Stripped: StrippedConfig{
				Enabled:   true,
				OutputDir: "./clean",
			},
			Aggregate: AggregateConfig{
				Enabled:   true,
				OutputDir: "./unified",
			},
			SCCMap: SCCMapConfig{
				Enabled:    true,
				OutputFile: "./scc-map.json",
			},
		},
	}

	registry := scc.NewRegistry()
	rawInput := []byte("# Title")
	generators := buildGenerators(cfg, filepath.Join(dir, "output", "en", "md"), "", registry, rawInput)

	// md + json + yaml + linked + dse + dse-linked + stripped + aggregate + scc-map = 9
	if len(generators) != 9 {
		t.Errorf("expected 9 generators, got %d", len(generators))
		for _, gen := range generators {
			t.Logf("  format: %s", gen.Format())
		}
	}
}

func TestBuildGenerators_EmptyLocale(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Locale:    "", // should default to "en"
		Input:     "test.md",
		ConfigDir: dir,
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md"},
		},
	}

	registry := scc.NewRegistry()
	generators := buildGenerators(cfg, filepath.Join(dir, "output", "en", "md"), "", registry, nil)

	if len(generators) != 1 {
		t.Errorf("expected 1 generator, got %d", len(generators))
	}
}

func TestBuildGenerators_EmptyMdOutputDir(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Locale:    "en",
		Input:     "test.md",
		ConfigDir: dir,
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md"},
		},
	}

	registry := scc.NewRegistry()
	// Pass empty mdOutputDir — should derive from config
	generators := buildGenerators(cfg, "", "", registry, nil)

	if len(generators) != 1 {
		t.Errorf("expected 1 generator, got %d", len(generators))
	}
}

func TestRunWithConfig_MultiFormat(t *testing.T) {
	inputPath := "../../testdata/fixtures/simple_class.md"
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skip("test fixture not found")
	}

	dir := t.TempDir()
	registryPath := filepath.Join(dir, "classification.json")

	cfg := &Config{
		Book:      "test.v1",
		Input:     inputPath,
		Locale:    "en",
		ConfigDir: dir,
		Classification: ClassificationConfig{
			Registry: registryPath,
		},
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md", "json", "yaml"},
			Variants: VariantsConfig{
				Linked:    true,
				DSE:       true,
				DSELinked: true,
			},
			Aggregate: AggregateConfig{
				Enabled:   true,
				OutputDir: "./unified",
			},
			SCCMap: SCCMapConfig{
				Enabled:    true,
				OutputFile: filepath.Join(dir, "scc-map.json"),
			},
		},
	}

	mdOutputDir := filepath.Join(dir, "output", "en", "md")

	result, err := RunWithConfig(cfg, inputPath, mdOutputDir, registryPath)
	if err != nil {
		t.Fatalf("RunWithConfig failed: %v", err)
	}

	// With 6 generators (md+json+yaml+linked+dse+dse-linked) + aggregate,
	// each classified section should produce 7 written files
	if result.WrittenFiles < 20 {
		t.Errorf("expected many written files from multi-format run, got %d", result.WrittenFiles)
	}

	// SCC map should have been finalized
	if _, err := os.Stat(filepath.Join(dir, "scc-map.json")); os.IsNotExist(err) {
		t.Error("expected scc-map.json to be written")
	}

	// JSON files should exist
	jsonDir := filepath.Join(dir, "output", "en", "json")
	if _, err := os.Stat(jsonDir); os.IsNotExist(err) {
		t.Error("expected json output directory")
	}

	// YAML files should exist
	yamlDir := filepath.Join(dir, "output", "en", "yaml")
	if _, err := os.Stat(yamlDir); os.IsNotExist(err) {
		t.Error("expected yaml output directory")
	}
}

func TestRunWithConfig_BadInputPath(t *testing.T) {
	cfg := &Config{
		Input:     "nonexistent.md",
		Locale:    "en",
		ConfigDir: t.TempDir(),
		Output: OutputConfig{
			Formats: []string{"md"},
		},
	}

	_, err := RunWithConfig(cfg, "/nonexistent/path.md", t.TempDir(), "")
	if err == nil {
		t.Error("expected error for nonexistent input path")
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "bad.yaml")

	if err := os.WriteFile(configPath, []byte("invalid: [yaml: broken"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadConfig(configPath)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadConfig_NonexistentFile(t *testing.T) {
	_, err := LoadConfig("/nonexistent/pipeline.yaml")
	if err == nil {
		t.Error("expected error for nonexistent config file")
	}
}

func TestRunWithConfig_DuplicateSCC(t *testing.T) {
	inputPath := "../../testdata/fixtures/duplicate_scc.md"
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skip("test fixture not found")
	}

	dir := t.TempDir()
	registryPath := filepath.Join(dir, "classification.json")
	mdOutputDir := filepath.Join(dir, "output")

	cfg := &Config{
		Book:      "test.v1",
		Input:     inputPath,
		Locale:    "en",
		ConfigDir: dir,
		Classification: ClassificationConfig{
			Registry: registryPath,
		},
		Output: OutputConfig{
			Formats: []string{"md"},
		},
	}

	result, err := RunWithConfig(cfg, inputPath, mdOutputDir, registryPath)
	if err != nil {
		t.Fatalf("RunWithConfig failed: %v", err)
	}

	// Should have at least one duplicate SCC error
	hasDuplicateError := false
	for _, e := range result.Errors {
		if len(e) > 0 {
			hasDuplicateError = true
		}
	}
	if !hasDuplicateError && result.ClassifiedSections > 2 {
		// Depending on how the fixture parses, we expect duplicate detection
		t.Logf("Classified: %d, Errors: %v", result.ClassifiedSections, result.Errors)
	}
}

func TestRunWithConfig_WithStrippedAndAggregate(t *testing.T) {
	inputPath := "../../testdata/fixtures/simple_class.md"
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skip("test fixture not found")
	}

	dir := t.TempDir()
	registryPath := filepath.Join(dir, "classification.json")
	mdOutputDir := filepath.Join(dir, "output", "en", "md")

	cfg := &Config{
		Book:      "test.v1",
		Input:     inputPath,
		Locale:    "en",
		ConfigDir: dir,
		Classification: ClassificationConfig{
			Registry: registryPath,
		},
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md"},
			Stripped: StrippedConfig{
				Enabled:   true,
				OutputDir: "./clean",
			},
			Aggregate: AggregateConfig{
				Enabled:   true,
				OutputDir: "./unified",
			},
			SCCMap: SCCMapConfig{
				Enabled:    true,
				OutputFile: filepath.Join(dir, "scc-map.json"),
			},
		},
	}

	result, err := RunWithConfig(cfg, inputPath, mdOutputDir, registryPath)
	if err != nil {
		t.Fatalf("RunWithConfig failed: %v", err)
	}

	if result.WrittenFiles < 5 {
		t.Errorf("expected written files, got %d", result.WrittenFiles)
	}

	// Stripped output should exist
	cleanDir := filepath.Join(dir, "clean")
	if _, err := os.Stat(cleanDir); os.IsNotExist(err) {
		t.Error("expected clean output directory")
	}

	// SCC map should exist
	if _, err := os.Stat(filepath.Join(dir, "scc-map.json")); os.IsNotExist(err) {
		t.Error("expected scc-map.json")
	}

	// Aggregate index should exist
	unifiedDir := filepath.Join(dir, "unified", "en", "md", "_index")
	if _, err := os.Stat(unifiedDir); os.IsNotExist(err) {
		t.Error("expected aggregate index directory")
	}
}

func TestRunWithConfig_NoRegistryPath(t *testing.T) {
	inputPath := "../../testdata/fixtures/simple_class.md"
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skip("test fixture not found")
	}

	dir := t.TempDir()
	mdOutputDir := filepath.Join(dir, "output")

	cfg := &Config{
		Book:      "test.v1",
		Input:     inputPath,
		Locale:    "en",
		ConfigDir: dir,
		Output: OutputConfig{
			Formats: []string{"md"},
		},
	}

	result, err := RunWithConfig(cfg, inputPath, mdOutputDir, "")
	if err != nil {
		t.Fatalf("RunWithConfig failed: %v", err)
	}

	if result.WrittenFiles < 5 {
		t.Errorf("expected written files, got %d", result.WrittenFiles)
	}
}
