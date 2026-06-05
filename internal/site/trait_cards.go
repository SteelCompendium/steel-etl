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
)

var (
	// a markdown heading line, with optional trailing {…attr_list…}
	traitHeadRe = regexp.MustCompile(`(?m)^(#{1,6})[ \t]+(.+?)[ \t]*(?:\{([^}]*)\})?[ \t]*$`)
	traitSCCRe  = regexp.MustCompile(`data-scc="([^"]+)"`)
	// a **Benefit:** / **Drawback:** lead-labeled paragraph → titled segment
	segLabelRe = regexp.MustCompile(`(?is)^\*\*\s*(benefit|drawback)s?\s*:?\s*\*\*\s*(.+)$`)
	// single-* italic, run AFTER bold has consumed every **…** pair
	traitItalicRe = regexp.MustCompile(`\*([^*\n]+)\*`)
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

	cls := "sc-trait"
	if leadProse {
		cls += " sc-trait--lead" // engraved drop-cap on the opening paragraph
	}

	return wrapTraitSection(cls, eyebrow, name, tag, bodyHTML)
}

// renderTraitNode renders a nested sub-trait (no eyebrow, no drop cap; its level
// pill is derived from the scc). Recurses through its own children.
func renderTraitNode(n *traitNode) string {
	intro, _ := parseTraitTree(n.content) // sub-headings already split into n.children
	bodyHTML, _ := renderTraitBody(intro, n.children)
	tag := traitTag("", n.scc)
	return wrapTraitSection("sc-trait", "", strings.TrimSpace(n.name), tag, bodyHTML)
}

// wrapTraitSection assembles one <section class="sc-trait …"> with header +
// body, as a single contiguous block (no blank lines).
func wrapTraitSection(cls, eyebrow, name, tag, bodyHTML string) string {
	dia := `<span class="sc-trait__dia"></span>`
	var b strings.Builder
	fmt.Fprintf(&b, "<section class=\"%s\" data-action=\"trait\">\n", cls)
	b.WriteString("<header class=\"sc-trait__head\">\n")
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
		case "list":
			b.WriteString(renderTraitList(tp))
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
	case isItalicPara(tp):
		return "flavor"
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

// traitEyebrow is the class/ancestry/kit context line (small-caps), title-cased.
func traitEyebrow(fm string) string {
	for _, key := range []string{"class", "ancestry", "kit"} {
		if v := strings.TrimSpace(parseFrontmatterField(fm, key)); v != "" {
			return titleCase(strings.ReplaceAll(v, "-", " "))
		}
	}
	return ""
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
