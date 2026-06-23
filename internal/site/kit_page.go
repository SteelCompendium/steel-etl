package site

import (
	"html"
	"strings"
)

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

	// Header band — backpack crest + "<Kind> Kit" eyebrow + name.
	sb.WriteString(`<div class="sc-kit__head"><span class="sc-crest sc-kit__crest"><span>` +
		crestSVG("kit") + `</span></span>` + "\n")
	sb.WriteString(`<div class="sc-kit__titles"><div class="sc-kit__eyebrow">` +
		html.EscapeString(kitKind(body)+" Kit") + `</div>` + "\n")
	sb.WriteString(`<div class="sc-kit__name">` + html.EscapeString(name) + `</div></div></div>` + "\n")

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
	stam := bonusShort(parseFrontmatterField(fm, "stamina_bonus"))
	spd := orZero(parseFrontmatterField(fm, "speed_bonus"))
	stab := orZero(parseFrontmatterField(fm, "stability_bonus"))
	dis := orZero(firstField(fm, "disengage_bonus", "disengage"))
	melee := orDash(strings.TrimSpace(parseFrontmatterField(fm, "melee_damage_bonus")))
	ranged := orDash(strings.TrimSpace(parseFrontmatterField(fm, "ranged_damage_bonus")))
	meleeDist := orDash(strings.TrimSpace(parseFrontmatterField(fm, "melee_distance_bonus")))
	rangedDist := orDash(strings.TrimSpace(parseFrontmatterField(fm, "ranged_distance_bonus")))
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
