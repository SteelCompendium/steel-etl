package content

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// SC-199: the 5th- and 9th-Level Weapon Enhancement pages shipped with every
// enhancement that follows Chargebreaker / Nova swallowed into the ability card,
// because the ability blockquote's `> ` prefix ran past the end of the ability.
//
// The fixture is a verbatim excerpt of the real book source at the broken commit
// (see the header comment in the .md) rather than a hand-written approximation —
// SC-155 was a fixture-shape-vs-corpus-shape miss and this guards the same class.

func labelsOf(issues []AbilityQuoteIssue) []string {
	out := make([]string, 0, len(issues))
	for _, is := range issues {
		out = append(out, fmt.Sprintf("%s/%s", is.Ability, is.Label))
	}
	return out
}

func TestScanOverExtendedAbilityQuotes_SC199BrokenCorpusExcerpt(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("testdata", "sc199_weapon_enhancements_prefix.md"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	got := labelsOf(ScanOverExtendedAbilityQuotes(string(data)))
	want := []string{
		"Stop Right There/Chilling II",
		"Stop Right There/Devastating",
		"Stop Right There/Disrupting II",
		"Stop Right There/Metamorphic",
		"Stop Right There/Silencing",
		"Stop Right There/Terrifying II",
		"Stop Right There/Thundering II",
		"Stop Right There/Vengeance II",
		"Nova/Terrifying III",
		"Nova/Thundering III",
		"Nova/Vengeance III",
		"Nova/Windcutting",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("findings mismatch\n got: %v\nwant: %v", got, want)
	}

	// The ability's own fields must never be reported — flagging **Trigger:** or
	// **Effect:** would mean a fix that truncates a real ability.
	for _, is := range ScanOverExtendedAbilityQuotes(string(data)) {
		switch is.Label {
		case "Trigger", "Effect", "Power Roll + Your Highest Characteristic Score":
			t.Errorf("ability field %q reported as foreign content", is.Label)
		}
	}
}

// The whole annotated corpus must be clean. This is the regression gate: it reads
// the real book sources, so re-introducing an over-extended quote anywhere in any
// book fails the build rather than shipping corrupted pages.
func TestScanOverExtendedAbilityQuotes_RealCorpusIsClean(t *testing.T) {
	books, err := filepath.Glob(filepath.Join("..", "..", "input", "*", "*.md"))
	if err != nil {
		t.Fatalf("glob input: %v", err)
	}
	if len(books) == 0 {
		t.Skip("skipping: no book sources available")
	}

	total := 0
	for _, path := range books {
		data, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", path, err)
		}
		for _, is := range ScanOverExtendedAbilityQuotes(string(data)) {
			total++
			t.Errorf("%s:%d: ability %q has swallowed the following sibling **%s:** — "+
				"the blockquote's `> ` prefix runs past the end of the ability body",
				filepath.Base(path), is.Line, is.Ability, is.Label)
		}
	}
	t.Logf("scanned %d book sources, %d findings", len(books), total)
}

func TestScanOverExtendedAbilityQuotes_LegitimateShapes(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{
			// The fixed shape: the ability ends at Effect, the siblings follow outside.
			name: "sc199 fixed shape",
			body: "**Chargebreaker:** While you wield this weapon, you have the following ability.\n" +
				"\n" +
				"> ###### Stop Right There\n" +
				">\n" +
				"> *Their momentum, your impact.*\n" +
				">\n" +
				"> | **[Melee](scc.v1:mcdm.heroes.v1/rule.combat/melee), Weapon** | **Free [triggered](scc.v1:mcdm.heroes.v1/rule.combat/triggered-action)** |\n" +
				"> |---------------------------|-------------------:|\n" +
				"> | **📏 Melee 1**            |   **🎯 One enemy** |\n" +
				">\n" +
				"> **Trigger:** The target willingly moves adjacent to you.\n" +
				">\n" +
				"> **Effect:** The target takes 5 damage.\n" +
				"\n" +
				"**Chilling II:** …\n" +
				"\n" +
				"**Metamorphic:** …\n" +
				"\n" +
				"- **Concealed:** …\n",
		},
		{
			// An ability that genuinely is the last thing in its section.
			name: "ability is last in section",
			body: "You have the following ability.\n\n" +
				"> ###### Nova\n>\n> *I am an eternal flame, baby!*\n>\n" +
				"> **[Power Roll](scc.v1:mcdm.heroes.v1/rule.dice/power-roll) + Your Highest Characteristic Score:**\n>\n" +
				"> - **≤11:** 7 fire damage\n> - **12-16:** 11 fire damage\n> - **17+:** 16 fire damage\n",
		},
		{
			// Bodies whose own labels look like sibling paragraphs to a naive rule.
			name: "known ability fields after Effect",
			body: "> ###### Entropic Bolt\n>\n" +
				"> **[Power Roll](scc.v1:mcdm.heroes.v1/rule.dice/power-roll) + Reason:**\n>\n" +
				"> - **≤11:** 2 damage\n> - **17+:** 6 damage\n>\n" +
				"> **Effect:** The target is slowed.\n>\n" +
				"> **Spend 1+ Insight:** The damage increases by 2 per insight spent.\n>\n" +
				"> **Persistent 1:** You maintain the bolt.\n>\n" +
				"> **Special:** Usable once per turn.\n>\n" +
				"> **Strained:** You take 3 corruption damage.\n",
		},
		{
			// Heroes book: the Zola sample negotiation. Headed blockquote, but not an
			// ability — its motivations genuinely belong inside.
			name: "headed non-ability blockquote",
			body: "> **Zola Honeycut Negotiation Stats**\n>\n" +
				"> - **Interest: 2**\n> - **Patience: 4**\n>\n" +
				"> ###### Motivations\n>\n" +
				"> **Benevolence:** Zola always gives her fellow thieves a bigger cut.\n>\n" +
				"> ###### Pitfall\n>\n" +
				"> **Higher Authority:** Zola has no interest in serving anyone but herself.\n",
		},
		{
			// Monster statblock abilities are headless blockquotes whose bodies carry
			// free-form sub-option labels under an Effect.
			name: "headless statblock ability with sub-options",
			body: "> **[Power Roll](scc.v1:mcdm.monsters.v1/rule.dice/power-roll) + 5:**\n>\n" +
				"> - **≤11:** 4 damage\n>\n" +
				"> **Effect:** The Director chooses one transformation.\n>\n" +
				"> **Head:** The target can't communicate.\n>\n" +
				"> **Legs:** The target is slowed.\n>\n" +
				"> **3 Malice:** The transformation is permanent.\n",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if issues := ScanOverExtendedAbilityQuotes(tc.body); len(issues) != 0 {
				t.Errorf("expected no findings, got %v", labelsOf(issues))
			}
		})
	}
}
