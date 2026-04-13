package context

// Metadata holds key-value pairs from an annotation or document frontmatter.
type Metadata map[string]string

// ContextStack tracks annotation metadata by heading depth.
// Index 0 is unused; indices 1-6 correspond to H1-H6.
type ContextStack struct {
	document Metadata
	levels   [7]Metadata
}

// NewContextStack creates a stack with document-level metadata.
func NewContextStack(docMeta Metadata) *ContextStack {
	return &ContextStack{document: docMeta}
}

// Push sets metadata at a heading level, clearing all deeper levels.
func (s *ContextStack) Push(headingLevel int, meta Metadata) {
	if headingLevel < 1 || headingLevel > 6 {
		return
	}
	for i := headingLevel; i <= 6; i++ {
		s.levels[i] = nil
	}
	s.levels[headingLevel] = meta
}

// Lookup searches for a key from the given level upward through
// all ancestor levels and finally the document metadata.
func (s *ContextStack) Lookup(fromLevel int, key string) (string, bool) {
	for i := fromLevel; i >= 1; i-- {
		if s.levels[i] != nil {
			if val, ok := s.levels[i][key]; ok {
				return val, true
			}
		}
	}
	if val, ok := s.document[key]; ok {
		return val, true
	}
	return "", false
}

// AncestorsOfType returns all non-nil ancestor metadata entries
// from shallowest to deepest, excluding the given level itself.
func (s *ContextStack) AncestorsOfType(fromLevel int) []Metadata {
	var ancestors []Metadata
	for i := 1; i < fromLevel; i++ {
		if s.levels[i] != nil {
			ancestors = append(ancestors, s.levels[i])
		}
	}
	return ancestors
}

// Current returns the metadata at the given heading level.
func (s *ContextStack) Current(headingLevel int) Metadata {
	if headingLevel < 1 || headingLevel > 6 {
		return nil
	}
	return s.levels[headingLevel]
}

// Document returns the document-level metadata.
func (s *ContextStack) Document() Metadata {
	return s.document
}
