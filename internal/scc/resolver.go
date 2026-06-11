package scc

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

// LinkMode controls how scc: links are resolved in content.
type LinkMode int

const (
	// LinkAll resolves every scc: link to a relative path.
	LinkAll LinkMode = iota
	// LinkFirst resolves only the first occurrence of each SCC code per call;
	// subsequent occurrences are stripped to display text.
	LinkFirst
	// LinkNone strips all scc: links, leaving only the display text.
	LinkNone
)

// mdLinkRe matches markdown links with scc: protocol URLs, with an optional
// scheme-version prefix (scc.vN:) and an optional trailing #format qualifier.
// Capture groups: 1=display text, 2=".vN" or "" (bare ⇒ v1), 3=canonical
// source/type/item code, 4="#format" or "" (reserved, ignored for identity).
// Format qualifiers (group 4) are lowercase per spec; an uppercase #JSON will
// not match and the link is left verbatim.
// Examples:
//
//	[Gouge](scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge)
//	[Gouge](scc.v1:mcdm.heroes.v1/feature.ability.fury.level-1/gouge#json)
//
// This is the scc-protocol-specific matcher. A protocol-agnostic variant (also
// named mdLinkRe) lives in internal/content/statblock_parse.go; keep both in
// sync if the markdown-link syntax ever changes.
var mdLinkRe = regexp.MustCompile(`\[([^\]]+)\]\(scc(\.v\d+)?:([a-zA-Z0-9._\-]+/[a-zA-Z0-9._\-]+/[a-zA-Z0-9._\-]+)(#[a-z][a-z0-9\-]*)?\)`)

// Resolver resolves SCC codes to relative file paths.
type Resolver struct {
	registry      *Registry
	ext           string // file extension for resolved paths (e.g., ".md")
	schemeVersion int    // scheme version this resolver resolves (from the registry)
}

// NewResolver creates a resolver backed by the given registry.
func NewResolver(registry *Registry, ext string) *Resolver {
	return &Resolver{registry: registry, ext: ext, schemeVersion: registry.SchemeVersion()}
}

// schemeVersionFromTag parses a ".vN" prefix tag (e.g. ".v1") into N. An empty
// tag is the bare scc: form, which is implicit scheme version 1.
func schemeVersionFromTag(tag string) int {
	if tag == "" {
		return 1
	}
	// The regex guarantees tag matches \.v\d+, so Atoi cannot fail here;
	// the error branch is defensive and unreachable in practice.
	n, err := strconv.Atoi(strings.TrimPrefix(tag, ".v"))
	if err != nil {
		return 1
	}
	return n
}

// ResolveLinks replaces scc: protocol links in markdown link syntax with relative file paths.
// The relativeTo parameter is the SCC code of the current file (used to compute relative paths).
// The mode parameter controls link density: LinkAll resolves all, LinkFirst deduplicates per call,
// and LinkNone strips all links to display text.
func (r *Resolver) ResolveLinks(content string, relativeTo string, mode LinkMode) string {
	if mode == LinkNone {
		return mdLinkRe.ReplaceAllString(content, "$1")
	}

	seen := make(map[string]bool)

	return mdLinkRe.ReplaceAllStringFunc(content, func(match string) string {
		sub := mdLinkRe.FindStringSubmatch(match)
		if len(sub) < 4 {
			return match
		}
		displayText := sub[1]
		versionTag := sub[2] // "" (bare ⇒ v1) or ".vN"
		sccCode := sub[3]    // canonical source/type/item; scheme prefix and #fragment excluded by the regex
		// sub[4] is the optional #format qualifier — reserved, ignored for identity.

		// Only links minted under this build's scheme version resolve against this
		// registry. A future scc.v2: reference must NOT silently resolve to v1 content.
		if v := schemeVersionFromTag(versionTag); v != r.schemeVersion {
			fmt.Fprintf(os.Stderr, "WARN: scc link scheme v%d not resolvable in this build (current v%d): %q\n", v, r.schemeVersion, sccCode)
			return displayText
		}

		// Resolve the SCC code (check registry, then aliases)
		resolvedCode := sccCode
		if !r.registry.Contains(resolvedCode) {
			if canonical, ok := r.registry.ResolveAlias(resolvedCode); ok {
				resolvedCode = canonical
			} else {
				// Unresolved link: warn and strip to display text
				fmt.Fprintf(os.Stderr, "WARN: unresolved scc link %q\n", sccCode)
				return displayText
			}
		}

		// In LinkFirst mode, only link the first occurrence of each code
		if mode == LinkFirst {
			if seen[resolvedCode] {
				return displayText
			}
			seen[resolvedCode] = true
		}

		return fmt.Sprintf("[%s](%s)", displayText, sccToRelPathFrom(resolvedCode, relativeTo, r.ext))
	})
}

// ResolveFrontmatter returns a deep copy of a frontmatter map with scc: links
// resolved in every string value (recursing through nested maps and slices).
// The input map is never mutated, since it is shared across output generators.
func (r *Resolver) ResolveFrontmatter(fm map[string]any, relativeTo string, mode LinkMode) map[string]any {
	if fm == nil {
		return nil
	}
	out := make(map[string]any, len(fm))
	for k, v := range fm {
		out[k] = r.resolveValue(v, relativeTo, mode)
	}
	return out
}

// resolveValue resolves scc: links within a single frontmatter value, deep-copying
// containers so the original is left untouched. Non-string scalars pass through.
func (r *Resolver) resolveValue(v any, relativeTo string, mode LinkMode) any {
	switch val := v.(type) {
	case string:
		return r.ResolveLinks(val, relativeTo, mode)
	case map[string]any:
		return r.ResolveFrontmatter(val, relativeTo, mode)
	case []any:
		out := make([]any, len(val))
		for i, elem := range val {
			out[i] = r.resolveValue(elem, relativeTo, mode)
		}
		return out
	case []string:
		out := make([]string, len(val))
		for i, elem := range val {
			out[i] = r.ResolveLinks(elem, relativeTo, mode)
		}
		return out
	default:
		return v
	}
}

// sccToRootPath converts an SCC code to a path relative to the output root.
func sccToRootPath(sccCode string, ext string) string {
	parts := strings.Split(sccCode, "/")
	if len(parts) < 2 {
		return sccCode
	}

	var pathParts []string
	for _, part := range parts[1:] {
		pathParts = append(pathParts, strings.Split(part, ".")...)
	}

	if len(pathParts) == 0 {
		return sccCode
	}

	pathParts[len(pathParts)-1] += ext
	return strings.Join(pathParts, "/")
}

// sccToRelPathFrom computes a relative path from the file at fromSCC to the file at toSCC.
// MkDocs resolves markdown links relative to the file that contains them, so paths must be
// relative to the current file's directory, not the output root.
func sccToRelPathFrom(toSCC string, fromSCC string, ext string) string {
	targetPath := sccToRootPath(toSCC, ext)

	if fromSCC == "" {
		return targetPath
	}

	fromPath := sccToRootPath(fromSCC, ext)
	fromDir := filepath.Dir(fromPath)

	rel, err := filepath.Rel(fromDir, targetPath)
	if err != nil {
		return targetPath
	}

	return filepath.ToSlash(rel)
}
