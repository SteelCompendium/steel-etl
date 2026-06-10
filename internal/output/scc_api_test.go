package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/site"
)

func TestSCCAPIGenerator(t *testing.T) {
	dir := t.TempDir()

	sections := []site.SectionConfig{
		{Name: "Browse", Include: []string{"class/", "feature/", "condition/", "kit/", "ancestry/"}},
		{Name: "Read", Include: []string{"chapter/"}},
	}

	gen := &SCCAPIGenerator{
		OutputDir: dir,
		BaseURL:   "https://steelcompendium.io/v2",
		Sections:  sections,
		Aliases:   map[string]string{"mcdm.heroes.v1/feature.ability.common/reactive-strike": "mcdm.heroes.v1/feature.ability.fury.level-1/reactive-strike"},
	}

	entries := []struct {
		scc    string
		parsed *content.ParsedContent
	}{
		{
			"mcdm.heroes.v1/class/fury",
			&content.ParsedContent{
				Frontmatter: map[string]any{"name": "Fury", "type": "class"},
				ItemID:      "fury",
			},
		},
		{
			"mcdm.heroes.v1/feature.ability.fury.level-1/gouge",
			&content.ParsedContent{
				Frontmatter: map[string]any{"name": "Gouge", "type": "ability"},
				ItemID:      "gouge",
			},
		},
		{
			"mcdm.heroes.v1/chapter/introduction",
			&content.ParsedContent{
				Frontmatter: map[string]any{"name": "Introduction", "type": "chapter"},
				ItemID:      "introduction",
			},
		},
		{
			"mcdm.heroes.v1/condition/dazed",
			&content.ParsedContent{
				Frontmatter: map[string]any{"name": "Dazed", "type": "condition"},
				ItemID:      "dazed",
			},
		},
	}

	for _, e := range entries {
		if err := gen.WriteSection(e.scc, e.parsed); err != nil {
			t.Fatalf("WriteSection %s: %v", e.scc, err)
		}
	}

	if err := gen.Finalize(); err != nil {
		t.Fatalf("Finalize: %v", err)
	}

	// Verify index.json
	t.Run("index.json", func(t *testing.T) {
		var idx apiIndex
		readJSON(t, filepath.Join(dir, "v1", "index.json"), &idx)

		if idx.Version != 1 {
			t.Errorf("version = %d, want 1", idx.Version)
		}
		if idx.TotalCodes != 4 {
			t.Errorf("total_codes = %d, want 4", idx.TotalCodes)
		}
		if idx.TotalAliases != 1 {
			t.Errorf("total_aliases = %d, want 1", idx.TotalAliases)
		}
		if idx.BaseURL != "https://steelcompendium.io/v2" {
			t.Errorf("base_url = %q", idx.BaseURL)
		}
		if idx.Endpoints.Registry != "/api/v1/scc.json" {
			t.Errorf("endpoints.registry = %q", idx.Endpoints.Registry)
		}
	})

	// Verify scc.json
	t.Run("scc.json", func(t *testing.T) {
		var reg apiRegistry
		readJSON(t, filepath.Join(dir, "v1", "scc.json"), &reg)

		if len(reg.Entries) != 4 {
			t.Fatalf("entries = %d, want 4", len(reg.Entries))
		}
		// Entries should be sorted
		if reg.Entries[0].SCC != "mcdm.heroes.v1/chapter/introduction" {
			t.Errorf("first entry = %q, want chapter/introduction", reg.Entries[0].SCC)
		}
		if len(reg.Aliases) != 1 {
			t.Errorf("aliases = %d, want 1", len(reg.Aliases))
		}
	})

	// Verify types.json
	t.Run("types.json", func(t *testing.T) {
		var types apiTypes
		readJSON(t, filepath.Join(dir, "v1", "types.json"), &types)

		if len(types.Types["class"]) != 1 {
			t.Errorf("class count = %d, want 1", len(types.Types["class"]))
		}
		if len(types.Types["chapter"]) != 1 {
			t.Errorf("chapter count = %d, want 1", len(types.Types["chapter"]))
		}
		if len(types.Types["condition"]) != 1 {
			t.Errorf("condition count = %d, want 1", len(types.Types["condition"]))
		}
		if len(types.Types["ability"]) != 1 {
			t.Errorf("ability count = %d, want 1", len(types.Types["ability"]))
		}
	})

	// Verify individual resolve files
	t.Run("resolve_files", func(t *testing.T) {
		var fury apiEntry
		readJSON(t, filepath.Join(dir, "v1", "resolve", "mcdm.heroes.v1", "class", "fury.json"), &fury)

		if fury.SCC != "mcdm.heroes.v1/class/fury" {
			t.Errorf("scc = %q", fury.SCC)
		}
		if fury.Name != "Fury" {
			t.Errorf("name = %q", fury.Name)
		}
		if fury.Source != "mcdm.heroes.v1" {
			t.Errorf("source = %q", fury.Source)
		}
	})

	// Verify URL mapping with sections
	t.Run("url_mapping", func(t *testing.T) {
		var fury apiEntry
		readJSON(t, filepath.Join(dir, "v1", "resolve", "mcdm.heroes.v1", "class", "fury.json"), &fury)
		if fury.URL != "https://steelcompendium.io/v2/Browse/class/fury/" {
			t.Errorf("fury url = %q, want Browse section", fury.URL)
		}

		var intro apiEntry
		readJSON(t, filepath.Join(dir, "v1", "resolve", "mcdm.heroes.v1", "chapter", "introduction.json"), &intro)
		if intro.URL != "https://steelcompendium.io/v2/Read/chapter/introduction/" {
			t.Errorf("intro url = %q, want Read section", intro.URL)
		}
	})

	// Verify alias resolve file
	t.Run("alias_resolve", func(t *testing.T) {
		// The alias doesn't have a real entry (reactive-strike wasn't added via WriteSection),
		// but it points to a canonical that doesn't exist in our test data.
		// In real usage the canonical entry would exist. For this test we just verify
		// the file was NOT created (since the canonical entry isn't in entries).
		aliasPath := filepath.Join(dir, "v1", "resolve", "mcdm.heroes.v1", "feature.ability.common", "reactive-strike.json")
		if _, err := os.Stat(aliasPath); !os.IsNotExist(err) {
			t.Errorf("alias resolve file should not exist when canonical is missing")
		}
	})
}

func TestSCCAPIGenerator_PrunesStaleResolveFiles(t *testing.T) {
	dir := t.TempDir()
	base := "https://steelcompendium.io/v2"
	stale := filepath.Join(dir, "v1", "resolve", "mcdm.heroes.v1/feature.trait.fury.level-1/old-code.json")
	kept := filepath.Join(dir, "v1", "resolve", "mcdm.heroes.v1/feature.fury.level-1/kept-code.json")

	// Run 1: two codes written.
	gen1 := &SCCAPIGenerator{OutputDir: dir, BaseURL: base}
	_ = gen1.WriteSection("mcdm.heroes.v1/feature.trait.fury.level-1/old-code",
		&content.ParsedContent{Frontmatter: map[string]any{"name": "Old", "type": "trait"}, ItemID: "old-code"})
	_ = gen1.WriteSection("mcdm.heroes.v1/feature.fury.level-1/kept-code",
		&content.ParsedContent{Frontmatter: map[string]any{"name": "Kept", "type": "feature"}, ItemID: "kept-code"})
	if err := gen1.Finalize(); err != nil {
		t.Fatalf("run 1 Finalize: %v", err)
	}
	if _, err := os.Stat(stale); err != nil {
		t.Fatalf("run 1 should have written %s: %v", stale, err)
	}

	// Run 2: fresh generator, same dir, with old-code removed from the registry.
	gen2 := &SCCAPIGenerator{OutputDir: dir, BaseURL: base}
	_ = gen2.WriteSection("mcdm.heroes.v1/feature.fury.level-1/kept-code",
		&content.ParsedContent{Frontmatter: map[string]any{"name": "Kept", "type": "feature"}, ItemID: "kept-code"})
	if err := gen2.Finalize(); err != nil {
		t.Fatalf("run 2 Finalize: %v", err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Errorf("stale resolve file should have been pruned: %s (stat err=%v)", stale, err)
	}
	if _, err := os.Stat(kept); err != nil {
		t.Errorf("kept resolve file should still exist: %v", err)
	}
}

func TestSCCAPIGenerator_AliasWithCanonical(t *testing.T) {
	dir := t.TempDir()

	gen := &SCCAPIGenerator{
		OutputDir: dir,
		BaseURL:   "https://example.com",
		Aliases:   map[string]string{"alias/code/x": "real/code/y"},
	}

	// Add the canonical entry
	if err := gen.WriteSection("real/code/y", &content.ParsedContent{
		Frontmatter: map[string]any{"name": "Y", "type": "test"},
		ItemID:      "y",
	}); err != nil {
		t.Fatal(err)
	}

	if err := gen.Finalize(); err != nil {
		t.Fatal(err)
	}

	// Alias resolve file should exist and point to canonical
	var aliasEntry apiEntry
	readJSON(t, filepath.Join(dir, "v1", "resolve", "alias", "code", "x.json"), &aliasEntry)

	if aliasEntry.SCC != "real/code/y" {
		t.Errorf("alias entry scc = %q, want real/code/y", aliasEntry.SCC)
	}
	if aliasEntry.Name != "Y" {
		t.Errorf("alias entry name = %q, want Y", aliasEntry.Name)
	}
}

func TestSCCAPIGenerator_Empty(t *testing.T) {
	gen := &SCCAPIGenerator{
		OutputDir: t.TempDir(),
		BaseURL:   "https://example.com",
	}

	if err := gen.Finalize(); err != nil {
		t.Fatalf("Finalize on empty should not error: %v", err)
	}
}

func TestSCCAPIGenerator_NilGuards(t *testing.T) {
	gen := &SCCAPIGenerator{OutputDir: t.TempDir(), BaseURL: "https://example.com"}

	if err := gen.WriteSection("", nil); err != nil {
		t.Errorf("empty scc should not error: %v", err)
	}
	if err := gen.WriteSection("some/code/x", nil); err != nil {
		t.Errorf("nil parsed should not error: %v", err)
	}
}

func TestSCCAPIGenerator_NoSections(t *testing.T) {
	dir := t.TempDir()

	gen := &SCCAPIGenerator{
		OutputDir: dir,
		BaseURL:   "https://example.com",
		Sections:  nil, // no site config
	}

	if err := gen.WriteSection("src/type/item", &content.ParsedContent{
		Frontmatter: map[string]any{"name": "Item", "type": "type"},
		ItemID:      "item",
	}); err != nil {
		t.Fatal(err)
	}

	if err := gen.Finalize(); err != nil {
		t.Fatal(err)
	}

	var entry apiEntry
	readJSON(t, filepath.Join(dir, "v1", "resolve", "src", "type", "item.json"), &entry)

	// Without sections, URL should have no section prefix
	if entry.URL != "https://example.com/type/item/" {
		t.Errorf("url = %q, want fallback without section", entry.URL)
	}
}

func TestSCCAPISchemeVersion(t *testing.T) {
	t.Run("explicit_version", func(t *testing.T) {
		dir := t.TempDir()
		gen := &SCCAPIGenerator{
			OutputDir:     dir,
			BaseURL:       "https://example.com",
			SchemeVersion: 2,
		}
		if err := gen.WriteSection("src/class/warrior", &content.ParsedContent{
			Frontmatter: map[string]any{"name": "Warrior", "type": "class"},
			ItemID:      "warrior",
		}); err != nil {
			t.Fatal(err)
		}
		if err := gen.Finalize(); err != nil {
			t.Fatalf("Finalize: %v", err)
		}

		// index.json must have scheme_version: 2
		var idx apiIndex
		readJSON(t, filepath.Join(dir, "v1", "index.json"), &idx)
		if idx.SchemeVersion != 2 {
			t.Errorf("index.json scheme_version = %d, want 2", idx.SchemeVersion)
		}

		// scc.json must have top-level scheme_version: 2
		var reg apiRegistry
		readJSON(t, filepath.Join(dir, "v1", "scc.json"), &reg)
		if reg.SchemeVersion != 2 {
			t.Errorf("scc.json scheme_version = %d, want 2", reg.SchemeVersion)
		}
		// entries[] must NOT each carry scheme_version (keep lean)
		raw, err := os.ReadFile(filepath.Join(dir, "v1", "scc.json"))
		if err != nil {
			t.Fatal(err)
		}
		var rawReg map[string]json.RawMessage
		if err := json.Unmarshal(raw, &rawReg); err != nil {
			t.Fatal(err)
		}
		var entries []json.RawMessage
		if err := json.Unmarshal(rawReg["entries"], &entries); err != nil {
			t.Fatal(err)
		}
		if len(entries) == 0 {
			t.Fatal("entries must not be empty")
		}
		for i, elem := range entries {
			if strings.Contains(string(elem), "scheme_version") {
				t.Errorf("entries[%d] should not contain scheme_version: %s", i, elem)
			}
		}

		// types.json must have top-level scheme_version: 2
		var types apiTypes
		readJSON(t, filepath.Join(dir, "v1", "types.json"), &types)
		if types.SchemeVersion != 2 {
			t.Errorf("types.json scheme_version = %d, want 2", types.SchemeVersion)
		}

		// per-entry resolve file must include scheme_version: 2
		resolveRaw, err := os.ReadFile(filepath.Join(dir, "v1", "resolve", "src", "class", "warrior.json"))
		if err != nil {
			t.Fatalf("read resolve file: %v", err)
		}
		var resolveEntry struct {
			SCC           string `json:"scc"`
			SchemeVersion int    `json:"scheme_version"`
		}
		if err := json.Unmarshal(resolveRaw, &resolveEntry); err != nil {
			t.Fatalf("unmarshal resolve file: %v", err)
		}
		if resolveEntry.SCC != "src/class/warrior" {
			t.Errorf("resolve scc = %q, want src/class/warrior", resolveEntry.SCC)
		}
		if resolveEntry.SchemeVersion != 2 {
			t.Errorf("resolve scheme_version = %d, want 2", resolveEntry.SchemeVersion)
		}
	})

	t.Run("zero_value_defaults_to_1", func(t *testing.T) {
		dir := t.TempDir()
		gen := &SCCAPIGenerator{
			OutputDir: dir,
			BaseURL:   "https://example.com",
			// SchemeVersion deliberately left at zero
		}
		if err := gen.WriteSection("src/class/fighter", &content.ParsedContent{
			Frontmatter: map[string]any{"name": "Fighter", "type": "class"},
			ItemID:      "fighter",
		}); err != nil {
			t.Fatal(err)
		}
		if err := gen.Finalize(); err != nil {
			t.Fatalf("Finalize: %v", err)
		}

		var idx apiIndex
		readJSON(t, filepath.Join(dir, "v1", "index.json"), &idx)
		if idx.SchemeVersion != 1 {
			t.Errorf("index.json scheme_version = %d, want 1 (default)", idx.SchemeVersion)
		}

		resolveRaw, err := os.ReadFile(filepath.Join(dir, "v1", "resolve", "src", "class", "fighter.json"))
		if err != nil {
			t.Fatalf("read resolve file: %v", err)
		}
		var resolveEntry struct {
			SchemeVersion int `json:"scheme_version"`
		}
		if err := json.Unmarshal(resolveRaw, &resolveEntry); err != nil {
			t.Fatalf("unmarshal resolve file: %v", err)
		}
		if resolveEntry.SchemeVersion != 1 {
			t.Errorf("resolve scheme_version = %d, want 1 (default)", resolveEntry.SchemeVersion)
		}
	})
}

func TestExtractSource(t *testing.T) {
	tests := []struct {
		scc  string
		want string
	}{
		{"mcdm.heroes.v1/class/fury", "mcdm.heroes.v1"},
		{"mcdm.monsters.v1/monster/chimera", "mcdm.monsters.v1"},
		{"noseparator", "noseparator"},
	}
	for _, tt := range tests {
		if got := extractSource(tt.scc); got != tt.want {
			t.Errorf("extractSource(%q) = %q, want %q", tt.scc, got, tt.want)
		}
	}
}

func TestMatchesSectionIncludes(t *testing.T) {
	browse := site.SectionConfig{
		Name:    "Browse",
		Include: []string{"class/", "feature/"},
		Exclude: []string{"feature/internal/"},
	}

	tests := []struct {
		path string
		want bool
	}{
		{"class/fury", true},
		{"feature/ability/fury/level-1/gouge", true},
		{"chapter/introduction", false},
		{"feature/internal/debug", false},
	}
	for _, tt := range tests {
		if got := matchesSectionIncludes(tt.path, browse); got != tt.want {
			t.Errorf("matchesSectionIncludes(%q) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func readJSON(t *testing.T, path string, v any) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	if err := json.Unmarshal(data, v); err != nil {
		t.Fatalf("unmarshal %s: %v", path, err)
	}
}
