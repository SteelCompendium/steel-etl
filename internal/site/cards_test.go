package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// inlineMD renders the small subset of inline markdown (links + emphasis) that
// frontmatter card values carry, and rewrites ".md" link targets to the served
// directory-URL form. The card markup nests these values inside non-attributed
// raw-HTML divs where md_in_html will not process them, so they are pre-rendered
// to HTML here. See inlineMD / dirURL in cards.go.
func TestInlineMD(t *testing.T) {
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"link rewritten to dir url", "[Brag](../skill/brag.md)", `<a href="../skill/brag/">Brag</a>`},
		{"emphasis", "*Quick Build:*", "<em>Quick Build:</em>"},
		{
			"mixed",
			"One skill (*Quick Build:* [Brag](../skill/brag.md), [Society](../skill/society.md).)",
			`One skill (<em>Quick Build:</em> <a href="../skill/brag/">Brag</a>, <a href="../skill/society/">Society</a>.)`,
		},
		{"plain", "Two skills from the crafting skill group", "Two skills from the crafting skill group"},
		{"empty", "", ""},
		{"escapes html", "a < b & c", "a &lt; b &amp; c"},
		{"no p wrapper", "hello", "hello"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := inlineMD(tc.in)
			if got != tc.want {
				t.Errorf("inlineMD(%q) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestDirURL(t *testing.T) {
	tests := []struct{ in, want string }{
		{"agent.md", "agent/"},
		{"../skill/sneak.md", "../skill/sneak/"},
		{"../skill/sneak.md#quick", "../skill/sneak/#quick"},
		{"index.md", ""},
		{"../skill/index.md", "../skill/"},
		{"already/dir/", "already/dir/"},
		{"https://example.com/x.md", "https://example.com/x.md"},
		{"#anchor", "#anchor"},
		{"mailto:a@b.c", "mailto:a@b.c"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := dirURL(tc.in); got != tc.want {
			t.Errorf("dirURL(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// careerCard's Skills line must emit a real, resolvable anchor (directory URL),
// not literal markdown — the regression the md_in_html-not-firing bug produced.
func TestCareerCardSkillsRendersLink(t *testing.T) {
	fm := "---\nname: Agent\ntype: career\nskills:\n" +
		"    - One skill from the interpersonal group (*Quick Build:* [Brag](../skill/brag.md).)\n---"
	out := careerCard(fm, "", "agent.md", "Agent")
	if !strings.Contains(out, `<a href="../skill/brag/">Brag</a>`) {
		t.Errorf("expected rendered directory-URL link in Skills line, got:\n%s", out)
	}
	if strings.Contains(out, "[Brag](") {
		t.Errorf("Skills line still contains literal markdown link:\n%s", out)
	}
	if strings.Contains(out, "*Quick Build:*") {
		t.Errorf("Skills line still contains literal emphasis markup:\n%s", out)
	}
	if strings.Contains(out, ".md") {
		t.Errorf("Skills line still contains a dead .md link:\n%s", out)
	}
}

// careerFlavor must drop the trailing "In defining your career…" prompt and
// keep only the lead-in flavor sentence.
func TestCareerFlavorStripsPrompt(t *testing.T) {
	body := "\nYou worked as a spy for a government or organization. In defining your career, think about the following questions:\n\n- Who did you work for?\n"
	if got, want := careerFlavor(body), "You worked as a spy for a government or organization."; got != want {
		t.Errorf("careerFlavor = %q, want %q", got, want)
	}
}

func TestCareerLanguageCount(t *testing.T) {
	tests := []struct{ in, want string }{
		{"Two languages", "Two"},
		{"One language", "One"},
		{"three Languages", "three"},
		{"2", "2"},
		{"", ""},
	}
	for _, tc := range tests {
		if got := careerLanguageCount(tc.in); got != tc.want {
			t.Errorf("careerLanguageCount(%q) = %q, want %q", tc.in, got, tc.want)
		}
	}
}

// careerCard reads the singular `language` key and shows the count word, not a dash.
func TestCareerCardLanguages(t *testing.T) {
	fm := "---\nname: Agent\ntype: career\nlanguage: Two languages\n---"
	out := careerCard(fm, "", "agent.md", "Agent")
	if !strings.Contains(out, ">Two<") {
		t.Errorf("expected Languages stat value 'Two', got:\n%s", out)
	}
}

// cultureCard must surface the body's "**Skill Options:**" line as a link.
func TestCultureCardSkillOptions(t *testing.T) {
	body := "\nRaised by scholars.\n\n**Skill Options:** One skill from the lore skill group. (*Quick Build:* [History](../skill/history.md).)\n"
	out := cultureCard("---\nname: Academic\ntype: culture\n---", body, "academic.md", "Academic")
	if !strings.Contains(out, "Skill Options") {
		t.Errorf("expected a Skill Options line, got:\n%s", out)
	}
	if !strings.Contains(out, `<a href="../skill/history/">History</a>`) {
		t.Errorf("expected rendered Skill Options link, got:\n%s", out)
	}
}

// kitCard shows raw equipment text, the per-echelon stamina label, and all four
// offense stats (melee/ranged damage + melee/ranged distance).
func TestKitCardStatsAndEquipment(t *testing.T) {
	fm := "---\nname: Guisarmier\ntype: kit\nequipment_text: You wear medium armor and wield a polearm.\n" +
		"melee_damage_bonus: +2/+2/+2\nmelee_distance_bonus: \"+1\"\nstamina_bonus: +6 per echelon\n---"
	out := kitCard(fm, "", "guisarmier.md", "Guisarmier")
	for _, want := range []string{
		"You wear medium armor and wield a polearm.",
		"Stamina per Echelon",
		"Melee Dmg", "Ranged Dmg", "Melee Dist", "Ranged Dist",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("kit card missing %q in:\n%s", want, out)
		}
	}
	// Unpopulated offense stats render an em-dash, not a zero.
	if !strings.Contains(out, "—") {
		t.Errorf("expected em-dash for empty offense stats, got:\n%s", out)
	}
}

// kitCard always uses the backpack crest (matching the Kits card on the Browse
// landing) regardless of martial/magic/psionic kind, and surfaces the body's
// first flavor line as a brief description.
func TestKitCardIconAndDescription(t *testing.T) {
	body := "The [Ranger](ranger.md) kit outfits you with medium armor and weapons for every challenge.\n\n## Equipment\n"
	out := kitCard("---\nname: Ranger\ntype: kit\n---", body, "ranger.md", "Ranger")
	if !strings.Contains(out, iconPaths["kit"]) {
		t.Errorf("kit card must use the backpack (kit) crest icon, got:\n%s", out)
	}
	if !strings.Contains(out, "The Ranger kit outfits you") {
		t.Errorf("kit card missing brief description line, got:\n%s", out)
	}
}

// kitCard must always emit the equipment line (a non-breaking space when the
// kit has none) so cards reserve the same vertical space and stay aligned.
func TestKitCardEquipmentAlwaysPresent(t *testing.T) {
	out := kitCard("---\nname: Boren\ntype: kit\n---", "", "boren.md", "Boren")
	if !strings.Contains(out, `<div class="sc-card__equip">&nbsp;</div>`) {
		t.Errorf("expected a placeholder equipment line, got:\n%s", out)
	}
}

// classIntro must recognize a "Basics" header even when it carries a stamped
// attr_list (the Beastheart class — a master class — classifies its Basics
// section, so the page render appends ` {data-scc="…"}`).
func TestClassIntroBasicsWithAttrList(t *testing.T) {
	body := "Intro flavor.\n\n## Basics {data-scc=\"mcdm.beastheart.v1/feature.trait.beastheart.level-1/basics\"}\n\n**Starting Characteristics:** Might 2.\n"
	got := classIntro(body)
	if got != "Intro flavor." {
		t.Errorf("classIntro = %q, want %q (Basics attr_list not matched → whole body leaked)", got, "Intro flavor.")
	}
}

// signatureFromBody must strip the trailing {data-scc=…} attr_list the
// heading-permalink pass stamps onto the ability heading.
func TestSignatureNameStripsAttrList(t *testing.T) {
	body := "## Signature Ability\n\n### Fade {data-scc=\"mcdm.heroes.v1/feature.ability.cloak-and-dagger/fade\"}\n\n| **Melee, Strike** | x |\n"
	name, _, _ := signatureFromBody(body)
	if name != "Fade" {
		t.Errorf("signature name = %q, want %q", name, "Fade")
	}
}

// classCard renders the full intro section (before "Basics"), preserving the
// flavor blockquote and dropping the Basics content.
func TestClassCardFullIntro(t *testing.T) {
	// Body as it appears post-render: injected "# Censor" H1 + rule, then flavor.
	body := "\n# Censor\n\n---\n\nDemons fear you.\n\nAs a [censor](censor.md), you fight.\n\n> \"We FIGHT!\"\n\n## Basics\n\n**Starting Characteristics:** Might 2.\n"
	out := classCard("---\nname: Censor\ntype: class\n---", body, "censor.md", "Censor")
	if !strings.Contains(out, "Demons fear you.") || !strings.Contains(out, "As a") {
		t.Errorf("expected full intro paragraphs, got:\n%s", out)
	}
	if strings.Contains(out, "<h1>") || strings.Contains(out, "<hr>") {
		t.Errorf("injected title/rule leaked into class card:\n%s", out)
	}
	if !strings.Contains(out, "<blockquote>") {
		t.Errorf("expected intro blockquote preserved, got:\n%s", out)
	}
	if strings.Contains(out, "Starting Characteristics") {
		t.Errorf("Basics content leaked into class card:\n%s", out)
	}
	if !strings.Contains(out, `<a href="censor/">censor</a>`) {
		t.Errorf("expected rewritten intro link, got:\n%s", out)
	}
}

// proseBlock preserves paragraph breaks and truncates at the rune budget.
func TestProseBlock(t *testing.T) {
	body := "\nFirst para.\n\nSecond para.\n\n## Heading\n\nNot included.\n"
	if got, want := proseBlock(body, 0), "First para.\n\nSecond para."; got != want {
		t.Errorf("proseBlock = %q, want %q", got, want)
	}
}

// card() must use the stretched-link structure: a <div> wrapper (never an <a>
// wrapping the whole card, which would nest the inner links) plus one overlay
// anchor pointing at the directory URL of the card's page.
func TestCardStretchedLinkStructure(t *testing.T) {
	out := card("agent.md", "briefcase", "Career", "Agent", "  <div class=\"sc-card__line\">x</div>\n")
	if !strings.HasPrefix(out, `<div class="sc-card sc-fil">`) {
		t.Errorf("card must be a <div> wrapper, got:\n%s", out)
	}
	if strings.Contains(out, `<a class="sc-card sc-fil"`) || strings.Contains(out, `<a class="sc-card sc-card--wide`) {
		t.Errorf("card wrapper must not be an <a> (breaks inner links):\n%s", out)
	}
	if !strings.Contains(out, `<a class="sc-card__link" href="agent/" aria-label="Agent"></a>`) {
		t.Errorf("expected stretched overlay link to directory URL, got:\n%s", out)
	}
}

func TestBuildCardsContentNestedSkillLeaf(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, "skill", "crafting")
	if err := os.MkdirAll(leaf, 0755); err != nil {
		t.Fatal(err)
	}
	// self-named container page (must be excluded from the card grid)
	os.WriteFile(filepath.Join(leaf, "crafting.md"),
		[]byte("---\nname: Crafting Skills\ntype: skill-group\n---\n\nOverview.\n"), 0644)
	os.WriteFile(filepath.Join(leaf, "alchemy.md"),
		[]byte("---\nname: Alchemy\ntype: skill\n---\n\nMake bombs and potions.\n"), 0644)
	os.WriteFile(filepath.Join(leaf, "carpentry.md"),
		[]byte("---\nname: Carpentry\ntype: skill\n---\n\nCreate items out of wood.\n"), 0644)

	content, ok := buildCardsContent(leaf, "crafting",
		[]string{"crafting.md", "alchemy.md", "carpentry.md"}, nil)
	if !ok {
		t.Fatalf("buildCardsContent ok=false, want true for nested skill leaf")
	}
	if !strings.Contains(content, ">Alchemy<") || !strings.Contains(content, ">Carpentry<") {
		t.Errorf("expected skill cards for Alchemy and Carpentry; got:\n%s", content)
	}
	if strings.Contains(content, "Crafting Skills") || strings.Contains(content, "crafting/\"") {
		t.Errorf("self-named container card should be excluded; got:\n%s", content)
	}
}

func TestBuildCardsContent_TreasureLeaf(t *testing.T) {
	root := t.TempDir()
	leaf := filepath.Join(root, "treasure", "1st-echelon", "consumable")
	if err := os.MkdirAll(leaf, 0755); err != nil {
		t.Fatal(err)
	}
	item := "---\nname: Black Ash Dart\ntype: treasure\ntreasure_type: consumable\nechelon: \"1\"\nkeywords:\n  - Magic\n---\n\n*A dart of compressed black ash.*\n\n**Keywords:** Magic\n\n**Item Prerequisite:** A pinch of black ash\n\n**Project Source:** Texts in Caelian\n\n**Project Roll Characteristic:** Reason\n\n**Project Goal:** 45\n\n**Effect:** As a maneuver, you make a ranged free strike using a black ash dart.\n"
	if err := os.WriteFile(filepath.Join(leaf, "black-ash-dart.md"), []byte(item), 0644); err != nil {
		t.Fatal(err)
	}

	content, ok := buildCardsContent(leaf, "consumable", []string{"black-ash-dart.md"}, nil)
	if !ok {
		t.Fatalf("buildCardsContent ok=false, want true for treasure leaf dir")
	}
	if !strings.Contains(content, "sc-cards") {
		t.Errorf("expected sc-cards wrapper, got:\n%s", content)
	}
	if !strings.Contains(content, "Black Ash Dart") {
		t.Errorf("expected item name in cards, got:\n%s", content)
	}
	// Leaf title comes from the dirName.
	if !strings.Contains(content, "# Consumable") {
		t.Errorf("expected '# Consumable' title, got:\n%s", content)
	}
	// Keywords render as tags.
	if !strings.Contains(content, "sc-card__tags") {
		t.Errorf("expected keyword tags, got:\n%s", content)
	}
	// Flavor is the first prose line, shown in full (no effect blurb), with the
	// fixed-height clamp class so cards stay the same height.
	if !strings.Contains(content, "A dart of compressed black ash.") {
		t.Errorf("expected flavor line, got:\n%s", content)
	}
	if !strings.Contains(content, "sc-card__flavor--clamp") {
		t.Errorf("expected fixed-height flavor clamp class, got:\n%s", content)
	}
	// Project goal & roll characteristic become stat sub-cards.
	if !strings.Contains(content, "Project Goal") || !strings.Contains(content, ">45<") {
		t.Errorf("expected Project Goal stat, got:\n%s", content)
	}
	if !strings.Contains(content, "Roll Characteristic") {
		t.Errorf("expected Roll Characteristic stat, got:\n%s", content)
	}
	// Item prerequisite & project source become wrapping label lines.
	if !strings.Contains(content, "A pinch of black ash") || !strings.Contains(content, "Texts in Caelian") {
		t.Errorf("expected prerequisite/source lines, got:\n%s", content)
	}
}

func TestGoalStat(t *testing.T) {
	cases := []struct {
		in, wantDisplay, wantTooltip string
	}{
		{"45", "45", ""},   // plain number → no tooltip
		{"450", "450", ""}, // multi-digit plain
		{"45 (yields 1d3 darts)", "45*", "45 (yields 1d3 darts)"}, // parenthetical → abbreviated + tooltip
		{"30 or 45 (see below)", "30*", "30 or 45 (see below)"},   // extra text after the leading number
		{"450 1st Level: ...", "450*", "450 1st Level: ..."},      // trailing prose (data quirk)
		{"see entry", "see entry", ""},                            // no leading number → passthrough
	}
	for _, c := range cases {
		gotD, gotT := goalStat(c.in)
		if gotD != c.wantDisplay || gotT != c.wantTooltip {
			t.Errorf("goalStat(%q) = (%q, %q), want (%q, %q)", c.in, gotD, gotT, c.wantDisplay, c.wantTooltip)
		}
	}
}

func TestBuildCardsContent_TreasureIntermediateFallsBack(t *testing.T) {
	root := t.TempDir()
	mid := filepath.Join(root, "treasure", "1st-echelon")
	if err := os.MkdirAll(filepath.Join(mid, "consumable"), 0755); err != nil {
		t.Fatal(err)
	}
	// Intermediate dir has a subdir → not a leaf → no cards.
	_, ok := buildCardsContent(mid, "1st-echelon", nil, []string{"consumable"})
	if ok {
		t.Errorf("buildCardsContent ok=true for intermediate dir, want false")
	}
}

func TestCardFlavor_PrefersFrontmatter(t *testing.T) {
	fm := "flavor: From the frontmatter field\n"
	body := "From the body prose paragraph.\n"
	if got := cardFlavor(fm, body); got != "From the frontmatter field" {
		t.Errorf("cardFlavor = %q, want frontmatter value", got)
	}
}

func TestCardFlavor_FallsBackToBody(t *testing.T) {
	fm := "name: Thing\n"
	body := "From the body prose paragraph.\n"
	if got := cardFlavor(fm, body); got != "From the body prose paragraph." {
		t.Errorf("cardFlavor = %q, want body fallback", got)
	}
}
