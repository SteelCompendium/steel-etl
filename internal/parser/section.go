package parser

// Section represents a heading-delimited section of the document.
type Section struct {
	Heading      string            // heading text (e.g., "Fury")
	HeadingLevel int               // 1-6
	Annotation   map[string]string // from annotation pre-pass (nil if unannotated)
	BodySource   string            // raw markdown body (between this heading and next)
	Children     []*Section
	Parent       *Section
}

// Document represents a fully parsed annotated markdown document.
type Document struct {
	Frontmatter map[string]any // from YAML frontmatter
	Sections    []*Section     // top-level sections (H1s, or whatever the highest level is)
	Source      []byte         // raw markdown source
}

// Type returns the @type annotation value, or empty string if unset.
func (s *Section) Type() string {
	if s.Annotation == nil {
		return ""
	}
	return s.Annotation["type"]
}

// ID returns the @id annotation value, or empty string if unset.
func (s *Section) ID() string {
	if s.Annotation == nil {
		return ""
	}
	return s.Annotation["id"]
}

// AllSections returns a flat list of this section and all descendants, depth-first.
func (s *Section) AllSections() []*Section {
	result := []*Section{s}
	for _, child := range s.Children {
		result = append(result, child.AllSections()...)
	}
	return result
}
