package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ContentParser extracts structured data from an annotated markdown section.
type ContentParser interface {
	// Type returns the @type value this parser handles.
	Type() string

	// Parse extracts structured metadata from a section.
	Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error)
}

// ParsedContent holds the extracted structured data from a section.
type ParsedContent struct {
	// Frontmatter fields for YAML output.
	Frontmatter map[string]any

	// The content body (markdown) with annotations stripped.
	Body string

	// PageBody is a full book-order render of this section's subtree (own body +
	// all nested descendants inline), used by reading-format outputs (md-linked).
	// Empty for sections where it is not populated; consumers fall back to Body.
	PageBody string

	// SCC classification components derived by the parser.
	TypePath []string // e.g., ["abilities", "fury"]
	ItemID   string   // e.g., "gouge"

	// Children holds parsed sub-content that should be embedded in the parent.
	// For example, a kit's signature ability is stored under "signature_ability".
	Children map[string]*ParsedContent
}
