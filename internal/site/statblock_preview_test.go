package site

import (
	"strings"
	"testing"
)

func TestRenderStatblockFeatureLine_Ability(t *testing.T) {
	f := sbFeature{
		Kind: "ability", Action: "main",
		Name: "Cutting Strike", Usage: "Main Action", Cost: "Signature",
	}
	got := renderStatblockFeatureLine(f)
	for _, want := range []string{
		`class="sb-prev__feat"`,
		`data-action="main"`,
		`class="sb__feat-glyph"`,
		`class="sb-prev__feat-name">Cutting Strike<`,
		`class="sb-prev__feat-usage">Main Action<`,
		`class="sb-prev__feat-cost">Signature<`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("feature line missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderStatblockFeatureLine_PassiveDefaultsTraitUsage(t *testing.T) {
	f := sbFeature{Kind: "passive", Action: "passive", Name: "Mob Tactics"}
	got := renderStatblockFeatureLine(f)
	if !strings.Contains(got, `class="sb-prev__feat-usage">Trait<`) {
		t.Errorf("passive feature should default usage to Trait:\n%s", got)
	}
}

func TestRenderStatblockFeatureLine_StripsLinks(t *testing.T) {
	f := sbFeature{Kind: "ability", Action: "triggered", Name: "Riposte",
		Cost: "2 [Malice](../malice/)"}
	got := renderStatblockFeatureLine(f)
	if strings.Contains(got, "](") || strings.Contains(got, "<a ") {
		t.Errorf("feature line must strip markdown links, got:\n%s", got)
	}
	if !strings.Contains(got, `class="sb-prev__feat-cost">2 Malice<`) {
		t.Errorf("link should reduce to its text:\n%s", got)
	}
}

func TestRenderStatblockPreviewCard(t *testing.T) {
	d := buildStatblockIsland(strings.TrimSpace(`
name: Goblin Cutter
organization: Minion
role: Harrier
level: 1
ev: "3"
size: 1S
speed: 6
stamina: "5"
stability: "0"
free_strike: "2"
might: "1"
agility: "2"
reason: "-1"
intuition: "0"
presence: "-1"
keywords:
    - Goblin
type: statblock`), "> ⭐️ **Mob Tactics**\n>\n> Deals 1 extra damage.")

	got := renderStatblockPreviewCard(d, "goblin-cutter.md", "")
	for _, want := range []string{
		`class="sb-wrap sb-prev"`,
		`data-role="harrier"`,
		`class="sb-prev__link" href="goblin-cutter/"`,
		`class="sc-head"`,
		`<h2 class="sc-head__slot sc-head__left-primary sc-head__slot--line">Goblin Cutter</h2>`,
		`class="sb__defenses"`,
		`class="sb__meta"`,
		`class="sb__chars"`,
		`class="sb-prev__feats"`,
		`class="sb-prev__feat"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("preview card missing %q in:\n%s", want, got)
		}
	}
	if strings.Contains(got, "](") {
		t.Errorf("preview card leaked a markdown link:\n%s", got)
	}
}

func TestRenderStatblockPreviewCard_SourceChip(t *testing.T) {
	d := buildStatblockIsland("name: Bound Imp\nrole: Support\nlevel: 1\ntype: statblock", "")
	withSrc := renderStatblockPreviewCard(d, "bound-imp.md", "Summoner")
	if !strings.Contains(withSrc, `class="sb-prev__src">Summoner<`) {
		t.Errorf("expected Summoner source chip:\n%s", withSrc)
	}
	noSrc := renderStatblockPreviewCard(d, "bound-imp.md", "")
	if strings.Contains(noSrc, "sb-prev__src") {
		t.Errorf("empty source must emit no chip:\n%s", noSrc)
	}
}

func TestStatblockPreviewCard_FeaturesFromCache(t *testing.T) {
	// A real statblock source (blockquote feature body) with an scc code.
	src := []byte("---\n" +
		"name: Cache Goblin\n" +
		"role: Harrier\n" +
		"level: 1\n" +
		"scc: mcdm.test.v1/monster.x.statblock/cache-goblin\n" +
		"type: statblock\n" +
		"---\n\n" +
		"> ⭐️ **Spooky Trait**\n>\n> Does a creepy thing.\n")

	// Transforming the page (as buildSection does) must populate the feature
	// cache keyed by scc.
	if _, ok := buildStatblockIslandPage(src); !ok {
		t.Fatal("expected statblock transform to fire")
	}

	// Now render a preview from a body that has NO blockquotes (simulating the
	// already-transformed on-disk leaf the group landing reads). Features must
	// come from the cache, not the body.
	fm, _ := splitFrontmatter(string(src))
	got := statblockPreviewCard(fm, "<div>already rendered .sb-wrap</div>", "cache-goblin.md", "Cache Goblin")
	if !strings.Contains(got, `class="sb-prev__feats"`) {
		t.Errorf("expected cached feature list in preview, got:\n%s", got)
	}
	if !strings.Contains(got, "Spooky Trait") {
		t.Errorf("expected cached feature name 'Spooky Trait' in preview, got:\n%s", got)
	}
}

func TestSbCardsOpen_DefaultAttrs(t *testing.T) {
	got := sbCardsOpen()
	for _, want := range []string{
		`class="sb-cards"`,
		`data-sbprev-stats="on"`,
		`data-sbprev-meta="off"`,
		`data-sbprev-chars="off"`,
		`data-sbprev-feats="off"`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("sbCardsOpen missing %q in: %s", want, got)
		}
	}
}

func TestStatblockPreview_UsesSharedHead(t *testing.T) {
	got := renderStatblockPreviewCard(sbIsland{KindNoun: "Monster", Name: "Goblin Cutter", Level: "1"}, "x/", "")
	if !strings.Contains(got, `<header class="sc-head">`) {
		t.Errorf("preview should use shared head:\n%s", got)
	}
}
