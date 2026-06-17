package site

// High-fantasy steel TRAIT / FEATURE cards — the illuminated "codex niche".
//
// Where ability_cards.go renders an ability/trait page as the raised `.sc-ability`
// plate, this renders a `type: trait` page as the flat, recessed `.sc-trait`
// niche: a colored left spine + embossed feature heading wrapping prose, lists,
// lead-ins, and — recursively — NESTED abilities and NESTED sub-traits.
//
// The page body is a book-faithful subtree render (RenderSubtree), so the trait's
// children arrive as a flat run of markdown headings (H2..H6) carrying
// `{data-scc="…"}` attrs: a `feature.ability.*` scc marks a nested ability, any
// other (or none) marks a nested sub-trait. We rebuild that heading tree by level
// and render each node — abilities via renderAbilityCard (reused, `ops`-free),
// sub-traits via renderTraitNode (recursive). Styled by
// docs/stylesheets/steel-traits.css. SITE-ONLY, like ability_cards.go.

import (
	"fmt"
	"html"
	"regexp"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

var (
	// a markdown heading line, with optional trailing {…attr_list…}
	traitHeadRe = regexp.MustCompile(`(?m)^(#{1,6})[ \t]+(.+?)[ \t]*(?:\{([^}]*)\})?[ \t]*$`)
	traitSCCRe  = regexp.MustCompile(`data-scc="([^"]+)"`)
	// a **Benefit:** / **Drawback:** lead-labeled paragraph → titled segment
	segLabelRe = regexp.MustCompile(`(?is)^\*\*\s*(benefit|drawback)s?\s*:?\s*\*\*\s*(.+)$`)
	// single-* italic, run AFTER bold has consumed every **…** pair
	traitItalicRe = regexp.MustCompile(`\*([^*\n]+)\*`)
	// a markdown table's header-separator row: only | : - and spaces
	tableSepRe = regexp.MustCompile(`^[|:\- ]+$`)
)

// traitInline renders the inline markdown trait prose carries — **bold**,
// *italic*, and [text](url) links — mirroring the JS rich() contract (the
// ability card's richInline deliberately omits italics, so traits get their own).
func traitInline(s string) string {
	s = html.EscapeString(s)
	s = mdBoldRe.ReplaceAllString(s, "<b>$1</b>")
	s = traitItalicRe.ReplaceAllString(s, "<em>$1</em>")
	s = mdLinkRe.ReplaceAllStringFunc(s, func(m string) string {
		sub := mdLinkRe.FindStringSubmatch(m)
		return fmt.Sprintf(`<a href="%s">%s</a>`, cardHref(sub[2]), sub[1])
	})
	return s
}

// traitNode is one heading in the rebuilt subtree: an ability leaf or a
// (recursive) sub-trait. content is the raw markdown between this heading and its
// first child (the node's own prose/lists); children are deeper headings.
type traitNode struct {
	level     int
	name      string
	scc       string
	isAbility bool
	content   string
	children  []*traitNode
}

// traitGlyph is the DrawSteelGlyphs codepoint for the trait crest — a
// PLACEHOLDER (mirrors ACTIONS.trait.glyph in steel-feature-browser.js), swapped
// here in one place when the official trait glyph lands.
const traitGlyph = "*"

// traitCrest renders the steel-shield crest shared by full + preview trait cards
// (the glyph is tinted with the trait accent via CSS).
func traitCrest() string {
	return `<span class="sc-crest sc-trait__crest"><span class="sc-trait__glyph">` + traitGlyph + "</span></span>\n"
}

// renderTraitCard builds the contiguous (no blank-line) `.sc-trait` HTML for a
// `type: trait` page so md_in_html passes it through verbatim.
func renderTraitCard(fm, body string) string {
	name := strings.TrimSpace(parseFrontmatterField(fm, "name"))
	if name == "" {
		name = "Trait"
	}
	eyebrow := traitEyebrow(fm)
	tag := traitTag(parseFrontmatterField(fm, "level"), parseFrontmatterField(fm, "scc"))

	intro, children := parseTraitTree(body)

	bodyHTML, leadProse := renderTraitBody(intro, children)

	cls := "sc-trait sc-trait--crest"
	if leadProse {
		cls += " sc-trait--lead" // engraved drop-cap on the opening paragraph
	}

	// Stash the sub-feature count (and single-grant phrase) on the root so the
	// index preview can show "N options" / "Grants the X maneuver" without
	// re-parsing the rendered tree.
	return wrapTraitSection(cls, traitFeatureAttrs(children), traitCrest(), eyebrow, name, tag, bodyHTML)
}

// renderTraitNode renders a nested sub-trait (no eyebrow, no crest, no drop cap;
// its level pill is derived from the scc). Recurses through its own children.
func renderTraitNode(n *traitNode) string {
	intro, _ := parseTraitTree(n.content) // sub-headings already split into n.children
	bodyHTML, _ := renderTraitBody(intro, n.children)
	tag := traitTag("", n.scc)
	return wrapTraitSection("sc-trait", "", "", "", strings.TrimSpace(n.name), tag, bodyHTML)
}

// traitFeatureAttrs returns the data-* attributes describing a trait's direct
// sub-features (its choice/option headings): data-sub="N" always, plus
// data-grant="the <Name> <action>" when it grants exactly one ability.
func traitFeatureAttrs(children []*traitNode) string {
	if len(children) == 0 {
		return ""
	}
	attrs := fmt.Sprintf(" data-sub=\"%d\"", len(children))
	if len(children) == 1 && children[0].isAbility {
		attrs += fmt.Sprintf(" data-grant=\"%s\"", html.EscapeString(singleGrantPhrase(children[0])))
	}
	return attrs
}

// singleGrantPhrase builds "the <Name> <action-type>" for a trait that grants a
// single ability, reading the action from the ability's 2×2 spec table.
func singleGrantPhrase(child *traitNode) string {
	label := "ability"
	for _, p := range paraSplitRe.Split(child.content, -1) {
		if strings.HasPrefix(strings.TrimSpace(p), "|") {
			if _, act, _, _ := parseAbilityTable(p); act != "" {
				label = strings.ToLower(actionInfo(act, "ability").label)
			}
			break
		}
	}
	return "the " + strings.TrimSpace(child.name) + " " + label
}

// wrapTraitSection assembles one <section class="sc-trait …"> with header +
// body, as a single contiguous block (no blank lines). attrs are extra section
// attributes (e.g. data-sub); crest is the optional crest HTML.
func wrapTraitSection(cls, attrs, crest, eyebrow, name, tag, bodyHTML string) string {
	dia := `<span class="sc-trait__dia"></span>`
	var b strings.Builder
	fmt.Fprintf(&b, "<section class=\"%s\" data-action=\"trait\"%s>\n", cls, attrs)
	b.WriteString("<header class=\"sc-trait__head\">\n")
	if crest != "" {
		b.WriteString(crest)
	}
	b.WriteString("<div class=\"sc-trait__titles\">\n")
	if eyebrow != "" {
		fmt.Fprintf(&b, "<div class=\"sc-trait__eyebrow\">%s%s</div>\n", dia, html.EscapeString(eyebrow))
	}
	fmt.Fprintf(&b, "<h3 class=\"sc-trait__name\">%s</h3>\n", html.EscapeString(name))
	b.WriteString("</div>\n")
	b.WriteString(tag)
	b.WriteString("</header>\n")
	b.WriteString("<div class=\"sc-trait__body\">\n")
	b.WriteString(bodyHTML)
	b.WriteString("</div>\n")
	b.WriteString("</section>\n")
	return b.String()
}

// renderTraitBody renders a node's own intro blocks, then (if any) its children
// wrapped in a single .sc-trait__nest rail. Returns leadProse=true when the first
// intro block is plain prose (eligible for the drop cap).
func renderTraitBody(intro string, children []*traitNode) (body string, leadProse bool) {
	var b strings.Builder
	first := true
	signatureHint := strings.Contains(strings.ToLower(intro), "signature")

	for _, p := range paraSplitRe.Split(intro, -1) {
		tp := strings.TrimSpace(p)
		if tp == "" {
			continue
		}
		kind := classifyTraitBlock(tp)
		switch kind {
		case "flavor":
			fmt.Fprintf(&b, "<p class=\"sc-trait__flavor\">%s</p>\n", traitInline(strings.Trim(tp, "*")))
		case "tiers":
			var t [3]string
			parseTiers(tp, &t)
			b.WriteString(tierPanelHTML("", "", t, traitInline))
		case "list":
			b.WriteString(renderTraitList(tp))
		case "table":
			b.WriteString(renderTraitTable(tp))
		case "callout":
			b.WriteString(renderTraitCallout(tp))
		case "leadin":
			fmt.Fprintf(&b, "<p class=\"sc-trait__leadin\"><span class=\"sc-trait__dia\"></span>%s</p>\n", traitInline(collapseLines(tp)))
		case "benefit", "drawback":
			b.WriteString(renderTraitSegment(tp))
		default: // text
			if first {
				leadProse = true
			}
			fmt.Fprintf(&b, "<p>%s</p>\n", traitInline(collapseLines(tp)))
		}
		first = false
	}

	if len(children) > 0 {
		b.WriteString("<div class=\"sc-trait__nest\">\n")
		for _, c := range children {
			if c.isAbility {
				b.WriteString(renderAbilityCard(synthAbilityFM(c.name, signatureHint), c.content))
			} else {
				b.WriteString(renderTraitNode(c))
			}
		}
		b.WriteString("</div>\n")
	}
	return b.String(), leadProse
}

// classifyTraitBlock buckets one intro paragraph block.
func classifyTraitBlock(tp string) string {
	switch {
	case isCalloutBlock(tp):
		return "callout"
	case isItalicPara(tp):
		return "flavor"
	case isTableBlock(tp):
		return "table"
	case isTierListBlock(tp):
		return "tiers"
	case isListBlock(tp):
		return "list"
	case segLabelRe.MatchString(tp):
		if strings.HasPrefix(strings.ToLower(strings.TrimLeft(tp, "* ")), "drawback") {
			return "drawback"
		}
		return "benefit"
	case isTraitLeadin(tp):
		return "leadin"
	default:
		return "text"
	}
}

// calloutTitleRe pulls the bold title from a callout's first de-quoted line.
var calloutTitleRe = regexp.MustCompile(`^\*\*(.+?)\*\*$`)

// isCalloutBlock reports whether a body block is a `@type: callout` annotation
// comment followed by its blockquote (the comment is the first non-blank line).
func isCalloutBlock(tp string) bool {
	for _, ln := range strings.Split(tp, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		return content.IsCalloutComment(t)
	}
	return false
}

// renderTraitCallout renders a `@type: callout` block as a recessed `.sc-callout`
// aside. It drops the annotation comment, strips the leading `>` blockquote markers,
// takes the first **bold** line as the title, and renders the remaining blocks
// (paragraphs + bullet lists) as the aside body — so callout content reads as a
// proper sidebar instead of leaking comment/quote markers as escaped text.
func renderTraitCallout(block string) string {
	// Drop the annotation comment line(s); de-blockquote the remainder.
	var md []string
	for _, ln := range strings.Split(block, "\n") {
		t := strings.TrimSpace(ln)
		if content.IsCalloutComment(t) {
			continue
		}
		t = strings.TrimPrefix(t, ">")
		t = strings.TrimPrefix(t, " ")
		md = append(md, t)
	}
	body := strings.TrimSpace(strings.Join(md, "\n"))

	var title string
	var rest []string
	titleTaken := false
	for _, blk := range paraSplitRe.Split(body, -1) {
		tb := strings.TrimSpace(blk)
		if tb == "" {
			continue
		}
		if !titleTaken {
			titleTaken = true
			if m := calloutTitleRe.FindStringSubmatch(tb); m != nil {
				title = m[1]
				continue
			}
		}
		rest = append(rest, tb)
	}

	var b strings.Builder
	b.WriteString("<aside class=\"sc-callout\" data-action=\"callout\">\n")
	if title != "" {
		b.WriteString("<div class=\"sc-callout__title\"><span class=\"sc-callout__dia\"></span>" + traitInline(title) + "</div>\n")
	}
	b.WriteString("<div class=\"sc-callout__body\">\n")
	for _, blk := range rest {
		if isListBlock(blk) {
			b.WriteString(renderTraitList(blk))
		} else {
			b.WriteString("<p>" + traitInline(collapseLines(blk)) + "</p>\n")
		}
	}
	b.WriteString("</div>\n</aside>\n")
	return b.String()
}

// isTraitLeadin reports whether a prose paragraph is the "You gain the following
// ability:" run-in that introduces a nested block.
func isTraitLeadin(tp string) bool {
	t := strings.TrimRight(tp, " \t")
	if strings.HasSuffix(t, ":") {
		return true
	}
	low := strings.ToLower(t)
	return strings.Contains(low, "following") &&
		(strings.Contains(low, "abilit") || strings.Contains(low, "trait") || strings.Contains(low, "benefit"))
}

// isTableBlock reports whether a block is a pipe table: a leading "|" row
// followed by a "|---|---|" header-separator row.
func isTableBlock(tp string) bool {
	lines := strings.Split(tp, "\n")
	if len(lines) < 2 || !strings.HasPrefix(strings.TrimSpace(lines[0]), "|") {
		return false
	}
	return isTableSepLine(strings.TrimSpace(lines[1]))
}

func isTableSepLine(t string) bool {
	return strings.Contains(t, "-") && tableSepRe.MatchString(t)
}

// renderTraitTable renders a markdown pipe table as a real HTML <table> (the
// first row is the header; the |---| separator is dropped). Cells carry rich
// inline markdown — links resolve through traitInline like the rest of the body.
func renderTraitTable(block string) string {
	var rows [][]string
	for _, ln := range strings.Split(block, "\n") {
		t := strings.TrimSpace(ln)
		if !strings.HasPrefix(t, "|") || isTableSepLine(t) {
			continue
		}
		rows = append(rows, splitRow(t))
	}
	if len(rows) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("<table><thead><tr>")
	for _, c := range rows[0] {
		sb.WriteString("<th>" + traitInline(c) + "</th>")
	}
	sb.WriteString("</tr></thead><tbody>")
	for _, r := range rows[1:] {
		sb.WriteString("<tr>")
		for _, c := range r {
			sb.WriteString("<td>" + traitInline(c) + "</td>")
		}
		sb.WriteString("</tr>")
	}
	sb.WriteString("</tbody></table>\n")
	return sb.String()
}

// renderTraitList renders a "- …" / "* …" bullet block as a <ul>.
func renderTraitList(block string) string {
	var sb strings.Builder
	sb.WriteString("<ul>")
	for _, ln := range strings.Split(block, "\n") {
		t := strings.TrimSpace(ln)
		if t == "" {
			continue
		}
		sb.WriteString("<li>" + traitInline(strings.TrimSpace(t[1:])) + "</li>")
	}
	sb.WriteString("</ul>\n")
	return sb.String()
}

// renderTraitSegment renders a **Benefit:**/**Drawback:** paragraph as a titled
// mini-panel with the matching tone tag.
func renderTraitSegment(tp string) string {
	m := segLabelRe.FindStringSubmatch(tp)
	label, tone := "Benefit", "benefit"
	if strings.EqualFold(strings.TrimSpace(m[1]), "drawback") {
		label, tone = "Drawback", "drawback"
	}
	return fmt.Sprintf(
		"<div class=\"sc-trait__seg\" data-tone=\"%s\"><div class=\"sc-trait__seg-head\"><span class=\"sc-trait__dia\"></span><span class=\"tag\">%s</span></div><div class=\"sc-trait__seg-body\"><p>%s</p></div></div>\n",
		tone, label, traitInline(collapseLines(m[2])))
}

// synthAbilityFM builds minimal frontmatter for a nested ability parsed from a
// heading (renderAbilityCard reads name/type/cost here; action type + power roll
// come from the body). A "signature ability" lead-in upgrades it to a Signature
// cost badge.
func synthAbilityFM(name string, signature bool) string {
	fm := "name: " + name + "\ntype: ability"
	if signature {
		fm += "\nsubtype: signature"
	}
	return fm
}

// featureNoun is the eyebrow noun for a card that renders in the recessed
// .sc-trait niche. That visual is shared by real ancestry/monster traits and by
// plain class features, but the LABEL must reflect the actual frontmatter type:
// "Trait" only for type: trait (the narrowed taxonomy — ancestry + monster
// passives), "Feature" for everything else (the plain feature umbrella).
func featureNoun(featureType string) string {
	if strings.TrimSpace(featureType) == "trait" {
		return "Trait"
	}
	return "Feature"
}

// traitEyebrow is the source context line: "<Class> Feature" / "<Ancestry> Trait"
// (small-caps), optionally suffixed with the subclass when present (e.g. an
// order/domain). The level lives in the right-hand tag, so it is not duplicated here.
func traitEyebrow(fm string) string {
	source := ""
	for _, key := range []string{"class", "ancestry", "kit"} {
		if v := strings.TrimSpace(parseFrontmatterField(fm, key)); v != "" {
			source = titleCase(strings.ReplaceAll(v, "-", " "))
			break
		}
	}
	label := strings.TrimSpace(source + " " + featureNoun(parseFrontmatterField(fm, "type")))
	if sub := strings.TrimSpace(parseFrontmatterField(fm, "subclass")); sub != "" {
		label += " · " + titleCase(strings.ReplaceAll(sub, "-", " "))
	}
	return label
}

var sccLevelRe = regexp.MustCompile(`level-(\d+)`)

// traitTag builds the right-side level pill, preferring the frontmatter `level`
// and falling back to a `level-N` segment in the scc. Empty when neither exists.
func traitTag(level, scc string) string {
	n := strings.TrimSpace(level)
	if n == "" {
		if m := sccLevelRe.FindStringSubmatch(scc); m != nil {
			n = m[1]
		}
	}
	if n == "" {
		return ""
	}
	return fmt.Sprintf("<div class=\"sc-trait__tag\">Level <span class=\"num\">%s</span></div>\n", html.EscapeString(n))
}

// parseTraitTree splits a trait page body into the leading intro markdown (before
// the first heading) and the heading subtree rebuilt by level.
func parseTraitTree(body string) (intro string, roots []*traitNode) {
	locs := traitHeadRe.FindAllStringSubmatchIndex(body, -1)
	if len(locs) == 0 {
		return strings.TrimSpace(body), nil
	}
	intro = strings.TrimSpace(body[:locs[0][0]])

	// flat list of headings with their content spans (heading end → next heading)
	var flat []*traitNode
	for i, loc := range locs {
		level := loc[3] - loc[2] // length of the leading '#' run
		name := strings.TrimSpace(body[loc[4]:loc[5]])
		scc := ""
		if loc[6] >= 0 {
			if m := traitSCCRe.FindStringSubmatch(body[loc[6]:loc[7]]); m != nil {
				scc = m[1]
			}
		}
		contentStart := loc[1]
		contentEnd := len(body)
		if i+1 < len(locs) {
			contentEnd = locs[i+1][0]
		}
		flat = append(flat, &traitNode{
			level:     level,
			name:      name,
			scc:       scc,
			isAbility: strings.Contains(scc, "feature.ability."),
			content:   strings.TrimSpace(body[contentStart:contentEnd]),
		})
	}

	// rebuild the tree by level using a stack of open ancestors
	var stack []*traitNode
	for _, n := range flat {
		for len(stack) > 0 && stack[len(stack)-1].level >= n.level {
			stack = stack[:len(stack)-1]
		}
		if len(stack) == 0 {
			roots = append(roots, n)
		} else {
			top := stack[len(stack)-1]
			top.children = append(top.children, n)
		}
		stack = append(stack, n)
	}
	return intro, roots
}
