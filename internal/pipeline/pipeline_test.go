package pipeline

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunPipelineOnFixture(t *testing.T) {
	inputPath := "../../testdata/fixtures/simple_class.md"
	outputDir := t.TempDir()
	registryPath := filepath.Join(t.TempDir(), "classification.json")

	result, err := Run(inputPath, outputDir, registryPath)
	if err != nil {
		t.Fatalf("Pipeline.Run failed: %v", err)
	}

	t.Logf("Total: %d, Parsed: %d, Skipped: %d, Classified: %d, Written: %d, Errors: %d",
		result.TotalSections, result.ParsedSections, result.SkippedSections,
		result.ClassifiedSections, result.WrittenFiles, len(result.Errors))

	for _, e := range result.Errors {
		t.Logf("  Error: %s", e)
	}

	if len(result.Errors) > 0 {
		t.Errorf("expected no errors, got %d", len(result.Errors))
	}

	// Should have written files for: chapter(1) + class(2) + trait(1) + ability(3) = 7
	if result.WrittenFiles < 5 {
		t.Errorf("expected at least 5 written files, got %d", result.WrittenFiles)
	}

	// Verify some output files exist (new paths: source/type/item)
	checkFileExists(t, outputDir, "chapter/classes.md")
	checkFileExists(t, outputDir, "class/fury.md")
	checkFileExists(t, outputDir, "class/shadow.md")
	checkFileExists(t, outputDir, "feature/ability/fury/level-1/brutal-slam.md")
	checkFileExists(t, outputDir, "feature/ability/fury/level-1/gouge.md")
	checkFileExists(t, outputDir, "feature/trait/fury/level-1/growing-ferocity.md")

	// Verify registry was written
	if _, err := os.Stat(registryPath); os.IsNotExist(err) {
		t.Error("expected classification.json to be written")
	}
}

func TestRunPipelineOnRealDocument(t *testing.T) {
	inputPath := "../../input/heroes/Draw Steel Heroes.md"
	if _, err := os.Stat(inputPath); os.IsNotExist(err) {
		t.Skipf("skipping: %v", err)
	}

	outputDir := t.TempDir()
	registryPath := filepath.Join(t.TempDir(), "classification.json")

	result, err := Run(inputPath, outputDir, registryPath)
	if err != nil {
		t.Fatalf("Pipeline.Run failed: %v", err)
	}

	t.Logf("Total: %d, Parsed: %d, Skipped: %d, Classified: %d, Written: %d, Errors: %d",
		result.TotalSections, result.ParsedSections, result.SkippedSections,
		result.ClassifiedSections, result.WrittenFiles, len(result.Errors))

	if len(result.Errors) > 10 {
		for i, e := range result.Errors {
			if i >= 10 {
				break
			}
			t.Logf("  Error: %s", e)
		}
		t.Errorf("too many errors: %d", len(result.Errors))
	}

	// Should have written 1000+ files
	if result.WrittenFiles < 900 {
		t.Errorf("expected 900+ written files, got %d", result.WrittenFiles)
	}

	// Spot-check some key files (new paths)
	checkFileExists(t, outputDir, "class/fury.md")
	checkFileExists(t, outputDir, "feature/ability/fury/level-1/brutal-slam.md")
	checkFileExists(t, outputDir, "chapter/classes.md")
}

func TestRunPipeline_SubheadingContentIncluded(t *testing.T) {
	inputPath := "../../testdata/fixtures/feature_with_subheadings.md"
	outputDir := t.TempDir()
	registryPath := filepath.Join(t.TempDir(), "classification.json")

	result, err := Run(inputPath, outputDir, registryPath)
	if err != nil {
		t.Fatalf("Pipeline.Run failed: %v", err)
	}

	t.Logf("Total: %d, Parsed: %d, Skipped: %d, Classified: %d, Written: %d, Errors: %d",
		result.TotalSections, result.ParsedSections, result.SkippedSections,
		result.ClassifiedSections, result.WrittenFiles, len(result.Errors))

	for _, e := range result.Errors {
		t.Logf("  Error: %s", e)
	}

	// --- Growing Ferocity should include both unannotated tables ---
	growingFerocityPath := filepath.Join(outputDir, "feature/trait/fury/level-1/growing-ferocity.md")
	growingFerocity := readFileContent(t, growingFerocityPath)

	if !strings.Contains(growingFerocity, "Berserker Growing Ferocity Table") {
		t.Error("Growing Ferocity output should contain 'Berserker Growing Ferocity Table' sub-heading")
	}
	if !strings.Contains(growingFerocity, "Knockback bonus equal to Might score.") {
		t.Error("Growing Ferocity output should contain Berserker table content")
	}
	if !strings.Contains(growingFerocity, "Reaver Growing Ferocity Table") {
		t.Error("Growing Ferocity output should contain 'Reaver Growing Ferocity Table' sub-heading")
	}
	if !strings.Contains(growingFerocity, "Agility") {
		t.Error("Growing Ferocity output should contain Reaver table content")
	}
	// Own content should also be present
	if !strings.Contains(growingFerocity, "You gain certain benefits") {
		t.Error("Growing Ferocity output should contain its own body text")
	}

	// --- 1st-Level Aspect Features should include the lookup table ---
	aspectFeaturesPath := filepath.Join(outputDir, "feature/trait/fury/level-1/1st-level-aspect-features.md")
	aspectFeatures := readFileContent(t, aspectFeaturesPath)

	if !strings.Contains(aspectFeatures, "1st-Level Aspect Features Table") {
		t.Error("Aspect Features output should contain the features table sub-heading")
	}
	if !strings.Contains(aspectFeatures, "Berserker") && !strings.Contains(aspectFeatures, "Kit, Primordial Strength") {
		t.Error("Aspect Features output should contain the features table content")
	}

	// --- Class (Fury) should include unannotated Basics + Advancement Table ---
	furyPath := filepath.Join(outputDir, "class/fury.md")
	fury := readFileContent(t, furyPath)

	if !strings.Contains(fury, "Basics") {
		t.Error("Fury class output should contain unannotated 'Basics' sub-section")
	}
	if !strings.Contains(fury, "Starting Characteristics") {
		t.Error("Fury class output should contain Basics content")
	}
	if !strings.Contains(fury, "Fury Advancement Table") {
		t.Error("Fury class output should contain the Advancement Table")
	}

	// --- Damaging Ferocity (level 2) should include its additions table ---
	damagingPath := filepath.Join(outputDir, "feature/trait/fury/level-2/damaging-ferocity.md")
	damaging := readFileContent(t, damagingPath)

	if !strings.Contains(damaging, "Damaging Ferocity Additions") {
		t.Error("Damaging Ferocity output should contain 'Damaging Ferocity Additions' table heading")
	}
	if !strings.Contains(damaging, "2 surges") {
		t.Error("Damaging Ferocity output should contain table content")
	}

	// --- Ability (Brutal Slam) should NOT contain other sections' content ---
	brutalSlamPath := filepath.Join(outputDir, "feature/ability/fury/level-1/brutal-slam.md")
	brutalSlam := readFileContent(t, brutalSlamPath)

	if !strings.Contains(brutalSlam, "Power Roll + Might") {
		t.Error("Brutal Slam should contain its own power roll")
	}
	if strings.Contains(brutalSlam, "Growing Ferocity") {
		t.Error("Brutal Slam should NOT contain Growing Ferocity content")
	}
}

func checkFileExists(t *testing.T, dir, relPath string) {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", relPath)
	}
}

func readFileContent(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read output file %s: %v", path, err)
	}
	return string(data)
}
