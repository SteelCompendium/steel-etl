package content

import (
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

var (
	nonAlphaNum   = regexp.MustCompile(`[^a-z0-9]+`)
	leadTrail     = regexp.MustCompile(`^-+|-+$`)
	costSuffixRe  = regexp.MustCompile(`\s*\((\d+\s+\w+)\)\s*$`)
	domainsLineRe = regexp.MustCompile(`(?m)^\*\*Domains:\*\*\s*(.+)$`)
)

// findAncestorID walks the context stack upward from the given level, looking
// for an ancestor with the specified @type, and returns its @id.
func findAncestorID(ctx *context.ContextStack, fromLevel int, targetType string) string {
	for level := fromLevel - 1; level >= 1; level-- {
		cur := ctx.Current(level)
		if cur == nil {
			continue
		}
		if cur["type"] == targetType {
			return cur["id"]
		}
	}
	return ""
}

// CleanHeading strips the cost suffix from a heading.
// "Alacrity of the Heart (11 Piety)" → "Alacrity of the Heart"
func CleanHeading(s string) string {
	return strings.TrimSpace(costSuffixRe.ReplaceAllString(s, ""))
}

// extractCostSuffix returns the cost embedded in a heading's trailing
// parenthetical (the part CleanHeading strips), e.g. "Barbed Tail (1 Point)"
// → "1 Point". Returns "" when the heading has no such suffix.
func extractCostSuffix(s string) string {
	if m := costSuffixRe.FindStringSubmatch(s); m != nil {
		return strings.TrimSpace(m[1])
	}
	return ""
}

// extractDomains pulls the comma-separated domain list from a god/saint body's
// "**Domains:**" line into a trimmed slice. Returns nil when the line is absent.
func extractDomains(body string) []string {
	m := domainsLineRe.FindStringSubmatch(body)
	if m == nil {
		return nil
	}
	var out []string
	for _, part := range strings.Split(m[1], ",") {
		if v := strings.TrimSpace(part); v != "" {
			out = append(out, v)
		}
	}
	return out
}

// headingName returns the entity's display name: the @name annotation override
// when present (used where the book heading differs from the entity, e.g. the
// "Devil Gods" section that defines the Lords of Hell), else the cleaned heading.
func headingName(s *parser.Section) string {
	if s.Annotation != nil {
		if n := strings.TrimSpace(s.Annotation["name"]); n != "" {
			return n
		}
	}
	return CleanHeading(s.Heading)
}

// Slugify converts a heading text to a URL-friendly slug.
// "Blood for Blood!" -> "blood-for-blood"
// "Brutal Slam" -> "brutal-slam"
// "Saint's Raiment" -> "saints-raiment"
func Slugify(s string) string {
	s = strings.ToLower(s)
	// Strip apostrophes before replacing non-alphanumeric (so "saint's" → "saints")
	s = strings.ReplaceAll(s, "'", "")
	s = strings.ReplaceAll(s, "\u2019", "") // right single quote
	s = nonAlphaNum.ReplaceAllString(s, "-")
	s = leadTrail.ReplaceAllString(s, "")
	return s
}
