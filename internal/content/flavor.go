package content

import (
	"regexp"
	"strings"
)

// contentMdLinkRe matches a markdown link [text](target) for stripping.
var contentMdLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

// firstFlavorParagraph returns the first prose paragraph of a section body,
// markdown-stripped, for use as the structured `flavor` field. It mirrors what
// the index-page cards display (first prose paragraph, links/emphasis removed)
// so the data field and the card stay in lockstep — the parser is the single
// source of truth and cards.go reads this value back from frontmatter.
//
// "Prose" excludes headings, tables, blockquotes, list items, horizontal
// rules, and bold "**Label:** value" stat lines (which look like prose but are
// structured data the parser lifts into their own fields).
func firstFlavorParagraph(body string) string {
	for _, raw := range strings.Split(body, "\n") {
		t := strings.TrimSpace(raw)
		if !isFlavorProse(t) {
			continue
		}
		if s := stripInlineMarkdown(t); s != "" {
			return s
		}
	}
	return ""
}

// isFlavorProse reports whether a trimmed line is flavor prose (not a heading,
// table, blockquote, list item, horizontal rule, or bold stat line).
func isFlavorProse(t string) bool {
	if t == "" || t == "---" {
		return false
	}
	if strings.HasPrefix(t, "#") || strings.HasPrefix(t, "|") ||
		strings.HasPrefix(t, ">") || strings.HasPrefix(t, "- ") ||
		strings.HasPrefix(t, "**") {
		return false
	}
	return true
}

// stripInlineMarkdown removes link syntax, bold/italic markers, and inline code
// backticks, returning clean descriptor text.
func stripInlineMarkdown(s string) string {
	s = contentMdLinkRe.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "`", "")
	return strings.TrimSpace(s)
}
