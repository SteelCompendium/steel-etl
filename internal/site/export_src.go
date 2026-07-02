package site

// Export-source island: carded leaf pages (ability/statblock/featureblock/
// kit/companion) get their ORIGINAL markdown body stashed in a hidden
// <template class="sc-src"> so the v2 "Copy as Markdown" control
// (sc-export.js) can read it client-side. <template> content is inert —
// browsers don't render it and the search indexer doesn't see it. SITE-ONLY.
// See workspace docs/superpowers/plans/2026-07-01-p10-card-exports.md.

import (
	"html"
	"regexp"
	"strings"
)

// appendSourceTemplate appends the pre-card markdown body to a carded page,
// carried in a data-src ATTRIBUTE (not element content — python-markdown
// re-enters raw HTML blocks at blank lines and would render the markdown).
// html.EscapeString escapes quotes, so the attribute is safe; newlines are
// legal inside attribute values.
func appendSourceTemplate(carded []byte, origBody string) []byte {
	// Newlines become &#10; so the whole tag stays on ONE line — python-
	// markdown's raw-HTML block detection is line-based and a multi-line
	// attribute lets markdown processing leak back in mid-tag.
	src := strings.ReplaceAll(html.EscapeString(strings.TrimSpace(origBody)), "\n", "&#10;")
	out := string(carded)
	out += "\n\n<template class=\"sc-src\" data-fmt=\"md\" data-src=\"" + src + "\"></template>\n"
	return []byte(out)
}

// srcTemplateRe matches a stashed source template (appendSourceTemplate) for
// removal when a leaf card is transcluded into a container page — containers
// must not accumulate one hidden source copy per embedded card.
var srcTemplateRe = regexp.MustCompile(`(?s)\n*<template class="sc-src"[^>]*>.*?</template>\n*`)

// dropSourceTemplate strips the sc-src island from card html.
func dropSourceTemplate(s string) string {
	return strings.TrimSpace(srcTemplateRe.ReplaceAllString(s, "\n"))
}
