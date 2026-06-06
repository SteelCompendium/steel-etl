package site

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// abilityLeaf is a minimal already-transformed ability leaf (frontmatter is what
// extractPreviewItem reads; body is the rendered card, ignored for abilities).
func abilityLeaf(name, action, cost string) string {
	fm := "---\nname: " + name + "\ntype: ability\nclass: censor\nlevel: \"1\"\n" +
		"action_type: " + action + "\ndistance: Ranged 10\ntarget: One enemy\n" +
		"flavor: A holy strike.\nkeywords:\n    - Magic\n    - Ranged\n"
	if cost != "" {
		fm += "cost: " + cost + "\n"
	}
	fm += "---\n\n<article class=\"sc-ability\"></article>\n"
	return fm
}

// traitLeaf is a trait leaf whose body is the rendered .sc-trait card carrying a
// lead-in and (optionally) a nested ability plate.
func traitLeaf(name string, withAbility bool) string {
	body := "<section class=\"sc-trait\" data-action=\"trait\">\n" +
		"<p class=\"sc-trait__leadin\"><span class=\"sc-trait__dia\"></span>You pass judgment on a foe.</p>\n"
	if withAbility {
		body += "<div class=\"sc-trait__nest\">\n" +
			"<article class=\"sc-ability sc-fil\" data-action=\"maneuver\">\n" +
			"<div class=\"sc-ability__eyebrow\"><span class=\"sc-ability__dia\"></span>Maneuver</div>\n" +
			"<h3 class=\"sc-ability__name\">Judgment</h3>\n</article>\n</div>\n"
	}
	body += "</section>\n"
	return "---\nname: " + name + "\ntype: trait\nclass: censor\nlevel: \"1\"\n---\n\n" + body
}

func TestExtractPreviewItem_Ability(t *testing.T) {
	// No cost + (implicitly) signature subtype is handled elsewhere; here cost set.
	fm, body := splitFrontmatter(abilityLeaf("Judgment", "Maneuver", ""))
	// signature via subtype
	fm += "\nsubtype: signature"
	it := extractPreviewItem(fm, body, "ability", "Censor")

	if it.Kind != "ability" {
		t.Fatalf("kind=%q want ability", it.Kind)
	}
	if it.Action != "maneuver" {
		t.Errorf("action=%q want maneuver", it.Action)
	}
	if it.Cost != "Signature" {
		t.Errorf("cost=%q want Signature (from subtype)", it.Cost)
	}
	if it.Distance != "Ranged 10" || it.Targets != "One enemy" {
		t.Errorf("distance/targets = %q / %q", it.Distance, it.Targets)
	}
	if len(it.Keywords) != 2 || it.Keywords[0] != "Magic" {
		t.Errorf("keywords=%v", it.Keywords)
	}

	it.Href = "judgment/"
	html := renderPrevCard(it, false)
	for _, want := range []string{
		`class="sc-prev sc-prev--ability sc-fil"`,
		`data-action="maneuver"`,
		`href="judgment/"`,
		`<span class="sc-prev__glyph">f</span>`,
		`>Maneuver</div>`,
		`<h3 class="sc-prev__name">Judgment</h3>`,
		`<div class="sc-prev__tag">Signature</div>`,
		`<span class="sc-prev__chip">Magic</span>`,
		`Ranged <b>10</b>`,
		`One enemy`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("ability preview missing %q\n%s", want, html)
		}
	}
	// context=false must not emit the "from …" meta.
	if strings.Contains(html, `>from<`) {
		t.Error("context=false card should omit the from-meta")
	}
}

func TestExtractPreviewItem_AbilityNumericCost(t *testing.T) {
	fm, body := splitFrontmatter(abilityLeaf("Censure", "Triggered", "3 Wrath"))
	it := extractPreviewItem(fm, body, "ability", "Censor")
	it.Href = "censure/"
	html := renderPrevCard(it, false)
	if !strings.Contains(html, `<span class="num">3</span> Wrath`) {
		t.Errorf("numeric cost not split into mono span:\n%s", html)
	}
	if !strings.Contains(html, `data-action="triggered"`) {
		t.Errorf("triggered action key missing:\n%s", html)
	}
}

func TestExtractPreviewItem_Trait(t *testing.T) {
	fm, body := splitFrontmatter(traitLeaf("Judgment", true))
	it := extractPreviewItem(fm, body, "trait", "Censor")

	if it.Action != "maneuver" {
		t.Errorf("trait action=%q want maneuver (from nested ability)", it.Action)
	}
	if it.Grants != "the Judgment maneuver" {
		t.Errorf("grants=%q want 'the Judgment maneuver'", it.Grants)
	}
	if it.Flavor != "You pass judgment on a foe." {
		t.Errorf("flavor=%q", it.Flavor)
	}

	it.Href = "judgment/"
	html := renderPrevCard(it, false)
	for _, want := range []string{
		`class="sc-prev sc-prev--trait sc-fil"`,
		`data-action="maneuver"`,
		`<div class="sc-prev__eyebrow"><span class="sc-prev__dia"></span>Censor</div>`,
		`<h3 class="sc-prev__name">Judgment</h3>`,
		`<div class="sc-prev__tag">Level <span class="num">1</span></div>`,
		`Grants the Judgment maneuver`,
		`You pass judgment on a foe.`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("trait preview missing %q\n%s", want, html)
		}
	}
}

func TestExtractPreviewItem_TraitNoAbility(t *testing.T) {
	fm, body := splitFrontmatter(traitLeaf("Inner Light", false))
	it := extractPreviewItem(fm, body, "trait", "Censor")
	if it.Action != "trait" {
		t.Errorf("action=%q want trait", it.Action)
	}
	if it.Grants != "" {
		t.Errorf("grants=%q want empty", it.Grants)
	}
	html := renderPrevCard(it, false)
	if strings.Contains(html, "sc-prev__grant") {
		t.Error("trait without nested ability should have no grant marker")
	}
}

func TestBuildFeatureIndex_FolderCards(t *testing.T) {
	root := t.TempDir()
	// feature/trait/<class>/<level>/ tree: trait/ is the index-of-indexes node.
	traitDir := filepath.Join(root, "feature", "trait")
	for _, cls := range []string{"censor", "conduit"} {
		writeFile(t, filepath.Join(traitDir, cls, "level-1", "a.md"), "---\nname: A\ntype: trait\n---\n")
		writeFile(t, filepath.Join(traitDir, cls, "level-1", "b.md"), "---\nname: B\ntype: trait\n---\n")
	}

	content, ok := buildFeatureIndexContent(traitDir, "trait", nil, []string{"censor", "conduit"})
	if !ok {
		t.Fatal("expected folder-card index for trait/")
	}
	for _, want := range []string{
		`# Traits`,
		`<div class="sc-folders sc-folders--lg">`, // 2 nodes → large
		`<a class="sc-folder" href="censor/">`,
		`<h3 class="sc-folder__name">Censor</h3>`,
		`<span class="sc-folder__count">2</span>`, // 2 leaves beneath censor
		`<span class="sc-folder__chev">`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("folder index missing %q\n%s", want, content)
		}
	}
}

func TestBuildFeatureIndex_PreviewCards(t *testing.T) {
	root := t.TempDir()
	lvlDir := filepath.Join(root, "feature", "ability", "censor", "level-1")
	writeFile(t, filepath.Join(lvlDir, "judgment.md"), abilityLeaf("Judgment", "Maneuver", "Signature"))
	writeFile(t, filepath.Join(lvlDir, "censure.md"), abilityLeaf("Censure", "Triggered", "3 Wrath"))

	content, ok := buildFeatureIndexContent(lvlDir, "level-1", []string{"censure.md", "judgment.md"}, nil)
	if !ok {
		t.Fatal("expected preview-card index for ability level dir")
	}
	for _, want := range []string{
		`# Censor — Level 1`,
		`<div class="sc-prevs">`,
		`class="sc-prev sc-prev--ability sc-fil"`,
		`href="judgment/"`,
		`href="censure/"`,
	} {
		if !strings.Contains(content, want) {
			t.Errorf("preview index missing %q\n%s", want, content)
		}
	}
}

func TestBuildFeatureIndex_SearchIslandOnLanding(t *testing.T) {
	root := t.TempDir()
	featureDir := filepath.Join(root, "feature")
	writeFile(t, filepath.Join(featureDir, "trait", "censor", "level-1", "judgment.md"), traitLeaf("Judgment", true))
	writeFile(t, filepath.Join(featureDir, "ability", "censor", "level-1", "judgment.md"), abilityLeaf("Judgment", "Maneuver", "Signature"))

	content, ok := buildFeatureIndexContent(featureDir, "feature", nil, []string{"ability", "trait"})
	if !ok {
		t.Fatal("expected folder index for feature/")
	}
	if !strings.Contains(content, `## Search & Filter`) {
		t.Error("feature landing missing Search & Filter section")
	}
	if !strings.Contains(content, `<div class="sc-browse-mount">`) ||
		!strings.Contains(content, `<script type="application/json" class="sc-browse-data">`) {
		t.Error("feature landing missing browse data island")
	}

	// The island must be valid JSON with the expected item shape + dir-URL hrefs.
	start := strings.Index(content, `class="sc-browse-data">`)
	rest := content[start:]
	open := strings.IndexByte(rest, '\n') + 1
	end := strings.Index(rest, "</script>")
	var items []browseItem
	if err := json.Unmarshal([]byte(strings.TrimSpace(rest[open:end])), &items); err != nil {
		t.Fatalf("island JSON invalid: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	var sawAbilityHref, sawDistanceBold bool
	for _, it := range items {
		if it.Href == "ability/censor/level-1/judgment/" {
			sawAbilityHref = true
			if it.Distance != "Ranged **10**" {
				t.Errorf("island distance=%q want bolded markdown", it.Distance)
			}
			sawDistanceBold = true
		}
	}
	if !sawAbilityHref {
		t.Errorf("island missing ability dir-URL href; items=%+v", items)
	}
	if !sawDistanceBold {
		t.Error("ability item distance not converted to **markdown**")
	}
}

func TestUnderFeatureOrTreasure(t *testing.T) {
	cases := map[string]bool{
		"/x/Browse/feature":              true,
		"/x/Browse/feature/trait/censor": true,
		"/x/Browse/treasure/1st-echelon": true,
		"/x/Browse/class":                false,
		"/x/Browse/kit":                  false,
	}
	for in, want := range cases {
		if got := underFeatureOrTreasure(in); got != want {
			t.Errorf("underFeatureOrTreasure(%q)=%v want %v", in, got, want)
		}
	}
}

func TestFeatureKind(t *testing.T) {
	cases := map[string]string{
		"/x/Browse/feature/trait/censor/level-1":    "trait",
		"/x/Browse/feature/ability/censor/level-1":  "ability",
		"/x/Browse/feature/ability/Kits":            "ability",
		"/x/Browse/treasure/1st-echelon/consumable": "",
		"/x/Browse/class":                           "",
	}
	for in, want := range cases {
		if got := featureKind(in); got != want {
			t.Errorf("featureKind(%q)=%q want %q", in, got, want)
		}
	}
}

// non-feature parent-of-leaves (e.g. a treasure leaf category) must NOT become
// preview cards — it falls through to the default list.
func TestBuildFeatureIndex_TreasureLeafFallsThrough(t *testing.T) {
	root := t.TempDir()
	dir := filepath.Join(root, "treasure", "1st-echelon", "consumable")
	writeFile(t, filepath.Join(dir, "potion.md"), "---\nname: Potion\ntype: treasure\n---\n")
	if _, ok := buildFeatureIndexContent(dir, "consumable", []string{"potion.md"}, nil); ok {
		t.Error("treasure leaf dir should fall through (no preview cards)")
	}
	_ = os.Stat // keep os import if unused elsewhere
}
