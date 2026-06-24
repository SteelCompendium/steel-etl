package site

import (
	"html"
	"strings"
)

// buildKitPage rewrites a `type: kit` page body into the unified High-Fantasy
// Steel `.sc-kit` plate. The signature-ability heading marker is preserved AFTER
// the (closed) plate so the embedItemCards post-pass transcludes its standalone
// `.sc-ability` card beneath it (steel-kit.css fuses the two into one card).
// Returns (newData, true) for kit pages; (data, false) otherwise. Frontmatter is
// preserved verbatim; injectH1 (next in buildSection) prepends the hidden "# Name".
//
// SITE-ONLY: runs against generated md-linked pages; the shared data repos are
// untouched.
func buildKitPage(data []byte) ([]byte, bool) {
	fm, body := splitFrontmatter(string(data))
	if parseFrontmatterField(fm, "type") != "kit" {
		return data, false
	}
	newBody := renderKitPlate(fm, body)
	if marker := kitSignatureMarker(body); marker != "" {
		// Blank line BEFORE the marker so MkDocs ends the raw-HTML plate block and
		// parses the heading; embed then splices the card here.
		newBody += "\n\n" + marker + "\n"
	}
	return []byte("---\n" + fm + "\n---\n\n" + newBody), true
}

// kitBonus renders a kit bonus value for a fixed grid slot: it strips a trailing
// "per echelon"-style qualifier (kept in the small label instead) and shows an em
// dash when the kit grants no such bonus, so every slot reads uniformly (the
// approved all-8 grid uses "—" for every absent bonus).
func kitBonus(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "—"
	}
	if i := strings.Index(strings.ToLower(s), " per "); i >= 0 {
		s = strings.TrimSpace(s[:i])
	}
	return s
}

// kitKind derives the kit's family label (Martial / Magic / Psionic) the same way
// the preview card (kitCard) does — from the signature ability's keyword line.
func kitKind(body string) string {
	_, _, keywords := signatureFromBody(body)
	switch {
	case strings.Contains(keywords, "Psionic"):
		return "Psionic"
	case strings.Contains(keywords, "Magic"):
		return "Magic"
	}
	return "Martial"
}

// kitSignatureMarker returns the kit body's signature-ability heading line — the
// sole {data-scc} heading on a kit page — verbatim (trailing whitespace trimmed),
// so it survives into the output for the embedItemCards post-pass to splice the
// `.sc-ability` card beneath it. Returns "" when the kit has no signature ability.
func kitSignatureMarker(body string) string {
	for _, line := range strings.Split(body, "\n") {
		if dataSCCHeadingRe.MatchString(line) {
			return strings.TrimRight(line, " \t")
		}
	}
	return ""
}

// renderKitPlate builds the contiguous (no blank-line) raw-HTML `.sc-kit` plate
// from frontmatter so md_in_html passes it through verbatim. The signature
// ability card is NOT rendered here — only its band head; the card is spliced
// beneath the (closed) plate by embedItemCards via the preserved {data-scc} marker.
func renderKitPlate(fm, body string) string {
	name := strings.TrimSpace(parseFrontmatterField(fm, "name"))

	var sb strings.Builder
	sb.WriteString(`<section class="sc-kit sc-fil">` + "\n")

	// Header — the shared 6-slot card header (never hand-rolled): backpack crest,
	// "<Kind> Kit" eyebrow, and the kit name as the primary. Class re-attaches the
	// kit plate's head separator chrome (steel-kit.css).
	sb.WriteString(renderCardHead(cardHeadSlots{
		Crest:       `<span class="sc-crest sc-kit__crest"><span>` + crestSVG("kit") + `</span></span>`,
		NameTag:     "h2",
		Class:       "sc-kit__head",
		LeftEyebrow: hLine(html.EscapeString(kitKind(body) + " Kit")),
		LeftPrimary: hLine(html.EscapeString(name)),
	}))
	sb.WriteString("\n")

	// Flavor — untruncated; links rendered to real anchors via inlineMD.
	if f := cardFlavor(fm, body); f != "" {
		sb.WriteString(`<div class="sc-kit__flavor">` + inlineMD(f) + `</div>` + "\n")
	}

	// Equipment band — verbatim sentence (links rendered). &nbsp; reserves the box
	// when a kit lacks equipment (matches kitCard).
	equip := strings.TrimSpace(parseFrontmatterField(fm, "equipment_text"))
	if equip == "" {
		equip = "&nbsp;"
	} else {
		equip = inlineMD(equip)
	}
	sb.WriteString(`<div class="sc-kit__band"><div class="sc-kit__band-head">Equipment</div>` + "\n")
	sb.WriteString(`<div class="sc-kit__equip">` + equip + `</div></div>` + "\n")

	// Kit Bonuses band — two rows of 4 fixed slots (mirrors kitCard exactly).
	stam := kitBonus(parseFrontmatterField(fm, "stamina_bonus"))
	spd := kitBonus(parseFrontmatterField(fm, "speed_bonus"))
	stab := kitBonus(parseFrontmatterField(fm, "stability_bonus"))
	dis := kitBonus(firstField(fm, "disengage_bonus", "disengage"))
	melee := kitBonus(parseFrontmatterField(fm, "melee_damage_bonus"))
	ranged := kitBonus(parseFrontmatterField(fm, "ranged_damage_bonus"))
	meleeDist := kitBonus(parseFrontmatterField(fm, "melee_distance_bonus"))
	rangedDist := kitBonus(parseFrontmatterField(fm, "ranged_distance_bonus"))
	sb.WriteString(`<div class="sc-kit__band"><div class="sc-kit__band-head">Kit Bonuses</div>` + "\n")
	sb.WriteString(statsBlock([][3]string{
		{stam, "Stamina per Echelon", ""}, {spd, "Speed", ""}, {stab, "Stability", ""}, {dis, "Disengage", ""},
	}))
	sb.WriteString(statsBlock([][3]string{
		{melee, "Melee Dmg", "is-dmg"}, {ranged, "Ranged Dmg", "is-dmg"},
		{meleeDist, "Melee Dist", ""}, {rangedDist, "Ranged Dist", ""},
	}))
	sb.WriteString(`</div>` + "\n")

	// Signature Ability band head — only when the body carries a signature ability;
	// the card itself is spliced beneath the plate by embedItemCards.
	if kitSignatureMarker(body) != "" {
		sb.WriteString(`<div class="sc-kit__band sc-kit__band--sig"><div class="sc-kit__band-head">Signature Ability</div></div>` + "\n")
	}

	sb.WriteString(`</section>` + "\n")
	return sb.String()
}
