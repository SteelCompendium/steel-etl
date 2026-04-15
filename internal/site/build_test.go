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
