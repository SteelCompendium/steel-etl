package site

import (
	"strings"
	"testing"
)

// A representative kit body as the md-linked page carries it (pre-embed): a
// flavor paragraph, ## Equipment / ## Kit Bonuses sections, and a Signature
// Ability section whose ability heading carries the {data-scc} marker stamped by
// RenderSubtree. signatureFromBody reads the keyword table row for kind detection.
const kitTestBody = `The Ranger kit outfits you with medium armor and weapons.

## Equipment

You wear medium armor and wield a bow and a medium weapon.

## Kit Bonuses

**Stamina Bonus:** +6 per echelon

**Speed Bonus:** +1

## Signature Ability

### Hamstring Shot {data-scc="mcdm.heroes.v1/feature.ability.ranger/hamstring-shot"}

| **Ranged, Strike, Weapon** | **Main action** |
|----------------------------|-----------------|
`

const kitTestFM = "equipment_text: You wear medium armor and wield a bow and a medium weapon.\n" +
	"flavor: The Ranger kit outfits you with medium armor and weapons.\n" +
	"melee_damage_bonus: +1/+1/+1\n" +
	"name: Ranger\n" +
	"ranged_distance_bonus: \"+5\"\n" +
	"speed_bonus: \"+1\"\n" +
	"stamina_bonus: +6 per echelon\n" +
	"type: kit"

func TestRenderKitPlate(t *testing.T) {
	got := renderKitPlate(kitTestFM, kitTestBody)
	wants := []string{
		`<section class="sc-kit sc-fil">`,
		`<div class="sc-kit__eyebrow">Martial Kit</div>`, // ranged/strike/weapon → not psionic/magic
		`<div class="sc-kit__name">Ranger</div>`,
		`sc-kit__crest`, // backpack crest
		`The Ranger kit outfits you with medium armor`, // flavor, untruncated
		`<div class="sc-kit__band-head">Equipment</div>`,
		`You wear medium armor and wield a bow and a medium weapon.`,
		`<div class="sc-kit__band-head">Kit Bonuses</div>`,
		`<div class="l">Stamina per Echelon</div>`,
		`<div class="l">Stability</div>`, // absent bonus still gets a fixed slot
		`<div class="l">Ranged Dist</div>`,
		`<div class="sc-kit__band-head">Signature Ability</div>`, // sig band head present (body has a marker)
		`</section>`,
	}
	for _, w := range wants {
		if !strings.Contains(got, w) {
			t.Errorf("plate missing %q\n--- got ---\n%s", w, got)
		}
	}
	// The +6-per-echelon stamina bonus is shortened to its value (per kitCard).
	if !strings.Contains(got, `+6`) || strings.Contains(got, `+6 per echelon`) {
		t.Errorf("stamina bonus should be shortened to its value\n%s", got)
	}
	// Every absent bonus renders as an em dash in its fixed slot (the approved
	// all-8 grid). Here stability and melee-distance are absent.
	if !strings.Contains(got, `<div class="v">—</div><div class="l">Stability</div>`) {
		t.Errorf("absent Stability bonus should render as an em dash\n%s", got)
	}
	if !strings.Contains(got, `<div class="v">—</div><div class="l">Melee Dist</div>`) {
		t.Errorf("absent Melee Dist bonus should render as an em dash\n%s", got)
	}
	// Contiguity: md_in_html requires no blank lines in the raw-HTML plate.
	if strings.Contains(got, "\n\n") {
		t.Errorf("plate must be a contiguous block (no blank lines) for md_in_html\n%s", got)
	}
}

func TestBuildKitPage_NonKitUnchanged(t *testing.T) {
	in := []byte("---\ntype: ability\nname: Dragon Breath\n---\n\nbody\n")
	out, ok := buildKitPage(in)
	if ok {
		t.Fatalf("expected ok=false for type: ability")
	}
	if string(out) != string(in) {
		t.Fatalf("non-kit data should pass through unchanged")
	}
}

func TestBuildKitPage_PreservesMarkerAfterPlate(t *testing.T) {
	in := []byte("---\n" + kitTestFM + "\n---\n\n" + kitTestBody + "\n")
	out, ok := buildKitPage(in)
	if !ok {
		t.Fatalf("expected ok=true for type: kit")
	}
	s := string(out)
	marker := `### Hamstring Shot {data-scc="mcdm.heroes.v1/feature.ability.ranger/hamstring-shot"}`
	if !strings.Contains(s, marker) {
		t.Fatalf("output must preserve the signature-ability {data-scc} marker for embed\n%s", s)
	}
	// The marker MUST come AFTER the closed plate, else embedItemCards' swallow
	// (to the next heading / EOF) would eat the </section> close.
	if strings.Index(s, "</section>") > strings.Index(s, marker) {
		t.Fatalf("marker must follow </section> so embed cannot swallow the plate close\n%s", s)
	}
	// Frontmatter is preserved verbatim.
	if !strings.HasPrefix(s, "---\n"+kitTestFM+"\n---\n\n") {
		t.Fatalf("frontmatter must be preserved verbatim\n%s", s)
	}
	// The original ability table markdown is dropped (embed supplies the card).
	if strings.Contains(s, "Ranged, Strike, Weapon") {
		t.Errorf("original ability markdown should be dropped; embed supplies the card\n%s", s)
	}
}

func TestBuildKitPage_NoSignatureAbility(t *testing.T) {
	body := "A simple kit.\n\n## Equipment\n\nA dagger.\n"
	fm := "equipment_text: A dagger.\nname: Simple\ntype: kit"
	out, ok := buildKitPage([]byte("---\n" + fm + "\n---\n\n" + body))
	if !ok {
		t.Fatalf("expected ok=true")
	}
	s := string(out)
	if strings.Contains(s, "Signature Ability") {
		t.Errorf("kit without a signature ability must omit the Signature Ability band\n%s", s)
	}
	if !strings.Contains(s, `<div class="sc-kit__band-head">Equipment</div>`) {
		t.Errorf("plate should still render without a signature ability\n%s", s)
	}
}

func TestKitKind(t *testing.T) {
	// kitKind reads keywords via signatureFromBody, which requires the kit's
	// "Signature Ability" section header (always present on real kit pages).
	sig := func(kw string) string {
		return "## Signature Ability\n\n### A\n\n| **" + kw + "** | x |\n"
	}
	cases := map[string]string{
		sig("Psionic, Strike"): "Psionic",
		sig("Magic, Ranged"):   "Magic",
		sig("Weapon, Melee"):   "Martial",
		"":                     "Martial", // no signature ability → default Martial
	}
	for body, want := range cases {
		if got := kitKind(body); got != want {
			t.Errorf("kitKind(%q) = %q, want %q", body, got, want)
		}
	}
}
