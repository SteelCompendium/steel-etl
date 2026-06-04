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
// Wire-up: at the very top of buildIndexContent() in build.go, add:
//
//	if cards, ok := buildCardsContent(dir, dirName, files, subdirs); ok {
//		return cards
//	}

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// richCardTypes lists index directories rendered as stat-cards. Only flat type
// dirs (no subdirs) qualify — feature/ability are nested and excluded.
var richCardTypes = map[string]bool{
	"kit": true, "class": true, "ancestry": true, "career": true,
	"treasure": true, "perk": true, "title": true, "complication": true,
	"culture": true, "condition": true, "skill": true,
	"movement": true, "negotiation": true,
}

// typeIcon maps a type to its crest icon for the simple (name + blurb) types.
var typeIcon = map[string]string{
	"condition": "zap", "skill": "star", "movement": "move", "negotiation": "speech",
}

// buildCardsContent returns rich-card index markup for a supported flat type.
// ok=false → caller falls back to the default browse-index list.
func buildCardsContent(dir, dirName string, files, subdirs []string) (content string, ok bool) {
	if !richCardTypes[dirName] || len(files) == 0 || len(subdirs) > 0 {
		return "", false
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })

	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n<div class=\"sc-cards\">\n")
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
		sb.WriteString(cardFor(dirName, fm, body, f, name))
	}
	sb.WriteString("</div>\n")
	return sb.String(), true
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
		return card(file, icon, titleCase(t), name, blurbBlock(bodyBlurb(body, 96)))
	}
}

// ── per-type builders ───────────────────────────────────────────────────────

func kitCard(fm, body, file, name string) string {
	stam := bonusShort(parseFrontmatterField(fm, "stamina_bonus"))
	spd := orZero(parseFrontmatterField(fm, "speed_bonus"))
	stab := orZero(parseFrontmatterField(fm, "stability_bonus"))
	armor, weapon := parseEquip(parseFrontmatterField(fm, "equipment_text"))
	melee := strings.TrimSpace(parseFrontmatterField(fm, "melee_damage_bonus"))
	ranged := strings.TrimSpace(parseFrontmatterField(fm, "ranged_damage_bonus"))
	dmgVal, dmgLabel := melee, "Melee Dmg"
	if melee == "" && ranged != "" {
		dmgVal, dmgLabel = ranged, "Ranged Dmg"
	} else if melee == "" {
		dmgVal = "\u2014"
	}
	sigName, sigType, keywords := signatureFromBody(body)
	kind, icon := "Martial", "shield"
	switch {
	case strings.Contains(keywords, "Psionic"):
		kind, icon = "Psionic", "wand"
	case strings.Contains(keywords, "Magic"):
		kind, icon = "Magic", "wand"
	}

	inner := lineBlock("", html.EscapeString(armor)+" armor &middot; "+html.EscapeString(weapon)+" weapon")
	inner += statsBlock([][3]string{{stam, "Stamina", ""}, {spd, "Speed", ""}, {stab, "Stability", ""}, {dmgVal, dmgLabel, "is-dmg"}})
	if sigName != "" {
		inner += fmt.Sprintf("  <div class=\"sc-card__sig\"><span class=\"sc-card__dot\" data-type=\"%s\"></span>"+
			"<span class=\"sc-card__sig-label\">Signature</span>"+
			"<span class=\"sc-card__sig-name\">%s</span></div>\n", sigType, html.EscapeString(sigName))
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
	if inner == "" {
		inner = blurbBlock(bodyBlurb(body, 96))
	}
	return card(file, "shield", "Class", name, inner)
}

func ancestryCard(fm, body, file, name string) string {
	if t := parseFrontmatterField(fm, "signature_trait_name"); t != "" {
		return card(file, "users", "Ancestry", name, lineBlock("Signature Trait", "<span class=\"hl\">"+html.EscapeString(t)+"</span>"))
	}
	return card(file, "users", "Ancestry", name, blurbBlock(bodyBlurb(body, 96)))
}

func careerCard(fm, body, file, name string) string {
	var stats [][3]string
	if v := parseFrontmatterField(fm, "renown"); v != "" {
		stats = append(stats, [3]string{v, "Renown", ""})
	}
	if v := parseFrontmatterField(fm, "wealth"); v != "" {
		stats = append(stats, [3]string{v, "Wealth", ""})
	}
	if v := parseFrontmatterField(fm, "project_points"); v != "" {
		stats = append(stats, [3]string{v, "Project Pts", ""})
	}
	inner := statsBlock(stats)
	if v := parseFrontmatterField(fm, "perk"); v != "" {
		inner += lineBlock("Perk", html.EscapeString(v))
	}
	if inner == "" {
		inner = blurbBlock(bodyBlurb(body, 96))
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
	if v := parseFrontmatterField(fm, "prerequisites"); v != "" {
		inner += lineBlock("Prerequisite", html.EscapeString(v))
	}
	inner += blurbBlock(bodyBlurb(body, 90))
	return card(file, "sparkles", label, name, inner)
}

func titleCard(fm, body, file, name string) string {
	label := "Title"
	if e := parseFrontmatterField(fm, "echelon"); e != "" {
		label = "Echelon " + e
	}
	blurb := parseFrontmatterField(fm, "effect")
	if blurb == "" {
		if bs := parseFrontmatterList(fm, "benefits"); len(bs) > 0 {
			blurb = bs[0]
		}
	}
	if blurb == "" {
		blurb = bodyBlurb(body, 96)
	}
	return card(file, "crown", label, name, blurbBlock(truncate(stripMD(blurb), 96)))
}

func complicationCard(fm, body, file, name string) string {
	ben := parseFrontmatterField(fm, "benefit")
	draw := parseFrontmatterField(fm, "drawback")
	inner := "  <div class=\"sc-card__bd\">\n"
	if ben != "" {
		inner += "    <div class=\"is-benefit\"><b>Benefit</b>" + html.EscapeString(truncate(stripMD(ben), 96)) + "</div>\n"
	}
	if draw != "" {
		inner += "    <div class=\"is-drawback\"><b>Drawback</b>" + html.EscapeString(truncate(stripMD(draw), 96)) + "</div>\n"
	}
	inner += "  </div>\n"
	if ben == "" && draw == "" {
		inner = blurbBlock(bodyBlurb(body, 96))
	}
	return card(file, "alert", "Complication", name, inner)
}

func cultureCard(fm, body, file, name string) string {
	var tags []string
	for _, k := range []string{"environment", "organization", "upbringing"} {
		if v := parseFrontmatterField(fm, k); v != "" {
			tags = append(tags, v)
		}
	}
	inner := tagsBlock(tags)
	if inner == "" {
		inner = blurbBlock(bodyBlurb(body, 96))
	}
	return card(file, "map", "Culture", name, inner)
}

// ── shared builders ─────────────────────────────────────────────────────────

func card(file, icon, typeLabel, name, inner string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "<a class=\"sc-card sc-fil\" href=\"%s\">\n", html.EscapeString(file))
	fmt.Fprintf(&sb, "  <div class=\"sc-card__head\"><span class=\"sc-crest\"><span>%s</span></span>\n", crestSVG(icon))
	fmt.Fprintf(&sb, "    <div><div class=\"sc-card__type\">%s</div>\n", html.EscapeString(typeLabel))
	fmt.Fprintf(&sb, "    <div class=\"sc-card__name\">%s</div></div></div>\n", html.EscapeString(name))
	sb.WriteString(inner)
	sb.WriteString("</a>\n")
	return sb.String()
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
			if len(s[0]) > 4 {
				style = " style=\"font-size:.72rem\""
			}
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

var (
	reArmor  = regexp.MustCompile(`(?i)wear\s+(?:an?\s+)?([a-z]+)\s+armor`)
	reWeapon = regexp.MustCompile(`(?i)wield\s+(?:an?\s+)?([a-z]+)\s+weapon`)
)

func parseEquip(text string) (armor, weapon string) {
	armor, weapon = "\u2014", "\u2014"
	if m := reArmor.FindStringSubmatch(text); m != nil {
		armor = titleCase(strings.ToLower(m[1]))
	}
	if m := reWeapon.FindStringSubmatch(text); m != nil {
		weapon = titleCase(strings.ToLower(m[1]))
	}
	return armor, weapon
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
)

func signatureFromBody(body string) (name, sigType, keywords string) {
	loc := reSigSection.FindStringIndex(body)
	if loc == nil {
		return "", "", ""
	}
	rest := body[loc[1]:]
	if m := reHeading.FindStringSubmatch(rest); m != nil {
		name = strings.TrimSpace(m[1])
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

// bodyBlurb returns the first prose sentence/paragraph of a page body,
// stripped of markdown and truncated.
func bodyBlurb(body string, max int) string {
	for _, raw := range strings.Split(body, "\n") {
		t := strings.TrimSpace(raw)
		if t == "" || t == "---" || strings.HasPrefix(t, "#") ||
			strings.HasPrefix(t, "|") || strings.HasPrefix(t, ">") || strings.HasPrefix(t, "- ") {
			continue
		}
		if s := stripMD(t); s != "" {
			return truncate(s, max)
		}
	}
	return ""
}

func truncate(s string, max int) string {
	r := []rune(strings.TrimSpace(s))
	if len(r) <= max {
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
