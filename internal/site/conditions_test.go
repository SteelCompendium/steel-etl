package site

import (
	"os"
	"reflect"
	"regexp"
	"sort"
	"strings"
	"testing"
)

// ── vocabulary ───────────────────────────────────────────────────────────────

// The Go condition vocabulary must equal the set of `@type: condition` entities
// in the book sources. This is the guard that catches a condition being missed:
// if an errata printing or a new supplement adds a tenth condition, extraction
// would silently never match it — here it fails the build instead.
func TestConditionVocabularyMatchesBookSources(t *testing.T) {
	books := []string{
		"../../input/heroes/Draw Steel Heroes.md",
		"../../input/monsters/Draw Steel Monsters.md",
		"../../input/beastheart/Draw Steel Beastheart.md",
		"../../input/summoner/Draw Steel Summoner.md",
	}
	re := regexp.MustCompile(`@type:\s*condition\s*\|\s*@id:\s*([a-z][a-z0-9-]*)`)
	found := map[string]bool{}
	read := 0
	for _, b := range books {
		data, err := os.ReadFile(b)
		if err != nil {
			continue // a book source may be absent in a partial checkout
		}
		read++
		for _, m := range re.FindAllStringSubmatch(string(data), -1) {
			found[m[1]] = true
		}
	}
	if read == 0 {
		t.Skip("no book sources available")
	}
	if len(found) == 0 {
		t.Fatal("no @type: condition annotations found — the annotation shape changed?")
	}
	var got []string
	for k := range found {
		got = append(got, k)
	}
	sort.Strings(got)
	want := append([]string(nil), conditionSlugs...)
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("conditionSlugs is out of sync with the book sources:\n  book: %v\n  go:   %v", got, want)
	}
}

func TestConditionVocabularyHasNoDuplicates(t *testing.T) {
	seen := map[string]bool{}
	for _, s := range conditionSlugs {
		if seen[s] {
			t.Errorf("duplicate condition slug %q", s)
		}
		seen[s] = true
		if !conditionSet[s] {
			t.Errorf("conditionSet missing %q", s)
		}
		if conditionWordRe[s] == nil {
			t.Errorf("conditionWordRe missing %q", s)
		}
	}
}

// Every condition in the vocabulary must be detectable in both the forms the
// corpus actually uses: a resolved md link, and a bare word. A regex or table
// that quietly drops one condition fails here.
func TestExtractConditions_EveryConditionDetectable(t *testing.T) {
	for _, slug := range conditionSlugs {
		linked := "**Effect:** The target is [" + slug + "](../../../../condition/" + slug + ".md) (save ends)."
		if got := extractConditions("", linked); !reflect.DeepEqual(got, []string{slug}) {
			t.Errorf("linked %q: got %v, want [%s]", slug, got, slug)
		}
		bare := "**Effect:** The target is " + slug + " until the end of the encounter."
		if got := extractConditions("", bare); !reflect.DeepEqual(got, []string{slug}) {
			t.Errorf("bare %q: got %v, want [%s]", slug, got, slug)
		}
	}
}

// ── extraction ───────────────────────────────────────────────────────────────

func TestExtractConditions(t *testing.T) {
	tests := []struct {
		name string
		fm   string
		body string
		want []string
	}{
		{
			name: "inflicted in power-roll tiers",
			fm:   "name: Phase Step\ntype: ability\n",
			body: "- **≤11:** 6 damage; M < WEAK, [dazed](../../../../condition/dazed.md)\n" +
				"- **17+:** 12 damage; M < STRONG, [dazed](../../../../condition/dazed.md)",
			want: []string{"dazed"},
		},
		{
			// A precondition, not an affliction the ability applies. v1 matches
			// it deliberately — see the design note in conditions.go.
			name: "referenced as a precondition still matches",
			fm:   "type: ability\n",
			body: "**Effect:** This ability has no effect on a [prone](../../../../condition/prone.md) target.",
			want: []string{"prone"},
		},
		{
			// Likewise removal — the ability is about slowed/weakened even
			// though it takes them away.
			name: "removal still matches",
			fm:   "type: ability\n",
			body: "**Effect:** The target is no longer [slowed](../../../../condition/slowed.md) or [weakened](../../../../condition/weakened.md).",
			want: []string{"slowed", "weakened"},
		},
		{
			// Censor's Judgment: taunted appears ONLY in an unlabelled bullet
			// list, which never reaches the `effects:` frontmatter. Extraction
			// from frontmatter alone would miss this (12 abilities do).
			name: "body-only bullet list beyond the effects frontmatter",
			fm:   "name: Judgment\ntype: ability\neffects:\n    - effect: The target is judged by you.\n      name: Effect\n",
			body: "**Effect:** The target is judged by you.\n\n" +
				"- If you damage a creature judged by you with a melee ability, the creature is " +
				"[taunted](../../../../condition/taunted.md) by you until the end of their next turn.",
			want: []string{"taunted"},
		},
		{
			// Assassinate. The ONLY mention is in flavor — a mention there says
			// nothing about the ability's mechanics, so it must not match.
			name: "flavor-only mention is excluded (frontmatter)",
			fm:   "name: Assassinate\ntype: ability\nflavor: A practiced attack will instantly kill an already weakened foe.\n",
			body: "**Effect:** The target is reduced to 0 Stamina.",
			want: nil,
		},
		{
			// Accelerate, as it appears in md-linked: flavor is BOTH a
			// frontmatter field and a leading italic paragraph.
			name: "flavor-only mention is excluded (italic body paragraph)",
			fm:   "name: Accelerate\ntype: ability\nflavor: To your ally, it seems as though the world has slowed down.\n",
			body: "*To your ally, it seems as though the world has slowed down.*\n\n" +
				"**Effect:** The target shifts up to a number of squares equal to your Reason score.",
			want: nil,
		},
		{
			// "you weaken your connection" / "then grab them" are flavor verbs,
			// not conditions — matching on the exact condition word (never a
			// stem) is what keeps them out.
			name: "verb stems are not conditions",
			fm:   "type: ability\nflavor: You suddenly strike an enemy, then grab them in a psionically enhanced grip.\n",
			body: "*You weaken your connection to this manifold.*\n\n**Effect:** You shift up to your speed.",
			want: nil,
		},
		{
			name: "ability name is not scanned",
			fm:   "name: Weakening Brand\ntype: ability\n",
			body: "**Effect:** Your strikes deal extra damage.",
			want: nil,
		},
		{
			// The rule glossary entry ABOUT conditions is rule/combat/condition.md
			// — no slug follows the slash, so it must not produce a match.
			name: "link to the conditions rule itself is not a condition",
			fm:   "type: ability\n",
			body: "**Effect:** The target gains a [condition](../../../../rule/combat/condition.md) of your choice.",
			want: nil,
		},
		{
			name: "frontmatter tiers alone",
			fm: "type: ability\ntier1: 7 damage; I < WEAK, [restrained](../../../../condition/restrained.md) (save ends)\n" +
				"trigger: The target is [grabbed](../../../../condition/grabbed.md).\n",
			body: "",
			want: []string{"grabbed", "restrained"},
		},
		{
			name: "canonical order regardless of appearance order",
			fm:   "type: ability\n",
			body: "**Effect:** weakened, then dazed, then bleeding.",
			want: []string{"bleeding", "dazed", "weakened"},
		},
		{
			name: "no conditions",
			fm:   "name: Dragon Breath\ntype: ability\nflavor: A furious exhalation.\n",
			body: "**Effect:** You choose the ability's damage type.",
			want: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := extractConditions(tc.fm, tc.body)
			if !reflect.DeepEqual(got, tc.want) {
				t.Errorf("extractConditions() = %v, want %v", got, tc.want)
			}
		})
	}
}

// A folded/multi-line flavor block must be dropped whole, not just its first line.
func TestFrontmatterWithoutFields_DropsContinuationLines(t *testing.T) {
	fm := "name: X\nflavor: >-\n  the world has\n  slowed down\ntier1: 2 damage; [dazed](../../condition/dazed.md)\n"
	got := frontmatterWithoutFields(fm, conditionsDropFields)
	if strings.Contains(got, "slowed") {
		t.Errorf("folded flavor block leaked through:\n%s", got)
	}
	if !strings.Contains(got, "tier1") {
		t.Errorf("dropped too much — tier1 is gone:\n%s", got)
	}
	if c := extractConditions(fm, ""); !reflect.DeepEqual(c, []string{"dazed"}) {
		t.Errorf("extractConditions = %v, want [dazed]", c)
	}
}

// ── card attribute round-trip ────────────────────────────────────────────────

func TestRenderAbilityCard_StampsConditions(t *testing.T) {
	fm := "name: Phase Step\ntype: ability\naction_type: Main action\nflavor: You weaken your connection.\n"
	body := "*You weaken your connection.*\n\n" +
		"**Power Roll + Agility:**\n\n" +
		"- **≤11:** 6 damage; M < WEAK, [dazed](../../../../condition/dazed.md)\n" +
		"- **12-16:** 8 damage; M < AVERAGE, [dazed](../../../../condition/dazed.md)\n" +
		"- **17+:** 12 damage; M < STRONG, [dazed](../../../../condition/dazed.md)"
	card := renderAbilityCard(fm, body, "")
	if !strings.Contains(card, `data-conditions="dazed"`) {
		t.Errorf("card missing data-conditions=\"dazed\":\n%s", card)
	}
	if strings.Contains(card, "weakened") {
		t.Errorf("flavor verb leaked into the card conditions:\n%s", card)
	}
	if got := conditionsFromCardHTML(card); !reflect.DeepEqual(got, []string{"dazed"}) {
		t.Errorf("conditionsFromCardHTML = %v, want [dazed]", got)
	}
}

func TestRenderAbilityCard_OmitsAttrWhenNoConditions(t *testing.T) {
	card := renderAbilityCard("name: Dragon Breath\ntype: ability\n", "**Effect:** 2 damage.", "")
	if strings.Contains(card, "data-conditions") {
		t.Errorf("empty condition set must omit the attribute:\n%s", card)
	}
}

// A trait/feature card must not carry the attribute — v1 scopes the facet to
// abilities.
func TestRenderCard_TraitHasNoConditions(t *testing.T) {
	out, ok := buildAbilityCardPage([]byte("---\nname: Grit\ntype: trait\nancestry: dwarf\n---\n\nYou can't be knocked [prone](../../condition/prone.md).\n"), nil)
	if !ok {
		t.Fatal("expected the trait page to be carded")
	}
	if strings.Contains(string(out), "data-conditions") {
		t.Errorf("trait card must not carry data-conditions:\n%s", out)
	}
}

func TestConditionsFromCardHTML(t *testing.T) {
	t.Run("ignores unknown slugs", func(t *testing.T) {
		if got := conditionsFromCardHTML(`<article data-conditions="dazed dazzled prone">`); !reflect.DeepEqual(got, []string{"dazed", "prone"}) {
			t.Errorf("got %v, want [dazed prone]", got)
		}
	})
	t.Run("takes only the page's own card", func(t *testing.T) {
		html := `<article class="sc-ability" data-conditions="prone"></article>` +
			`<article class="sc-ability" data-conditions="bleeding"></article>`
		if got := conditionsFromCardHTML(html); !reflect.DeepEqual(got, []string{"prone"}) {
			t.Errorf("got %v, want [prone]", got)
		}
	})
	t.Run("absent attribute", func(t *testing.T) {
		if got := conditionsFromCardHTML(`<article class="sc-ability"></article>`); got != nil {
			t.Errorf("got %v, want nil", got)
		}
	})
}

// A rendered card that found no conditions must stay empty — re-deriving from
// the rendered HTML would re-admit the flavor <p> and the sc-src template.
func TestConditionsForPreview_HonoursEmptyCard(t *testing.T) {
	fm := "name: Assassinate\ntype: ability\nflavor: kill an already weakened foe.\n"
	body := `<article class="sc-ability sc-fil" data-action="main">` +
		`<p class="sc-ability__flavor">kill an already weakened foe.</p></article>`
	if got := conditionsForPreview(fm, body); got != nil {
		t.Errorf("conditionsForPreview = %v, want nil (flavor must not leak back in)", got)
	}
}

func TestConditionsForPreview_DerivesForUncardedPage(t *testing.T) {
	fm := "name: X\ntype: ability\n"
	body := "**Effect:** The target is [prone](../../condition/prone.md)."
	if got := conditionsForPreview(fm, body); !reflect.DeepEqual(got, []string{"prone"}) {
		t.Errorf("conditionsForPreview = %v, want [prone]", got)
	}
}
