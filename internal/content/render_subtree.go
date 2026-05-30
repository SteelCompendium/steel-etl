package content

import (
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// RenderSubtree serializes a section's entire subtree as book-order markdown:
// the section's own immediate body, followed by every descendant (annotated or
// not) inline in document order. Heading levels are normalized so the section
// itself occupies the page's H1 (added separately by the site builder via H1
// injection) and descendants nest by their source depth. Ability statblocks
// (sections with @type: ability), which are blockquoted in source, are
// un-blockquoted to match how standalone ability pages render; genuine flavor
// blockquotes (which are not ability sections) are preserved.
//
// scc: links are left in their raw form; the md-linked generator resolves them
// relative to the page's own SCC code.
func RenderSubtree(section *parser.Section) string {
	return renderSubtree(section, section.HeadingLevel)
}

func renderSubtree(section *parser.Section, rootLevel int) string {
	var parts []string

	if body := nodeBody(section); body != "" {
		parts = append(parts, body)
	}

	for _, child := range section.Children {
		level := 1 + (child.HeadingLevel - rootLevel)
		if level < 1 {
			level = 1
		}
		if level > 6 {
			level = 6
		}
		heading := strings.Repeat("#", level) + " " + CleanHeading(child.Heading)
		childBody := renderSubtree(child, rootLevel)
		if childBody != "" {
			parts = append(parts, heading+"\n\n"+childBody)
		} else {
			parts = append(parts, heading)
		}
	}

	return strings.Join(parts, "\n\n")
}

// nodeBody returns a section's immediate body, un-blockquoted for ability
// sections (whose statblocks are blockquoted in source).
func nodeBody(section *parser.Section) string {
	body := section.BodySource
	if section.Type() == "ability" {
		body = stripBlockquotePrefix(body)
	}
	return body
}
