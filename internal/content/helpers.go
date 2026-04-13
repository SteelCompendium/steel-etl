package content

import (
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
)

var (
	nonAlphaNum    = regexp.MustCompile(`[^a-z0-9]+`)
	leadTrail      = regexp.MustCompile(`^-+|-+$`)
	costSuffixRe   = regexp.MustCompile(`\s*\(\d+\s+\w+\)\s*$`)
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
