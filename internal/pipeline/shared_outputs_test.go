package pipeline

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// RunSharedOutputs should regenerate the cross-book aggregate over the union of
// items from multiple books, so secondary-book entries land alongside the primary.
func TestRunSharedOutputs_SpansBooks(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Locale:    "en",
		ConfigDir: dir,
		Output: OutputConfig{
			BaseDir: "./output",
			Formats: []string{"md"}, // must be ignored by the shared pass
			Aggregate: AggregateConfig{
				Enabled:   true,
				OutputDir: "./unified",
			},
		},
	}

	items := []ClassifiedItem{
		{
			SCCCode: "mcdm.heroes.v1/feature.ability.fury.level-1/brutal-slam",
			Parsed: &content.ParsedContent{
				Frontmatter: map[string]any{"name": "Brutal Slam", "type": "ability"},
				Body:        "heroes body",
				TypePath:    []string{"feature", "ability", "fury", "level-1"},
				ItemID:      "brutal-slam",
			},
		},
		{
			SCCCode: "mcdm.beastheart.v1/feature.ability.beastheart.level-6/sic-em",
			Parsed: &content.ParsedContent{
				Frontmatter: map[string]any{"name": "Sic 'Em!", "type": "ability", "subclass": "guardian"},
				Body:        "beastheart body",
				TypePath:    []string{"feature", "ability", "beastheart", "level-6"},
				ItemID:      "sic-em",
			},
		},
	}

	if err := RunSharedOutputs(cfg, items); err != nil {
		t.Fatalf("RunSharedOutputs: %v", err)
	}

	// The shared pass must NOT emit per-book formats (md), only the shared aggregate.
	if _, err := os.Stat(filepath.Join(dir, "output", "en", "md")); !os.IsNotExist(err) {
		t.Error("shared pass should not write per-book md output")
	}

	// Both books' aggregate files should exist under the unified tree.
	aggBase := filepath.Join(dir, "unified", "en", "md")
	for _, rel := range []string{
		filepath.Join("feature", "ability", "fury", "level-1", "brutal-slam.md"),
		filepath.Join("feature", "ability", "beastheart", "level-6", "sic-em.md"),
	} {
		if _, err := os.Stat(filepath.Join(aggBase, rel)); os.IsNotExist(err) {
			t.Errorf("expected aggregate file %s to exist", rel)
		}
	}
}
