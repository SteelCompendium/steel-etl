package content

import (
	"regexp"
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
// sccBySection maps a descendant section to its final (post-override) SCC code.
// Each descendant heading that has a code gets an attr_list `{data-scc="<code>"}`
// marker so the v2 client can offer a stable /scc/<code>/ permalink on that
// heading's anchor icon. A nil map emits no markers. Headings without a code
// (structural sections) are left plain. attr_list (enabled in v2/mkdocs.yml)
// turns the marker into a data-scc attribute on the rendered <hN> without
// affecting the toc-generated heading id.
//
// An unclassified inline statblock (a `@type: statblock | @classify: false`
// descendant — has no SCC code, so never appears in sccBySection) instead gets a
// `{data-sb-inline="true"}` marker. The v2 embed_cards post-pass keys off it to
// build a .sb-wrap card from the inline markdown; data/ keeps the raw table.
//
// scc: links in bodies are left in their raw form; the md-linked generator
// resolves them relative to the page's own SCC code.
func RenderSubtree(section *parser.Section, sccBySection map[*parser.Section]string) string {
	return renderSubtree(section, section.HeadingLevel, sccBySection)
}

func renderSubtree(section *parser.Section, rootLevel int, sccBySection map[*parser.Section]string) string {
	var parts []string

	if body := nodeBody(section, section.HeadingLevel == rootLevel); body != "" {
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
		var attrs []string
		if code := sccBySection[child]; code != "" {
			attrs = append(attrs, `data-scc="`+code+`"`)
		} else if child.Type() == "statblock" && child.NoClassify() {
			// Unclassified inline statblock (@classify: false): no SCC code, so it
			// carries a data-sb-inline marker instead. The raw stat table + feature
			// blockquotes still render here verbatim (faithful for the data/ output);
			// the v2 embed_cards post-pass upgrades the marked region to a .sb-wrap card.
			attrs = append(attrs, `data-sb-inline="true"`)
		}
		// Preserve the point cost CleanHeading strips (ancestry purchased traits:
		// "(1 Point)") so embedded card renders (ancestry/Read pages) can show it;
		// the standalone leaf reads it from frontmatter instead.
		if cost := extractCostSuffix(child.Heading); cost != "" {
			attrs = append(attrs, `data-cost="`+cost+`"`)
		}
		if len(attrs) > 0 {
			heading += ` {` + strings.Join(attrs, " ") + `}`
		}
		childBody := renderSubtree(child, rootLevel, sccBySection)
		if childBody != "" {
			parts = append(parts, heading+"\n\n"+childBody)
		} else {
			parts = append(parts, heading)
		}
	}

	return strings.Join(parts, "\n\n")
}

// nodeBody returns a section's immediate body, un-blockquoted for ability
// sections (whose statblocks are blockquoted in source), with any overflow
// (7+ hash) heading demoted to bold. When isRoot is true (this section is the
// page's own root, not a descendant), incidental `@owner: loose` callouts are
// stripped — they belong to the section's broader context, not to its own page.
func nodeBody(section *parser.Section, isRoot bool) string {
	body := section.BodySource
	if section.Type() == "ability" {
		body = stripBlockquotePrefix(body)
	}
	if isRoot {
		body = stripLooseCallouts(body)
	}
	return demoteOverflowHeadings(body)
}

// overflowHeadingRe matches an ATX heading deeper than H6 (7+ leading '#').
// CommonMark caps headings at H6, so these render as literal hashes. Draw Steel
// statblocks use H8 for retainer "Level N … Advancement Ability" sub-labels,
// which are intentionally not collected as sections (they fold into the
// statblock body); demote them to bold so they don't leak as raw '########'.
var overflowHeadingRe = regexp.MustCompile(`(?m)^#{7,}[ \t]+(.+?)[ \t]*$`)

// demoteOverflowHeadings rewrites every 7+-hash heading line to a bold label.
func demoteOverflowHeadings(body string) string {
	return overflowHeadingRe.ReplaceAllString(body, `**$1**`)
}
