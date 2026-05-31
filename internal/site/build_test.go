package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupSourceDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	files := map[string]string{
		"class/fury.md":                                    "---\nname: Fury\ntype: class\n---\n\nFury description.",
		"class/shadow.md":                                  "---\nname: Shadow\ntype: class\n---\n\nShadow description.",
		"feature/ability/fury/level-1/gouge.md":            "---\nname: Gouge\ntype: ability\n---\n\nGouge text.",
		"feature/ability/fury/level-1/brutal-slam.md":      "---\nname: Brutal Slam\ntype: ability\n---\n\nSlam text.",
		"feature/trait/fury/level-1/growing-ferocity.md":   "---\nname: Growing Ferocity\ntype: trait\n---\n\nFerocity text.",
		"condition/dazed.md":                               "---\nname: Dazed\ntype: condition\n---\n\nDazed text.",
		"chapter/classes.md":                               "# Classes\n\nChapter intro.",
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
	if !strings.Contains(content, "[Fury](fury.md)") {
		t.Error("class index missing Fury link")
	}
	if !strings.Contains(content, "[Shadow](shadow.md)") {
		t.Error("class index missing Shadow link")
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
	// Subdirectories that themselves contain content are rendered as collapsible
	// <details> sections with a linked <summary>, not plain markdown links.
	if !strings.Contains(content, `<summary><a href="ability/">Abilities</a></summary>`) {
		t.Error("feature index missing ability subdir summary link")
	}
	if !strings.Contains(content, `<summary><a href="trait/">Traits</a></summary>`) {
		t.Error("feature index missing trait subdir summary link")
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
		"feature/ability/fury/level-1/gouge.md":              "---\nname: Gouge\n---\n\nGouge text.",
		"feature/ability/arcane-archer/exploding-arrow.md":   "---\nname: Exploding Arrow\n---\n\nArrow text.",
		"kit/arcane-archer.md":                               "---\nname: Arcane Archer\ntype: kit\n---\n\nKit desc.",
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
		{Name: "Browse", Include: []string{"class/", "feature/", "condition/", "ancestry/", "kit/"}},
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := rewriteSectionLinks(tt.content, tt.srcRelPath, tt.destRelPath, tt.sectionName, sections)
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
