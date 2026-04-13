package parser

import (
	"regexp"
	"strings"
)

// Annotation holds metadata extracted from an HTML comment annotation.
type Annotation struct {
	Fields  map[string]string // @key -> value (without @ prefix)
	Line    int               // 1-based line number where the annotation starts
	EndLine int               // 1-based line number where the annotation ends
}

// EndMarker represents an <!-- @end: id --> marker.
type EndMarker struct {
	ID   string
	Line int // 1-based line number
}

// singleLineRe matches single-line annotations: <!-- @key: value | @key: value -->
var singleLineRe = regexp.MustCompile(`^<!--\s+((?:@[\w-]+:\s*[^|>]+(?:\|\s*)?)+)\s*-->$`)

// multiLineOpenRe matches the opening of a multi-line annotation.
var multiLineOpenRe = regexp.MustCompile(`^<!--\s*$`)

// multiLineCloseRe matches the closing of a multi-line annotation.
var multiLineCloseRe = regexp.MustCompile(`^\s*-->\s*$`)

// fieldRe matches a single @key: value pair.
var fieldRe = regexp.MustCompile(`@([\w-]+):\s*(.+?)(?:\s*\||\s*$)`)

// endMarkerRe matches <!-- @end: id --> markers.
var endMarkerRe = regexp.MustCompile(`^<!--\s*@end:\s*([\w-]+)\s*-->$`)

// ExtractAnnotations scans raw markdown text and returns all annotations
// with their fields and line numbers. End markers are excluded.
func ExtractAnnotations(input string) []Annotation {
	lines := strings.Split(input, "\n")
	var annotations []Annotation
	var inMultiLine bool
	var multiLineStart int
	var multiLineFields map[string]string

	for i, line := range lines {
		lineNum := i + 1 // 1-based
		trimmed := strings.TrimSpace(line)

		if inMultiLine {
			if multiLineCloseRe.MatchString(trimmed) {
				// End of multi-line block
				if len(multiLineFields) > 0 {
					annotations = append(annotations, Annotation{
						Fields:  multiLineFields,
						Line:    multiLineStart,
						EndLine: lineNum,
					})
				}
				inMultiLine = false
				multiLineFields = nil
				continue
			}
			// Parse field within multi-line block
			matches := fieldRe.FindStringSubmatch(trimmed)
			if matches != nil {
				multiLineFields[matches[1]] = strings.TrimSpace(matches[2])
			}
			continue
		}

		// Check for end marker (skip)
		if endMarkerRe.MatchString(trimmed) {
			continue
		}

		// Check for single-line annotation
		if singleLineRe.MatchString(trimmed) {
			fields := parseSingleLine(trimmed)
			if len(fields) > 0 {
				annotations = append(annotations, Annotation{
					Fields:  fields,
					Line:    lineNum,
					EndLine: lineNum,
				})
			}
			continue
		}

		// Check for multi-line annotation opening
		if multiLineOpenRe.MatchString(trimmed) {
			inMultiLine = true
			multiLineStart = lineNum
			multiLineFields = make(map[string]string)
			continue
		}
	}

	return annotations
}

// parseSingleLine extracts @key: value pairs from a single-line annotation.
func parseSingleLine(line string) map[string]string {
	fields := make(map[string]string)
	// Strip comment delimiters
	inner := line
	inner = strings.TrimPrefix(inner, "<!--")
	inner = strings.TrimSuffix(inner, "-->")
	inner = strings.TrimSpace(inner)

	// Split by pipe and parse each segment
	segments := strings.Split(inner, "|")
	for _, seg := range segments {
		seg = strings.TrimSpace(seg)
		matches := fieldRe.FindStringSubmatch(seg)
		if matches != nil {
			fields[matches[1]] = strings.TrimSpace(matches[2])
		}
	}
	return fields
}

// ExtractEndMarkers scans raw markdown text and returns all end markers.
func ExtractEndMarkers(input string) []EndMarker {
	lines := strings.Split(input, "\n")
	var markers []EndMarker

	for i, line := range lines {
		lineNum := i + 1
		trimmed := strings.TrimSpace(line)
		matches := endMarkerRe.FindStringSubmatch(trimmed)
		if matches != nil {
			markers = append(markers, EndMarker{
				ID:   matches[1],
				Line: lineNum,
			})
		}
	}
	return markers
}
