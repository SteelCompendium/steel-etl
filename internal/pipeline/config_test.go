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

func TestResolveInputPath_EnglishLocale(t *testing.T) {
	cfg := &Config{
		Input:     "./input/heroes/Draw Steel Heroes.md",
		Locale:    "en",
		I18nDir:   "./input/i18n",
		ConfigDir: t.TempDir(),
	}

	got := cfg.ResolveInputPath()
	want := cfg.ResolvePath(cfg.Input)
	if got != want {
		t.Errorf("English locale should use default input\ngot:  %s\nwant: %s", got, want)
	}
}

func TestResolveInputPath_NoI18nDir(t *testing.T) {
	cfg := &Config{
		Input:     "./input/heroes/Draw Steel Heroes.md",
		Locale:    "es",
		I18nDir:   "", // not set
		ConfigDir: t.TempDir(),
	}

	got := cfg.ResolveInputPath()
	want := cfg.ResolvePath(cfg.Input)
	if got != want {
		t.Errorf("No i18n_dir should fall back to default input\ngot:  %s\nwant: %s", got, want)
	}
}

func TestResolveInputPath_LocaleFileExists(t *testing.T) {
	dir := t.TempDir()

	// Create the locale-specific input file
	i18nDir := filepath.Join(dir, "input", "i18n", "es")
	if err := os.MkdirAll(i18nDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(i18nDir, "Draw Steel Heroes.md"), []byte("# Héroes"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Input:     "./input/heroes/Draw Steel Heroes.md",
		Locale:    "es",
		I18nDir:   "./input/i18n",
		ConfigDir: dir,
	}

	got := cfg.ResolveInputPath()
	want := filepath.Join(dir, "input", "i18n", "es", "Draw Steel Heroes.md")
	if got != want {
		t.Errorf("Should resolve to locale-specific input\ngot:  %s\nwant: %s", got, want)
	}
}

func TestResolveInputPath_LocaleFileMissing(t *testing.T) {
	dir := t.TempDir()

	// Create i18n dir but no locale file
	if err := os.MkdirAll(filepath.Join(dir, "input", "i18n", "fr"), 0755); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{
		Input:     "./input/heroes/Draw Steel Heroes.md",
		Locale:    "fr",
		I18nDir:   "./input/i18n",
		ConfigDir: dir,
	}

	got := cfg.ResolveInputPath()
	want := cfg.ResolvePath(cfg.Input)
	if got != want {
		t.Errorf("Missing locale file should fall back to default\ngot:  %s\nwant: %s", got, want)
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

func baseMultiBookConfig() *Config {
	cfg := &Config{
		Book:      "mcdm.heroes.v1",
		Input:     "./input/heroes/Draw Steel Heroes.md",
		Locale:    "en",
		ConfigDir: "/proj",
	}
	cfg.Output.BaseDir = "../data/data-rules"
	cfg.Output.Formats = []string{"md", "json"}
	cfg.Output.Variants = VariantsConfig{Linked: true, DSE: true}
	cfg.Output.Aggregate = AggregateConfig{Enabled: true, OutputDir: "../data/data-unified"}
	cfg.Output.SCCAPI = SCCAPIConfig{Enabled: true, OutputDir: "../data/api"}
	cfg.Output.SCCMap = SCCMapConfig{Enabled: true, OutputFile: "./scc.json"}
	cfg.Output.Stripped = StrippedConfig{Enabled: true, OutputDir: "../data/clean"}
	return cfg
}

func TestEffectiveBookConfig_InheritsAndOverridesIdentity(t *testing.T) {
	cfg := baseMultiBookConfig()
	b := BookConfig{
		Book:  "mcdm.monsters.v1",
		Input: "./input/monsters/Draw Steel Monsters.md",
	}

	eff := cfg.EffectiveBookConfig(b)

	if eff.Book != "mcdm.monsters.v1" {
		t.Errorf("Book = %q, want mcdm.monsters.v1", eff.Book)
	}
	if eff.Input != b.Input {
		t.Errorf("Input = %q, want %q", eff.Input, b.Input)
	}
	// Inherits base output base_dir + formats when the book overrides nothing.
	if eff.Output.BaseDir != "../data/data-rules" {
		t.Errorf("BaseDir = %q, want inherited ../data/data-rules", eff.Output.BaseDir)
	}
	if got, want := eff.Output.Formats, []string{"md", "json"}; !equalStrings(got, want) {
		t.Errorf("Formats = %v, want inherited %v", got, want)
	}
	// Variants are inherited.
	if !eff.Output.Variants.Linked || !eff.Output.Variants.DSE {
		t.Errorf("Variants not inherited: %+v", eff.Output.Variants)
	}
}

func TestEffectiveBookConfig_DisablesSharedOutputs(t *testing.T) {
	cfg := baseMultiBookConfig()
	eff := cfg.EffectiveBookConfig(BookConfig{Book: "mcdm.monsters.v1", Input: "x.md"})

	if eff.Output.Aggregate.Enabled {
		t.Error("Aggregate should be disabled for a secondary book")
	}
	if eff.Output.SCCAPI.Enabled {
		t.Error("SCCAPI should be disabled for a secondary book")
	}
	if eff.Output.SCCMap.Enabled {
		t.Error("SCCMap should be disabled for a secondary book")
	}
	if eff.Output.Stripped.Enabled {
		t.Error("Stripped should be disabled for a secondary book")
	}
	// The base config must NOT be mutated by deriving a book config.
	if !cfg.Output.Aggregate.Enabled || !cfg.Output.SCCAPI.Enabled {
		t.Error("base config shared outputs were mutated by EffectiveBookConfig")
	}
}

func TestEffectiveBookConfig_BookOverridesBaseDirAndFormats(t *testing.T) {
	cfg := baseMultiBookConfig()
	b := BookConfig{
		Book:  "mcdm.monsters.v1",
		Input: "x.md",
	}
	b.Output.BaseDir = "../data/data-bestiary"
	b.Output.Formats = []string{"yaml"}

	eff := cfg.EffectiveBookConfig(b)

	if eff.Output.BaseDir != "../data/data-bestiary" {
		t.Errorf("BaseDir = %q, want overridden ../data/data-bestiary", eff.Output.BaseDir)
	}
	if got, want := eff.Output.Formats, []string{"yaml"}; !equalStrings(got, want) {
		t.Errorf("Formats = %v, want overridden %v", got, want)
	}
	// Overriding the derived config's formats must not leak into the base config.
	if got, want := cfg.Output.Formats, []string{"md", "json"}; !equalStrings(got, want) {
		t.Errorf("base Formats mutated = %v, want %v", got, want)
	}
}

func TestBookOutputDir(t *testing.T) {
	cfg := &Config{
		ConfigDir: "/repo/steel-etl",
		Output:    OutputConfig{BaseDir: "../data/data-unified", Dir: "heroes"},
	}
	got := cfg.BookOutputDir("en")
	want := "/repo/data/data-unified/en/books/heroes"
	if got != want {
		t.Fatalf("BookOutputDir = %q, want %q", got, want)
	}
}

func TestEffectiveBookConfigCarriesDir(t *testing.T) {
	base := &Config{
		ConfigDir: "/repo/steel-etl",
		Output:    OutputConfig{BaseDir: "../data/data-unified", Dir: "heroes", Formats: []string{"md"}},
	}
	eff := base.EffectiveBookConfig(BookConfig{
		Book:   "mcdm.monsters.v1",
		Input:  "./input/monsters/x.md",
		Output: OutputConfig{BaseDir: "../data/data-unified", Dir: "monsters"},
	})
	if eff.Output.Dir != "monsters" {
		t.Fatalf("eff.Output.Dir = %q, want monsters", eff.Output.Dir)
	}
}

func equalStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}
