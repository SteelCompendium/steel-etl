package site

// Rich category-index cards for the Steel Compendium MkDocs site.
//
// Replaces the flat ".browse-index" link list with ".sc-card" stat-cards for
// the index types listed in richCardTypes (starting with "kit"). All data is
// read from frontmatter that the content parser already emits, plus the page
// body for the signature ability — so NO change to the shared data repos is
// required. Styled by docs/stylesheets/steel-redesign.css.
//
// Wire-up: add the following 3 lines at the very top of buildIndexContent()
// (in build.go), before `title := dirToTitle(dirName)`:
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

// richCardTypes lists the index directories rendered as stat-cards instead of
// the default link list. Extend (alphabetically: ancestry, career, …) as each
// type's card is designed.
var richCardTypes = map[string]bool{
	"kit": true,
}

// buildCardsContent returns rich-card index markup for a supported type.
// ok=false → caller falls back to the default browse-index list.
func buildCardsContent(dir, dirName string, files, subdirs []string) (content string, ok bool) {
	if !richCardTypes[dirName] || len(files) == 0 || len(subdirs) > 0 {
		return "", false
	}
	sort.Slice(files, func(i, j int) bool { return naturalLess(files[i], files[j]) })

	var sb strings.Builder
	sb.WriteString("# " + dirToTitle(dirName) + "\n\n---\n\n")
	sb.WriteString("<div class=\"sc-cards\">\n")
	for _, f := range files {
		switch dirName {
		case "kit":
			sb.WriteString(kitCard(dir, f))
		}
	}
	sb.WriteString("</div>\n")
	return sb.String(), true
}

// ── Kit card ────────────────────────────────────────────────────────────────

func kitCard(dir, file string) string {
	data, err := os.ReadFile(filepath.Join(dir, file))
	if err != nil {
		return ""
	}
	fm, body := splitFrontmatter(string(data))

	name := parseFrontmatterField(fm, "name")
	if name == "" {
		name = fileToTitle(file)
	}
	stam := bonusShort(parseFrontmatterField(fm, "stamina_bonus"))
	spd := orZero(parseFrontmatterField(fm, "speed_bonus"))
	stab := orZero(parseFrontmatterField(fm, "stability_bonus"))
	armor, weapon := parseEquip(parseFrontmatterField(fm, "equipment_text"))

	// Damage box: prefer melee; fall back to ranged when a kit is ranged-only.
	melee := strings.TrimSpace(parseFrontmatterField(fm, "melee_damage_bonus"))
	ranged := strings.TrimSpace(parseFrontmatterField(fm, "ranged_damage_bonus"))
	dmgVal, dmgLabel := melee, "Melee Dmg"
	if melee == "" && ranged != "" {
		dmgVal, dmgLabel = ranged, "Ranged Dmg"
	} else if melee == "" {
		dmgVal = "\u2014"
	}

	sigName, sigType, keywords := signatureFromBody(body)

	// Crest + type label derived from the signature keywords (data-driven).
	kind, icon := "Martial", crestSVGShield
	switch {
	case strings.Contains(keywords, "Psionic"):
		kind, icon = "Psionic", crestSVGWand
	case strings.Contains(keywords, "Magic"):
		kind, icon = "Magic", crestSVGWand
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "<a class=\"sc-card sc-fil\" href=\"%s\">\n", html.EscapeString(file))
	fmt.Fprintf(&sb, "  <div class=\"sc-card__head\"><span class=\"sc-crest\"><span>%s</span></span>\n", icon)
	fmt.Fprintf(&sb, "    <div><div class=\"sc-card__type\">%s Kit</div>\n", html.EscapeString(kind))
	fmt.Fprintf(&sb, "    <div class=\"sc-card__name\">%s</div></div></div>\n", html.EscapeString(name))
	fmt.Fprintf(&sb, "  <div class=\"sc-card__equip\">%s armor &middot; %s weapon</div>\n",
		html.EscapeString(armor), html.EscapeString(weapon))
	sb.WriteString("  <div class=\"sc-card__stats\">\n")
	statBox(&sb, stam, "Stamina", false)
	statBox(&sb, spd, "Speed", false)
	statBox(&sb, stab, "Stability", false)
	statBox(&sb, dmgVal, dmgLabel, true)
	sb.WriteString("  </div>\n")
	if sigName != "" {
		fmt.Fprintf(&sb, "  <div class=\"sc-card__sig\"><span class=\"sc-card__dot\" data-type=\"%s\"></span>"+
			"<span class=\"sc-card__sig-label\">Signature</span>"+
			"<span class=\"sc-card__sig-name\">%s</span></div>\n",
			sigType, html.EscapeString(sigName))
	}
	sb.WriteString("</a>\n")
	return sb.String()
}

func statBox(sb *strings.Builder, val, label string, dmg bool) {
	cls := "sc-card__stat"
	style := ""
	if dmg {
		cls += " is-dmg"
		if len(val) > 4 { // shrink the long "+2/+2/+2" triple to fit the box
			style = " style=\"font-size:.72rem\""
		}
	}
	fmt.Fprintf(sb, "    <div class=\"%s\"><div class=\"v\"%s>%s</div><div class=\"l\">%s</div></div>\n",
		cls, style, html.EscapeString(val), html.EscapeString(label))
}

// ── frontmatter helpers ─────────────────────────────────────────────────────

// bonusShort trims trailing descriptors like " per echelon" → "+3".
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

// parseEquip pulls "Light"/"Medium" out of "You wear light armor and wield a
// medium weapon."; unknown parts become an em dash.
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

var (
	reSigSection = regexp.MustCompile(`(?mi)^#{1,6}\s+Signature Ability\s*$`)
	reHeading    = regexp.MustCompile(`(?m)^#{1,6}\s+(.+?)\s*$`)
)

// signatureFromBody extracts the signature ability's name, a coarse type (for
// the dot color), and its keyword string from a kit page body. Returns empty
// strings when no signature ability is present (the card omits the row).
func signatureFromBody(body string) (name, sigType, keywords string) {
	loc := reSigSection.FindStringIndex(body)
	if loc == nil {
		return "", "", ""
	}
	rest := body[loc[1]:]

	if m := reHeading.FindStringSubmatch(rest); m != nil {
		name = strings.TrimSpace(m[1])
	}
	// First markdown table row of bold keywords, e.g.
	//   | **Melee, Psionic, Strike, Weapon** | **Main action** |
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

// ── crest icons (inline SVG so generated HTML needs no emoji processing) ──────

const crestSVGShield = `<svg viewBox="0 0 24 24" width="19" height="19" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="M12 22s8-4 8-10V5l-8-3-8 3v7c0 6 8 10 8 10Z"/></svg>`

const crestSVGWand = `<svg viewBox="0 0 24 24" width="19" height="19" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round" stroke-linejoin="round"><path d="m3 21 9-9"/><path d="M15 4V2"/><path d="M15 16v-2"/><path d="M8 9h2"/><path d="M20 9h2"/><path d="M17.8 11.8 19 13"/><path d="M15 9h.01"/><path d="M17.8 6.2 19 5"/><path d="M12.2 6.2 11 5"/></svg>`
