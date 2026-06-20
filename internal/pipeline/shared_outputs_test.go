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
			Formats: []string{"md", "json"}, // per-book writes are skipped; aggregate spans all formats
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

	// Both books' aggregate files should exist under the unified tree (md).
	aggMd := filepath.Join(dir, "unified", "en", "unified", "md")
	for _, rel := range []string{
		filepath.Join("feature", "ability", "fury", "level-1", "brutal-slam.md"),
		filepath.Join("feature", "ability", "beastheart", "level-6", "sic-em.md"),
	} {
		if _, err := os.Stat(filepath.Join(aggMd, rel)); os.IsNotExist(err) {
			t.Errorf("expected aggregate md file %s to exist", rel)
		}
	}

	// The aggregate now spans all formats: a json aggregate file must exist too.
	aggJSON := filepath.Join(dir, "unified", "en", "unified", "json",
		"feature", "ability", "fury", "level-1", "brutal-slam.json")
	if _, err := os.Stat(aggJSON); os.IsNotExist(err) {
		t.Error("expected all-format aggregate to write a json file")
	}
}
