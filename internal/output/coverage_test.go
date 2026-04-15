package output

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// Format() method coverage for all generators
func TestAllGeneratorFormats(t *testing.T) {
	tests := []struct {
		name   string
		gen    Generator
		format string
	}{
		{"MarkdownGenerator", &MarkdownGenerator{}, "md"},
		{"JSONGenerator", &JSONGenerator{}, "json"},
		{"YAMLGenerator", &YAMLGenerator{}, "yaml"},
		{"DSEGenerator", &DSEGenerator{}, "md-dse"},
		{"AggregateGenerator", &AggregateGenerator{}, "aggregate"},
		{"SCCMapGenerator", &SCCMapGenerator{}, "scc-map"},
		{"StrippedGenerator", &StrippedGenerator{}, "stripped"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.gen.Format(); got != tt.format {
				t.Errorf("%s.Format() = %q, want %q", tt.name, got, tt.format)
			}
		})
	}
}

// dirOf edge cases
func TestDirOf(t *testing.T) {
	tests := []struct {
		path string
		want string
	}{
		{"/home/user/file.txt", "/home/user"},
		{"relative/path/file.txt", "relative/path"},
		{"file.txt", "."},
		{"dir/", "dir"},
		{"/root", ""},
	}

	for _, tt := range tests {
		got := dirOf(tt.path)
		if got != tt.want {
			t.Errorf("dirOf(%q) = %q, want %q", tt.path, got, tt.want)
		}
	}
}

// getStringOr fallback path
func TestGetStringOr(t *testing.T) {
	m := map[string]any{
		"present": "value",
		"empty":   "",
		"number":  42,
	}

	tests := []struct {
		key      string
		fallback string
		want     string
	}{
		{"present", "default", "value"},
		{"missing", "default", "default"},
		{"empty", "default", "default"},   // empty string falls through to default
		{"number", "default", "default"},  // non-string falls through to default
	}

	for _, tt := range tests {
		got := getStringOr(m, tt.key, tt.fallback)
		if got != tt.want {
			t.Errorf("getStringOr(%q, %q) = %q, want %q", tt.key, tt.fallback, got, tt.want)
		}
	}
}

// SCCToFilePath edge cases
func TestSCCToFilePath_EdgeCases(t *testing.T) {
	tests := []struct {
		scc  string
		ext  string
		want string
	}{
		// Single component (no slash)
		{"nosource", ".md", "unknown.md"},
		// Empty string
		{"", ".md", "unknown.md"},
	}

	for _, tt := range tests {
		got := SCCToFilePath(tt.scc, tt.ext)
		if got != tt.want {
			t.Errorf("SCCToFilePath(%q, %q) = %q, want %q", tt.scc, tt.ext, got, tt.want)
		}
	}
}

// copyFrontmatter returns a shallow copy, not the same map
func TestCopyFrontmatter(t *testing.T) {
	src := map[string]any{"a": "1", "b": "2"}
	dst := copyFrontmatter(src)

	if len(dst) != 2 {
		t.Errorf("expected 2 entries, got %d", len(dst))
	}

	// Modify the copy, original should not change
	dst["c"] = "3"
	if _, ok := src["c"]; ok {
		t.Error("modifying copy should not affect source")
	}
}

// buildEffects edge cases
func TestBuildEffects_BodyOnly(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Test",
			"type": "ability",
		},
		Body: "Just a body, no structured effects.",
	}

	effects := buildEffects(parsed)
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect from body, got %d", len(effects))
	}
	if effects[0]["effect"] != "Just a body, no structured effects." {
		t.Error("expected body as fallback effect")
	}
}

func TestBuildEffects_NoEffectsNoBody(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{"name": "Test"},
	}

	effects := buildEffects(parsed)
	if len(effects) != 0 {
		t.Errorf("expected 0 effects for empty parsed, got %d", len(effects))
	}
}

func TestBuildEffects_SpendOnly(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":  "Test",
			"spend": "The target is dazed until end of turn.",
		},
	}

	effects := buildEffects(parsed)
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect for spend, got %d", len(effects))
	}
	if effects[0]["name"] != "Spend" {
		t.Error("expected spend effect name")
	}
}

func TestBuildEffects_AllFields(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"effect":                    "Main effect text.",
			"power_roll_characteristic": "Might",
			"tier1":                     "4 damage",
			"tier2":                     "7 damage",
			"tier3":                     "10 damage",
			"spend":                     "Extra spend effect.",
		},
		Body: "Body text (should be ignored since structured effects exist).",
	}

	effects := buildEffects(parsed)
	if len(effects) != 3 {
		t.Fatalf("expected 3 effects (effect + power roll + spend), got %d", len(effects))
	}

	// First: main effect
	if effects[0]["effect"] != "Main effect text." {
		t.Error("expected main effect")
	}
	// Second: power roll
	if effects[1]["roll"] != "Power Roll + Might" {
		t.Error("expected power roll")
	}
	if effects[1]["tier1"] != "4 damage" {
		t.Error("expected tier1")
	}
	// Third: spend
	if effects[2]["name"] != "Spend" {
		t.Error("expected spend effect")
	}
}

// buildDSEFrontmatter edge cases
func TestBuildDSEFrontmatter_TraitType(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Growing Ferocity",
			"type": "trait",
		},
		ItemID: "growing-ferocity",
	}

	fm := buildDSEFrontmatter("mcdm.heroes.v1/feature.trait.fury.level-1/growing-ferocity", parsed)

	if fm["feature_type"] != "trait" {
		t.Errorf("expected feature_type=trait, got %v", fm["feature_type"])
	}
	if fm["action_type"] != "feature" {
		t.Errorf("traits should have action_type=feature, got %v", fm["action_type"])
	}
	if fm["source"] != "mcdm.heroes.v1" {
		t.Errorf("expected source=mcdm.heroes.v1, got %v", fm["source"])
	}
}

func TestBuildDSEFrontmatter_NoCost(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "Dazed",
			"type": "condition",
		},
		ItemID: "dazed",
	}

	fm := buildDSEFrontmatter("mcdm.heroes.v1/condition/dazed", parsed)

	// Conditions should not have feature_type or cost fields
	if _, ok := fm["feature_type"]; ok {
		t.Error("conditions should not have feature_type")
	}
	if _, ok := fm["cost_amount"]; ok {
		t.Error("conditions should not have cost_amount")
	}
}

// buildDSFeatureBlock with various fields
func TestBuildDSFeatureBlock_WithOptionalFields(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name":        "Test Ability",
			"type":        "ability",
			"cost":        "3 Ferocity",
			"flavor":      "A flavorful description.",
			"keywords":    []string{"Melee", "Strike"},
			"action_type": "Main action",
			"distance":    "Melee 1",
			"target":      "One creature",
			"trigger":     "When an enemy moves adjacent",
		},
		Body: "Body text.",
	}

	out, err := buildDSFeatureBlock(parsed)
	if err != nil {
		t.Fatalf("buildDSFeatureBlock failed: %v", err)
	}

	if out == "" {
		t.Fatal("expected non-empty output")
	}
}

// writeFile helper test
func TestWriteFile(t *testing.T) {
	dir := t.TempDir()
	err := writeFile(dir, "mcdm.heroes.v1/class/fury", ".json", []byte(`{"name":"Fury"}`))
	if err != nil {
		t.Fatalf("writeFile failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dir, "class", "fury.json"))
	if err != nil {
		t.Fatalf("failed to read written file: %v", err)
	}

	if string(data) != `{"name":"Fury"}` {
		t.Errorf("unexpected content: %s", data)
	}
}

// AggregateGenerator nil/empty guard paths
func TestAggregateGenerator_NilAndEmpty(t *testing.T) {
	gen := &AggregateGenerator{BaseDir: t.TempDir()}

	if err := gen.WriteSection("some/code", nil); err != nil {
		t.Errorf("expected nil error for nil parsed, got %v", err)
	}
	if err := gen.WriteSection("", &content.ParsedContent{}); err != nil {
		t.Errorf("expected nil error for empty SCC, got %v", err)
	}
}

func TestAggregateGenerator_FinalizeEmpty(t *testing.T) {
	gen := &AggregateGenerator{BaseDir: t.TempDir()}

	// Finalize with no sections should be a no-op
	if err := gen.Finalize(); err != nil {
		t.Errorf("expected nil error for empty finalize, got %v", err)
	}
}

// SCCMapGenerator nil/empty guard
func TestSCCMapGenerator_NilAndEmpty(t *testing.T) {
	gen := &SCCMapGenerator{OutputPath: filepath.Join(t.TempDir(), "map.json")}

	if err := gen.WriteSection("some/code", nil); err != nil {
		t.Errorf("expected nil error for nil parsed, got %v", err)
	}
	if err := gen.WriteSection("", &content.ParsedContent{}); err != nil {
		t.Errorf("expected nil error for empty SCC, got %v", err)
	}
}

// DSEGenerator nil/empty guard
func TestDSEGenerator_NilAndEmpty(t *testing.T) {
	gen := &DSEGenerator{BaseDir: t.TempDir()}

	if err := gen.WriteSection("some/code", nil); err != nil {
		t.Errorf("expected nil error for nil parsed, got %v", err)
	}
	if err := gen.WriteSection("", &content.ParsedContent{}); err != nil {
		t.Errorf("expected nil error for empty SCC, got %v", err)
	}
}

// BuildMarkdownFile with empty body
func TestBuildMarkdownFile_EmptyBody(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{"name": "Test"},
		Body:        "",
	}

	out, err := BuildMarkdownFile(parsed)
	if err != nil {
		t.Fatalf("BuildMarkdownFile failed: %v", err)
	}

	// Should have frontmatter but no body content after the closing ---
	if out == "" {
		t.Fatal("expected non-empty output")
	}
}

// StrippedGenerator with empty raw input
func TestStrippedGenerator_EmptyInput(t *testing.T) {
	dir := t.TempDir()
	outputPath := filepath.Join(dir, "clean", "empty.md")

	gen := &StrippedGenerator{
		OutputPath: outputPath,
		RawInput:   []byte(""),
	}

	if err := gen.Finalize(); err != nil {
		t.Fatalf("Finalize with empty input failed: %v", err)
	}

	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output: %v", err)
	}

	if len(data) != 0 {
		t.Errorf("expected empty output, got %q", string(data))
	}
}

// parseCost edge case: empty string
func TestParseCost_Empty(t *testing.T) {
	amount, resource := parseCost("")
	if amount != "" || resource != "" {
		t.Errorf("parseCost(\"\") = (%q, %q), want (\"\", \"\")", amount, resource)
	}
}
