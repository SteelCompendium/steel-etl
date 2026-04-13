package scc

import (
	"regexp"
	"strings"
)

// sccLinkRe matches scc: protocol links in markdown content.
// Examples:
//
//	scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge
//	[Gouge](scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge)
var sccLinkRe = regexp.MustCompile(`scc:([a-zA-Z0-9._\-]+/[a-zA-Z0-9._\-]+/[a-zA-Z0-9._\-]+)`)

// Resolver resolves SCC codes to relative file paths.
type Resolver struct {
	registry *Registry
	ext      string // file extension for resolved paths (e.g., ".md")
}

// NewResolver creates a resolver backed by the given registry.
func NewResolver(registry *Registry, ext string) *Resolver {
	return &Resolver{registry: registry, ext: ext}
}

// ResolveLinks replaces all scc: protocol links in the content with relative file paths.
// The relativeTo parameter is the SCC code of the current file (used to compute relative paths).
func (r *Resolver) ResolveLinks(content string, relativeTo string) string {
	return sccLinkRe.ReplaceAllStringFunc(content, func(match string) string {
		sccCode := strings.TrimPrefix(match, "scc:")

		// Check if the code exists in the registry (or resolve aliases)
		if !r.registry.Contains(sccCode) {
			if canonical, ok := r.registry.ResolveAlias(sccCode); ok {
				sccCode = canonical
			} else {
				// Unknown SCC code -- leave the link as-is
				return match
			}
		}

		return sccToRelPath(sccCode, r.ext)
	})
}

// sccToRelPath converts an SCC code to a relative path from the output root.
// This matches output.SCCToFilePath but avoids a circular import.
func sccToRelPath(sccCode string, ext string) string {
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
