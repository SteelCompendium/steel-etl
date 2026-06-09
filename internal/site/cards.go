package site

// Rich category-index cards for the Steel Compendium MkDocs site.
//
// Replaces the flat ".browse-index" link list with ".sc-card" stat-cards for
// every flat leaf index type (kit, class, ancestry, career, treasure, perk,
// title, complication, culture, condition, skill, movement, negotiation).
// The nested aggregations (feature, ability) keep the existing list/expand UI.
//
// ALL data comes from frontmatter the content parser already emits, plus each
// page body for blurbs / the kit signature ability — so NO change to the shared
// data repos. Styled by docs/stylesheets/steel-redesign.css.
//
// Two card shapes:
//   - grid cards  (default): multi-column .sc-cards grid of stat-cards.
//   - WIDE cards  (complication, perk): full-width editorial rows for types
//     with hundreds of entries and longer text — .sc-cards--wide.
//
// Wire-up: at the very top of buildIndexContent() in build.go, add:
//
//	if cards, ok := buildCardsContent(dir, dirName, files, subdirs); ok {
//		return cards
//	}

import (
	"bytes"
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
)

// richCardTypes lists index directories rendered as stat-cards. Only flat type
// dirs (no subdirs) qualify — feature/ability are nested and excluded.
var richCardTypes = map[string]bool{
	"kit": true, "class": true, "ancestry": true, "career": true,
	"treasure": true, "perk": true, "title": true, "complication": true,
	"culture": true, "condition": true, "skill": true,
	"movement": true, "negotiation": true,
}

// wideCardTypes render as full-width editorial rows (.sc-cards--wide) instead of
// the multi-column grid — used for the high-count, text-led types.
var wideCardTypes = map[string]bool{
	"complication": true, "perk": true,
}

// buildCardsContent returns rich-card index markup for a supported flat type.
// ok=false → caller falls back to the default browse-index list.
func buildCardsContent(dir, dirName string, files, subdirs []string) (content string, ok bool) {
	leaf := len(subdirs) == 0 && len(files) > 0
	var cardType string
	switch {
	// Rule (glossary) leaves are nested one level under rule/<group>; render their
	// terms as rule cards keyed by the group. This MUST precede richCardTypes:
	// several group names (treasure, negotiation) collide with rich card types, and
	// every rule term should render as the same rule card regardless of its group.
	case leaf && pathHasSegment(dir, "rule"):
		cardType = "rule"
	case richCardTypes[dirName]:
		cardType = dirName
	// Treasure leaves are nested (treasure/<tier>/<category>); render their items as
	// treasure cards even though the leaf dirName isn't "treasure".
	case leaf && pathHasSegment(dir, "treasure"):
		cardType = "treasure"
	// Skill leaves are nested (skill/<group>/<item>); render their items as skill
	// cards. The self-named <group>.md container page is dropped below.
	case leaf && pathHasSegment(dir, "skill"):
		cardType = "skill"
		files = dropSelfNamed(files, dirName)
	default:
		return "", false
	}
	if len(files) == 0 || len(subdirs) > 0 {
		return "", false
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })

	wrapper := "sc-cards"
	if wideCardTypes[cardType] {
		wrapper = "sc-cards sc-cards--wide"
	}

	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n<div class=\"" + wrapper + "\">\n")
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(dir, f))
		if err != nil {
			continue
		}
		fm, body := splitFrontmatter(string(data))
		name := parseFrontmatterField(fm, "name")
		if name == "" {
			name = fileToTitle(f)
		}
		sb.WriteString(cardFor(cardType, dirName, fm, body, f, name))
	}
	sb.WriteString("</div>\n")
	return sb.String(), true
}

// dropSelfNamed removes the self-named container page (<dirName>.md) from a leaf
// directory's file list, so a skill-group's landing page doesn't appear as a
// card inside its own group grid.
func dropSelfNamed(files []string, dirName string) []string {
	self := dirName + ".md"
	out := files[:0:0]
	for _, f := range files {
		if f != self {
			out = append(out, f)
		}
	}
	return out
}

// pathHasSegment reports whether any path segment of dir equals seg.
func pathHasSegment(dir, seg string) bool {
	for _, p := range strings.Split(filepath.ToSlash(dir), "/") {
		if p == seg {
			return true
		}
	}
	return false
}

func cardFor(t, dirName, fm, body, file, name string) string {
	switch t {
	case "kit":
		return kitCard(fm, body, file, name)
	case "rule":
		return ruleCard(fm, body, file, name, dirName)
	case "class":
		return classCard(fm, body, file, name)
	case "ancestry":
		return ancestryCard(fm, body, file, name)
	case "career":
		return careerCard(fm, body, file, name)
	case "treasure":
		return treasureCard(fm, body, file, name)
	case "perk":
		return perkCard(fm, body, file, name)
	case "title":
		return titleCard(fm, body, file, name)
	case "complication":
		return complicationCard(fm, body, file, name)
	case "culture":
		return cultureCard(fm, body, file, name)
	default: // condition, skill, movement, negotiation
		max := 96
		if t == "skill" { // skills are short — give them room so they don't ellipsize
			max = 220
		}
		// iconPaths is keyed by type, so the type name doubles as the icon key
		// (crestSVG falls back to the scroll glyph for any unmapped type).
		return card(file, t, titleCase(t), name, blurbBlock(bodyBlurb(body, max)))
	}
}

// ── per-type builders ───────────────────────────────────────────────────────

func kitCard(fm, body, file, name string) string {
	stam := bonusShort(parseFrontmatterField(fm, "stamina_bonus"))
	spd := orZero(parseFrontmatterField(fm, "speed_bonus"))
	stab := orZero(parseFrontmatterField(fm, "stability_bonus"))
	dis := orZero(firstField(fm, "disengage_bonus", "disengage"))
	melee := orDash(strings.TrimSpace(parseFrontmatterField(fm, "melee_damage_bonus")))
	ranged := orDash(strings.TrimSpace(parseFrontmatterField(fm, "ranged_damage_bonus")))
	meleeDist := orDash(strings.TrimSpace(parseFrontmatterField(fm, "melee_distance_bonus")))
	rangedDist := orDash(strings.TrimSpace(parseFrontmatterField(fm, "ranged_distance_bonus")))
	sigName, sigType, keywords := signatureFromBody(body)
	// The crest is always the backpack (matching the Kits card on the Browse
	// landing); only the type label distinguishes martial/magic/psionic kits.
	kind := "Martial"
	switch {
	case strings.Contains(keywords, "Psionic"):
		kind = "Psionic"
	case strings.Contains(keywords, "Magic"):
		kind = "Magic"
	}

	// Equipment: show the raw source sentence verbatim — parsing it into
	// armor/weapon tokens lost nuance (e.g. "one or two light weapons"). The
	// line is always emitted (a non-breaking space when a kit lacks equipment)
	// so every card reserves the same vertical space and stays aligned.
	equip := html.EscapeString(strings.TrimSpace(parseFrontmatterField(fm, "equipment_text")))
	if equip == "" {
		equip = "&nbsp;"
	}
	inner := "  <div class=\"sc-card__equip\">" + equip + "</div>\n"
	// First flavor line — the kit's brief description (links stripped, truncated).
	if desc := cardFlavor(fm, body); desc != "" {
		inner += flavorDiv(desc, 160)
	}
	// Row 1 — defensive / movement stats. Stamina bonuses are per echelon.
	inner += statsBlock([][3]string{{stam, "Stamina per Echelon", ""}, {spd, "Speed", ""}, {stab, "Stability", ""}, {dis, "Disengage", ""}})
	// Row 2 — offensive stats. Melee & ranged damage and distance can each be
	// populated independently, so all four show on every card.
	inner += statsBlock([][3]string{
		{melee, "Melee Dmg", "is-dmg"},
		{ranged, "Ranged Dmg", "is-dmg"},
		{meleeDist, "Melee Dist", ""},
		{rangedDist, "Ranged Dist", ""},
	})
	if sigName != "" {
		inner += sigBlock(sigType, sigName)
	}
	return card(file, "kit", kind+" Kit", name, inner)
}

func classCard(fm, body, file, name string) string {
	inner := ""
	if hr := parseFrontmatterField(fm, "heroic_resource"); hr != "" {
		inner += lineBlock("Heroic Resource", "<span class=\"hl\">"+html.EscapeString(hr)+"</span>")
	}
	if chars := parseFrontmatterList(fm, "primary_characteristics"); len(chars) > 0 {
		inner += tagsBlock(chars)
	}
	// Full intro section — everything before the "Basics" header — rendered with
	// its paragraph/blockquote structure (and newlines) preserved. The class's
	// flavor is important context for the index, so it is shown in full.
	if sec := blockMD(classIntro(body)); sec != "" {
		inner += "  <div class=\"sc-card__intro\">" + sec + "</div>\n"
	}
	if inner == "" {
		inner = blurbBlock(bodyBlurb(body, 96))
	}
	return card(file, "class", "Class", name, inner)
}

func ancestryCard(fm, body, file, name string) string {
	inner := ""
	if t := parseFrontmatterField(fm, "signature_trait_name"); t != "" {
		inner += lineBlock("Signature Trait", "<span class=\"hl\">"+html.EscapeString(t)+"</span>")
	}
	// First flavor paragraph, shown in full (no truncation).
	if f := cardFlavor(fm, body); f != "" {
		inner += flavorDiv(f, 0)
	}
	if inner == "" {
		inner = blurbBlock(bodyBlurb(body, 96))
	}
	return card(file, "ancestry", "Ancestry", name, inner)
}

func careerCard(fm, body, file, name string) string {
	inner := ""
	// First line of flavor (minus the "In defining your career…" boilerplate).
	if f := strings.TrimSpace(parseFrontmatterField(fm, "flavor")); f != "" {
		inner += flavorDiv(f, 200)
	} else if f := careerFlavor(body); f != "" {
		inner += flavorDiv(f, 200)
	}
	// Four standard numeric fields as stat boxes. The source carries the
	// languages count as a "<Count> language(s)" string under the singular
	// `language` key (a YAML list under `languages` is also supported); reduce
	// it to its leading count word so it fits the stat box.
	lang := ""
	if list := parseFrontmatterList(fm, "languages"); len(list) > 0 {
		lang = fmt.Sprintf("%d", len(list))
	} else {
		lang = orDash(careerLanguageCount(firstField(fm, "language", "languages")))
	}
	inner += statsBlock([][3]string{
		{lang, "Languages", ""},
		{orDash(parseFrontmatterField(fm, "project_points")), "Project Pts", ""},
		{orDash(parseFrontmatterField(fm, "renown")), "Renown", ""},
		{orDash(parseFrontmatterField(fm, "wealth")), "Wealth", ""},
	})
	// Skills & Perk are long / non-standard — render as wrapping text lines.
	if sk := joinField(fm, "skills"); sk != "" {
		inner += lineBlock("Skills", inlineMD(sk))
	}
	if pk := strings.TrimSpace(parseFrontmatterField(fm, "perk")); pk != "" {
		inner += lineBlock("Perk", inlineMD(pk))
	}
	return card(file, "career", "Career", name, inner)
}

func treasureCard(fm, body, file, name string) string {
	tt := titleCase(strings.ReplaceAll(parseFrontmatterField(fm, "treasure_type"), "-", " "))
	if tt == "" {
		tt = "Treasure"
	}
	inner := ""
	// Keywords stay pinned to the top as tags.
	if kw := parseFrontmatterList(fm, "keywords"); len(kw) > 0 {
		inner += tagsBlock(kw)
	}
	// Flavor: the first prose paragraph (the italic descriptor), shown in full (no
	// truncation). The --clamp modifier reserves a fixed 3-line height so a 1-line
	// and a 3-line descriptor still produce the same card height.
	if f := cardFlavor(fm, body); f != "" {
		inner += "  <div class=\"sc-card__flavor sc-card__flavor--clamp\">" + html.EscapeString(f) + "</div>\n"
	}
	// Project goal & roll characteristic are short, fixed values — render them as
	// stat sub-cards (the kit-card treatment). They live as "**Label:**" lines in
	// the body; the parser doesn't lift them into frontmatter.
	var stats []statCell
	if v := firstNonEmpty(parseFrontmatterField(fm, "project_goal"), bodyLabeledLine(body, "Project Goal")); v != "" {
		disp, tip := goalStat(stripMD(v))
		stats = append(stats, statCell{val: disp, label: "Project Goal", title: tip})
	}
	if v := firstNonEmpty(parseFrontmatterField(fm, "project_roll_characteristic"), bodyLabeledLine(body, "Project Roll Characteristic")); v != "" {
		stats = append(stats, statCell{val: stripMD(v), label: "Roll Characteristic"})
	}
	inner += statsCells(stats)
	// Item prerequisite & project source are free text — wrapping label lines
	// (the Career card's skill/perk treatment).
	if v := bodyLabeledLine(body, "Item Prerequisite"); v != "" {
		inner += lineBlock("Prerequisite", inlineMD(v))
	}
	if v := bodyLabeledLine(body, "Project Source"); v != "" {
		inner += lineBlock("Source", inlineMD(v))
	}
	return card(file, "treasure", tt, name, inner)
}

func perkCard(fm, body, file, name string) string {
	label := "Perk"
	if g := titleCase(strings.ReplaceAll(parseFrontmatterField(fm, "perk_group"), "-", " ")); g != "" {
		label = g + " Perk"
	}
	inner := ""
	if v := firstField(fm, "prerequisites", "prerequisite"); v != "" {
		inner += "  <div class=\"sc-card__line\"><b>Prerequisite</b> <span class=\"hl\">" + html.EscapeString(v) + "</span></div>\n"
	}
	// Perk text: the leading run of prose paragraphs, with their paragraph
	// breaks preserved, truncated generously (these are the wide cards).
	if b := proseBlock(body, 480); b != "" {
		inner += flavorDivML(b)
	}
	return wideCard(file, "perk", label, name, inner)
}

func titleCard(fm, body, file, name string) string {
	label := "Title"
	if e := parseFrontmatterField(fm, "echelon"); e != "" {
		label = "Echelon " + e
	}
	inner := ""
	// First paragraph is flavor — shown in full (no truncation).
	if f := cardFlavor(fm, body); f != "" {
		inner += flavorDiv(f, 0)
	}
	if v := firstField(fm, "prerequisites", "prerequisite"); v != "" {
		inner += "  <div class=\"sc-card__line\"><b>Prerequisite</b> <span class=\"hl\">" + html.EscapeString(v) + "</span></div>\n"
	}
	return card(file, "title", label, name, inner)
}

func complicationCard(fm, body, file, name string) string {
	// Take the 1–2 line description/flavor that sits ABOVE the benefit/drawback.
	flavor := strings.TrimSpace(parseFrontmatterField(fm, "flavor"))
	if flavor == "" {
		flavor = complicationFlavor(body)
	}
	if flavor == "" {
		// Combined "Benefit and Drawback:" entries have no lead-in — fall back to
		// the benefit (or drawback) text so the card isn't empty.
		flavor = stripMD(firstField(fm, "benefit", "drawback"))
	}
	// Description/flavor shown in full (no truncation).
	inner := flavorDiv(flavor, 0)
	return wideCard(file, "complication", "Complication", name, inner)
}

func cultureCard(fm, body, file, name string) string {
	var tags []string
	for _, k := range []string{"environment", "organization", "upbringing"} {
		if v := parseFrontmatterField(fm, k); v != "" {
			tags = append(tags, v)
		}
	}
	inner := tagsBlock(tags)
	// First flavor paragraph.
	if f := cardFlavor(fm, body); f != "" {
		inner += flavorDiv(f, 240)
	}
	// "Skill Options" lives in the body as a bolded "**Skill Options:**" lead-in,
	// not frontmatter — fall back to extracting it from the body.
	so := joinField(fm, "skill_options")
	if so == "" {
		so = bodyLabeledLine(body, "Skill Options")
	}
	if so != "" {
		inner += lineBlock("Skill Options", inlineMD(so))
	}
	if inner == "" {
		inner = blurbBlock(bodyBlurb(body, 96))
	}
	return card(file, "culture", "Culture", name, inner)
}

// ruleCard renders a glossary term: crest + its group as the type label (e.g.
// "Dice", "Combat") + the term name + the first prose line of its definition.
// Rule pages carry no structured fields beyond name/scc/type, so the blurb is
// the only body content — a generous cap keeps short definitions whole.
func ruleCard(fm, body, file, name, groupDir string) string {
	return card(file, "rule", dirToTitle(groupDir), name, blurbBlock(bodyBlurb(body, 200)))
}

// ── shared builders ─────────────────────────────────────────────────────────

// card and wideCard use the "stretched link" pattern: the card is a <div>, and a
// single absolutely-positioned overlay <a class="sc-card__link"> is the whole-card
// click target. Inner content (incl. the markdown links inlineMD renders) stays in
// normal flow and sits above the overlay via CSS z-index, so those links remain
// independently clickable. A bare <a> wrapping the card would make any inner link an
// invalid <a>-in-<a> that browsers split apart, fragmenting the card. The overlay
// gets an aria-label so it announces the card name. See docs/stylesheets/steel-redesign.css.

func card(file, icon, typeLabel, name, inner string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<div class=\"sc-card sc-fil\">\n")
	fmt.Fprintf(&sb, "  <a class=\"sc-card__link\" href=\"%s\" aria-label=\"%s\"></a>\n",
		html.EscapeString(dirURL(file)), html.EscapeString(name))
	fmt.Fprintf(&sb, "  <div class=\"sc-card__head\"><span class=\"sc-crest\"><span>%s</span></span>\n", crestSVG(icon))
	fmt.Fprintf(&sb, "    <div><div class=\"sc-card__type\">%s</div>\n", html.EscapeString(typeLabel))
	fmt.Fprintf(&sb, "    <div class=\"sc-card__name\">%s</div></div></div>\n", html.EscapeString(name))
	sb.WriteString(inner)
	sb.WriteString("</div>\n")
	return sb.String()
}

// wideCard is the full-width editorial row: crest · name column · body column.
// inner is raw HTML (already escaped where needed).
func wideCard(file, icon, typeLabel, name, inner string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<div class=\"sc-card sc-card--wide sc-fil\">\n")
	fmt.Fprintf(&sb, "  <a class=\"sc-card__link\" href=\"%s\" aria-label=\"%s\"></a>\n",
		html.EscapeString(dirURL(file)), html.EscapeString(name))
	fmt.Fprintf(&sb, "  <span class=\"sc-crest lg\"><span>%s</span></span>\n", crestSVG(icon))
	fmt.Fprintf(&sb, "  <div class=\"sc-card__namecol\"><div class=\"sc-card__type\">%s</div>"+
		"<div class=\"sc-card__name\">%s</div></div>\n", html.EscapeString(typeLabel), html.EscapeString(name))
	fmt.Fprintf(&sb, "  <div class=\"sc-card__body\">%s</div>\n", inner)
	sb.WriteString("</div>\n")
	return sb.String()
}

func sigBlock(sigType, sigName string) string {
	return fmt.Sprintf("  <div class=\"sc-card__sig\"><span class=\"sc-card__dot\" data-type=\"%s\"></span>"+
		"<span class=\"sc-card__sig-label\">Signature</span>"+
		"<span class=\"sc-card__sig-name\">%s</span></div>\n", sigType, html.EscapeString(sigName))
}

// statCell is one cell of a stat grid: value, label, an optional extra class on
// the cell, and an optional native tooltip (title attr) on the value.
type statCell struct {
	val, label, cls, title string
}

// statsBlock renders an N-column stat grid. Each entry is {value, label, extraClass}.
func statsBlock(stats [][3]string) string {
	cells := make([]statCell, len(stats))
	for i, s := range stats {
		cells[i] = statCell{val: s[0], label: s[1], cls: s[2]}
	}
	return statsCells(cells)
}

// statsCells is the underlying stat-grid renderer (statsBlock is the [][3]string
// shorthand). A cell with a non-empty title gets a native hover tooltip on its
// value; that value is also raised above the card's stretched-link overlay (the
// .has-tip class) so the tooltip is reachable.
func statsCells(cells []statCell) string {
	if len(cells) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "  <div class=\"sc-card__stats\" style=\"grid-template-columns:repeat(%d,1fr)\">\n", len(cells))
	for _, c := range cells {
		cls := "sc-card__stat"
		style := ""
		if c.cls != "" {
			cls += " " + c.cls
		}
		if len([]rune(c.val)) > 4 { // long values (e.g. +1/+1/+1) need a smaller face
			style = " style=\"font-size:.72rem\""
		}
		vcls, title := "v", ""
		if c.title != "" {
			vcls, title = "v has-tip", " title=\""+html.EscapeString(c.title)+"\""
		}
		fmt.Fprintf(&sb, "    <div class=\"%s\"><div class=\"%s\"%s%s>%s</div><div class=\"l\">%s</div></div>\n",
			cls, vcls, style, title, html.EscapeString(c.val), html.EscapeString(c.label))
	}
	sb.WriteString("  </div>\n")
	return sb.String()
}

func tagsBlock(tags []string) string {
	if len(tags) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("  <div class=\"sc-card__tags\">")
	for _, t := range tags {
		sb.WriteString("<span class=\"sc-tag\">" + html.EscapeString(t) + "</span>")
	}
	sb.WriteString("</div>\n")
	return sb.String()
}

// inlineMDRenderer is the default CommonMark renderer (WithUnsafe off, so raw
// HTML in source is escaped). Reused for every inlineMD call.
var inlineMDRenderer = goldmark.New()

var hrefAttrRe = regexp.MustCompile(`href="([^"]*)"`)

// dirURL converts a relative ".md" link target to the directory-URL form MkDocs
// serves under use_directory_urls (e.g. "agent.md" → "agent/",
// "../skill/sneak.md" → "../skill/sneak/", "index.md" → "" so "../skill/index.md"
// → "../skill/"). Cards are emitted as raw HTML that MkDocs never post-processes,
// so the ".md" links are dead (404) unless rewritten here. External, anchor-only,
// and non-.md hrefs pass through unchanged. A "#fragment" suffix is preserved.
func dirURL(href string) string {
	if href == "" || strings.HasPrefix(href, "#") ||
		strings.HasPrefix(href, "http://") || strings.HasPrefix(href, "https://") ||
		strings.HasPrefix(href, "mailto:") {
		return href
	}
	path, frag := href, ""
	if i := strings.IndexByte(href, '#'); i >= 0 {
		path, frag = href[:i], href[i:]
	}
	if strings.HasSuffix(path, ".md") {
		path = strings.TrimSuffix(path, ".md")
		if base := path[strings.LastIndexByte(path, '/')+1:]; base == "index" {
			path = path[:len(path)-len("index")] // ".../index" → ".../"
		} else {
			path += "/"
		}
	}
	return path + frag
}

// rewriteHrefs applies dirURL to every href="…" in a fragment of HTML.
func rewriteHrefs(htmlFrag string) string {
	return hrefAttrRe.ReplaceAllStringFunc(htmlFrag, func(m string) string {
		return `href="` + dirURL(hrefAttrRe.FindStringSubmatch(m)[1]) + `"`
	})
}

// inlineMD renders a string of inline markdown (links, emphasis, bold, code) to
// HTML for embedding directly in a card's raw-HTML divs.
//
// Why pre-render here instead of relying on md_in_html: card values nest several
// levels deep inside non-attributed elements (sc-cards > a.sc-card > div.sc-card__line).
// md_in_html only processes content whose entire ancestor chain carries a
// `markdown` attribute, and adding those attributes mangles the <a>-wrapped card
// (the anchor gets split by stray <p> tags). So the markdown is converted to HTML
// at build time and embedded verbatim. goldmark's block <p> wrapper is stripped to
// keep the result inline; raw HTML is escaped by goldmark, so this is XSS-safe.
func inlineMD(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := inlineMDRenderer.Convert([]byte(s), &buf); err != nil {
		return html.EscapeString(s) // fall back to escaped literal on render error
	}
	out := strings.TrimSpace(buf.String())
	// Strip the single block <p>…</p> wrapper goldmark adds (joinField produces a
	// single line, so there is exactly one). Multi-paragraph values keep their
	// inner breaks but lose only the outermost wrapper, which is acceptable here.
	out = strings.TrimSuffix(strings.TrimPrefix(out, "<p>"), "</p>")
	// goldmark renders ".md" link targets verbatim; rewrite them to the served
	// directory-URL form so the links resolve (see dirURL).
	out = rewriteHrefs(out)
	return strings.TrimSpace(out)
}

// blockMD renders a multi-paragraph block of markdown to HTML, keeping the
// block structure (paragraphs, blockquotes, lists) so newlines are preserved —
// unlike inlineMD, which strips the single <p> wrapper for inline values. Used
// for the class intro section. ".md" link targets are rewritten to the served
// directory-URL form; raw HTML in source is escaped by goldmark (XSS-safe).
func blockMD(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	var buf bytes.Buffer
	if err := inlineMDRenderer.Convert([]byte(s), &buf); err != nil {
		return html.EscapeString(s)
	}
	return rewriteHrefs(strings.TrimSpace(buf.String()))
}

// lineBlock writes a "Label value" line. value is already HTML (may contain a
// <span class="hl">); label is plain text (empty label → value only).
func lineBlock(label, valueHTML string) string {
	if label == "" {
		return "  <div class=\"sc-card__line\">" + valueHTML + "</div>\n"
	}
	return "  <div class=\"sc-card__line\"><b>" + html.EscapeString(label) + "</b> " + valueHTML + "</div>\n"
}

func blurbBlock(text string) string {
	if strings.TrimSpace(text) == "" {
		return ""
	}
	return "  <div class=\"sc-card__blurb\">" + html.EscapeString(text) + "</div>\n"
}

// flavorDiv wraps prose flavor. max<=0 → no truncation.
func flavorDiv(text string, max int) string {
	t := strings.TrimSpace(text)
	if t == "" {
		return ""
	}
	if max > 0 {
		t = truncate(t, max)
	}
	return "  <div class=\"sc-card__flavor\">" + html.EscapeString(t) + "</div>\n"
}

// flavorDivML wraps multi-line prose flavor, preserving its line breaks as <br>
// (the escaped text would otherwise collapse newlines in HTML).
func flavorDivML(text string) string {
	t := strings.TrimSpace(text)
	if t == "" {
		return ""
	}
	esc := strings.ReplaceAll(html.EscapeString(t), "\n", "<br>\n")
	return "  <div class=\"sc-card__flavor\">" + esc + "</div>\n"
}

func crestSVG(icon string) string {
	p := iconPaths[icon]
	if p == "" {
		p = iconPaths["scroll"]
	}
	// Filled render to match the landing's Material icons (the .sc-crest CSS sets
	// fill: currentColor; MDI glyphs are solid-fill designed, so no stroke).
	return `<svg viewBox="0 0 24 24" width="19" height="19" fill="currentColor">` + p + `</svg>`
}

// ── frontmatter / body helpers ──────────────────────────────────────────────

func bonusShort(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.Index(strings.ToLower(s), " per "); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	if s == "" {
		return "0"
	}
	return s
}

func orZero(s string) string {
	if s = strings.TrimSpace(s); s == "" {
		return "0"
	}
	return s
}

func orDash(s string) string {
	if s = strings.TrimSpace(s); s == "" {
		return "\u2014"
	}
	return s
}

// firstField returns the first non-empty frontmatter field among keys.
func firstField(fm string, keys ...string) string {
	for _, k := range keys {
		if v := strings.TrimSpace(parseFrontmatterField(fm, k)); v != "" {
			return v
		}
	}
	return ""
}

// joinField returns a field as a string — joining YAML lists with ", ".
func joinField(fm, key string) string {
	if list := parseFrontmatterList(fm, key); len(list) > 0 {
		return strings.Join(list, ", ")
	}
	return strings.TrimSpace(parseFrontmatterField(fm, key))
}

// parseFrontmatterList reads a YAML list (block "- item" lines, or inline
// "[a, b]") for a top-level key.
func parseFrontmatterList(fm, key string) []string {
	if v := parseFrontmatterField(fm, key); strings.HasPrefix(v, "[") {
		var out []string
		for _, p := range strings.Split(strings.Trim(v, "[]"), ",") {
			if p = strings.TrimSpace(strings.Trim(strings.TrimSpace(p), "\"'")); p != "" {
				out = append(out, p)
			}
		}
		return out
	}
	lines := strings.Split(fm, "\n")
	var out []string
	for i := 0; i < len(lines); i++ {
		if lines[i] != key+":" {
			continue
		}
		for j := i + 1; j < len(lines); j++ {
			t := strings.TrimSpace(lines[j])
			if strings.HasPrefix(t, "- ") {
				out = append(out, strings.Trim(strings.TrimSpace(t[2:]), "\"'"))
			} else if t == "" {
				continue
			} else {
				break
			}
		}
		break
	}
	return out
}

var (
	reSigSection = regexp.MustCompile(`(?mi)^#{1,6}\s+Signature Ability\s*$`)
	reHeading    = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*$`)
	reMdLink     = regexp.MustCompile(`\[([^\]]*)\]\([^)]*\)`)
	// trailing attr_list (e.g. ` {data-scc="…"}`) the heading-permalink pass
	// stamps onto headings — not part of the visible heading text.
	reHeadingAttr = regexp.MustCompile(`\s*\{[^}]*\}\s*$`)
)

func signatureFromBody(body string) (name, sigType, keywords string) {
	loc := reSigSection.FindStringIndex(body)
	if loc == nil {
		return "", "", ""
	}
	rest := body[loc[1]:]
	if m := reHeading.FindStringSubmatch(rest); m != nil {
		name = strings.TrimSpace(reHeadingAttr.ReplaceAllString(m[1], ""))
	}
	for _, line := range strings.Split(rest, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "|") || !strings.Contains(line, "**") {
			continue
		}
		if a := strings.Index(line, "**"); a >= 0 {
			if b := strings.Index(line[a+2:], "**"); b >= 0 {
				keywords = line[a+2 : a+2+b]
			}
		}
		break
	}
	sigType = "passive"
	switch {
	case strings.Contains(keywords, "Ranged"):
		sigType = "ranged"
	case strings.Contains(keywords, "Strike"):
		sigType = "strike"
	case strings.Contains(keywords, "Area"):
		sigType = "area"
	}
	return name, sigType, keywords
}

func stripMD(s string) string {
	s = reMdLink.ReplaceAllString(s, "$1")
	s = strings.ReplaceAll(s, "**", "")
	s = strings.ReplaceAll(s, "*", "")
	s = strings.ReplaceAll(s, "`", "")
	return strings.TrimSpace(s)
}

// isProse reports whether a trimmed line is body prose (not heading/table/list/rule).
func isProse(t string) bool {
	if t == "" || t == "---" {
		return false
	}
	return !strings.HasPrefix(t, "#") && !strings.HasPrefix(t, "|") &&
		!strings.HasPrefix(t, ">") && !strings.HasPrefix(t, "- ")
}

// cardFlavor returns the structured `flavor` frontmatter field (the parser is
// the single source of truth), falling back to the first body prose paragraph
// for pages produced before the field existed. Keeps card output stable while
// making the data field authoritative.
func cardFlavor(fm, body string) string {
	if f := strings.TrimSpace(parseFrontmatterField(fm, "flavor")); f != "" {
		return f
	}
	return firstProse(body)
}

// firstNonEmpty returns the first trimmed-non-empty value among the args.
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if s := strings.TrimSpace(v); s != "" {
			return s
		}
	}
	return ""
}

// firstProse returns the first prose paragraph of a page body, markdown-stripped.
func firstProse(body string) string {
	for _, raw := range strings.Split(body, "\n") {
		t := strings.TrimSpace(raw)
		if !isProse(t) {
			continue
		}
		if s := stripMD(t); s != "" {
			return s
		}
	}
	return ""
}

// careerFlavor returns the first flavor paragraph with the trailing "In
// defining your career…" prompt (and everything after it) removed. The books
// append that prompt to the end of the lead-in sentence, e.g. "You worked as a
// spy… In defining your career, think about the following questions:".
func careerFlavor(body string) string {
	for _, raw := range strings.Split(body, "\n") {
		t := strings.TrimSpace(raw)
		if !isProse(t) {
			continue
		}
		if i := strings.Index(t, "In defining your career"); i >= 0 {
			t = strings.TrimSpace(t[:i])
		}
		if s := stripMD(t); s != "" {
			return s
		}
	}
	return ""
}

// careerLanguageCount reduces a languages descriptor like "Two languages" to
// its leading count word ("Two"). Input without a trailing "language(s)" word
// is returned unchanged.
func careerLanguageCount(s string) string {
	s = strings.TrimSpace(s)
	low := strings.ToLower(s)
	for _, suf := range []string{" languages", " language"} {
		if strings.HasSuffix(low, suf) {
			return strings.TrimSpace(s[:len(s)-len(suf)])
		}
	}
	return s
}

// leadingNum matches the leading run of digits in a string.
var leadingNum = regexp.MustCompile(`^\d+`)

// goalStat reduces a Project Goal value to its leading number for the stat box.
// A goal with extra detail (e.g. "45 (yields 1d3 darts)") renders as "45*" with
// the full text as the value's hover tooltip; a plain number renders as-is with
// no tooltip.
func goalStat(goal string) (display, tooltip string) {
	goal = strings.TrimSpace(goal)
	num := leadingNum.FindString(goal)
	if num == "" || num == goal {
		return goal, ""
	}
	return num + "*", goal
}

// bodyLabeledLine returns the inline-markdown value following a "**Label:**"
// bold lead-in in the body (e.g. "**Skill Options:**"), or "" if absent.
func bodyLabeledLine(body, label string) string {
	prefix := "**" + label + ":**"
	for _, raw := range strings.Split(body, "\n") {
		t := strings.TrimSpace(raw)
		if strings.HasPrefix(t, prefix) {
			return strings.TrimSpace(t[len(prefix):])
		}
	}
	return ""
}

// complicationFlavor returns the description/flavor that sits above the
// Benefit/Drawback block (skipping those bolded lines).
func complicationFlavor(body string) string {
	for _, raw := range strings.Split(body, "\n") {
		t := strings.TrimSpace(raw)
		if !isProse(t) {
			continue
		}
		l := strings.ToLower(t)
		if strings.HasPrefix(l, "**benefit") || strings.HasPrefix(l, "**drawback") ||
			strings.HasPrefix(l, "benefit:") || strings.HasPrefix(l, "drawback:") {
			continue
		}
		if s := stripMD(t); s != "" {
			return s
		}
	}
	return ""
}

// reBasicsHeading matches the "Basics" section header that ends a class's
// intro flavor section. The trailing attr_list is optional: classified Basics
// sections (e.g. Beastheart's) carry a stamped ` {data-scc="…"}` suffix.
var reBasicsHeading = regexp.MustCompile(`(?mi)^#{1,6}\s+Basics\s*(?:\{[^}]*\})?\s*$`)

// classIntro returns the class's intro section: the markdown from the start of
// the body up to (but excluding) the "Basics" header, with the page renderer's
// injected "# Name" H1 (and the rule under it) stripped. Falls back to the
// whole body if no Basics header is present.
func classIntro(body string) string {
	if loc := reBasicsHeading.FindStringIndex(body); loc != nil {
		body = body[:loc[0]]
	}
	return strings.TrimSpace(stripLeadingTitle(body))
}

// stripLeadingTitle drops a leading injected H1 ("# Name") and the horizontal
// rule the page renderer places directly under it, so the intro starts at the
// flavor prose instead of repeating the card's name.
func stripLeadingTitle(body string) string {
	lines := strings.Split(body, "\n")
	i := 0
	skipBlank := func() {
		for i < len(lines) && strings.TrimSpace(lines[i]) == "" {
			i++
		}
	}
	skipBlank()
	if i < len(lines) && strings.HasPrefix(strings.TrimSpace(lines[i]), "# ") {
		i++
		skipBlank()
		if i < len(lines) && strings.TrimSpace(lines[i]) == "---" {
			i++
		}
	}
	return strings.Join(lines[i:], "\n")
}

// bodyBlurb returns the first prose sentence/paragraph, truncated.
func bodyBlurb(body string, max int) string {
	return truncate(firstProse(body), max)
}

// proseBlock collects the leading run of prose paragraphs (markdown-stripped),
// preserving the blank-line breaks between them, and truncates to max runes
// (max<=0 → no limit). Stops at the first non-prose block (heading, table,
// blockquote, list) once prose has started.
func proseBlock(body string, max int) string {
	var lines []string
	started := false
	for _, raw := range strings.Split(body, "\n") {
		t := strings.TrimSpace(raw)
		if t == "" {
			if started {
				lines = append(lines, "")
			}
			continue
		}
		if !isProse(t) {
			if started {
				break
			}
			continue
		}
		if s := stripMD(t); s != "" {
			lines = append(lines, s)
			started = true
		}
	}
	out := strings.TrimSpace(strings.Join(lines, "\n"))
	if max > 0 {
		out = truncate(out, max)
	}
	return out
}

func truncate(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if max <= 0 || len(r) <= max {
		return string(r)
	}
	return strings.TrimSpace(string(r[:max])) + "\u2026"
}

// ── crest icon paths (inline SVG, no emoji processing needed) ────────────────

// iconPaths are filled Material Design Icon glyphs keyed by card type, matching
// the crest icons on the Browse landing grid (static_content/docs/Browse/index.md)
// so every index page's cards carry the same icon as that category's landing
// card. Rendered filled (see crestSVG) — the landing's :material-…: icons are
// fill-designed too, so an outline (stroke) glyph here would not match.
var iconPaths = map[string]string{
	"kit":          `<path d="M16,5V4A2,2 0 0,0 14,2H10A2,2 0 0,0 8,4V5A4,4 0 0,0 4,9V20A2,2 0 0,0 6,22H18A2,2 0 0,0 20,20V9A4,4 0 0,0 16,5M10,4H14V5H10V4M12,9L14,11L12,13L10,11L12,9M18,16H9V18H8V16H6V15H18V16Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                             // bag-personal
	"class":        `<path d="M12 1L3 5V11C3 16.5 6.8 21.7 12 23C17.2 21.7 21 16.5 21 11V5L12 1M15 15H13V18H11V15H9V13H11L10 7.1L12 5.5L14 7.1L13 13H15V15Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                    // shield-sword
	"ancestry":     `<path d="M12,5.5A3.5,3.5 0 0,1 15.5,9A3.5,3.5 0 0,1 12,12.5A3.5,3.5 0 0,1 8.5,9A3.5,3.5 0 0,1 12,5.5M5,8C5.56,8 6.08,8.15 6.53,8.42C6.38,9.85 6.8,11.27 7.66,12.38C7.16,13.34 6.16,14 5,14A3,3 0 0,1 2,11A3,3 0 0,1 5,8M19,8A3,3 0 0,1 22,11A3,3 0 0,1 19,14C17.84,14 16.84,13.34 16.34,12.38C17.2,11.27 17.62,9.85 17.47,8.42C17.92,8.15 18.44,8 19,8M5.5,18.25C5.5,16.18 8.41,14.5 12,14.5C15.59,14.5 18.5,16.18 18.5,18.25V20H5.5V18.25M0,20V18.5C0,17.11 1.89,15.94 4.45,15.6C3.86,16.28 3.5,17.22 3.5,18.25V20H0M24,20H20.5V18.25C20.5,17.22 20.14,16.28 19.55,15.6C22.11,15.94 24,17.11 24,18.5V20Z"/>`, // account-group
	"career":       `<path d="M10 16V15H3L3 19C3 20.11 3.89 21 5 21H19C20.11 21 21 20.11 21 19V15H14V16H10M20 7H16V5L14 3H10L8 5V7H4C2.9 7 2 7.9 2 9V12C2 13.11 2.89 14 4 14H10V12H14V14H20C21.1 14 22 13.1 22 12V9C22 7.9 21.1 7 20 7M14 7H10V5H14V7Z"/>`,                                                                                                                                                                                                                                                                                                                                                                         // briefcase-variant
	"treasure":     `<path d="M5,4H19A3,3 0 0,1 22,7V11H15V10H9V11H2V7A3,3 0 0,1 5,4M11,11H13V13H11V11M2,12H9V13L11,15H13L15,13V12H22V20H2V12Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 // treasure-chest
	"perk":         `<path d="M16,9H19L14,16M10,9H14L12,17M5,9H8L10,16M15,4H17L19,7H16M11,4H13L14,7H10M7,4H9L8,7H5M6,2L2,8L12,22L22,8L18,2H6Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                  // diamond-stone
	"title":        `<path d="M5 16L3 5L8.5 10L12 4L15.5 10L21 5L19 16H5M19 19C19 19.6 18.6 20 18 20H6C5.4 20 5 19.6 5 19V18H19V19Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            // crown
	"complication": `<path d="M23,12L20.56,9.22L20.9,5.54L17.29,4.72L15.4,1.54L12,3L8.6,1.54L6.71,4.72L3.1,5.53L3.44,9.21L1,12L3.44,14.78L3.1,18.47L6.71,19.29L8.6,22.47L12,21L15.4,22.46L17.29,19.28L20.9,18.46L20.56,14.78L23,12M13,17H11V15H13V17M13,13H11V7H13V13Z"/>`,                                                                                                                                                                                                                                                                                                                                                         // alert-decagram
	"culture":      `<path d="M15,19L9,16.89V5L15,7.11M20.5,3C20.44,3 20.39,3 20.34,3L15,5.1L9,3L3.36,4.9C3.15,4.97 3,5.15 3,5.38V20.5A0.5,0.5 0 0,0 3.5,21C3.55,21 3.61,21 3.66,20.97L9,18.9L15,21L20.64,19.1C20.85,19 21,18.85 21,18.62V3.5A0.5,0.5 0 0,0 20.5,3Z"/>`,                                                                                                                                                                                                                                                                                                                                                            // map
	"condition":    `<path d="M11 15H6L13 1V9H18L11 23V15Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                     // lightning-bolt
	"skill":        `<path d="M12,1L9,9L1,12L9,15L12,23L15,15L23,12L15,9L12,1Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                 // star-four-points
	"movement":     `<path d="M13,11H18L16.5,9.5L17.92,8.08L21.84,12L17.92,15.92L16.5,14.5L18,13H13V18L14.5,16.5L15.92,17.92L12,21.84L8.08,17.92L9.5,16.5L11,18V13H6L7.5,14.5L6.08,15.92L2.16,12L6.08,8.08L7.5,9.5L6,11H11V6L9.5,7.5L8.08,6.08L12,2.16L15.92,6.08L14.5,7.5L13,6V11Z"/>`,                                                                                                                                                                                                                                                                                                                                            // arrow-all
	"negotiation":  `<path d="M17,12V3A1,1 0 0,0 16,2H3A1,1 0 0,0 2,3V17L6,13H16A1,1 0 0,0 17,12M21,6H19V15H6V17A1,1 0 0,0 7,18H18L22,22V7A1,1 0 0,0 21,6Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                     // forum
	// Books-tab crests (Read section). book = generic book-card fallback;
	// chapter = shared chapter-card glyph; sword-cross/paw/skull = per-book
	// icons referenced by site.yaml's books[].icon. See cards_book.go.
	"book":        `<path d="M12 21.5C10.65 20.65 8.2 20 6.5 20C4.85 20 3.15 20.3 1.75 21.05C1.65 21.1 1.6 21.1 1.5 21.1C1.25 21.1 1 20.85 1 20.6V6C1.6 5.55 2.25 5.25 3 5C4.11 4.65 5.33 4.5 6.5 4.5C8.45 4.5 10.55 4.9 12 6C13.45 4.9 15.55 4.5 17.5 4.5C18.67 4.5 19.89 4.65 21 5C21.75 5.25 22.4 5.55 23 6V20.6C23 20.85 22.75 21.1 22.5 21.1C22.4 21.1 22.35 21.1 22.25 21.05C20.85 20.3 19.15 20 17.5 20C15.8 20 13.35 20.65 12 21.5M12 8V19.5C13.35 18.65 15.8 18 17.5 18C18.7 18 19.9 18.15 21 18.5V7C19.9 6.65 18.7 6.5 17.5 6.5C15.8 6.5 13.35 7.15 12 8M13 11.5C14.11 10.82 15.6 10.5 17.5 10.5C18.41 10.5 19.26 10.59 20 10.78V9.23C19.13 9.08 18.29 9 17.5 9C15.73 9 14.23 9.28 13 9.84V11.5M17.5 11.67C15.79 11.67 14.29 11.93 13 12.46V14.15C14.11 13.5 15.6 13.16 17.5 13.16C18.54 13.16 19.38 13.24 20 13.4V11.9C19.13 11.74 18.29 11.67 17.5 11.67M20 14.57C19.13 14.41 18.29 14.33 17.5 14.33C15.67 14.33 14.17 14.6 13 15.13V16.82C14.11 16.16 15.6 15.83 17.5 15.83C18.54 15.83 19.38 15.91 20 16.07V14.57Z"/>`, // book-open-variant
	"chapter":     `<path d="M19 2L14 6.5V17.5L19 13V2M6.5 5C4.55 5 2.45 5.4 1 6.5V21.16C1 21.41 1.25 21.66 1.5 21.66C1.6 21.66 1.65 21.59 1.75 21.59C3.1 20.94 5.05 20.5 6.5 20.5C8.45 20.5 10.55 20.9 12 22C13.35 21.15 15.8 20.5 17.5 20.5C19.15 20.5 20.85 20.81 22.25 21.56C22.35 21.61 22.4 21.59 22.5 21.59C22.75 21.59 23 21.34 23 21.09V6.5C22.4 6.05 21.75 5.75 21 5.5V19C19.9 18.65 18.7 18.5 17.5 18.5C15.8 18.5 13.35 19.15 12 20V6.5C10.55 5.4 8.45 5 6.5 5Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                       // book-open-page-variant
	"sword-cross": `<path d="M6.2,2.44L18.1,14.34L20.22,12.22L21.63,13.63L19.16,16.1L22.34,19.28C22.73,19.67 22.73,20.3 22.34,20.69L21.63,21.4C21.24,21.79 20.61,21.79 20.22,21.4L17,18.23L14.56,20.7L13.15,19.29L15.27,17.17L3.37,5.27V2.44H6.2M15.89,10L20.63,5.26V2.44H17.8L13.06,7.18L15.89,10M10.94,15L8.11,12.13L5.9,14.34L3.78,12.22L2.37,13.63L4.84,16.1L1.66,19.29C1.27,19.68 1.27,20.31 1.66,20.7L2.37,21.41C2.76,21.8 3.39,21.8 3.78,21.41L7,18.23L9.44,20.7L10.85,19.29L8.73,17.17L10.94,15Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         // sword-cross
	"paw":         `<path d="M8.35,3C9.53,2.83 10.78,4.12 11.14,5.9C11.5,7.67 10.85,9.25 9.67,9.43C8.5,9.61 7.24,8.32 6.87,6.54C6.5,4.77 7.17,3.19 8.35,3M15.5,3C16.69,3.19 17.35,4.77 17,6.54C16.62,8.32 15.37,9.61 14.19,9.43C13,9.25 12.35,7.67 12.72,5.9C13.08,4.12 14.33,2.83 15.5,3M3,7.6C4.14,7.11 5.69,8 6.5,9.55C7.26,11.13 7,12.79 5.87,13.28C4.74,13.77 3.2,12.89 2.41,11.32C1.62,9.75 1.9,8.08 3,7.6M21,7.6C22.1,8.08 22.38,9.75 21.59,11.32C20.8,12.89 19.26,13.77 18.13,13.28C17,12.79 16.74,11.13 17.5,9.55C18.31,8 19.86,7.11 21,7.6M19.33,18.38C19.37,19.32 18.65,20.36 17.79,20.75C16,21.57 13.88,19.87 11.89,19.87C9.9,19.87 7.76,21.64 6,20.75C5,20.26 4.31,18.96 4.44,17.88C4.62,16.39 6.41,15.59 7.47,14.5C8.88,13.09 9.88,10.44 11.89,10.44C13.89,10.44 14.95,13.05 16.3,14.5C17.41,15.72 19.26,16.75 19.33,18.38Z"/>`,                                                                                                                                                                                        // paw
	"skull":       `<path d="M12,2A9,9 0 0,0 3,11C3,14.03 4.53,16.82 7,18.47V22H9V19H11V22H13V19H15V22H17V18.46C19.47,16.81 21,14 21,11A9,9 0 0,0 12,2M8,11A2,2 0 0,1 10,13A2,2 0 0,1 8,15A2,2 0 0,1 6,13A2,2 0 0,1 8,11M16,11A2,2 0 0,1 18,13A2,2 0 0,1 16,15A2,2 0 0,1 14,13A2,2 0 0,1 16,11M12,14L13.5,17H10.5L12,14Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                         // skull
	"scroll":      `<path d="M17.8,20C17.4,21.2 16.3,22 15,22H5C3.3,22 2,20.7 2,19V18H5L14.2,18C14.6,19.2 15.7,20 17,20H17.8M19,2C20.7,2 22,3.3 22,5V6H20V5C20,4.4 19.6,4 19,4C18.4,4 18,4.4 18,5V18H17C16.4,18 16,17.6 16,17V16H5V5C5,3.3 6.3,2 8,2H19M8,6V8H15V6H8M8,10V12H14V10H8Z"/>`,                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                                            // script-text
	// rule = the rules/glossary tree's crest (rule/ landing folders + term cards).
	"rule": `<path d="M12 21.5C10.65 20.65 8.2 20 6.5 20C4.85 20 3.15 20.3 1.75 21.05C1.65 21.1 1.6 21.1 1.5 21.1C1.25 21.1 1 20.85 1 20.6V6C1.6 5.55 2.25 5.25 3 5C4.11 4.65 5.33 4.5 6.5 4.5C8.45 4.5 10.55 4.9 12 6C13.45 4.9 15.55 4.5 17.5 4.5C18.67 4.5 19.89 4.65 21 5C21.75 5.25 22.4 5.55 23 6V20.6C23 20.85 22.75 21.1 22.5 21.1C22.4 21.1 22.35 21.1 22.25 21.05C20.85 20.3 19.15 20 17.5 20C15.8 20 13.35 20.65 12 21.5M12 8V19.5C13.35 18.65 15.8 18 17.5 18C18.7 18 19.9 18.15 21 18.5V7C19.9 6.65 18.7 6.5 17.5 6.5C15.8 6.5 13.35 7.15 12 8Z"/>`, // book-open-variant
}
