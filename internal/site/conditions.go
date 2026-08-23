package site

// SC-90 — CONDITION FACET EXTRACTION for the feature Search & Filter island.
//
// Draw Steel has nine rules conditions (Heroes book, `@type: condition`); each
// is its own SCC entity `mcdm.heroes.v1/condition/<slug>` and its own Browse
// page. Ability prose names them constantly, but NOTHING in the data contract
// records which ability touches which condition — so the facet is DERIVED here,
// at `steel-etl site` time, from text the pipeline already produced.
//
// SITE-ONLY, like ability_cards.go / bestiary_search.go: the shared data repos
// and the JSON/YAML schemas are untouched. (Promoting `conditions:` to real
// ability frontmatter + both schema copies is a separate, contract-level change
// — see the SC-90 report.)
//
// ── WHAT COUNTS AS A MATCH ───────────────────────────────────────────────────
// v1 matches ANY MENTION in an ability's MECHANICAL text. It deliberately does
// NOT distinguish inflicting a condition ("M<AVERAGE, dazed") from requiring
// one ("this ability has no effect on a prone target") from removing one ("is
// no longer slowed or weakened"). Rationale:
//
//   - The ticket asks for "conditions mentioned".
//   - Removal/requirement phrasings are ~0.5% of abilities (3 of 621 use "no
//     longer <condition>"), so a relation split would add fragile NLP for a
//     handful of rows while risking silent drops. An over-inclusive filter is
//     recoverable by eye; a filter that hides matches is not.
//
// FLAVOR IS EXCLUDED, which is the distinction that actually matters: without
// it, "A practiced attack will instantly kill an already weakened foe"
// (Assassinate) and "the world has slowed down" (Accelerate) would pollute the
// weakened/slowed facets with abilities that have no mechanical relationship to
// them. conditionMechanicalText drops the frontmatter `flavor:`/`name:` blocks
// and every wholly-italic body paragraph — the same paragraphs renderAbilityCard
// itself treats as flavor, so card and facet agree by construction.
//
// ── WHY BOTH A LINK SIGNAL AND A WORD SIGNAL ─────────────────────────────────
// The books are fully link-swept, so nearly every mention is an `scc:` link
// resolved to `condition/<slug>.md` (or `condition/<slug>/` once rendered) —
// that is the precise signal. The word-boundary signal is the safety net for a
// mention the sweep missed; on today's corpus the two agree exactly on
// mechanical text. Both are cheap, so we take the union.
//
// ── WHY THE BODY, NOT JUST FRONTMATTER ───────────────────────────────────────
// Ability frontmatter (`effects`, `tier1..3`, `target`, `trigger`) is LOSSY: it
// captures only labelled "**Effect:**"-style paragraphs. Censor's Judgment
// names `taunted` solely in an unlabelled bullet list, and 11 other abilities
// have the same shape — frontmatter-only extraction misses all 12. So the scan
// runs over the raw md-linked BODY (plus frontmatter, as a union).

import (
	"regexp"
	"sort"
	"strings"
)

// conditionSlugs is the canonical Draw Steel condition vocabulary — the nine
// `@type: condition | @id: <slug>` entities in the Heroes book (no other book
// defines any). Kept in rules order, which is alphabetical.
//
// ⚠️ This list must equal the set of condition ids in the book sources;
// TestConditionVocabularyMatchesBookSources enforces that, so a tenth condition
// added by an errata printing fails the build instead of silently vanishing
// from the facet.
//
// (The DSE plugin's ConditionManager is NOT authoritative here: it files
// `taunted` under pseudoConditions, so iterating it would drop a real
// condition. See draw-steel-elements/src/utils/Conditions.ts.)
var conditionSlugs = []string{
	"bleeding",
	"dazed",
	"frightened",
	"grabbed",
	"prone",
	"restrained",
	"slowed",
	"taunted",
	"weakened",
}

var conditionSet = func() map[string]bool {
	m := make(map[string]bool, len(conditionSlugs))
	for _, s := range conditionSlugs {
		m[s] = true
	}
	return m
}()

var (
	// A resolved condition link, in either shape the pipeline produces:
	//   md-linked body   → [taunted](../../../../condition/taunted.md)
	//   rendered card    → href="../../../../condition/taunted/"
	//   raw scc: link    → scc.v1:mcdm.heroes.v1/condition/taunted
	// The trailing "/" in the pattern keeps `rule/combat/condition.md` (the
	// glossary entry ABOUT conditions) from matching.
	conditionLinkRe = regexp.MustCompile(`condition/([a-z][a-z-]*)`)

	// Word-boundary net for an un-swept mention. Built from conditionSlugs so a
	// new condition needs no second edit.
	conditionWordRe = func() map[string]*regexp.Regexp {
		m := make(map[string]*regexp.Regexp, len(conditionSlugs))
		for _, s := range conditionSlugs {
			m[s] = regexp.MustCompile(`(?i)\b` + s + `\b`)
		}
		return m
	}()

	// data-conditions="dazed prone" — the attribute renderAbilityCard stamps on
	// the .sc-ability <article>, read back by extractPreviewItem when the leaf
	// page has already been card-rendered. Same read-back trick as data-grant /
	// data-sub on trait cards.
	reDataConditions = regexp.MustCompile(`data-conditions="([^"]*)"`)

	// Top-level YAML key at column 0 ("flavor:", "effects:"); an indented
	// "  name: Effect" inside the effects list is NOT one.
	fmTopKeyRe = regexp.MustCompile(`^([A-Za-z_][A-Za-z0-9_-]*):`)
)

// conditionsDropFields are the frontmatter keys whose values are prose ABOUT the
// ability rather than its mechanics. `flavor` is the one that actually produces
// false positives today; `name` is dropped defensively so a future ability
// called e.g. "Prone to Violence" cannot join the prone facet on its title.
var conditionsDropFields = map[string]bool{"flavor": true, "name": true}

// extractConditions returns the conditions mentioned in an ability's mechanical
// text, in canonical order. fm is the page's YAML frontmatter (without the ---
// fences); body is the RAW md-linked markdown body, before card rendering.
// Returns nil when nothing matches, so the JSON key is omitted.
func extractConditions(fm, body string) []string {
	text := conditionMechanicalText(fm, body)
	if text == "" {
		return nil
	}
	seen := make(map[string]bool, 4)
	for _, m := range conditionLinkRe.FindAllStringSubmatch(text, -1) {
		if conditionSet[m[1]] {
			seen[m[1]] = true
		}
	}
	for slug, re := range conditionWordRe {
		if !seen[slug] && re.MatchString(text) {
			seen[slug] = true
		}
	}
	if len(seen) == 0 {
		return nil
	}
	out := make([]string, 0, len(seen))
	for _, slug := range conditionSlugs { // canonical order, not map order
		if seen[slug] {
			out = append(out, slug)
		}
	}
	return out
}

// conditionMechanicalText joins the parts of an ability page that carry
// mechanics: its frontmatter minus the flavor/name blocks, and its body minus
// every wholly-italic (flavor) paragraph.
func conditionMechanicalText(fm, body string) string {
	var b strings.Builder
	b.WriteString(frontmatterWithoutFields(fm, conditionsDropFields))
	for _, p := range paraSplitRe.Split(body, -1) {
		tp := strings.TrimSpace(p)
		if tp == "" || isItalicPara(tp) {
			continue
		}
		b.WriteByte('\n')
		b.WriteString(tp)
	}
	return b.String()
}

// frontmatterWithoutFields drops the named TOP-LEVEL keys and their continuation
// lines (folded scalars, nested blocks) from a frontmatter string. Indented keys
// of the same name — "  name: Effect" inside the effects list — are kept.
func frontmatterWithoutFields(fm string, drop map[string]bool) string {
	var keep []string
	dropping := false
	for _, line := range strings.Split(fm, "\n") {
		if m := fmTopKeyRe.FindStringSubmatch(line); m != nil {
			dropping = drop[m[1]]
		}
		if !dropping {
			keep = append(keep, line)
		}
	}
	return strings.Join(keep, "\n")
}

// conditionsAttr renders the data-conditions attribute for an ability card, or
// "" when the ability mentions no condition. Slugs are [a-z-]+ by construction,
// so no escaping is required.
func conditionsAttr(conds []string) string {
	if len(conds) == 0 {
		return ""
	}
	return ` data-conditions="` + strings.Join(conds, " ") + `"`
}

// conditionsFromCardHTML reads the conditions back off an already-rendered
// .sc-ability card. Only the FIRST data-conditions on the page is the page's own
// card (container pages may transclude other leaves' cards below it).
func conditionsFromCardHTML(body string) []string {
	m := reDataConditions.FindStringSubmatch(body)
	if m == nil {
		return nil
	}
	var out []string
	for _, f := range strings.Fields(m[1]) {
		if conditionSet[f] {
			out = append(out, f)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return conditionIndex(out[i]) < conditionIndex(out[j])
	})
	return out
}

// conditionsForPreview resolves one leaf ability page's conditions for the
// Search & Filter island.
//
// By the time index pages are built the leaf body is ALREADY the rendered
// .sc-ability card, so the authoritative answer is the data-conditions attribute
// renderAbilityCard stamped from the raw markdown. An empty attribute means
// "derived, none found" and must be honoured — re-deriving from rendered HTML
// would re-admit the flavor line and the sc-src markdown template, which is
// exactly where the false positives live (Assassinate's "already weakened foe").
// Only a page that never went through the card renderer falls through to a live
// derivation.
func conditionsForPreview(fm, body string) []string {
	if strings.Contains(body, `class="sc-ability`) {
		return conditionsFromCardHTML(body)
	}
	return extractConditions(fm, body)
}

func conditionIndex(slug string) int {
	for i, s := range conditionSlugs {
		if s == slug {
			return i
		}
	}
	return len(conditionSlugs)
}
