package content

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// ClassParser handles @type: class sections.
type ClassParser struct{}

func (p *ClassParser) Type() string { return "class" }

func (p *ClassParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	fm := map[string]any{
		"name": section.Heading,
		"type": "class",
	}

	body := section.FullBodySource()

	if f := firstFlavorParagraph(body); f != "" {
		fm["flavor"] = f
	}

	// Extract heroic resource from body
	if hr := extractHeroicResource(body); hr != "" {
		fm["heroic_resource"] = hr
	}

	// Extract primary characteristics
	if v := extractField(body, "Primary Characteristics"); v != "" {
		fm["primary_characteristics"] = splitCommaList(v)
	} else if v := extractField(body, "Starting Characteristics"); v != "" {
		// Heroes 1.x phrasing: "You start with a Might of 2 and an Agility of
		// 2, and you can choose one of the following arrays…" — the named
		// begin-at-2 characteristics are the class's primaries.
		if names := startingCharRe.FindAllStringSubmatch(stripInlineMarkdown(v), -1); len(names) > 0 {
			var list []string
			for _, m := range names {
				list = append(list, m[1])
			}
			fm["primary_characteristics"] = list
		}
	}

	// Base stats for the class landing card (schema fields exist since 2.0.0)
	if n, ok := extractIntField(body, "Starting Stamina at 1st Level"); ok {
		fm["starting_stamina"] = n
	}
	if n, ok := extractIntField(body, "Stamina Gained at 2nd and Higher Levels"); ok {
		fm["stamina_per_level"] = n
	}
	if n, ok := extractIntField(body, "Recoveries"); ok {
		fm["recoveries"] = n
	}

	// Extract potency fields
	if v := extractField(body, "Weak Potency"); v != "" {
		fm["weak_potency"] = v
	}
	if v := extractField(body, "Average Potency"); v != "" {
		fm["average_potency"] = v
	}
	if v := extractField(body, "Strong Potency"); v != "" {
		fm["strong_potency"] = v
	}

	// Extract skills (natural language text, not a clean comma-separated list)
	if v := extractField(body, "Skill"); v != "" {
		fm["skills"] = []string{v}
	} else if v := extractField(body, "Skills"); v != "" {
		fm["skills"] = []string{v}
	}

	return &ParsedContent{
		Frontmatter: fm,
		Body:        body,
		TypePath:    []string{"class"},
		ItemID:      id,
	}, nil
}

// startingCharRe pulls the begin-at-2 characteristic names out of the
// "Starting Characteristics" prose (after link-stripping).
var startingCharRe = regexp.MustCompile(`(Might|Agility|Reason|Intuition|Presence) of 2`)

// extractIntField reads a bold-labeled field expected to hold a bare integer.
func extractIntField(body, fieldName string) (int, bool) {
	v := extractField(body, fieldName)
	if v == "" {
		return 0, false
	}
	n, err := strconv.Atoi(strings.TrimSpace(v))
	if err != nil {
		return 0, false
	}
	return n, true
}

// extractHeroicResource looks for "Heroic Resource: X" in the body.
func extractHeroicResource(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		// Match **Heroic Resource: Ferocity** or similar
		line = strings.ReplaceAll(line, "**", "")
		if strings.HasPrefix(line, "Heroic Resource:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Heroic Resource:"))
		}
	}
	return ""
}
