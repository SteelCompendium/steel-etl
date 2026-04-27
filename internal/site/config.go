package site

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config defines how SCC-based output maps to a MkDocs site structure.
type Config struct {
	// Source directory containing steel-etl md-linked output
	SourceDir string `yaml:"source_dir"`

	// MkDocs docs directory (output)
	DocsDir string `yaml:"docs_dir"`

	// Sections define the tab layout
	Sections []SectionConfig `yaml:"sections"`

	// SearchExclude lists sections where pages get search: exclude: true frontmatter
	SearchExclude []string `yaml:"search_exclude"`

	// StaticContent is a directory whose contents are copied over docs (overrides)
	StaticContent string `yaml:"static_content"`

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
	cfg.SourceDir = cfg.ResolvePath(cfg.SourceDir)
	cfg.DocsDir = cfg.ResolvePath(cfg.DocsDir)
	if cfg.StaticContent != "" {
		cfg.StaticContent = cfg.ResolvePath(cfg.StaticContent)
	}

	return &cfg, nil
}
