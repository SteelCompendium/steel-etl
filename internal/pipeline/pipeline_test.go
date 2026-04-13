package pipeline

import (
	"os"
	"path/filepath"
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

func checkFileExists(t *testing.T, dir, relPath string) {
	t.Helper()
	fullPath := filepath.Join(dir, relPath)
	if _, err := os.Stat(fullPath); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", relPath)
	}
}
