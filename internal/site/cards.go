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

// typeIcon maps a type to its crest icon for the simple (name + blurb) types.
var typeIcon = map[string]string{
	"condition": "zap", "skill": "star", "movement": "move", "negotiation": "speech",
}

// buildCardsContent returns rich-card index markup for a supported flat type.
// ok=false → caller falls back to the default browse-index list.
func buildCardsContent(dir, dirName string, files, subdirs []string) (content string, ok bool) {
	cardType := dirName
	if !richCardTypes[dirName] {
		// Treasure leaves are nested (treasure/<tier>/<category>); render their
		// items as treasure cards even though the leaf dirName isn't "treasure".
		if len(subdirs) == 0 && len(files) > 0 && pathHasSegment(dir, "treasure") {
			cardType = "treasure"
		} else {
			return "", false
		}
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
		sb.WriteString(cardFor(cardType, fm, body, f, name))
	}
	sb.WriteString("</div>\n")
	return sb.String(), true
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

func cardFor(t, fm, body, file, name string) string {
	switch t {
	case "kit":
		return kitCard(fm, body, file, name)
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
		icon := typeIcon[t]
		if icon == "" {
			icon = "scroll"
		}
		max := 96
		if t == "skill" { // skills are short — give them room so they don't ellipsize
			max = 220
		}
		return card(file, icon, titleCase(t), name, blurbBlock(bodyBlurb(body, max)))
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
	kind, icon := "Martial", "shield"
	switch {
	case strings.Contains(keywords, "Psionic"):
		kind, icon = "Psionic", "wand"
	case strings.Contains(keywords, "Magic"):
		kind, icon = "Magic", "wand"
	}

	inner := ""
	// Equipment: show the raw source sentence verbatim — parsing it into
	// armor/weapon tokens lost nuance (e.g. "one or two light weapons").
	if equip := strings.TrimSpace(parseFrontmatterField(fm, "equipment_text")); equip != "" {
		inner += "  <div class=\"sc-card__equip\">" + html.EscapeString(equip) + "</div>\n"
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
	return card(file, icon, kind+" Kit", name, inner)
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
	return card(file, "shield", "Class", name, inner)
}

func ancestryCard(fm, body, file, name string) string {
	inner := ""
	if t := parseFrontmatterField(fm, "signature_trait_name"); t != "" {
		inner += lineBlock("Signature Trait", "<span class=\"hl\">"+html.EscapeString(t)+"</span>")
	}
	// First flavor paragraph, shown in full (no truncation).
	if f := firstProse(body); f != "" {
		inner += flavorDiv(f, 0)
	}
	if inner == "" {
		inner = blurbBlock(bodyBlurb(body, 96))
	}
	return card(file, "users", "Ancestry", name, inner)
}

func careerCard(fm, body, file, name string) string {
	inner := ""
	// First line of flavor (minus the "In defining your career…" boilerplate).
	if f := careerFlavor(body); f != "" {
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
	return card(file, "briefcase", "Career", name, inner)
}

func treasureCard(fm, body, file, name string) string {
	tt := titleCase(strings.ReplaceAll(parseFrontmatterField(fm, "treasure_type"), "-", " "))
	if tt == "" {
		tt = "Treasure"
	}
	var stats [][3]string
	if v := parseFrontmatterField(fm, "level"); v != "" {
		stats = append(stats, [3]string{v, "Level", ""})
	}
	if v := parseFrontmatterField(fm, "rarity"); v != "" {
		stats = append(stats, [3]string{v, "Rarity", ""})
	}
	inner := statsBlock(stats)
	if kw := parseFrontmatterList(fm, "keywords"); len(kw) > 0 {
		inner += tagsBlock(kw)
	}
	if eff := parseFrontmatterField(fm, "effect"); eff != "" {
		inner += blurbBlock(truncate(stripMD(eff), 90))
	} else if inner == "" {
		inner = blurbBlock(bodyBlurb(body, 96))
	}
	return card(file, "gem", tt, name, inner)
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
	return wideCard(file, "sparkles", label, name, inner)
}

func titleCard(fm, body, file, name string) string {
	label := "Title"
	if e := parseFrontmatterField(fm, "echelon"); e != "" {
		label = "Echelon " + e
	}
	inner := ""
	// First paragraph is flavor — shown in full (no truncation).
	if f := firstProse(body); f != "" {
		inner += flavorDiv(f, 0)
	}
	if v := firstField(fm, "prerequisites", "prerequisite"); v != "" {
		inner += "  <div class=\"sc-card__line\"><b>Prerequisite</b> <span class=\"hl\">" + html.EscapeString(v) + "</span></div>\n"
	}
	return card(file, "crown", label, name, inner)
}

func complicationCard(fm, body, file, name string) string {
	// Take the 1–2 line description/flavor that sits ABOVE the benefit/drawback.
	flavor := complicationFlavor(body)
	if flavor == "" {
		// Combined "Benefit and Drawback:" entries have no lead-in — fall back to
		// the benefit (or drawback) text so the card isn't empty.
		flavor = stripMD(firstField(fm, "benefit", "drawback"))
	}
	// Description/flavor shown in full (no truncation).
	inner := flavorDiv(flavor, 0)
	return wideCard(file, "alert", "Complication", name, inner)
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
	if f := firstProse(body); f != "" {
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
	return card(file, "map", "Culture", name, inner)
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

// statsBlock renders an N-column stat grid. Each entry is {value, label, extraClass}.
func statsBlock(stats [][3]string) string {
	if len(stats) == 0 {
		return ""
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "  <div class=\"sc-card__stats\" style=\"grid-template-columns:repeat(%d,1fr)\">\n", len(stats))
	for _, s := range stats {
		cls := "sc-card__stat"
		style := ""
		if s[2] != "" {
			cls += " " + s[2]
		}
		if len([]rune(s[0])) > 4 { // long values (e.g. +1/+1/+1) need a smaller face
			style = " style=\"font-size:.72rem\""
		}
		fmt.Fprintf(&sb, "    <div class=\"%s\"><div class=\"v\"%s>%s</div><div class=\"l\">%s</div></div>\n",
			cls, style, html.EscapeString(s[0]), html.EscapeString(s[1]))
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
	return `<svg viewBox="0 0 24 24" width="19" height="19" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round">` + p + `</svg>`
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
// intro flavor section.
var reBasicsHeading = regexp.MustCompile(`(?mi)^#{1,6}\s+Basics\s*$`)

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

var iconPaths = map[string]string{
	"shield":    `<path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/>`,
	"wand":      `<path d="m3 21 9-9"/><path d="M15 4V2"/><path d="M15 16v-2"/><path d="M8 9h2"/><path d="M20 9h2"/><path d="M17.8 11.8 19 13"/><path d="M15 9h.01"/><path d="M17.8 6.2 19 5"/><path d="M12.2 6.2 11 5"/>`,
	"users":     `<path d="M16 21v-2a4 4 0 0 0-4-4H6a4 4 0 0 0-4 4v2"/><circle cx="9" cy="7" r="4"/><path d="M22 21v-2a4 4 0 0 0-3-3.87"/><path d="M16 3.13a4 4 0 0 1 0 7.75"/>`,
	"briefcase": `<rect width="20" height="14" x="2" y="7" rx="2"/><path d="M16 21V5a2 2 0 0 0-2-2h-4a2 2 0 0 0-2 2v16"/>`,
	"gem":       `<path d="M6 3h12l4 6-10 13L2 9Z"/><path d="M11 3 8 9l4 13 4-13-3-6"/><path d="M2 9h20"/>`,
	"crown":     `<path d="M11.6 3.3a.5.5 0 0 1 .9 0l2.3 4.6a.5.5 0 0 0 .7.2l4.2-2.4a.5.5 0 0 1 .7.6l-2.3 9a1 1 0 0 1-1 .8H6.9a1 1 0 0 1-1-.8l-2.3-9a.5.5 0 0 1 .7-.6l4.2 2.4a.5.5 0 0 0 .7-.2Z"/><path d="M5 21h14"/>`,
	"alert":     `<path d="m21.73 18-8-14a2 2 0 0 0-3.48 0l-8 14A2 2 0 0 0 4 21h16a2 2 0 0 0 1.73-3Z"/><path d="M12 9v4"/><path d="M12 17h.01"/>`,
	"map":       `<path d="M14.1 4.2a2 2 0 0 0-1.3 0L8.4 5.6a2 2 0 0 1-1.3 0L4 4.5A1 1 0 0 0 3 5.4v12.8a1 1 0 0 0 .7.9l3.4 1.1a2 2 0 0 0 1.3 0l4.4-1.4a2 2 0 0 1 1.3 0l3.1 1.1a1 1 0 0 0 1.3-.9V6.4a1 1 0 0 0-.7-.9Z"/><path d="M9 5v15"/><path d="M15 4v15"/>`,
	"sparkles":  `<path d="M12 3l1.9 5.8a2 2 0 0 0 1.3 1.3L21 12l-5.8 1.9a2 2 0 0 0-1.3 1.3L12 21l-1.9-5.8a2 2 0 0 0-1.3-1.3L3 12l5.8-1.9a2 2 0 0 0 1.3-1.3Z"/>`,
	"star":      `<path d="M12 2l3.09 6.26L22 9.27l-5 4.87 1.18 6.88L12 17.77l-6.18 3.25L7 14.14 2 9.27l6.91-1.01L12 2z"/>`,
	"zap":       `<path d="M4 14a1 1 0 0 1-.78-1.63l9.9-10.2a.5.5 0 0 1 .86.46l-1.92 6.02A1 1 0 0 0 13 10h7a1 1 0 0 1 .78 1.63l-9.9 10.2a.5.5 0 0 1-.86-.46l1.92-6.02A1 1 0 0 0 11 14z"/>`,
	"move":      `<path d="M5 9l-3 3 3 3"/><path d="M9 5l3-3 3 3"/><path d="M15 19l-3 3-3-3"/><path d="M19 9l3 3-3 3"/><path d="M2 12h20"/><path d="M12 2v20"/>`,
	"speech":    `<path d="M7.9 20A9 9 0 1 0 4 16.1L2 22Z"/>`,
	"scroll":    `<path d="M19 17V5a2 2 0 0 0-2-2H4"/><path d="M8 21h12a2 2 0 0 0 2-2v-1a1 1 0 0 0-1-1H11a1 1 0 0 0-1 1v1a2 2 0 1 1-4 0V5a2 2 0 1 0-4 0v2a1 1 0 0 0 1 1h3"/>`,
}
