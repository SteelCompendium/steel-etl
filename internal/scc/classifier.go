package scc

import "strings"

// Classify builds an SCC string from source, type path, and item ID.
//
// Examples:
//
//	Classify("mcdm.heroes.v1", ["abilities", "fury"], "gouge")
//	  → "mcdm.heroes.v1/abilities.fury/gouge"
//
//	Classify("mcdm.heroes.v1", ["classes"], "fury")
//	  → "mcdm.heroes.v1/classes/fury"
func Classify(source string, typePath []string, itemID string) string {
	var parts []string

	if source != "" {
		parts = append(parts, source)
	}

	if len(typePath) > 0 {
		typeComponent := strings.Join(typePath, ".")
		parts = append(parts, typeComponent)
	}

	parts = append(parts, itemID)
	return strings.Join(parts, "/")
}
