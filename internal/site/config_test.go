package site

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadSiteConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "site.yaml")

	content := `
source_dir: ../output/en/md-linked
docs_dir: ./docs
sections:
  - name: Browse
    include:
      - class/
      - feature/
    sort: natural
  - name: Read
    title: Rulebook Chapters
    include:
      - chapter/
search_exclude:
  - Read
static_content: ./static
`
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadSiteConfig(path)
	if err != nil {
		t.Fatalf("LoadSiteConfig failed: %v", err)
	}

	// Paths are resolved relative to the config file's directory
	wantSource := filepath.Join(dir, "../output/en/md-linked")
	if cfg.SourceDir != wantSource {
		t.Errorf("SourceDir = %q, want %q", cfg.SourceDir, wantSource)
	}
	wantDocs := filepath.Join(dir, "docs")
	if cfg.DocsDir != wantDocs {
		t.Errorf("DocsDir = %q, want %q", cfg.DocsDir, wantDocs)
	}
	if len(cfg.Sections) != 2 {
		t.Errorf("expected 2 sections, got %d", len(cfg.Sections))
	}
	if cfg.Sections[0].Name != "Browse" {
		t.Errorf("first section = %q", cfg.Sections[0].Name)
	}
	if cfg.Sections[0].Sort != "natural" {
		t.Errorf("sort = %q", cfg.Sections[0].Sort)
	}
	if cfg.Sections[1].Title != "Rulebook Chapters" {
		t.Errorf("title = %q", cfg.Sections[1].Title)
	}
	if len(cfg.SearchExclude) != 1 || cfg.SearchExclude[0] != "Read" {
		t.Errorf("SearchExclude = %v", cfg.SearchExclude)
	}
	wantStatic := filepath.Join(dir, "static")
	if cfg.StaticContent != wantStatic {
		t.Errorf("StaticContent = %q, want %q", cfg.StaticContent, wantStatic)
	}
	if cfg.ConfigDir != dir {
		t.Errorf("ConfigDir = %q, want %q", cfg.ConfigDir, dir)
	}
}

func TestLoadSiteConfig_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	os.WriteFile(path, []byte("invalid: [yaml: broken"), 0644)

	_, err := LoadSiteConfig(path)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoadSiteConfig_NonexistentFile(t *testing.T) {
	_, err := LoadSiteConfig("/nonexistent/site.yaml")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestLoadSCCMap(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "scc-map.json")

	content := `[
  {"scc": "mcdm.heroes.v1/condition/dazed", "path": "condition/dazed.md", "name": "Dazed", "type": "condition"},
  {"scc": "mcdm.heroes.v1/class/fury", "path": "class/fury.md", "name": "Fury", "type": "class"}
]`
	os.WriteFile(path, []byte(content), 0644)

	entries, err := LoadSCCMap(path)
	if err != nil {
		t.Fatalf("LoadSCCMap failed: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}

	// Should be sorted by SCC
	if entries[0].SCC != "mcdm.heroes.v1/class/fury" {
		t.Errorf("first entry = %q, expected fury (sorted)", entries[0].SCC)
	}
}

func TestLoadSCCMap_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.json")
	os.WriteFile(path, []byte("not json"), 0644)

	_, err := LoadSCCMap(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestLoadSCCMap_NonexistentFile(t *testing.T) {
	_, err := LoadSCCMap("/nonexistent/scc-map.json")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestBuild_EmptySource(t *testing.T) {
	srcDir := t.TempDir() // empty
	docsDir := filepath.Join(t.TempDir(), "docs")
	os.MkdirAll(docsDir, 0755)

	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Sections:  []SectionConfig{{Name: "Browse"}},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if result.CopiedFiles != 0 {
		t.Errorf("expected 0 files from empty source, got %d", result.CopiedFiles)
	}
}

func TestBuild_NonexistentDocsDir(t *testing.T) {
	srcDir := setupSourceDir(t)
	docsDir := filepath.Join(t.TempDir(), "nonexistent", "docs")

	cfg := &Config{
		SourceDir: srcDir,
		DocsDir:   docsDir,
		Sections:  []SectionConfig{{Name: "Browse", Include: []string{"class/"}}},
	}

	result, err := Build(cfg)
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	// Should create docs dir and copy files
	if result.CopiedFiles < 1 {
		t.Errorf("expected files to be copied, got %d", result.CopiedFiles)
	}
}

func TestCopyStaticContent(t *testing.T) {
	srcDir := t.TempDir()
	destDir := t.TempDir()

	// Create nested static content
	os.MkdirAll(filepath.Join(srcDir, "Browse"), 0755)
	os.WriteFile(filepath.Join(srcDir, "Browse", "override.md"), []byte("override"), 0644)
	os.WriteFile(filepath.Join(srcDir, "index.md"), []byte("home"), 0644)

	count, err := copyStaticContent(srcDir, destDir)
	if err != nil {
		t.Fatalf("copyStaticContent failed: %v", err)
	}

	if count != 2 {
		t.Errorf("expected 2 files copied, got %d", count)
	}

	checkExists(t, destDir, "Browse/override.md")
	checkExists(t, destDir, "index.md")
}
