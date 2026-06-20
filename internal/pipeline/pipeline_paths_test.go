package pipeline

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/output"
	"github.com/SteelCompendium/steel-etl/internal/scc"
)

// baseDirOf extracts the BaseDir field from the known generator types.
func baseDirOf(g output.Generator) string {
	switch t := g.(type) {
	case *output.MarkdownGenerator:
		return t.BaseDir
	case *output.JSONGenerator:
		return t.BaseDir
	case *output.YAMLGenerator:
		return t.BaseDir
	case *output.LinkedGenerator:
		return t.BaseDir
	case *output.DSEGenerator:
		return t.BaseDir
	case *output.DSELinkedGenerator:
		return t.BaseDir
	}
	return ""
}

func TestPerBookPathsIncludeBooksSlug(t *testing.T) {
	cfg := &Config{
		Book:      "mcdm.heroes.v1",
		Locale:    "en",
		ConfigDir: "/repo/steel-etl",
		Output: OutputConfig{
			BaseDir:  "../data/data-unified",
			Dir:      "heroes",
			Formats:  []string{"md", "json", "yaml"},
			Variants: VariantsConfig{Linked: true, DSE: true, DSELinked: true},
		},
	}
	mdOut := filepath.Join(cfg.BookOutputDir("en"), "md")
	gens := buildGenerators(cfg, mdOut, "", scc.NewRegistry(), nil)

	wantSuffixes := map[string]bool{
		filepath.Join("en", "books", "heroes", "md"):            false,
		filepath.Join("en", "books", "heroes", "json"):          false,
		filepath.Join("en", "books", "heroes", "yaml"):          false,
		filepath.Join("en", "books", "heroes", "md-linked"):     false,
		filepath.Join("en", "books", "heroes", "md-dse"):        false,
		filepath.Join("en", "books", "heroes", "md-dse-linked"): false,
	}
	for _, g := range gens {
		bd := baseDirOf(g)
		for suf := range wantSuffixes {
			if strings.HasSuffix(bd, suf) {
				wantSuffixes[suf] = true
			}
		}
	}
	for suf, seen := range wantSuffixes {
		if !seen {
			t.Errorf("no generator BaseDir ended with %q", suf)
		}
	}
}
