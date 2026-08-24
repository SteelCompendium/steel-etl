package content

import (
	"regexp"
	"strings"
)

// Over-extended ability-statblock blockquotes (SC-199).
//
// A named ability card is transcribed as a blockquote that leads with its own
// heading:
//
//	**Chargebreaker:** While you wield this weapon, you have the following ability.
//
//	> ###### Stop Right There
//	>
//	> *Their momentum, your impact.*
//	>
//	> | **Melee, Strike, Weapon** | **Free triggered** |
//	> |---------------------------|-------------------:|
//	> | **📏 Melee 1**            |   **🎯 One enemy** |
//	>
//	> **Trigger:** The target willingly moves adjacent to you.
//	>
//	> **Effect:** The target takes 5 damage.
//
//	**Chilling II:** …
//
// The blockquote *is* the ability's boundary: nothing downstream re-derives it, so
// if the `> ` prefix runs one paragraph too far, every following sibling is silently
// swallowed into the ability card in all six output formats and on the site. That is
// exactly what happened to the 5th- and 9th-Level Weapon Enhancement sections of the
// Heroes book (SC-199): eight and four sibling enhancements respectively rendered
// inside the "Stop Right There" / "Nova" cards.
//
// The shape is invisible to a proofreader (the quoted text reads correctly) and
// invisible to the parsers (a blockquote is a blockquote), so it is caught here
// instead: inside a *headed* ability blockquote, every bold-lead paragraph must be
// a known ability field. A label that is not — appearing after the ability body has
// already started — is content that belongs outside the quote.
//
// Deliberately conservative, because truncating a real ability is worse than the
// bug it guards against:
//
//   - Only *headed* blockquotes are considered. Monster statblock abilities are
//     headless blockquotes nested inside a statblock section (~1,370 in the corpus);
//     there is no "following sibling" for them to swallow, and their bodies carry
//     free-form sub-option labels (**Head:** / **Legs:** / **Torso:** under an
//     **Effect:**) that no allowlist can enumerate.
//   - The block must already look like an ability (at least one known field), so
//     headed non-ability blockquotes — e.g. the Heroes book's "Zola Honeycut
//     Negotiation Stats" sample, whose **Benevolence:** / **Protection:** motivations
//     genuinely belong inside — are never flagged.
//   - Only labels *after* the last known field are reported, so an ability that is
//     the last thing in its section, and ability bodies made entirely of known
//     fields and power-roll tiers, stay clean.
//
// Over the four annotated book sources this rule reports 12 findings before the
// SC-199 fix and 0 after — see ability_quote_lint_test.go.

// abilityQuoteHeadingRe matches a heading line inside a blockquote. Levels run past
// CommonMark's H6 cap because the Monsters book uses H7/H9 (see docs/statblocks.md).
var abilityQuoteHeadingRe = regexp.MustCompile(`^#{1,9}\s`)

// abilityQuoteLabelRe extracts the label from a bold-lead paragraph
// (`**Effect:** …`, `**Chilling II:** …`). The colon may sit inside or outside the
// bold run; both forms occur in the sources.
var abilityQuoteLabelRe = regexp.MustCompile(`^\*\*(.{1,70}?):?\*\*`)

// abilityQuoteLinkRe strips `[text](target)` down to `text` so a linked label
// (`**[Power Roll](scc.v1:…) + Might:**`) is matched on its prose.
var abilityQuoteLinkRe = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)

// abilityQuoteListItemRe matches a list-item line. Power-roll tiers (`- **≤11:** …`)
// and option lists (`- **Concealed:** …`) are body content, not paragraph labels.
var abilityQuoteListItemRe = regexp.MustCompile(`^\s*[-*+]\s`)

// abilityQuoteFieldRe is the closed vocabulary of ability-body paragraph labels,
// derived by enumerating every bold-lead label in every ability blockquote across
// the four book sources (heroes, monsters, beastheart, summoner).
var abilityQuoteFieldRe = regexp.MustCompile(`(?i)^(` +
	`Effect|Trigger|Special|Requirement|Prerequisite|Alternate Effect|Notes?` +
	`|Persistent(?: \d+)?` +
	`|Strained` +
	`|Mark Benefit` +
	`|Spend \d+\+? .+` + // Spend 1 Ferocity, Spend 2+ Drama, …
	`|Power Roll(?: .+)?` + // Power Roll + Might, Power Roll + Your Highest …
	`|(?:\d+\+? )?Malice` + // Malice, 1+ Malice, 3 Malice
	`)$`)

// AbilityQuoteIssue is one bold-lead paragraph that sits inside a named ability
// blockquote but is not part of the ability.
type AbilityQuoteIssue struct {
	Ability string // heading text of the ability the quote opens with
	Label   string // the bold-lead label that should not be inside the quote
	Line    int    // 1-based line number within the scanned body
}

// ScanOverExtendedAbilityQuotes reports bold-lead paragraphs that a named ability
// blockquote has swallowed from the content that should follow it.
func ScanOverExtendedAbilityQuotes(body string) []AbilityQuoteIssue {
	lines := strings.Split(body, "\n")
	var out []AbilityQuoteIssue

	for i := 0; i < len(lines); {
		if !strings.HasPrefix(lines[i], ">") {
			i++
			continue
		}
		// Maximal run of blockquote lines. A blank (unprefixed) line ends the quote
		// in the sources: every ability blockquote keeps a bare ">" between its
		// paragraphs.
		j := i
		for j < len(lines) && strings.HasPrefix(lines[j], ">") {
			j++
		}
		out = append(out, scanAbilityQuoteBlock(lines[i:j], i+1)...)
		i = j
	}
	return out
}

// scanAbilityQuoteBlock examines one blockquote run. startLine is the 1-based line
// number of block[0] in the enclosing body.
func scanAbilityQuoteBlock(block []string, startLine int) []AbilityQuoteIssue {
	heading := ""
	type label struct {
		idx   int
		text  string
		known bool
	}
	var labels []label
	lastKnown := -1

	for k, raw := range block {
		line := strings.TrimPrefix(strings.TrimPrefix(raw, ">"), " ")
		if heading == "" && abilityQuoteHeadingRe.MatchString(line) {
			heading = strings.TrimSpace(strings.TrimLeft(line, "#"))
			continue
		}
		// Table rows carry the keyword/distance/target cells; list items carry
		// power-roll tiers and option lists. Neither is a paragraph label.
		if strings.HasPrefix(line, "|") || abilityQuoteListItemRe.MatchString(line) {
			continue
		}
		m := abilityQuoteLabelRe.FindStringSubmatch(abilityQuoteLinkRe.ReplaceAllString(line, "$1"))
		if m == nil {
			continue
		}
		text := strings.TrimSpace(m[1])
		known := abilityQuoteFieldRe.MatchString(text)
		if known {
			lastKnown = k
		}
		labels = append(labels, label{idx: k, text: text, known: known})
	}

	// Not a named ability card, or the quote never starts an ability body.
	if heading == "" || lastKnown < 0 {
		return nil
	}

	var out []AbilityQuoteIssue
	for _, l := range labels {
		if l.idx > lastKnown && !l.known {
			out = append(out, AbilityQuoteIssue{
				Ability: heading,
				Label:   l.text,
				Line:    startLine + l.idx,
			})
		}
	}
	return out
}
