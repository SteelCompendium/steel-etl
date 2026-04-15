package site

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

// SCCMapEntry matches the output of SCCMapGenerator.
type SCCMapEntry struct {
	SCC  string `json:"scc"`
	Path string `json:"path"`
	Name string `json:"name"`
	Type string `json:"type"`
}

// BuildResult holds the outcome of a site build.
type BuildResult struct {
	CopiedFiles   int
	Sections      int
	NavFiles      int
	SearchExclude int
	Errors        []string
}

// Build generates the MkDocs site structure from steel-etl output.
func Build(cfg *Config) (*BuildResult, error) {
	result := &BuildResult{}

	// Clean docs dir (except protected paths)
	if err := cleanDocsDir(cfg.DocsDir); err != nil {
		return nil, fmt.Errorf("clean docs: %w", err)
	}

	// Read scc-to-path.json if available (for metadata), but primarily
	// we walk the source directory to copy files
	entries, err := walkSourceDir(cfg.SourceDir)
	if err != nil {
		return nil, fmt.Errorf("walk source: %w", err)
	}

	// Map files to sections
	for _, section := range cfg.Sections {
		count, errs := buildSection(cfg, section, entries)
		result.CopiedFiles += count
		result.Errors = append(result.Errors, errs...)
		result.Sections++
	}

	// Write .nav.yml files
	for _, section := range cfg.Sections {
		if err := writeNavYaml(cfg.DocsDir, section); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("nav %s: %v", section.Name, err))
		} else {
			result.NavFiles++
		}
	}

	// Apply search exclusion
	for _, sectionName := range cfg.SearchExclude {
		count, errs := applySearchExclusion(cfg.DocsDir, sectionName)
		result.SearchExclude += count
		result.Errors = append(result.Errors, errs...)
	}

	// Copy static content overrides
	if cfg.StaticContent != "" {
		count, err := copyStaticContent(cfg.StaticContent, cfg.DocsDir)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("static content: %v", err))
		}
		result.CopiedFiles += count
	}

	return result, nil
}

// sourceEntry represents a markdown file found in the source directory.
type sourceEntry struct {
	relPath string // relative to source dir (e.g., "class/fury.md")
	absPath string
}

func walkSourceDir(dir string) ([]sourceEntry, error) {
	var entries []sourceEntry
	err := filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}
		rel, _ := filepath.Rel(dir, path)
		entries = append(entries, sourceEntry{relPath: rel, absPath: path})
		return nil
	})
	return entries, err
}

// buildSection copies matching files from source into the section directory.
func buildSection(cfg *Config, section SectionConfig, entries []sourceEntry) (int, []string) {
	sectionDir := filepath.Join(cfg.DocsDir, section.Name)
	count := 0
	var errs []string

	for _, entry := range entries {
		if !matchesSection(entry.relPath, section) {
			continue
		}

		// Determine destination path within the section
		destRel := entry.relPath
		destPath := filepath.Join(sectionDir, destRel)

		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			errs = append(errs, fmt.Sprintf("mkdir %s: %v", destPath, err))
			continue
		}

		data, err := os.ReadFile(entry.absPath)
		if err != nil {
			errs = append(errs, fmt.Sprintf("read %s: %v", entry.absPath, err))
			continue
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", destPath, err))
			continue
		}
		count++
	}

	return count, errs
}

// matchesSection checks if a file's relative path matches the section's include/exclude rules.
func matchesSection(relPath string, section SectionConfig) bool {
	// Normalize path separators
	normalized := filepath.ToSlash(relPath)

	// Check excludes first
	for _, excl := range section.Exclude {
		if strings.HasPrefix(normalized, excl) {
			return false
		}
	}

	// If no includes, match everything
	if len(section.Include) == 0 {
		return true
	}

	// Check includes
	for _, incl := range section.Include {
		if strings.HasPrefix(normalized, incl) {
			return true
		}
	}

	return false
}

// writeNavYaml creates a .nav.yml file for the section.
func writeNavYaml(docsDir string, section SectionConfig) error {
	sectionDir := filepath.Join(docsDir, section.Name)
	if _, err := os.Stat(sectionDir); os.IsNotExist(err) {
		return nil // section dir doesn't exist, skip
	}

	nav := map[string]any{}
	if section.Title != "" {
		nav["title"] = section.Title
	}
	if section.Sort != "" {
		nav["sort"] = map[string]string{
			"type": section.Sort,
			"by":   "title",
		}
	}

	// If the only content is title or sort, use simpler YAML format
	data, err := yaml.Marshal(nav)
	if err != nil {
		return fmt.Errorf("marshal nav: %w", err)
	}

	return os.WriteFile(filepath.Join(sectionDir, ".nav.yml"), data, 0644)
}

// applySearchExclusion adds search: exclude: true frontmatter to all .md files in a section.
func applySearchExclusion(docsDir, sectionName string) (int, []string) {
	sectionDir := filepath.Join(docsDir, sectionName)
	if _, err := os.Stat(sectionDir); os.IsNotExist(err) {
		return 0, nil
	}

	count := 0
	var errs []string

	filepath.Walk(sectionDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".md") {
			return nil
		}

		data, err := os.ReadFile(path)
		if err != nil {
			errs = append(errs, fmt.Sprintf("read %s: %v", path, err))
			return nil
		}

		content := string(data)
		if strings.HasPrefix(content, "---\n") {
			// Has frontmatter — inject search exclude after opening ---
			rest := content[4:]
			content = "---\nsearch:\n  exclude: true\n" + rest
		} else {
			// No frontmatter — prepend it
			content = "---\nsearch:\n  exclude: true\n---\n\n" + content
		}

		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			errs = append(errs, fmt.Sprintf("write %s: %v", path, err))
			return nil
		}
		count++
		return nil
	})

	return count, errs
}

// cleanDocsDir removes generated content from the docs directory,
// preserving protected paths (stylesheets, javascripts, Media, index.md, etc.)
func cleanDocsDir(docsDir string) error {
	protected := map[string]bool{
		"javascripts":  true,
		"stylesheets":  true,
		"Media":        true,
		"index.md":     true,
		"preferences.md": true,
		".nav.yml":     true,
	}

	entries, err := os.ReadDir(docsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(docsDir, 0755)
		}
		return err
	}

	for _, entry := range entries {
		if protected[entry.Name()] {
			continue
		}
		path := filepath.Join(docsDir, entry.Name())
		if err := os.RemoveAll(path); err != nil {
			return fmt.Errorf("remove %s: %w", path, err)
		}
	}

	return nil
}

// copyStaticContent copies static content overrides into the docs directory.
func copyStaticContent(srcDir, docsDir string) (int, error) {
	count := 0
	err := filepath.Walk(srcDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}

		rel, _ := filepath.Rel(srcDir, path)
		dest := filepath.Join(docsDir, rel)

		if err := os.MkdirAll(filepath.Dir(dest), 0755); err != nil {
			return err
		}

		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		if err := os.WriteFile(dest, data, 0644); err != nil {
			return err
		}
		count++
		return nil
	})
	return count, err
}

// LoadSCCMap reads scc-to-path.json and returns the entries sorted by SCC code.
func LoadSCCMap(path string) ([]SCCMapEntry, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read scc map %s: %w", path, err)
	}

	var entries []SCCMapEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return nil, fmt.Errorf("parse scc map: %w", err)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].SCC < entries[j].SCC
	})

	return entries, nil
}
