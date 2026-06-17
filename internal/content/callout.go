package content

import (
	"regexp"
	"strings"
)

// Callout annotations are body-level directives (not section types):
//
//	<!-- @type: callout | @owner: self|loose -->
//	> blockquote text...
//
// @owner records what the callout semantically belongs to. `self` belongs to the
// immediate enclosing header and always renders; `loose` is incidental (the publisher
// just had whitespace there) and is stripped from the body of the page that is rooted
// at the section the callout sits in. Key order and trailing whitespace are tolerated,
// matching the parser's single-line annotation form.

// calloutKnownOwners is the Phase 1 value set. The space is intentionally open so a
// later coarser scope (e.g. "chapter") or an SCC reference can be added without a
// grammar change.
var calloutKnownOwners = map[string]bool{"self": true, "loose": true}

// calloutCommentLineRe matches a one-line HTML comment carrying @type: callout.
var calloutCommentLineRe = regexp.MustCompile(`^<!--.*@type:\s*callout\b.*-->\s*$`)

// ownerValueRe extracts the @owner value from such a comment.
var ownerValueRe = regexp.MustCompile(`@owner:\s*([\w-]+)`)

// blankLineRunRe collapses 3+ newlines (left after removing a callout from the middle
// of a body) back to a single blank line.
var blankLineRunRe = regexp.MustCompile(`\n{3,}`)

// CalloutAnnotation is a parsed callout comment, used by validation.
type CalloutAnnotation struct {
	Owner      string // @owner value, "" if absent
	HasOwner   bool
	OwnerKnown bool // Owner is in calloutKnownOwners
}

// IsCalloutComment reports whether a line is a `@type: callout` annotation comment
// (any @owner). The site card builder uses this to render a surviving callout block
// as an aside instead of leaking its comment/blockquote markers as text.
func IsCalloutComment(line string) bool {
	return calloutCommentLineRe.MatchString(strings.TrimSpace(line))
}

// isLooseCalloutComment reports whether a single line is a callout comment with
// @owner: loose.
func isLooseCalloutComment(line string) bool {
	t := strings.TrimSpace(line)
	if !calloutCommentLineRe.MatchString(t) {
		return false
	}
	m := ownerValueRe.FindStringSubmatch(t)
	return m != nil && m[1] == "loose"
}

// ScanCallouts returns one CalloutAnnotation per callout comment line in body.
func ScanCallouts(body string) []CalloutAnnotation {
	var out []CalloutAnnotation
	for _, line := range strings.Split(body, "\n") {
		t := strings.TrimSpace(line)
		if !calloutCommentLineRe.MatchString(t) {
			continue
		}
		ca := CalloutAnnotation{}
		if m := ownerValueRe.FindStringSubmatch(t); m != nil {
			ca.Owner = m[1]
			ca.HasOwner = true
			ca.OwnerKnown = calloutKnownOwners[m[1]]
		}
		out = append(out, ca)
	}
	return out
}

// stripLooseCallouts removes every `@owner: loose` callout comment and the contiguous
// blockquote run that immediately follows it. Callouts with any other owner and untagged
// blockquotes are left untouched. Operates line-wise because a blockquote spans lines.
func stripLooseCallouts(body string) string {
	if !strings.Contains(body, "callout") { // cheap guard: nothing to do
		return body
	}
	lines := strings.Split(body, "\n")
	out := make([]string, 0, len(lines))
	for i := 0; i < len(lines); i++ {
		if !isLooseCalloutComment(lines[i]) {
			out = append(out, lines[i])
			continue
		}
		// Skip the comment line, any blank lines, then the blockquote run.
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		for j < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[j]), ">") {
			j++
		}
		i = j - 1 // for-loop ++ lands on the first line after the run
	}
	result := strings.Join(out, "\n")
	result = blankLineRunRe.ReplaceAllString(result, "\n\n")
	return strings.TrimSpace(result)
}
