package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// A coded descendant heading on an aggregate page must carry its own SCC code
// as a data-scc attr_list marker in the rendered PageBody, so the v2 client can
// offer a stable /scc/<code>/ permalink on that heading. Assertions are
// structural (presence + suffix), NOT exact codes — the classifier's TypePath/
// ItemID for a given type are an implementation detail this test must not pin.
//
// Uses the md-linked generator (the only format that emits PageBody) over the
// existing simple_class fixture, whose Fury class subtree contains coded
// abilities (Brutal Slam, Gouge) and feature/feature-group headings.
func TestPageBody_SubheadingsCarryDataSCC(t *testing.T) {
	inputPath := "../../testdata/fixtures/simple_class.md"
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

	// md-linked output lands at <baseDir>/<locale>/md-linked/...
	furyPath := filepath.Join(baseDir, "en", "md-linked", "class", "fury.md")
	data, err := os.ReadFile(furyPath)
	if err != nil {
		t.Fatalf("read fury md-linked page: %v", err)
	}
	body := string(data)

	if !strings.Contains(body, `data-scc="`) {
		t.Errorf("fury PageBody carries no data-scc markers:\n%s", body)
	}
	if !strings.Contains(body, `Gouge {data-scc="`) {
		t.Errorf("Gouge heading missing data-scc marker:\n%s", body)
	}
	if !strings.Contains(body, `/gouge"}`) {
		t.Errorf("Gouge data-scc value should end in /gouge:\n%s", body)
	}
}
