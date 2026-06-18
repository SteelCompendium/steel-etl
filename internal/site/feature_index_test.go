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

// traitLeaf is a trait leaf whose body is the rendered .sc-trait card. attrs are
// the data-* markers trait_cards.go stamps on the root (data-sub / data-grant).
func traitLeaf(name, attrs string) string {
	body := "<section class=\"sc-trait sc-trait--crest\" data-action=\"trait\"" + attrs + ">\n" +
		"<header class=\"sc-trait__head\"></header>\n" +
		"<div class=\"sc-trait__body\">\n" +
		"<p class=\"sc-trait__leadin\"><span class=\"sc-trait__dia\"></span>You pass judgment on a foe.</p>\n" +
		"</div>\n</section>\n"
	return "---\nname: " + name + "\ntype: trait\nclass: censor\nlevel: \"1\"\n---\n\n" + body
}

// A plain feature (type: feature) preview keeps its kind as "feature" (not coerced
// to the dir-derived "trait") so the eyebrow reads "<Source> Feature" and the
// Search & Filter Type facet can offer Feature as its own bucket.
func TestExtractPreviewItem_PlainFeatureKindAndEyebrow(t *testing.T) {
	leaf := "---\nname: Summoner Strike\ntype: feature\nclass: summoner\nlevel: \"1\"\n---\n\n" +
		"<section class=\"sc-trait sc-trait--crest\" data-action=\"trait\">\n" +
		"<p class=\"sc-trait__leadin\"><span class=\"sc-trait__dia\"></span>You have the following ability.</p>\n</section>\n"
	fm, body := splitFrontmatter(leaf)
	it := extractPreviewItem(fm, body, "trait", "Summoner")
	if it.Kind != "feature" {
		t.Errorf("plain feature kind=%q want feature (must not coerce to trait)", it.Kind)
	}
	it.Href = "summoner-strike/"
	html := renderPrevCard(it, false)
	if !strings.Contains(html, "Summoner Feature") {
		t.Errorf("plain feature preview eyebrow should read \"Summoner Feature\"\n%s", html)
	}
	if strings.Contains(html, "Summoner Trait") {
		t.Errorf("plain feature preview must not be labelled a Trait\n%s", html)
	}
}

func TestFeatureKind_PlainFeatureDir(t *testing.T) {
	// Plain features now live at feature/<class>/... with no kind segment; they
	// should be treated as the recessed niche ("trait") for preview cards.
	if got := featureKind("Browse/feature/elementalist/level-1"); got != "trait" {
		t.Errorf("featureKind(plain feature dir) = %q, want trait", got)
	}
	if got := featureKind("Browse/feature/ability/Kits"); got != "ability" {
		t.Errorf("featureKind(ability dir) = %q, want ability", got)
	}
	if got := featureKind("Browse/feature/trait/dwarf"); got != "trait" {
		t.Errorf("featureKind(ancestry trait dir) = %q, want trait", got)
	}
	if got := featureKind("Browse/treasure/artifact"); got != "" {
		t.Errorf("featureKind(non-feature dir) = %q, want empty", got)
	}
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

// Keywords/distance/flavor carry SCC cross-reference links in the data (which
// the site rewrites to ".md" relative links). A preview card is itself an <a>,
// so it can't nest keyword <a> links — the values must be reduced to plain
// display text (no literal "[Melee](…)" leaking through).
func TestExtractPreviewItem_LinkedKeywordsStripped(t *testing.T) {
	fm := "---\nname: Holy Strike\ntype: ability\nclass: censor\nlevel: \"1\"\n" +
		"action_type: Main action\n" +
		"distance: '[Melee](../../../../rule/combat/melee.md) 1'\n" +
		"target: One creature\n" +
		"flavor: You amplify the power of your [judgment](../level-1/judgment.md).\n" +
		"keywords:\n    - '[Melee](../../../../rule/combat/melee.md)'\n" +
		"    - '[Strike](../../../../rule/combat/strike.md)'\n    - Weapon\n---\n\n" +
		"<article class=\"sc-ability\"></article>\n"
	fmData, body := splitFrontmatter(fm)
	it := extractPreviewItem(fmData, body, "ability", "Censor")

	if got := strings.Join(it.Keywords, ","); got != "Melee,Strike,Weapon" {
		t.Errorf("keywords=%q want plain display text", got)
	}
	if it.Distance != "Melee 1" {
		t.Errorf("distance=%q want 'Melee 1'", it.Distance)
	}
	if strings.Contains(it.Flavor, "[judgment]") || strings.Contains(it.Flavor, "](") {
		t.Errorf("flavor leaked link markdown: %q", it.Flavor)
	}

	it.Href = "holy-strike/"
	html := renderPrevCard(it, false)
	if strings.Contains(html, "](") || strings.Contains(html, "[Melee") {
		t.Errorf("preview card leaked link markdown\n%s", html)
	}
	for _, want := range []string{
		`<span class="sc-prev__chip">Melee</span>`,
		`<span class="sc-prev__chip">Strike</span>`,
		`<span class="sc-prev__chip">Weapon</span>`,
		`Melee <b>1</b>`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("preview card missing %q\n%s", want, html)
		}
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

func TestExtractPreviewItem_Trait_SingleGrant(t *testing.T) {
	fm, body := splitFrontmatter(traitLeaf("Judgment", ` data-sub="1" data-grant="the Judgment maneuver"`))
	it := extractPreviewItem(fm, body, "trait", "Censor")

	if it.Action != "trait" {
		t.Errorf("trait action=%q want trait (unified accent)", it.Action)
	}
	if it.Grants != "the Judgment maneuver" {
		t.Errorf("grants=%q want 'the Judgment maneuver'", it.Grants)
	}
	if it.Source != "class" || it.Klass != "Censor" {
		t.Errorf("source/klass = %q / %q want class / Censor", it.Source, it.Klass)
	}
	if it.Flavor != "You pass judgment on a foe." {
		t.Errorf("flavor=%q", it.Flavor)
	}

	it.Href = "judgment/"
	html := renderPrevCard(it, false)
	for _, want := range []string{
		`class="sc-prev sc-prev--trait sc-fil"`,
		`data-action="trait"`,
		`<span class="sc-crest sc-prev__crest"><span class="sc-prev__glyph">*</span></span>`,
		`<div class="sc-prev__eyebrow"><span class="sc-prev__dia"></span>Censor Trait</div>`,
		`<h3 class="sc-prev__name">Judgment</h3>`,
		`<div class="sc-prev__tag">Level <span class="num">1</span></div>`,
		`Grants the Judgment maneuver`,
	} {
		if !strings.Contains(html, want) {
			t.Errorf("trait preview missing %q\n%s", want, html)
		}
	}
}

func TestExtractPreviewItem_Trait_OptionCount(t *testing.T) {
	fm, body := splitFrontmatter(traitLeaf("Censor Abilities", ` data-sub="12"`))
	it := extractPreviewItem(fm, body, "trait", "Censor")
	if it.Grants != "" || it.Options != 12 {
		t.Fatalf("grants/options = %q / %d want '' / 12", it.Grants, it.Options)
	}
	html := renderPrevCard(it, false)
	if !strings.Contains(html, "12 options") || strings.Contains(html, "abilities") {
		t.Errorf("option-count marker wrong:\n%s", html)
	}
}

func TestExtractPreviewItem_Trait_Subclass(t *testing.T) {
	leaf := "---\nname: Wrath\ntype: trait\nclass: censor\nsubclass: exorcist\nlevel: \"1\"\n---\n\n" +
		"<section class=\"sc-trait sc-trait--crest\" data-action=\"trait\"></section>\n"
	fm, body := splitFrontmatter(leaf)
	it := extractPreviewItem(fm, body, "trait", "Censor")
	if it.Subclass != "Exorcist" {
		t.Fatalf("subclass=%q want Exorcist", it.Subclass)
	}
	if !strings.Contains(renderPrevCard(it, false), "Censor Trait · Exorcist") {
		t.Errorf("eyebrow should append subclass:\n%s", renderPrevCard(it, false))
	}
}

func TestExtractPreviewItem_TraitNoSubfeatures(t *testing.T) {
	fm, body := splitFrontmatter(traitLeaf("Inner Light", ""))
	it := extractPreviewItem(fm, body, "trait", "Censor")
	if it.Action != "trait" || it.Grants != "" || it.Options != 0 {
		t.Errorf("action/grants/options = %q / %q / %d", it.Action, it.Grants, it.Options)
	}
	if strings.Contains(renderPrevCard(it, false), "sc-prev__grant") {
		t.Error("trait with no sub-features should have no foot marker")
	}
}

func TestSourceFromMeta(t *testing.T) {
	cases := []struct{ fm, src, klass string }{
		{"class: censor\n", "class", "Censor"},
		{"ancestry: dwarf\n", "ancestry", "Dwarf"},
		{"kit: arcane-archer\n", "kit", "Arcane Archer"},
		{"class: beastheart\ncompanion: basilisk\n", "class", "Beastheart"},
		{"name: x\n", "other", "fallback"},
	}
	for _, c := range cases {
		if got := sourceFromMeta(c.fm); got != c.src {
			t.Errorf("sourceFromMeta(%q)=%q want %q", c.fm, got, c.src)
		}
		if got := klassFromMeta(c.fm, "fallback"); got != c.klass {
			t.Errorf("klassFromMeta(%q)=%q want %q", c.fm, got, c.klass)
		}
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

// A hub-and-spoke companion level dir (feature/companion/<class>/<species>/level-N)
// has no literal "trait" path segment, so klassFromDir yields "" — the title must
// drop the dangling em-dash and read just "Level N", not " — Level N".
func TestBuildFeatureIndex_PreviewCards_CompanionLevelTitle(t *testing.T) {
	root := t.TempDir()
	lvlDir := filepath.Join(root, "feature", "companion", "beastheart", "wolf", "level-6")
	writeFile(t, filepath.Join(lvlDir, "call-of-the-wild.md"), traitLeaf("Call of the Wild", ""))

	content, ok := buildFeatureIndexContent(lvlDir, "level-6", []string{"call-of-the-wild.md"}, nil)
	if !ok {
		t.Fatal("expected preview-card index for companion level dir")
	}
	if !strings.Contains(content, "# Level 6\n") {
		t.Errorf("companion level title should be %q, got:\n%s", "# Level 6", content)
	}
	if strings.Contains(content, "— Level 6") {
		t.Errorf("companion level title has dangling em-dash:\n%s", content)
	}
}

func TestBuildFeatureIndex_SearchIslandOnLanding(t *testing.T) {
	root := t.TempDir()
	featureDir := filepath.Join(root, "feature")
	writeFile(t, filepath.Join(featureDir, "trait", "censor", "level-1", "judgment.md"), traitLeaf("Judgment", ` data-sub="1" data-grant="the Judgment maneuver"`))
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

func TestBuildFeatureIndex_RuleLanding(t *testing.T) {
	root := t.TempDir()
	// rule/<group>/<term>.md tree: rule/ is the index-of-indexes node whose
	// children are the topic-group folders.
	ruleDir := filepath.Join(root, "rule")
	writeFile(t, filepath.Join(ruleDir, "dice", "power-roll.md"), "---\nname: Power Rolls\ntype: rule\n---\n")
	writeFile(t, filepath.Join(ruleDir, "dice", "edge.md"), "---\nname: Edge\ntype: rule\n---\n")
	writeFile(t, filepath.Join(ruleDir, "combat", "flanking.md"), "---\nname: Flanking\ntype: rule\n---\n")

	content, ok := buildFeatureIndexContent(ruleDir, "rule", nil, []string{"combat", "dice"})
	if !ok {
		t.Fatal("expected folder-card index for rule/")
	}
	for _, want := range []string{
		`# Rule`,
		`<div class="sc-folders sc-folders--lg">`, // 2 groups → large
		`<a class="sc-folder" href="dice/">`,
		`<h3 class="sc-folder__name">Dice</h3>`,
		`<span class="sc-folder__count">2</span>`, // 2 terms beneath dice
		`<path d="M12 21.5`,                       // rule (book) crest on the folders
		"glossary entry",                          // the rule landing intro
	} {
		if !strings.Contains(content, want) {
			t.Errorf("rule folder index missing %q\n%s", want, content)
		}
	}
	// The cross-tree Search & Filter island is feature-only — rule must not get it.
	if strings.Contains(content, "Search & Filter") {
		t.Errorf("rule landing should not carry the feature Search & Filter island\n%s", content)
	}
}

func TestUsesFolderIndex(t *testing.T) {
	cases := map[string]bool{
		"/x/Browse/feature":              true,
		"/x/Browse/feature/trait/censor": true,
		"/x/Browse/treasure/1st-echelon": true,
		"/x/Browse/skill":                true,
		"/x/Browse/skill/crafting":       true,
		"/x/Browse/rule":                 true,
		"/x/Browse/rule/dice":            true,
		"/x/Browse/class":                false,
		"/x/Browse/kit":                  false,
	}
	for in, want := range cases {
		if got := usesFolderIndex(in); got != want {
			t.Errorf("usesFolderIndex(%q)=%v want %v", in, got, want)
		}
	}
}

func TestBuildFeatureIndexContentSkillRoot(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skill")
	craft := filepath.Join(skillRoot, "crafting")
	if err := os.MkdirAll(craft, 0755); err != nil {
		t.Fatal(err)
	}
	// self-named container + two skills → count should be 2, not 3
	for _, f := range []string{"crafting.md", "alchemy.md", "carpentry.md"} {
		os.WriteFile(filepath.Join(craft, f), []byte("---\nname: X\n---\n"), 0644)
	}
	content, ok := buildFeatureIndexContent(skillRoot, "skill", nil, []string{"crafting"})
	if !ok {
		t.Fatalf("buildFeatureIndexContent ok=false, want true for skill root")
	}
	if !strings.Contains(content, `<a class="sc-folder" href="crafting/">`) ||
		!strings.Contains(content, `<h3 class="sc-folder__name">Crafting</h3>`) {
		t.Errorf("expected a Crafting folder card; got:\n%s", content)
	}
	if !strings.Contains(content, `<span class="sc-folder__count">2</span>`) {
		t.Errorf("expected count 2 (container excluded); got:\n%s", content)
	}
	if !strings.Contains(content, iconPaths["skill"]) {
		t.Errorf("expected the skill crest glyph on group folder cards; got:\n%s", content)
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

func TestUsesFolderIndex_Religion(t *testing.T) {
	// The religion umbrella (Browse/religion, children god/ + saint/) must route
	// through the .sc-folder card renderer, like feature/treasure/rule trees.
	if !usesFolderIndex("Browse/religion") {
		t.Error("usesFolderIndex(Browse/religion) = false, want true")
	}
	// Leaf grids stay flat .sc-card (handled by buildCardsContent), but the path
	// still contains a religion segment — that's fine; they have files, not subdirs,
	// so buildFeatureIndexContent's subdir gate skips them regardless.
	if !usesFolderIndex("Browse/religion/god") {
		t.Error("usesFolderIndex(Browse/religion/god) = false, want true (segment match)")
	}
}

func TestFolderCrestIcon_Religion(t *testing.T) {
	cases := map[string]string{"god": "god", "saint": "title"}
	for child, want := range cases {
		if got := folderCrestIcon("Browse/religion", child); got != want {
			t.Errorf("folderCrestIcon(religion, %q) = %q, want %q", child, got, want)
		}
	}
}
