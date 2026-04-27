package parser

import (
	"bytes"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmparser "github.com/yuin/goldmark/parser"
	"github.com/yuin/goldmark/text"
	meta "github.com/yuin/goldmark-meta"
)

// ParseDocument parses annotated markdown into a structured Document.
// It extracts YAML frontmatter, runs the annotation pre-pass, walks the
// goldmark AST, and builds a nested section tree.
func ParseDocument(source []byte) (*Document, error) {
	// 1. Extract annotations from raw text
	annotations := ExtractAnnotations(string(source))
	annotationByLine := indexAnnotationsByEndLine(annotations)

	// 2. Parse markdown with goldmark (including frontmatter)
	md := goldmark.New(goldmark.WithExtensions(meta.Meta))
	ctx := gmparser.NewContext()
	reader := text.NewReader(source)
	tree := md.Parser().Parse(reader, gmparser.WithContext(ctx))

	// 3. Extract frontmatter
	frontmatter := meta.Get(ctx)

	// 4. Collect heading info by walking AST
	headings := collectHeadings(tree, source)

	// 4b. Collect blockquote headings (> ######) that goldmark doesn't parse as headings
	bqHeadings := collectBlockquoteHeadings(source)
	headings = mergeHeadings(headings, bqHeadings)

	// 5. Associate annotations with headings
	associateAnnotations(headings, annotationByLine, source)

	// 6. Build section tree with body slicing
	sections := buildSectionTree(headings, source)

	return &Document{
		Frontmatter: frontmatter,
		Sections:    sections,
		Source:      source,
	}, nil
}

// headingInfo holds intermediate data about a heading found in the AST.
type headingInfo struct {
	text       string
	level      int
	lineNum    int // 1-based line number of the heading
	byteOffset int // byte offset of the heading line start in source
	annotation map[string]string
}

// indexAnnotationsByEndLine maps the annotation's EndLine to the annotation,
// so we can look up "is there an annotation that ends just before this heading?"
func indexAnnotationsByEndLine(annotations []Annotation) map[int]*Annotation {
	m := make(map[int]*Annotation, len(annotations))
	for i := range annotations {
		m[annotations[i].EndLine] = &annotations[i]
	}
	return m
}

// collectHeadings walks the goldmark AST and extracts all heading nodes.
// Headings inside blockquotes are skipped — they are handled separately by
// collectBlockquoteHeadings() which assigns them the correct tree level.
func collectHeadings(tree ast.Node, source []byte) []*headingInfo {
	var headings []*headingInfo

	ast.Walk(tree, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}

		// Skip entire blockquote subtrees — headings inside blockquotes
		// are handled by collectBlockquoteHeadings() at the correct level.
		if n.Kind() == ast.KindBlockquote {
			return ast.WalkSkipChildren, nil
		}

		heading, ok := n.(*ast.Heading)
		if !ok {
			return ast.WalkContinue, nil
		}

		// Extract heading text
		var buf bytes.Buffer
		for child := heading.FirstChild(); child != nil; child = child.NextSibling() {
			if child.Kind() == ast.KindText {
				segment := child.(*ast.Text).Segment
				buf.Write(segment.Value(source))
			}
		}

		// Compute line number from byte offset
		lineStart := 0
		if heading.Lines().Len() > 0 {
			lineStart = heading.Lines().At(0).Start
		}
		lineNum := bytes.Count(source[:lineStart], []byte("\n")) + 1

		headings = append(headings, &headingInfo{
			text:       buf.String(),
			level:      heading.Level,
			lineNum:    lineNum,
			byteOffset: lineStart,
		})

		return ast.WalkContinue, nil
	})

	return headings
}

// blockquoteH6Re matches "> ###### Heading Text" lines (H6+ inside blockquotes).
// Goldmark doesn't parse headings inside blockquotes regardless of level,
// so these must be detected via regex.
var blockquoteH6Re = regexp.MustCompile(`^>\s*(#{6,})\s+(.+)$`)

// collectBlockquoteHeadings scans source for "> ######" patterns that goldmark
// doesn't parse as heading nodes. Goldmark ignores headings inside blockquotes
// regardless of level, so these are detected via regex and injected as synthetic
// headings. For tree building purposes, we treat them as level 4 (same as regular
// abilities that appear under feature-groups at H3).
func collectBlockquoteHeadings(source []byte) []*headingInfo {
	lines := strings.Split(string(source), "\n")
	var headings []*headingInfo

	for i, line := range lines {
		matches := blockquoteH6Re.FindStringSubmatch(line)
		if matches == nil {
			continue
		}
		headingText := strings.TrimSpace(matches[2])
		lineNum := i + 1

		headings = append(headings, &headingInfo{
			text:    headingText,
			level:   4, // treat as H4 for tree structure (abilities nest under H3 feature-groups)
			lineNum: lineNum,
		})
	}
	return headings
}

// mergeHeadings combines AST headings and synthetic blockquote headings,
// sorted by line number.
func mergeHeadings(a, b []*headingInfo) []*headingInfo {
	merged := make([]*headingInfo, 0, len(a)+len(b))
	merged = append(merged, a...)
	merged = append(merged, b...)

	// Sort by line number (stable to preserve relative order within each source)
	sort.SliceStable(merged, func(i, j int) bool {
		return merged[i].lineNum < merged[j].lineNum
	})
	return merged
}

// associateAnnotations links annotations to their nearest following heading.
// An annotation applies to the heading on the line immediately after the annotation's EndLine,
// allowing for blank lines between them.
func associateAnnotations(headings []*headingInfo, annotationByLine map[int]*Annotation, source []byte) {
	lines := strings.Split(string(source), "\n")

	for _, h := range headings {
		// Look backward from the heading line to find an annotation
		// The annotation's EndLine should be on a line before the heading,
		// with only blank lines between them.
		for checkLine := h.lineNum - 1; checkLine >= 1; checkLine-- {
			// If this line is blank, keep looking
			if checkLine <= len(lines) && strings.TrimSpace(lines[checkLine-1]) == "" {
				continue
			}
			// Check if there's an annotation ending on this line
			if ann, ok := annotationByLine[checkLine]; ok {
				h.annotation = ann.Fields
			}
			break // stop at first non-blank line whether annotation or not
		}
	}
}

// buildSectionTree creates the nested section tree from the flat heading list.
// Body content is sliced from the source between heading boundaries.
func buildSectionTree(headings []*headingInfo, source []byte) []*Section {
	if len(headings) == 0 {
		return nil
	}

	// Create Section objects
	sections := make([]*Section, len(headings))
	for i, h := range headings {
		sections[i] = &Section{
			Heading:      h.text,
			HeadingLevel: h.level,
			Annotation:   h.annotation,
		}
	}

	// Assign body content: the body of section[i] is from end of its heading line
	// to the start of the next heading at the same or higher level (or section[i+1] for body-only)
	lines := strings.Split(string(source), "\n")
	for i, h := range headings {
		bodyStart := h.lineNum // line after the heading
		var bodyEnd int
		if i+1 < len(headings) {
			bodyEnd = headings[i+1].lineNum - 1
		} else {
			bodyEnd = len(lines)
		}

		// Find where the annotation for the next heading starts (if any)
		// so we exclude it from this section's body
		if i+1 < len(headings) && headings[i+1].annotation != nil {
			// Walk backward from the next heading to find annotation start
			for checkLine := headings[i+1].lineNum - 1; checkLine >= bodyStart; checkLine-- {
				trimmed := strings.TrimSpace(safeGetLine(lines, checkLine-1))
				if trimmed == "" {
					continue
				}
				if strings.HasPrefix(trimmed, "<!--") || strings.HasPrefix(trimmed, "@") || trimmed == "-->" {
					bodyEnd = checkLine - 1
					continue
				}
				break
			}
		}

		// Extract body lines (skip the heading line itself)
		if bodyStart < bodyEnd {
			bodyLines := lines[bodyStart:bodyEnd]
			sections[i].BodySource = strings.TrimSpace(strings.Join(bodyLines, "\n"))
		}
	}

	// Build parent-child relationships
	var roots []*Section
	var stack []*Section // stack of current ancestors

	for _, sec := range sections {
		// Pop stack until we find a parent (lower heading level)
		for len(stack) > 0 && stack[len(stack)-1].HeadingLevel >= sec.HeadingLevel {
			stack = stack[:len(stack)-1]
		}

		if len(stack) == 0 {
			roots = append(roots, sec)
		} else {
			parent := stack[len(stack)-1]
			sec.Parent = parent
			parent.Children = append(parent.Children, sec)
		}

		stack = append(stack, sec)
	}

	return roots
}

func safeGetLine(lines []string, idx int) string {
	if idx < 0 || idx >= len(lines) {
		return ""
	}
	return lines[idx]
}
