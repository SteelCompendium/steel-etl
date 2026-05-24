package pipeline

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/scc"
	"gopkg.in/yaml.v3"
)

// Config represents the full pipeline.yaml configuration.
type Config struct {
	Book           string              `yaml:"book"`
	Input          string              `yaml:"input"`
	Locale         string              `yaml:"locale"`
	I18nDir        string              `yaml:"i18n_dir"` // e.g., "./input/i18n" — locale input files live at {i18n_dir}/{locale}/...
	Classification ClassificationConfig `yaml:"classification"`
	Output         OutputConfig        `yaml:"output"`
	Parsers        ParsersConfig       `yaml:"parsers"`
	Books          []BookConfig        `yaml:"books"`

	// Resolved paths (set after loading)
	ConfigDir string `yaml:"-"`
}

type ClassificationConfig struct {
	Registry string `yaml:"registry"`
	Freeze   bool   `yaml:"freeze"`
}

type OutputConfig struct {
	BaseDir   string          `yaml:"base_dir"`
	Formats   []string        `yaml:"formats"`
	Variants  VariantsConfig  `yaml:"variants"`
	LinkMode  string          `yaml:"link_mode"`
	Stripped  StrippedConfig  `yaml:"stripped"`
	Aggregate AggregateConfig `yaml:"aggregate"`
	SCCMap    SCCMapConfig    `yaml:"scc_map"`
	SCCAPI    SCCAPIConfig    `yaml:"scc_api"`
}

// ParseLinkMode converts the string LinkMode config value to the typed enum.
func (o *OutputConfig) ParseLinkMode() scc.LinkMode {
	switch strings.ToLower(o.LinkMode) {
	case "first":
		return scc.LinkFirst
	case "none":
		return scc.LinkNone
	default:
		return scc.LinkAll
	}
}

type VariantsConfig struct {
	Linked    bool `yaml:"linked"`
	DSE       bool `yaml:"dse"`
	DSELinked bool `yaml:"dse_linked"`
}

type StrippedConfig struct {
	Enabled   bool   `yaml:"enabled"`
	OutputDir string `yaml:"output_dir"`
}

type AggregateConfig struct {
	Enabled   bool   `yaml:"enabled"`
	OutputDir string `yaml:"output_dir"`
}

type SCCMapConfig struct {
	Enabled    bool   `yaml:"enabled"`
	OutputFile string `yaml:"output_file"`
}

type SCCAPIConfig struct {
	Enabled    bool   `yaml:"enabled"`
	OutputDir  string `yaml:"output_dir"`
	BaseURL    string `yaml:"base_url"`
	SiteConfig string `yaml:"site_config"` // path to site.yaml for section mapping
}

type ParsersConfig struct {
	Overrides []ParserOverride `yaml:"overrides"`
}

type ParserOverride struct {
	Type   string `yaml:"type"`
	Parser string `yaml:"parser"`
}

// BookConfig represents a secondary book in a multi-book pipeline.
type BookConfig struct {
	Book   string       `yaml:"book"`
	Input  string       `yaml:"input"`
	Output OutputConfig `yaml:"output"`
}

// LoadConfig reads and parses a pipeline.yaml file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	cfg.ConfigDir = filepath.Dir(path)

	// Defaults
	if cfg.Locale == "" {
		cfg.Locale = "en"
	}
	if len(cfg.Output.Formats) == 0 {
		cfg.Output.Formats = []string{"md"}
	}

	return &cfg, nil
}

// ResolvePath resolves a path relative to the config file directory.
func (c *Config) ResolvePath(p string) string {
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(c.ConfigDir, p)
}

// HasFormat returns true if the given format is in the output formats list.
func (c *Config) HasFormat(format string) bool {
	for _, f := range c.Output.Formats {
		if f == format {
			return true
		}
	}
	return false
}

// ResolveInputPath returns the input file path for the configured locale.
// For "en" (or when no i18n_dir is set), returns the default input path.
// For other locales, looks for the input file under {i18n_dir}/{locale}/.
// The translated file is expected to mirror the default input's basename.
func (c *Config) ResolveInputPath() string {
	defaultPath := c.ResolvePath(c.Input)

	if c.Locale == "en" || c.Locale == "" || c.I18nDir == "" {
		return defaultPath
	}

	base := filepath.Base(c.Input)
	localePath := filepath.Join(c.ResolvePath(c.I18nDir), c.Locale, base)

	// Fall back to default if the locale file doesn't exist
	if _, err := os.Stat(localePath); os.IsNotExist(err) {
		return defaultPath
	}

	return localePath
}
