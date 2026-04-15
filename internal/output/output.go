package output

import "github.com/SteelCompendium/steel-etl/internal/content"

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
