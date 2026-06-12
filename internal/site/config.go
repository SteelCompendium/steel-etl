package site

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config defines how SCC-based output maps to a MkDocs site structure.
type Config struct {
	// Source directory containing steel-etl md-linked output (legacy singular form)
	SourceDir string `yaml:"source_dir"`

	// SourceDirs lists multiple md-linked output directories to merge (multi-book).
	// If empty, the singular SourceDir is used. Resolved relative to ConfigDir.
	SourceDirs []string `yaml:"source_dirs"`

	// MkDocs docs directory (output)
	DocsDir string `yaml:"docs_dir"`

	// Sections define the tab layout
	Sections []SectionConfig `yaml:"sections"`

	// SearchExclude lists sections where pages get search: exclude: true frontmatter
	SearchExclude []string `yaml:"search_exclude"`

	// StaticContent is a directory whose contents are copied over docs (overrides)
	StaticContent string `yaml:"static_content"`

	// Books maps a book's SCC prefix to a display folder/label/order for
	// per-book section grouping (used by sections with GroupByBook=true).
	Books []BookConfig `yaml:"books,omitempty"`

	// Registry is the path to the pipeline's SCC registry (classification.json),
	// used to read per-book printing provenance for page stamps. Optional —
	// empty disables stamping. Resolved relative to ConfigDir.
	Registry string `yaml:"registry,omitempty"`

	// ConfigDir is the directory containing the config file (set automatically).
	// All relative paths are resolved against this directory.
	ConfigDir string `yaml:"-"`
}

// ResolvePath resolves a path relative to the config file directory.
func (c *Config) ResolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.ConfigDir, p)
}

// normalizeSources resolves source paths relative to the config directory and
// folds the legacy singular SourceDir into SourceDirs when no list is given.
func (c *Config) normalizeSources() {
	if c.SourceDir != "" {
		c.SourceDir = c.ResolvePath(c.SourceDir)
	}
	for i, d := range c.SourceDirs {
		c.SourceDirs[i] = c.ResolvePath(d)
	}
	if len(c.SourceDirs) == 0 && c.SourceDir != "" {
		c.SourceDirs = []string{c.SourceDir}
	}
}

// SourceDirList returns the resolved source directories, falling back to the
// legacy singular SourceDir for configs constructed without normalizeSources.
func (c *Config) SourceDirList() []string {
	if len(c.SourceDirs) > 0 {
		return c.SourceDirs
	}
	if c.SourceDir != "" {
		return []string{c.SourceDir}
	}
	return nil
}

// SectionConfig maps SCC content types to a site section (tab).
type SectionConfig struct {
	// Name is the tab/directory name (e.g., "Browse", "Read", "Bestiary")
	Name string `yaml:"name"`

	// Title overrides the directory title in navigation
	Title string `yaml:"title,omitempty"`

	// Include lists SCC type prefixes to include (e.g., "class", "feature/ability")
	// Empty means include everything not matched by other sections.
	Include []string `yaml:"include,omitempty"`

	// Exclude lists SCC type prefixes to exclude from this section
	Exclude []string `yaml:"exclude,omitempty"`

	// Sort configures how pages are sorted in navigation
	Sort string `yaml:"sort,omitempty"` // "natural", "alphabetical"

	// GroupBy groups pages into subdirectories by this SCC path component
	// e.g., "class" groups abilities by their class (fury/, shadow/, etc.)
	GroupBy string `yaml:"group_by,omitempty"`

	// Groups remap subdirectories into a named group folder based on cross-referencing
	// another source type directory. For example, kit abilities in feature/ability/ can
	// be moved under feature/ability/Kits/ by matching against the kit/ source directory.
	Groups []GroupConfig `yaml:"groups,omitempty"`

	// GroupByBook places each page into a per-book subfolder (derived from the
	// page's scc prefix via Config.Books) instead of its SCC type path, and
	// emits source-ordered nav + per-book index pages.
	GroupByBook bool `yaml:"group_by_book,omitempty"`
}

// BookConfig maps a book's SCC prefix (substring before the first '/') to a
// display folder slug, human label, and sort order for the Read tab.
type BookConfig struct {
	Key    string `yaml:"key"`
	Folder string `yaml:"folder"`
	Label  string `yaml:"label"`
	Order  int    `yaml:"order"`
	// Description is a hand-authored blurb shown on the book's card in the
	// Read-section landing index. Optional.
	Description string `yaml:"description,omitempty"`
	// Icon is the iconPaths key for the book card's crest (e.g. "sword-cross").
	// Empty falls back to the generic "book" glyph. See iconPaths in cards.go.
	Icon string `yaml:"icon,omitempty"`
}

// BookByKey returns the BookConfig whose Key matches, and whether it was found.
func (c *Config) BookByKey(key string) (BookConfig, bool) {
	for _, b := range c.Books {
		if b.Key == key {
			return b, true
		}
	}
	return BookConfig{}, false
}

// GroupConfig moves subdirectories into a named group folder based on
// cross-referencing another source type directory.
type GroupConfig struct {
	// MatchType is the source type directory to cross-reference (e.g., "kit").
	// If a subdirectory name matches a file in this source directory, it is grouped.
	MatchType string `yaml:"match_type"`

	// From is the path prefix to match (e.g., "feature/ability").
	From string `yaml:"from"`

	// Label is the group directory name (e.g., "Kits").
	Label string `yaml:"label"`

	// Flatten collapses {parent}/{child}.md into a single {parent}-{child}.md
	// file directly under Label/, and rewrites the file's frontmatter "name"
	// to "Parent Title (Original Name)" so the page heading and nav title
	// show both the matched parent and the child name. Used for kits where
	// each parent has exactly one child page.
	Flatten bool `yaml:"flatten,omitempty"`
}

// LoadSiteConfig reads a site configuration file.
// Relative paths in the config are resolved against the config file's directory.
func LoadSiteConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read site config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse site config: %w", err)
	}

	cfg.ConfigDir = filepath.Dir(path)
	if !filepath.IsAbs(cfg.ConfigDir) {
		abs, err := filepath.Abs(cfg.ConfigDir)
		if err != nil {
			return nil, fmt.Errorf("resolve config dir: %w", err)
		}
		cfg.ConfigDir = abs
	}

	// Resolve all paths relative to the config file directory
	cfg.normalizeSources()
	cfg.DocsDir = cfg.ResolvePath(cfg.DocsDir)
	if cfg.StaticContent != "" {
		cfg.StaticContent = cfg.ResolvePath(cfg.StaticContent)
	}

	return &cfg, nil
}
