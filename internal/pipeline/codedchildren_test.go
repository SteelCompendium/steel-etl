package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A fixture advancement-features block's members are parser-emitted coded children;
// the pipeline must register each one's SCC code (the registry file records it)
// while the base/container codes are untouched.
func TestPipeline_FixtureCodedChildrenRegistered(t *testing.T) {
	inputPath := "../../testdata/fixtures/fixture_advancement.md"
	baseDir := t.TempDir()
	registryPath := filepath.Join(t.TempDir(), "classification.json")

	cfg := &Config{
		Input:  inputPath,
		Locale: "en",
		Output: OutputConfig{
			BaseDir:  baseDir,
			Variants: VariantsConfig{Linked: true},
		},
		Classification: ClassificationConfig{Registry: registryPath},
	}
	if _, err := RunWithConfig(cfg, inputPath, "", registryPath); err != nil {
		t.Fatalf("pipeline run: %v", err)
	}

	data, err := os.ReadFile(registryPath)
	if err != nil {
		t.Fatalf("read registry: %v", err)
	}
	reg := string(data)
	for _, code := range []string{
		"feature.fixture.demon.the-boil.level-5/soul-rancor",
		"feature.fixture.demon.the-boil.level-9/fester-field",
		"monster.fixture.demon.advancement-features/the-boil", // container unchanged
		"monster.fixture.demon.featureblock/the-boil",         // base unchanged
	} {
		if !strings.Contains(reg, code) {
			t.Errorf("registry missing expected code %q", code)
		}
	}
}
