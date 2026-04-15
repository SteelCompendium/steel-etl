package site

import (
	"fmt"
	"os"

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
func LoadSiteConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read site config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse site config: %w", err)
	}

	return &cfg, nil
}
