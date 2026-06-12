package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/scc"
)

func setupSourceDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"class/fury.md":                                  "---\nname: Fury\ntype: class\n---\n\nFury description.",
		"class/shadow.md":                                "---\nname: Shadow\ntype: class\n---\n\nShadow description.",
		"feature/ability/fury/level-1/gouge.md":          "---\nname: Gouge\ntype: ability\n---\n\nGouge text.",
		"feature/ability/fury/level-1/brutal-slam.md":    "---\nname: Brutal Slam\ntype: ability\n---\n\nSlam text.",
		"feature/trait/fury/level-1/growing-ferocity.md": "---\nname: Growing Ferocity\ntype: trait\n---\n\nFerocity text.",
		"condition/dazed.md":                             "---\nname: Dazed\ntype: condition\n---\n\nDazed text.",
		"chapter/classes.md":                             "# Classes\n\nChapter intro.",
	}

	for rel, content := range files {
		path := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func TestBuild_BasicSections(t *testing.T) {
	srcDir := setupSourceDir(t)
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Sections: []SectionConfig{
			{
				Name:    "Browse",
				Include: []string{"class/", "feature/", "condition/"},
				Sort:    "natural",
			},
			{
				Name:    "Read",
				Title:   "Rulebook Chapters",
				Include: []string{"chapter/"},
			},
		},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("Error: %s", e)
		}
	}

	// Browse should have class and feature files
	checkExists(t, docsDir, "Browse/class/fury.md")
	checkExists(t, docsDir, "Browse/feature/ability/fury/level-1/gouge.md")
	checkExists(t, docsDir, "Browse/condition/dazed.md")

	// Read should have chapter files
	checkExists(t, docsDir, "Read/chapter/classes.md")

	// Browse should NOT have chapter files
	checkNotExists(t, docsDir, "Browse/chapter/classes.md")

	// Nav files should exist
	checkExists(t, docsDir, "Browse/.nav.yml")
	checkExists(t, docsDir, "Read/.nav.yml")

	if result.Sections != 2 {
		t.Errorf("expected 2 sections, got %d", result.Sections)
	}
}

func TestBuild_SearchExclusion(t *testing.T) {
	srcDir := setupSourceDir(t)
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Sections: []SectionConfig{
			{Name: "Read", Include: []string{"chapter/"}},
		},
		SearchExclude: []string{"Read"},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if result.SearchExclude == 0 {
		t.Error("expected search exclusion to be applied")
	}

	// Verify frontmatter was injected
	data, err := os.ReadFile(filepath.Join(docsDir, "Read", "chapter", "classes.md"))
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	content := string(data)
	if !strings.Contains(content, "search:") || !strings.Contains(content, "exclude: true") {
		t.Error("search exclusion frontmatter not found")
	}
}

func TestBuild_StaticContentOverride(t *testing.T) {
	srcDir := setupSourceDir(t)
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	// Create static content with an override
	staticDir := filepath.Join(t.TempDir(), "static")
	os.MkdirAll(filepath.Join(staticDir, "Browse"), 0755)
	os.WriteFile(filepath.Join(staticDir, "Browse", "custom.md"), []byte("# Custom page"), 0644)

	cfg := &Config{
		SourceDir:     srcDir,
		DocsDir:       docsDir,
		Sections:      []SectionConfig{{Name: "Browse", Include: []string{"class/"}}},
		StaticContent: staticDir,
	}

	_, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Custom file should exist
	checkExists(t, docsDir, "Browse/custom.md")
	// Generated file should also exist
	checkExists(t, docsDir, "Browse/class/fury.md")
}

func TestBuild_GeneratesIndexPages(t *testing.T) {
	srcDir := setupSourceDir(t)
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Sections: []SectionConfig{
			{
				Name:    "Browse",
				Include: []string{"class/", "feature/", "condition/"},
				Sort:    "natural",
			},
		},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Index pages should be generated for type directories
	checkExists(t, docsDir, "Browse/class/index.md")
	checkExists(t, docsDir, "Browse/condition/index.md")
	checkExists(t, docsDir, "Browse/feature/index.md")
	checkExists(t, docsDir, "Browse/feature/ability/index.md")
	checkExists(t, docsDir, "Browse/feature/trait/index.md")

	// Section root should NOT get a generated index
	checkNotExists(t, docsDir, "Browse/index.md")

	if result.IndexPages == 0 {
		t.Error("expected index pages to be generated")
	}

	// Verify class index content
	data, err := os.ReadFile(filepath.Join(docsDir, "Browse", "class", "index.md"))
	if err != nil {
		t.Fatalf("read class index: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "# Classes") {
		t.Error("class index missing title")
	}
	// `class` is a rich-card index type (see richCardTypes in cards.go): entries
	// render as stat-cards linking to each page, not plain markdown links.
	if !strings.Contains(content, `<div class="sc-cards">`) {
		t.Error("class index should render as a card grid")
	}
	// Card links use the served directory-URL form (foo.md → foo/), not a dead
	// ".md" path that 404s under use_directory_urls.
	if !strings.Contains(content, `href="fury/"`) || !strings.Contains(content, `<div class="sc-card__name">Fury</div>`) {
		t.Error("class index missing Fury card")
	}
	if !strings.Contains(content, `href="shadow/"`) || !strings.Contains(content, `<div class="sc-card__name">Shadow</div>`) {
		t.Error("class index missing Shadow card")
	}

	// Verify feature index lists subdirectories
	data, err = os.ReadFile(filepath.Join(docsDir, "Browse", "feature", "index.md"))
	if err != nil {
		t.Fatalf("read feature index: %v", err)
	}
	content = string(data)
	if !strings.Contains(content, "# Features") {
		t.Error("feature index missing title")
	}
	// The feature/ landing is an index-of-indexes node (its children are
	// directories), so it renders navigational folder cards (steel-indexes.css),
	// not the old collapsible <details> list.
	if !strings.Contains(content, `<div class="sc-folders`) {
		t.Error("feature index should render folder cards")
	}
	if !strings.Contains(content, `<a class="sc-folder" href="ability/">`) ||
		!strings.Contains(content, `<h3 class="sc-folder__name">Abilities</h3>`) {
		t.Error("feature index missing Abilities folder card")
	}
	if !strings.Contains(content, `<a class="sc-folder" href="trait/">`) ||
		!strings.Contains(content, `<h3 class="sc-folder__name">Traits</h3>`) {
		t.Error("feature index missing Traits folder card")
	}
}

func TestBuild_StaticOverridesIndex(t *testing.T) {
	srcDir := setupSourceDir(t)
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	// Create static content with a custom index override
	staticDir := filepath.Join(t.TempDir(), "static")
	os.MkdirAll(filepath.Join(staticDir, "Browse", "class"), 0755)
	os.WriteFile(
		filepath.Join(staticDir, "Browse", "class", "index.md"),
		[]byte("# Custom Classes Index\n"),
		0644,
	)

	cfg := &Config{
		SourceDir:     srcDir,
		DocsDir:       docsDir,
		Sections:      []SectionConfig{{Name: "Browse", Include: []string{"class/"}}},
		StaticContent: staticDir,
	}

	_, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Static override should replace generated index
	data, err := os.ReadFile(filepath.Join(docsDir, "Browse", "class", "index.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "# Custom Classes Index\n" {
		t.Errorf("static override did not replace generated index: %s", data)
	}
}

func TestMatchesSection(t *testing.T) {
	tests := []struct {
		name    string
		relPath string
		section SectionConfig
		want    bool
	}{
		{"include match", "class/fury.md", SectionConfig{Include: []string{"class/"}}, true},
		{"include no match", "condition/dazed.md", SectionConfig{Include: []string{"class/"}}, false},
		{"exclude", "chapter/intro.md", SectionConfig{Include: []string{"chapter/", "class/"}, Exclude: []string{"chapter/"}}, false},
		{"no include matches all", "anything.md", SectionConfig{}, true},
		{"prefix match", "feature/ability/fury/level-1/gouge.md", SectionConfig{Include: []string{"feature/"}}, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := matchesSection(tt.relPath, tt.section)
			if got != tt.want {
				t.Errorf("matchesSection(%q) = %v, want %v", tt.relPath, got, tt.want)
			}
		})
	}
}

func TestCleanDocsDir_PreservesProtected(t *testing.T) {
	dir := t.TempDir()

	// Create protected and unprotected content
	os.MkdirAll(filepath.Join(dir, "javascripts"), 0755)
	os.WriteFile(filepath.Join(dir, "javascripts", "app.js"), []byte(""), 0644)
	os.MkdirAll(filepath.Join(dir, "stylesheets"), 0755)
	os.WriteFile(filepath.Join(dir, "index.md"), []byte("# Home"), 0644)
	os.MkdirAll(filepath.Join(dir, "Browse"), 0755)
	os.WriteFile(filepath.Join(dir, "Browse", "test.md"), []byte("test"), 0644)

	if err := cleanDocsDir(dir); err != nil {
		t.Fatalf("cleanDocsDir failed: %v", err)
	}

	// Protected should remain
	checkExists(t, dir, "javascripts/app.js")
	checkExists(t, dir, "index.md")

	// Unprotected should be gone
	checkNotExists(t, dir, "Browse/test.md")
}

func TestApplySearchExclusion_WithFrontmatter(t *testing.T) {
	dir := t.TempDir()
	sectionDir := filepath.Join(dir, "Read")
	os.MkdirAll(sectionDir, 0755)

	// File with existing frontmatter
	os.WriteFile(filepath.Join(sectionDir, "chapter.md"), []byte("---\nname: Chapter\n---\n\n# Content"), 0644)

	count, errs := applySearchExclusion(dir, "Read")
	if len(errs) > 0 {
		t.Errorf("errors: %v", errs)
	}
	if count != 1 {
		t.Errorf("expected 1 file, got %d", count)
	}

	data, _ := os.ReadFile(filepath.Join(sectionDir, "chapter.md"))
	content := string(data)
	if !strings.HasPrefix(content, "---\nsearch:\n  exclude: true\n") {
		t.Errorf("search exclusion not injected correctly:\n%s", content)
	}
}

func TestApplySearchExclusion_WithoutFrontmatter(t *testing.T) {
	dir := t.TempDir()
	sectionDir := filepath.Join(dir, "FullBook")
	os.MkdirAll(sectionDir, 0755)

	os.WriteFile(filepath.Join(sectionDir, "book.md"), []byte("# Full Book"), 0644)

	count, _ := applySearchExclusion(dir, "FullBook")
	if count != 1 {
		t.Errorf("expected 1 file, got %d", count)
	}

	data, _ := os.ReadFile(filepath.Join(sectionDir, "book.md"))
	content := string(data)
	if !strings.HasPrefix(content, "---\nsearch:\n  exclude: true\n---\n\n# Full Book") {
		t.Errorf("search exclusion not prepended correctly:\n%s", content)
	}
}

func TestBuild_Groups(t *testing.T) {
	srcDir := t.TempDir()
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	files := map[string]string{
		"feature/ability/fury/level-1/gouge.md":            "---\nname: Gouge\n---\n\nGouge text.",
		"feature/ability/arcane-archer/exploding-arrow.md": "---\nname: Exploding Arrow\n---\n\nArrow text.",
		"kit/arcane-archer.md":                             "---\nname: Arcane Archer\ntype: kit\n---\n\nKit desc.",
	}
	for rel, content := range files {
		path := filepath.Join(srcDir, rel)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}

	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Sections: []SectionConfig{
			{
				Name:    "Browse",
				Include: []string{"feature/", "kit/"},
				Groups: []GroupConfig{
					{MatchType: "kit", From: "feature/ability", Label: "Kits"},
				},
			},
		},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	if len(result.Errors) > 0 {
		for _, e := range result.Errors {
			t.Errorf("Error: %s", e)
		}
	}

	// Kit ability should be remapped under Kits/
	checkExists(t, docsDir, "Browse/feature/ability/Kits/arcane-archer/exploding-arrow.md")

	// Class ability should NOT be remapped
	checkExists(t, docsDir, "Browse/feature/ability/fury/level-1/gouge.md")

	// Kit ability should NOT exist at original path
	checkNotExists(t, docsDir, "Browse/feature/ability/arcane-archer/exploding-arrow.md")
}

func TestApplyGroups(t *testing.T) {
	srcDir := t.TempDir()
	// Create a kit file for cross-reference
	os.MkdirAll(filepath.Join(srcDir, "kit"), 0755)
	os.WriteFile(filepath.Join(srcDir, "kit", "arcane-archer.md"), []byte(""), 0644)

	groups := []GroupConfig{
		{MatchType: "kit", From: "feature/ability", Label: "Kits"},
	}

	tests := []struct {
		name       string
		relPath    string
		wantPath   string
		wantParent string
	}{
		{"kit ability remapped", "feature/ability/arcane-archer/exploding-arrow.md", "feature/ability/Kits/arcane-archer/exploding-arrow.md", ""},
		{"class ability unchanged", "feature/ability/fury/level-1/gouge.md", "feature/ability/fury/level-1/gouge.md", ""},
		{"unrelated path unchanged", "class/fury.md", "class/fury.md", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotParent := applyGroups(tt.relPath, groups, srcDir)
			if gotPath != tt.wantPath {
				t.Errorf("applyGroups(%q) path = %q, want %q", tt.relPath, gotPath, tt.wantPath)
			}
			if gotParent != tt.wantParent {
				t.Errorf("applyGroups(%q) parent = %q, want %q", tt.relPath, gotParent, tt.wantParent)
			}
		})
	}
}

func TestApplyGroups_Flatten(t *testing.T) {
	srcDir := t.TempDir()
	os.MkdirAll(filepath.Join(srcDir, "kit"), 0755)
	os.WriteFile(filepath.Join(srcDir, "kit", "battlemind.md"), []byte(""), 0644)
	os.WriteFile(filepath.Join(srcDir, "kit", "arcane-archer.md"), []byte(""), 0644)

	groups := []GroupConfig{
		{MatchType: "kit", From: "feature/ability", Label: "Kits", Flatten: true},
	}

	tests := []struct {
		name       string
		relPath    string
		wantPath   string
		wantParent string
	}{
		{
			"flatten kit ability",
			"feature/ability/battlemind/unmooring.md",
			"feature/ability/Kits/battlemind-unmooring.md",
			"battlemind",
		},
		{
			"flatten multi-word kit",
			"feature/ability/arcane-archer/exploding-arrow.md",
			"feature/ability/Kits/arcane-archer-exploding-arrow.md",
			"arcane-archer",
		},
		{
			"class ability unchanged",
			"feature/ability/fury/level-1/gouge.md",
			"feature/ability/fury/level-1/gouge.md",
			"",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPath, gotParent := applyGroups(tt.relPath, groups, srcDir)
			if gotPath != tt.wantPath {
				t.Errorf("applyGroups(%q) path = %q, want %q", tt.relPath, gotPath, tt.wantPath)
			}
			if gotParent != tt.wantParent {
				t.Errorf("applyGroups(%q) parent = %q, want %q", tt.relPath, gotParent, tt.wantParent)
			}
		})
	}
}

func TestBuild_GroupsFlatten(t *testing.T) {
	srcDir := t.TempDir()
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	files := map[string]string{
		"feature/ability/fury/level-1/gouge.md":            "---\nname: Gouge\n---\n\nGouge text.",
		"feature/ability/battlemind/unmooring.md":          "---\nname: Unmooring\ntype: ability\n---\n\nUnmooring text.",
		"feature/ability/arcane-archer/exploding-arrow.md": "---\nname: Exploding Arrow\ntype: ability\n---\n\nArrow text.",
		"kit/battlemind.md":                                "---\nname: Battlemind\ntype: kit\n---\n\nKit desc.",
		"kit/arcane-archer.md":                             "---\nname: Arcane Archer\ntype: kit\n---\n\nKit desc.",
	}
	for rel, content := range files {
		path := filepath.Join(srcDir, rel)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}

	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Sections: []SectionConfig{
			{
				Name:    "Browse",
				Include: []string{"feature/", "kit/"},
				Groups: []GroupConfig{
					{MatchType: "kit", From: "feature/ability", Label: "Kits", Flatten: true},
				},
			},
		},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	for _, e := range result.Errors {
		t.Errorf("Error: %s", e)
	}

	// Flattened kit ability lives directly under Kits/ as parent-child.md
	checkExists(t, docsDir, "Browse/feature/ability/Kits/battlemind-unmooring.md")
	checkExists(t, docsDir, "Browse/feature/ability/Kits/arcane-archer-exploding-arrow.md")

	// The original nested locations should NOT exist
	checkNotExists(t, docsDir, "Browse/feature/ability/Kits/battlemind/unmooring.md")
	checkNotExists(t, docsDir, "Browse/feature/ability/battlemind/unmooring.md")

	// Class ability remains nested under its level dir
	checkExists(t, docsDir, "Browse/feature/ability/fury/level-1/gouge.md")

	// Frontmatter name and H1 should both reflect the combined "Parent (Child)" form
	data, err := os.ReadFile(filepath.Join(docsDir, "Browse", "feature", "ability", "Kits", "battlemind-unmooring.md"))
	if err != nil {
		t.Fatalf("read flattened: %v", err)
	}
	content := string(data)
	if !strings.Contains(content, "name: Battlemind (Unmooring)") {
		t.Errorf("expected combined frontmatter name, got:\n%s", content)
	}
	if !strings.Contains(content, "# Battlemind (Unmooring)") {
		t.Errorf("expected combined H1, got:\n%s", content)
	}

	// Multi-word kit slugs should title-case "arcane-archer" → "Arcane Archer"
	data, err = os.ReadFile(filepath.Join(docsDir, "Browse", "feature", "ability", "Kits", "arcane-archer-exploding-arrow.md"))
	if err != nil {
		t.Fatalf("read flattened multi: %v", err)
	}
	if !strings.Contains(string(data), "# Arcane Archer (Exploding Arrow)") {
		t.Errorf("expected Arcane Archer combined H1, got:\n%s", data)
	}
}

func TestCombineFrontmatterName(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		parent string
		want   string
	}{
		{
			"combines simple slug",
			"---\nname: Unmooring\ntype: ability\n---\n\nBody.",
			"battlemind",
			"---\nname: Battlemind (Unmooring)\ntype: ability\n---\n\nBody.",
		},
		{
			"title-cases multi-word slug",
			"---\nname: Exploding Arrow\n---\n\nBody.",
			"arcane-archer",
			"---\nname: Arcane Archer (Exploding Arrow)\n---\n\nBody.",
		},
		{
			"no frontmatter unchanged",
			"# Title\n\nBody.",
			"battlemind",
			"# Title\n\nBody.",
		},
		{
			"no name field unchanged",
			"---\ntype: ability\n---\n\nBody.",
			"battlemind",
			"---\ntype: ability\n---\n\nBody.",
		},
		{
			"does not touch indented name keys",
			"---\nname: Outer\nnested:\n  name: Inner\n---\n\nBody.",
			"parent",
			"---\nname: Parent (Outer)\nnested:\n  name: Inner\n---\n\nBody.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(combineFrontmatterName([]byte(tt.input), tt.parent))
			if got != tt.want {
				t.Errorf("combineFrontmatterName:\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestInjectH1(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{
			name:  "adds h1 and hr from frontmatter",
			input: "---\nname: Devil\ntype: ancestry\n---\n\nFlavor text.",
			want:  "---\nname: Devil\ntype: ancestry\n---\n\n# Devil\n\n---\n\nFlavor text.",
		},
		{
			name:  "adds hr after existing h1 without duplicating h1",
			input: "---\nname: Devil\n---\n\n# Devil\n\nFlavor text.",
			want:  "---\nname: Devil\n---\n\n# Devil\n\n---\n\nFlavor text.",
		},
		{
			name:  "does not duplicate hr when one already follows h1",
			input: "---\nname: Devil\n---\n\n# Devil\n\n---\n\nFlavor text.",
			want:  "---\nname: Devil\n---\n\n# Devil\n\n---\n\nFlavor text.",
		},
		{
			name:  "skips if no frontmatter",
			input: "# Just a page\n\nContent.",
			want:  "# Just a page\n\nContent.",
		},
		{
			name:  "skips if no name field",
			input: "---\ntype: ancestry\n---\n\nContent.",
			want:  "---\ntype: ancestry\n---\n\nContent.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := string(injectH1([]byte(tt.input)))
			if got != tt.want {
				t.Errorf("injectH1:\n got: %q\nwant: %q", got, tt.want)
			}
		})
	}
}

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"level-1", "level-2", true},
		{"level-2", "level-10", true},
		{"level-10", "level-2", false},
		{"level-1", "level-1", false},
		{"level-9", "level-10", true},
		{"abc", "abd", true},
		{"a1b", "a2b", true},
		{"a10b", "a2b", false},
		{"foo", "foo1", true},
	}

	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			got := naturalLess(tt.a, tt.b)
			if got != tt.want {
				t.Errorf("naturalLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestRewriteSectionLinks(t *testing.T) {
	sections := []SectionConfig{
		{Name: "Browse", Include: []string{"class/", "feature/", "condition/", "ancestry/", "kit/", "skill/", "champion/"}},
		{Name: "Read", Include: []string{"chapter/"}},
	}

	tests := []struct {
		name        string
		content     string
		srcRelPath  string
		destRelPath string
		sectionName string
		want        string
	}{
		{
			name:        "chapter links to ancestry cross-section",
			content:     "See [Human](../ancestry/human.md) for details.",
			srcRelPath:  "chapter/background.md",
			destRelPath: "chapter/background.md",
			sectionName: "Read",
			want:        "See [Human](../../Browse/ancestry/human.md) for details.",
		},
		{
			name:        "chapter links to class cross-section",
			content:     "Pick a [class](../class/fury.md).",
			srcRelPath:  "chapter/classes.md",
			destRelPath: "chapter/classes.md",
			sectionName: "Read",
			want:        "Pick a [class](../../Browse/class/fury.md).",
		},
		{
			name:        "same section link unchanged",
			content:     "See [Classes](classes.md) chapter.",
			srcRelPath:  "chapter/background.md",
			destRelPath: "chapter/background.md",
			sectionName: "Read",
			want:        "See [Classes](classes.md) chapter.",
		},
		{
			name:        "browse to browse same section unchanged",
			content:     "See [Human](../ancestry/human.md).",
			srcRelPath:  "class/fury.md",
			destRelPath: "class/fury.md",
			sectionName: "Browse",
			want:        "See [Human](../ancestry/human.md).",
		},
		{
			name:        "browse links to chapter cross-section",
			content:     "Read the [introduction](../chapter/introduction.md).",
			srcRelPath:  "class/fury.md",
			destRelPath: "class/fury.md",
			sectionName: "Browse",
			want:        "Read the [introduction](../../Read/chapter/introduction.md).",
		},
		{
			name:        "multiple cross-section links",
			content:     "See [Human](../ancestry/human.md) and [Fury](../class/fury.md).",
			srcRelPath:  "chapter/background.md",
			destRelPath: "chapter/background.md",
			sectionName: "Read",
			want:        "See [Human](../../Browse/ancestry/human.md) and [Fury](../../Browse/class/fury.md).",
		},
		{
			name:        "http links unchanged",
			content:     "Visit [site](https://example.com) and [Human](../ancestry/human.md).",
			srcRelPath:  "chapter/background.md",
			destRelPath: "chapter/background.md",
			sectionName: "Read",
			want:        "Visit [site](https://example.com) and [Human](../../Browse/ancestry/human.md).",
		},
		{
			name:        "no matching section leaves link unchanged",
			content:     "See [unknown](../unknown/thing.md).",
			srcRelPath:  "chapter/background.md",
			destRelPath: "chapter/background.md",
			sectionName: "Read",
			want:        "See [unknown](../unknown/thing.md).",
		},
		{
			name:        "group-remapped dest cross-section",
			content:     "See [Classes](../../../chapter/classes.md).",
			srcRelPath:  "feature/ability/arcane-archer/exploding-arrow.md",
			destRelPath: "feature/ability/Kits/arcane-archer-exploding-arrow.md",
			sectionName: "Browse",
			want:        "See [Classes](../../../../Read/chapter/classes.md).",
		},
		{
			// A skill-group landing page is relocated to skill/<member>/index.md,
			// so inbound links must resolve there, not to skill/group/<member>.md.
			name:        "skill-group landing link relocates to member index",
			content:     "the [lore skill group](../skill/group/lore.md)",
			srcRelPath:  "class/censor.md",
			destRelPath: "class/censor.md",
			sectionName: "Browse",
			want:        "the [lore skill group](../skill/lore/index.md)",
		},
		{
			// A statblock page hoists away its statblock/ segment, so inbound
			// links must drop it too (real link: a Summoner chapter → champion).
			name:        "statblock link hoists away statblock segment",
			content:     "see [Avatar](../champion/undead/statblock/avatar-of-death.md)",
			srcRelPath:  "chapter/summoner-advice.md",
			destRelPath: "summoner/summoner-advice.md",
			sectionName: "Read",
			want:        "see [Avatar](../../Browse/champion/undead/avatar-of-death.md)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteSectionLinks(tt.content, tt.srcRelPath, tt.destRelPath, tt.sectionName, "", sections, nil)
			if got != tt.want {
				t.Errorf("rewriteSectionLinks():\n  got  %q\n  want %q", got, tt.want)
			}
		})
	}
}

func TestBuild_CrossSectionLinks(t *testing.T) {
	srcDir := t.TempDir()
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	files := map[string]string{
		"chapter/background.md": "---\nname: Background\ntype: chapter\n---\n\nChoose a [human](../ancestry/human.md) ancestry.",
		"ancestry/human.md":     "---\nname: Human\ntype: ancestry\n---\n\nHuman description.",
	}
	for rel, content := range files {
		path := filepath.Join(srcDir, rel)
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, []byte(content), 0644)
	}

	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Sections: []SectionConfig{
			{Name: "Browse", Include: []string{"ancestry/"}},
			{Name: "Read", Include: []string{"chapter/"}},
		},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}
	for _, e := range result.Errors {
		t.Errorf("Error: %s", e)
	}

	data, err := os.ReadFile(filepath.Join(docsDir, "Read", "chapter", "background.md"))
	if err != nil {
		t.Fatalf("read chapter: %v", err)
	}
	content := string(data)

	if strings.Contains(content, "../ancestry/human.md") {
		t.Error("cross-section link was not rewritten; still points to flat ../ancestry/human.md")
	}
	if !strings.Contains(content, "../../Browse/ancestry/human.md") {
		t.Errorf("expected rewritten cross-section link to Browse, got:\n%s", content)
	}
}

func checkExists(t *testing.T, base, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(base, rel)); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", rel)
	}
}

func checkNotExists(t *testing.T, base, rel string) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(base, rel)); err == nil {
		t.Errorf("expected file to NOT exist: %s", rel)
	}
}

func TestWalkSourceDirsMerges(t *testing.T) {
	a := t.TempDir()
	b := t.TempDir()
	if err := os.MkdirAll(filepath.Join(a, "class"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(b, "class"), 0755); err != nil {
		t.Fatal(err)
	}
	os.WriteFile(filepath.Join(a, "class", "fury.md"), []byte("---\nname: Fury\n---\n"), 0644)
	os.WriteFile(filepath.Join(b, "class", "beastheart.md"), []byte("---\nname: Beastheart\n---\n"), 0644)

	entries, err := walkSourceDirs([]string{a, b})
	if err != nil {
		t.Fatalf("walk: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}
	// Each entry must remember which source dir it came from.
	for _, e := range entries {
		if e.sourceDir != a && e.sourceDir != b {
			t.Errorf("entry %q has unexpected sourceDir %q", e.relPath, e.sourceDir)
		}
	}
}

func TestBookKeyFromSCC(t *testing.T) {
	cases := map[string]string{
		"mcdm.heroes.v1/chapter/introduction": "mcdm.heroes.v1",
		"mcdm.beastheart.v1/chapter/rewards":  "mcdm.beastheart.v1",
		"":                                    "",
		"noslash":                             "noslash",
	}
	for in, want := range cases {
		if got := bookKeyFromSCC(in); got != want {
			t.Errorf("bookKeyFromSCC(%q)=%q want %q", in, got, want)
		}
	}
}

func TestParseFrontmatterOrder(t *testing.T) {
	fm := "name: Rewards\nscc: mcdm.beastheart.v1/chapter/rewards\ntype: chapter\norder: 3\n"
	if got := parseFrontmatterInt(fm, "order", -1); got != 3 {
		t.Errorf("order=%d want 3", got)
	}
	if got := parseFrontmatterInt("name: x\n", "order", 99); got != 99 {
		t.Errorf("missing order default=%d want 99", got)
	}
}

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
}

func TestBuildGroupsReadByBook(t *testing.T) {
	dir := t.TempDir()
	heroesSrc := filepath.Join(dir, "src-heroes")
	beastSrc := filepath.Join(dir, "src-beast")
	docs := filepath.Join(dir, "docs")

	writeFile(t, filepath.Join(heroesSrc, "chapter", "introduction.md"),
		"---\nname: Introduction\nscc: mcdm.heroes.v1/chapter/introduction\ntype: chapter\norder: 0\n---\n\nHero intro.\n")
	writeFile(t, filepath.Join(heroesSrc, "chapter", "classes.md"),
		"---\nname: Classes\nscc: mcdm.heroes.v1/chapter/classes\ntype: chapter\norder: 7\n---\n\nClasses.\n")
	writeFile(t, filepath.Join(beastSrc, "chapter", "rewards.md"),
		"---\nname: Rewards\nscc: mcdm.beastheart.v1/chapter/rewards\ntype: chapter\norder: 2\n---\n\nRewards.\n")

	cfg := &Config{
		SourceDirs: []string{heroesSrc, beastSrc},
		DocsDir:    docs,
		Books: []BookConfig{
			{Key: "mcdm.heroes.v1", Folder: "heroes", Label: "Draw Steel Heroes", Order: 1},
			{Key: "mcdm.beastheart.v1", Folder: "beastheart", Label: "Draw Steel: Beastheart", Order: 2},
		},
		Sections: []SectionConfig{{Name: "Read", Include: []string{"chapter/"}, GroupByBook: true}},
	}

	if _, err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}
	for _, p := range []string{
		filepath.Join(docs, "Read", "heroes", "introduction.md"),
		filepath.Join(docs, "Read", "heroes", "classes.md"),
		filepath.Join(docs, "Read", "beastheart", "rewards.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s to exist: %v", p, err)
		}
	}
	if _, err := os.Stat(filepath.Join(docs, "Read", "chapter")); err == nil {
		t.Errorf("Read/chapter should not exist under group_by_book")
	}
}

func TestBuildBookNavAndIndexes(t *testing.T) {
	dir := t.TempDir()
	heroesSrc := filepath.Join(dir, "src-heroes")
	beastSrc := filepath.Join(dir, "src-beast")
	docs := filepath.Join(dir, "docs")
	writeFile(t, filepath.Join(heroesSrc, "chapter", "introduction.md"),
		"---\nname: Introduction\nscc: mcdm.heroes.v1/chapter/introduction\ntype: chapter\norder: 0\n---\n\nHero intro.\n")
	writeFile(t, filepath.Join(heroesSrc, "chapter", "classes.md"),
		"---\nname: Classes\nscc: mcdm.heroes.v1/chapter/classes\ntype: chapter\norder: 7\n---\n\nClasses.\n")
	writeFile(t, filepath.Join(beastSrc, "chapter", "rewards.md"),
		"---\nname: Rewards\nscc: mcdm.beastheart.v1/chapter/rewards\ntype: chapter\norder: 2\n---\n\nRewards.\n")
	writeFile(t, filepath.Join(beastSrc, "chapter", "the-beastheart-and-the-faeries.md"),
		"---\nname: The Beastheart & The Faeries\nscc: mcdm.beastheart.v1/chapter/the-beastheart-and-the-faeries\ntype: chapter\norder: 0\n---\n\nFiction.\n")
	cfg := &Config{
		SourceDirs: []string{heroesSrc, beastSrc},
		DocsDir:    docs,
		Books: []BookConfig{
			{Key: "mcdm.heroes.v1", Folder: "heroes", Label: "Draw Steel Heroes", Order: 1},
			{Key: "mcdm.beastheart.v1", Folder: "beastheart", Label: "Draw Steel: Beastheart", Order: 2},
		},
		Sections: []SectionConfig{{Name: "Read", Title: "Rulebook Chapters", Include: []string{"chapter/"}, GroupByBook: true}},
	}
	if _, err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}

	heroesNav, _ := os.ReadFile(filepath.Join(docs, "Read", "heroes", ".nav.yml"))
	if !strings.Contains(string(heroesNav), "Draw Steel Heroes") {
		t.Errorf("heroes nav missing label:\n%s", heroesNav)
	}
	if i, c := strings.Index(string(heroesNav), "introduction.md"), strings.Index(string(heroesNav), "classes.md"); i < 0 || c < 0 || i > c {
		t.Errorf("heroes nav not in source order:\n%s", heroesNav)
	}
	beastNav, _ := os.ReadFile(filepath.Join(docs, "Read", "beastheart", ".nav.yml"))
	if f, r := strings.Index(string(beastNav), "the-beastheart-and-the-faeries.md"), strings.Index(string(beastNav), "rewards.md"); f < 0 || r < 0 || f > r {
		t.Errorf("beastheart nav not in source order:\n%s", beastNav)
	}

	readNav, _ := os.ReadFile(filepath.Join(docs, "Read", ".nav.yml"))
	if h, b := strings.Index(string(readNav), "heroes"), strings.Index(string(readNav), "beastheart"); h < 0 || b < 0 || h > b {
		t.Errorf("Read nav not in book order:\n%s", readNav)
	}

	for _, p := range []string{
		filepath.Join(docs, "Read", "heroes", "index.md"),
		filepath.Join(docs, "Read", "beastheart", "index.md"),
		filepath.Join(docs, "Read", "index.md"),
	} {
		if _, err := os.Stat(p); err != nil {
			t.Errorf("expected %s: %v", p, err)
		}
	}
	beastIdx, _ := os.ReadFile(filepath.Join(docs, "Read", "beastheart", "index.md"))
	// Card names are HTML-escaped by card(), so "&" appears as "&amp;".
	if f, r := strings.Index(string(beastIdx), "The Beastheart &amp; The Faeries"), strings.Index(string(beastIdx), "Rewards"); f < 0 || r < 0 || f > r {
		t.Errorf("beastheart index not in source order:\n%s", beastIdx)
	}
}

func TestBuildBookIndexesUseCards(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	docs := filepath.Join(dir, "docs")
	writeFile(t, filepath.Join(src, "chapter", "ancestries.md"),
		"---\nname: Ancestries\nscc: mcdm.heroes.v1/chapter/ancestries\ntype: chapter\norder: 3\n---\n\nFantastic peoples inhabit the worlds of Draw Steel.\n")
	cfg := &Config{
		SourceDirs: []string{src},
		DocsDir:    docs,
		Books: []BookConfig{
			{Key: "mcdm.heroes.v1", Folder: "heroes", Label: "Heroes", Order: 1,
				Icon: "sword-cross", Description: "The core rulebook."},
		},
		Sections: []SectionConfig{{Name: "Read", Title: "Books", Include: []string{"chapter/"}, GroupByBook: true}},
	}
	if _, err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}

	// Section landing: a book card grid with the book's description + icon.
	landing, _ := os.ReadFile(filepath.Join(docs, "Read", "index.md"))
	for _, want := range []string{`<div class="sc-cards">`, `class="sc-card`, ">Heroes<", "The core rulebook.", iconPaths["sword-cross"]} {
		if !strings.Contains(string(landing), want) {
			t.Errorf("Read landing missing %q:\n%s", want, landing)
		}
	}
	if strings.Contains(string(landing), "browse-index") {
		t.Errorf("Read landing should no longer use browse-index:\n%s", landing)
	}

	// Per-book index: a chapter card grid with the auto-extracted blurb.
	idx, _ := os.ReadFile(filepath.Join(docs, "Read", "heroes", "index.md"))
	for _, want := range []string{`<div class="sc-cards">`, ">Ancestries<", "Fantastic peoples inhabit the worlds of Draw Steel.", iconPaths["chapter"]} {
		if !strings.Contains(string(idx), want) {
			t.Errorf("heroes index missing %q:\n%s", want, idx)
		}
	}
}

func TestBuildGroupByBookRewritesIntraBookLinks(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	docs := filepath.Join(dir, "docs")
	// the-basics links to a sibling chapter (downtime-projects) the way the ETL
	// emits it for the flat chapter/ layout: a bare basename.
	writeFile(t, filepath.Join(src, "chapter", "the-basics.md"),
		"---\nname: The Basics\nscc: mcdm.heroes.v1/chapter/the-basics\ntype: chapter\norder: 1\n---\n\nSee [downtime](downtime-projects.md).\n")
	writeFile(t, filepath.Join(src, "chapter", "downtime-projects.md"),
		"---\nname: Downtime Projects\nscc: mcdm.heroes.v1/chapter/downtime-projects\ntype: chapter\norder: 2\n---\n\nDowntime.\n")
	cfg := &Config{
		SourceDirs: []string{src},
		DocsDir:    docs,
		Books:      []BookConfig{{Key: "mcdm.heroes.v1", Folder: "heroes", Label: "Draw Steel Heroes", Order: 1}},
		Sections:   []SectionConfig{{Name: "Read", Include: []string{"chapter/"}, GroupByBook: true}},
	}
	if _, err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}
	data, err := os.ReadFile(filepath.Join(docs, "Read", "heroes", "the-basics.md"))
	if err != nil {
		t.Fatal(err)
	}
	// Link must resolve within the same book folder, NOT to Read/chapter/.
	if strings.Contains(string(data), "chapter/downtime-projects.md") {
		t.Errorf("link still points to old chapter/ path:\n%s", data)
	}
	if !strings.Contains(string(data), "(downtime-projects.md)") {
		t.Errorf("expected sibling link (downtime-projects.md), got:\n%s", data)
	}
}

func TestRewriteSectionLinks_GroupByBookTarget(t *testing.T) {
	sections := []SectionConfig{
		{Name: "Browse", Include: []string{"class/", "condition/"}},
		{Name: "Read", Include: []string{"chapter/"}, GroupByBook: true},
	}
	// A Browse class page (heroes book) linking to the "tests" chapter must
	// resolve into Read/<book>/, not Read/<type>/ or Read/chapter/.
	got := rewriteSectionLinks(
		"See [Tests](../chapter/tests.md).",
		"class/censor.md", "class/censor.md", "Browse", "heroes", sections, nil)
	want := "See [Tests](../../Read/heroes/tests.md)."
	if got != want {
		t.Errorf("Browse->Read GroupByBook link:\n  got  %q\n  want %q", got, want)
	}

	// A Read chapter (beastheart) linking to a sibling chapter stays in its book.
	got2 := rewriteSectionLinks(
		"See [Rewards](rewards.md).",
		"chapter/the-beastheart-class.md", "beastheart/the-beastheart-class.md", "Read", "beastheart", sections, nil)
	want2 := "See [Rewards](rewards.md)."
	if got2 != want2 {
		t.Errorf("Read intra-book link:\n  got  %q\n  want %q", got2, want2)
	}
}

// TestRewriteSectionLinks_KitFlattenTarget verifies an inbound link to a kit
// ability is rewritten to the flattened destination
// (feature/ability/<Label>/<kit>-<ability>.md), mirroring applyGroups. The
// flatten is gated on <sourceDir>/kit/<kit>.md existing, so a real source dir
// is required.
func TestRewriteSectionLinks_KitFlattenTarget(t *testing.T) {
	src := t.TempDir()
	// applyGroups stats kit/sniper.md to confirm "sniper" is a kit ability.
	writeFile(t, filepath.Join(src, "kit", "sniper.md"), "---\nname: Sniper\n---\n")

	sections := []SectionConfig{
		{
			Name:    "Browse",
			Include: []string{"class/", "feature/"},
			Groups:  []GroupConfig{{MatchType: "kit", From: "feature/ability", Label: "Kits", Flatten: true}},
		},
	}

	// A class page linking to a kit ability at its un-flattened path must
	// resolve to the flattened Kits/ page.
	got := rewriteSectionLinks(
		"See [Patient Shot](../feature/ability/sniper/patient-shot.md).",
		"class/tactician.md", "class/tactician.md", "Browse", "", sections, []string{src})
	want := "See [Patient Shot](../feature/ability/Kits/sniper-patient-shot.md)."
	if got != want {
		t.Errorf("kit-flatten link:\n  got  %q\n  want %q", got, want)
	}

	// A non-kit ability (no kit/<x>.md) is left at its un-flattened path.
	got2 := rewriteSectionLinks(
		"See [Gouge](../feature/ability/fury/gouge.md).",
		"class/tactician.md", "class/tactician.md", "Browse", "", sections, []string{src})
	want2 := "See [Gouge](../feature/ability/fury/gouge.md)."
	if got2 != want2 {
		t.Errorf("non-kit ability link should be unchanged:\n  got  %q\n  want %q", got2, want2)
	}
}

func TestBuildBookPlaceholderForEmptyBook(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	docs := filepath.Join(dir, "docs")
	// Only heroes has a chapter; bestiary is configured but has no content.
	writeFile(t, filepath.Join(src, "chapter", "introduction.md"),
		"---\nname: Introduction\nscc: mcdm.heroes.v1/chapter/introduction\ntype: chapter\norder: 0\n---\n\nHero intro.\n")
	cfg := &Config{
		SourceDirs: []string{src},
		DocsDir:    docs,
		Books: []BookConfig{
			{Key: "mcdm.heroes.v1", Folder: "heroes", Label: "Heroes", Order: 1},
			{Key: "mcdm.monsters.v1", Folder: "bestiary", Label: "Bestiary", Order: 2},
		},
		Sections: []SectionConfig{{Name: "Read", Title: "Books", Include: []string{"chapter/"}, GroupByBook: true}},
	}
	if _, err := Build(cfg); err != nil {
		t.Fatalf("build: %v", err)
	}
	// Empty book still gets a folder + placeholder index + nav.
	idx, err := os.ReadFile(filepath.Join(docs, "Read", "bestiary", "index.md"))
	if err != nil {
		t.Fatalf("expected bestiary placeholder index: %v", err)
	}
	if !strings.Contains(string(idx), "Bestiary") {
		t.Errorf("placeholder index missing label:\n%s", idx)
	}
	if _, err := os.Stat(filepath.Join(docs, "Read", "bestiary", ".nav.yml")); err != nil {
		t.Errorf("expected bestiary .nav.yml: %v", err)
	}
	// Landing + section nav include the empty book, after heroes.
	readNav, _ := os.ReadFile(filepath.Join(docs, "Read", ".nav.yml"))
	if h, b := strings.Index(string(readNav), "heroes"), strings.Index(string(readNav), "bestiary"); h < 0 || b < 0 || h > b {
		t.Errorf("Read nav missing/ordered wrong:\n%s", readNav)
	}
	landing, _ := os.ReadFile(filepath.Join(docs, "Read", "index.md"))
	if !strings.Contains(string(landing), "bestiary/") {
		t.Errorf("landing index missing bestiary:\n%s", landing)
	}
}

func TestGroupLandingIndexDest(t *testing.T) {
	cases := []struct {
		in       string
		wantDest string
		wantOK   bool
	}{
		{"skill/group/crafting.md", "skill/crafting/index.md", true},
		{"monster/group/goblins.md", "monster/goblins/index.md", true},
		{"skill/crafting/cooking.md", "", false},           // leaf skill, not a landing
		{"monster/goblins/statblock/cutter.md", "", false}, // statblock
		{"feature/ability/fury/level-1/gouge.md", "", false},
	}
	for _, c := range cases {
		dest, ok := groupLandingIndexDest(c.in)
		if ok != c.wantOK || dest != c.wantDest {
			t.Errorf("groupLandingIndexDest(%q) = (%q,%v), want (%q,%v)",
				c.in, dest, ok, c.wantDest, c.wantOK)
		}
	}
}

func TestStripLeadingHeading(t *testing.T) {
	in := "# Crafting\n\n---\n\n<div class=\"sc-cards\">X</div>\n"
	want := "<div class=\"sc-cards\">X</div>\n"
	if got := stripLeadingHeading(in); got != want {
		t.Errorf("stripLeadingHeading = %q, want %q", got, want)
	}
	// No leading heading → unchanged.
	plain := "no heading here"
	if got := stripLeadingHeading(plain); got != plain {
		t.Errorf("stripLeadingHeading(plain) = %q, want unchanged", got)
	}
}

func TestLoreIntro(t *testing.T) {
	// Skill-group body: H1 + intro, then a "## Skills Table" + inline child skills.
	// Truncate at the first H2 → keep only H1 + intro.
	in := "# Crafting Skills\n\n---\n\nIntro prose.\n\n## Crafting Skills Table\n\n| Skill | Use |\n|---|---|\n| Alchemy | bombs |\n\n## Alchemy\n\nMake bombs.\n"
	want := "# Crafting Skills\n\n---\n\nIntro prose."
	if got := loreIntro(in); got != want {
		t.Errorf("loreIntro = %q, want %q", got, want)
	}
	// Monster lore: no H2 → kept whole (trimmed).
	noH2 := "# Goblins\n\nThey are crafty."
	if got := loreIntro(noH2); got != noH2 {
		t.Errorf("loreIntro(noH2) = %q, want unchanged", got)
	}
}

func TestMergeGroupLanding(t *testing.T) {
	dir := t.TempDir()
	landing := "---\nname: Crafting Skills\nscc: mcdm.heroes.v1/skill.group/crafting\ntype: skill-group\n---\n# Crafting Skills\n\nThe crafting group makes things.\n\n## Crafting Skills Table\n\n| Skill | Desc |\n|---|---|\n| Cooking | food |\n\n## Cooking\n\nCook food.\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(landing), 0644); err != nil {
		t.Fatal(err)
	}
	generated := "# Crafting\n\n---\n\n<div class=\"sc-cards\">CARDS</div>\n"
	got := mergeGroupLanding(dir, generated)

	if !strings.Contains(got, "scc: mcdm.heroes.v1/skill.group/crafting") {
		t.Error("merged index lost the scc frontmatter")
	}
	if !strings.Contains(got, "The crafting group makes things.") {
		t.Error("merged index lost the lore")
	}
	if strings.Contains(got, "| Cooking | food |") {
		t.Error("merged index kept the redundant skills table")
	}
	if !strings.Contains(got, "<div class=\"sc-cards\">CARDS</div>") {
		t.Error("merged index lost the generated card grid")
	}
	if strings.Contains(got, "# Crafting\n\n---") {
		t.Error("merged index kept the generated duplicate H1")
	}
}

func TestMergeGroupLandingNoSCCPassthrough(t *testing.T) {
	dir := t.TempDir()
	// index.md without scc (a normal generated index) → generated returned as-is.
	os.WriteFile(filepath.Join(dir, "index.md"), []byte("# X\n\n---\n\nlist"), 0644)
	generated := "# X\n\n---\n\nlist"
	if got := mergeGroupLanding(dir, generated); got != generated {
		t.Errorf("mergeGroupLanding passthrough = %q, want unchanged", got)
	}
}

func TestBuild_PrintingStamps(t *testing.T) {
	srcDir := t.TempDir()
	classDir := filepath.Join(srcDir, "class")
	os.MkdirAll(classDir, 0755)
	page := "---\nname: Fury\nscc: mcdm.heroes.v1/class/fury\ntype: class\n---\n\nBody.\n"
	if err := os.WriteFile(filepath.Join(classDir, "fury.md"), []byte(page), 0644); err != nil {
		t.Fatal(err)
	}

	regPath := filepath.Join(t.TempDir(), "classification.json")
	reg := scc.NewRegistry()
	reg.Add("mcdm.heroes.v1/class/fury")
	reg.SetBookPrinting("mcdm.heroes.v1", "1.01b")
	if err := reg.Save(regPath); err != nil {
		t.Fatal(err)
	}

	docsDir := filepath.Join(t.TempDir(), "docs")
	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Registry:  regPath,
		Books:     []BookConfig{{Key: "mcdm.heroes.v1", Label: "Heroes"}},
		Sections: []SectionConfig{
			{Name: "Browse", Include: []string{"class/"}},
		},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build: %v", err)
	}
	if result.PrintingStamps == 0 {
		t.Fatal("expected at least one printing stamp")
	}

	data, err := os.ReadFile(filepath.Join(docsDir, "Browse", "class", "fury.md"))
	if err != nil {
		t.Fatal(err)
	}
	got := string(data)
	if !strings.Contains(got, "printing: \"1.01b\"") {
		t.Errorf("missing printing frontmatter:\n%s", got)
	}
	if !strings.Contains(got, "printing_book: \"Heroes\"") {
		t.Errorf("missing printing_book frontmatter:\n%s", got)
	}
}
