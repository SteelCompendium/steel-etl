package output

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

// Generator writes output files for parsed sections.
type Generator interface {
	// Format returns the generator's format identifier (e.g., "md", "json", "yaml").
	Format() string

	// WriteSection writes a single parsed section to the output directory.
	WriteSection(sccCode string, parsed *content.ParsedContent) error
}

// BulkGenerator can process the entire input or all sections at once.
// Used for generators that operate on the full document (e.g., stripped, aggregate).
type BulkGenerator interface {
	// Finalize is called after all sections have been written.
	// Use for aggregation, index generation, etc.
	Finalize() error
}

// Cleanable is implemented by generators that write to a directory which
// should be wiped before each run to remove stale output from previous builds.
type Cleanable interface {
	CleanDir() string
}

// CleanGeneratorDirs removes and re-creates the output directory for each
// generator that implements Cleanable. Call before writing any sections.
func CleanGeneratorDirs(generators []Generator) error {
	for _, gen := range generators {
		c, ok := gen.(Cleanable)
		if !ok {
			continue
		}
		dir := c.CleanDir()
		if dir == "" {
			continue
		}
		if err := os.RemoveAll(dir); err != nil {
			return fmt.Errorf("clean %s (%s): %w", gen.Format(), dir, err)
		}
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("recreate %s (%s): %w", gen.Format(), dir, err)
		}
		fmt.Fprintf(os.Stderr, "Cleaned: %s\n", filepath.Base(dir))
	}
	return nil
}
