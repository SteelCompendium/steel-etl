package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbedCardSectionsDefault(t *testing.T) {
	if got := embedCardSections(&Config{}); len(got) != 1 || got[0] != "Browse" {
		t.Errorf("default = %v, want [Browse]", got)
	}
	cfg := &Config{EmbedCardSections: []string{"Browse", "Read"}}
	if got := embedCardSections(cfg); len(got) != 2 || got[1] != "Read" {
		t.Errorf("explicit = %v, want [Browse Read]", got)
	}
}

func TestLeafCard(t *testing.T) {
	// A card-able leaf as buildSection writes it: frontmatter + injected H1
	// + hr + the finished card HTML.
	ability := "---\nname: Repent\nscc: x/feature.ability.censor.level-1/repent\ntype: ability\n---\n\n# Repent\n\n---\n\n<article class=\"sc-ability\">REPENT CARD</article>"
	scc, entry, ok := leafCard(ability)
	if !ok {
		t.Fatal("expected card-able ability leaf")
	}
	if scc != "x/feature.ability.censor.level-1/repent" {
		t.Errorf("scc = %q", scc)
	}
	if entry.html != `<article class="sc-ability">REPENT CARD</article>` {
		t.Errorf("card = %q (H1/hr not stripped?)", entry.html)
	}
	if entry.standalone {
		t.Error("ability is a recursive-container type, not standalone")
	}

	// A statblock leaf is card-able AND standalone (a feature card can't hold it).
	statblock := "---\nname: Ensnarer\nscc: x/monster.minion.summoner.demon.statblock/ensnarer\ntype: statblock\n---\n\n# Ensnarer\n\n---\n\n<div class=\"sb-wrap\">SB</div>"
	if _, e, ok := leafCard(statblock); !ok || !e.standalone {
		t.Errorf("statblock should be card-able + standalone (ok=%v standalone=%v)", ok, e.standalone)
	}

	// A beastheart companion feature-group leaf keeps its advancement-features
	// section (a separately-carded standalone entity) inline below the .sb-wrap on
	// its own page. The card captured for inline embedding must be ONLY the
	// .sb-wrap — the advancement section is embedded on its own under its sibling
	// heading, so leaving it here would duplicate it and, at the leaf's native ##
	// depth, break TOC nesting on container pages.
	companion := "---\nname: Bear\nscc: x/monster.companion.beastheart.statblock/bear\ntype: feature-group\n---\n\n# Bear\n\n---\n\n<div class=\"sb-wrap\">BEAR-SB</div>\n\n" +
		`## Bear Advancement Features {data-scc="x/monster.companion.beastheart.advancement-features/bear"}` +
		"\n\n<div class=\"fb-wrap\">BEAR-FEATUREBLOCK</div>"
	_, e, ok := leafCard(companion)
	if !ok || !e.standalone {
		t.Fatalf("companion feature-group should be card-able + standalone (ok=%v standalone=%v)", ok, e.standalone)
	}
	if e.html != `<div class="sb-wrap">BEAR-SB</div>` {
		t.Errorf("companion card should be just the .sb-wrap, got:\n%s", e.html)
	}

	// A non-card-able type (a class container page) is rejected.
	class := "---\nname: Censor\nscc: x/class.censor\ntype: class\n---\n\n# Censor\n\nbody"
	if _, _, ok := leafCard(class); ok {
		t.Error("class type should not be card-able")
	}

	// A page with no scc is rejected.
	noscc := "---\nname: X\ntype: ability\n---\n\n# X\n\n---\n\ncard"
	if _, _, ok := leafCard(noscc); ok {
		t.Error("missing scc should be rejected")
	}
}

func TestSpliceCards(t *testing.T) {
	body := strings.Join([]string{
		"",
		"# Censor",
		"",
		"---",
		"",
		"## Basics",
		"",
		"Class flavor paragraph.",
		"",
		"## 1st-Level Features",
		"",
		`### Wrath {data-scc="W"}`,
		"",
		"wrath inlined body",
		"",
		"#### Wrath in Combat",
		"",
		"combat sub body",
		"",
		`### Judgment {data-scc="J"}`,
		"",
		`#### Judgment {data-scc="JA"}`,
		"",
		"ability inlined body",
		"",
		`### Unknown {data-scc="U"}`,
		"",
		"unknown body",
		"",
		`### Portfolio {data-scc="P"}`,
		"",
		"portfolio flavor prose",
		"",
		"#### Demon Signature Minion",
		"",
		`##### Ensnarer {data-scc="M"}`,
		"",
		"ensnarer inlined statblock markdown",
		"",
		`### Censor Order {data-scc="CO"}`,
		"",
		"order inlined body",
	}, "\n")

	cards := map[string]cardEntry{
		"W":  {html: "<section>WRATH-CARD</section>"},
		"J":  {html: "<section>JUDGMENT-CARD</section>"},
		"JA": {html: "<section>JUDGMENT-ABILITY-CARD</section>"}, // present, but swallowed under J (recursive feature card already holds it)
		"CO": {html: "<section>ORDER-CARD</section>"},
		"P":  {html: "<section>PORTFOLIO-CARD</section>"},        // a feature whose sub-tree holds a standalone statblock
		"M":  {html: "<div>ENSNARER-SB</div>", standalone: true}, // minion statblock
		// "U" intentionally absent — not card-able.
	}

	got, n := spliceCards(body, "x/class.censor", "", cards)
	if n != 4 { // W, J, CO, M (P is descended, not carded)
		t.Fatalf("spliced %d cards, want 4", n)
	}

	// Structural headings + page body preserved.
	for _, keep := range []string{"# Censor", "## Basics", "Class flavor paragraph.", "## 1st-Level Features"} {
		if !strings.Contains(got, keep) {
			t.Errorf("dropped structural content %q", keep)
		}
	}
	// Card-able headings kept (TOC + permalink anchor) and cards inserted.
	for _, keep := range []string{
		`### Wrath {data-scc="W"}`, "WRATH-CARD",
		`### Judgment {data-scc="J"}`, "JUDGMENT-CARD",
		`### Censor Order {data-scc="CO"}`, "ORDER-CARD",
	} {
		if !strings.Contains(got, keep) {
			t.Errorf("missing kept heading/card %q", keep)
		}
	}
	// Inlined markdown bodies of replaced items are gone (swallowed). The nested
	// ability under Judgment is swallowed and NOT separately carded (no dup).
	for _, gone := range []string{
		"wrath inlined body", "combat sub body", "#### Wrath in Combat",
		"ability inlined body", `#### Judgment {data-scc="JA"}`, "JUDGMENT-ABILITY-CARD",
		"order inlined body",
	} {
		if strings.Contains(got, gone) {
			t.Errorf("inlined body %q should have been swallowed", gone)
		}
	}
	// Unknown (non-card-able) heading + its body left untouched.
	if !strings.Contains(got, `### Unknown {data-scc="U"}`) || !strings.Contains(got, "unknown body") {
		t.Error("non-card-able heading must be left intact with its body")
	}
	// Portfolio (a feature with a standalone statblock descendant) is NOT carded
	// monolithically — it is descended so the inner statblock gets its own card.
	if strings.Contains(got, "PORTFOLIO-CARD") {
		t.Error("Portfolio should be descended, not carded (would hide its minion statblock)")
	}
	for _, keep := range []string{
		`### Portfolio {data-scc="P"}`, "portfolio flavor prose",
		"#### Demon Signature Minion", `##### Ensnarer {data-scc="M"}`, "ENSNARER-SB",
	} {
		if !strings.Contains(got, keep) {
			t.Errorf("descend-mode should keep %q", keep)
		}
	}
	// The minion statblock's own inlined markdown is swallowed by ITS card.
	if strings.Contains(got, "ensnarer inlined statblock markdown") {
		t.Error("statblock inlined markdown should be replaced by its card")
	}
}

func TestSpliceCards_NestedStandalone(t *testing.T) {
	// A companion statblock (standalone) with its abilities AND a nested
	// advancement-features featureblock (also standalone) under it. The statblock
	// card holds the abilities but NOT the featureblock, so the swallow must stop
	// at the featureblock, which then gets its own card.
	body := strings.Join([]string{
		`### Bear {data-scc="S"}`, "",
		"raw statblock grid markdown", "",
		`#### Backhand {data-scc="A"}`, "",
		"ability markdown", "",
		`#### Bear Advancement Features {data-scc="F"}`, "",
		`##### Foe Thresher {data-scc="FT"}`, "",
		"foe thresher markdown with a [link](../../bad/path.md)", "",
		`### Boar {data-scc="S2"}`, "",
		"boar grid markdown",
	}, "\n")
	cards := map[string]cardEntry{
		"S":  {html: "<div>BEAR-SB</div>", standalone: true},
		"A":  {html: "<article>BACKHAND</article>"}, // ability, in the statblock card
		"F":  {html: "<div>BEAR-FEATUREBLOCK</div>", standalone: true},
		"S2": {html: "<div>BOAR-SB</div>", standalone: true},
		// "FT" has no leaf (embedded in the featureblock) — absent from the map.
	}
	got, n := spliceCards(body, "", "", cards)
	if n != 3 { // Bear statblock, Bear featureblock, Boar statblock
		t.Fatalf("spliced %d, want 3", n)
	}
	for _, want := range []string{"BEAR-SB", "BEAR-FEATUREBLOCK", "BOAR-SB",
		`### Bear {data-scc="S"}`, `#### Bear Advancement Features {data-scc="F"}`, `### Boar {data-scc="S2"}`} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	// The featureblock was NOT eaten by the statblock; the orphaned Foe Thresher
	// markdown (and its broken link) is swallowed by the featureblock card.
	for _, gone := range []string{"raw statblock grid markdown", "ability markdown",
		`##### Foe Thresher`, "foe thresher markdown", "bad/path.md"} {
		if strings.Contains(got, gone) {
			t.Errorf("should have been swallowed: %q", gone)
		}
	}
}

func TestRebaseLinks(t *testing.T) {
	// Leaf URL dir Browse/monster/companion/beastheart/basilisk (file dir
	// Browse/monster/companion/beastheart); container Browse/class/beastheart
	// (file dir Browse/class).
	from, to := "Browse/monster/companion/beastheart/basilisk", "Browse/class/beastheart"

	// (1) Directory-style href (a final URL): rebased against the URL dir.
	//     leaf 4-up -> Browse/movement/teleport ; from container URL dir -> ../../movement/teleport/
	if got := rebaseLinks(`<a href="../../../../movement/teleport/">x</a>`, from, to); !strings.Contains(got, `href="../../movement/teleport/"`) {
		t.Errorf("directory href not URL-rebased: %s", got)
	}
	// (2) Markdown directory link (final URL): rebased against the URL dir.
	if got := rebaseLinks(`[t](../../../../movement/teleport/)`, from, to); !strings.Contains(got, `](../../movement/teleport/)`) {
		t.Errorf("markdown directory link not URL-rebased: %s", got)
	}
	// (3) Markdown .md link: MkDocs resolves it from the FILE dir. leaf file dir
	//     3-up -> Browse/movement/forced-movement.md ; from container file dir (Browse/class) -> ../movement/forced-movement.md
	if got := rebaseLinks(`[m](../../../movement/forced-movement.md)`, from, to); !strings.Contains(got, `](../movement/forced-movement.md)`) {
		t.Errorf("markdown .md link not file-rebased: %s", got)
	}
	// (4) href .md link: also file-dir based.
	if got := rebaseLinks(`<a href="../../../condition/prone.md">p</a>`, from, to); !strings.Contains(got, `href="../condition/prone.md"`) {
		t.Errorf("href .md link not file-rebased: %s", got)
	}
	// (5) Anchor-only and external links untouched.
	ext := `<a href="#frag">f</a> <a href="https://x.io/">e</a>`
	if got := rebaseLinks(ext, from, to); !strings.Contains(got, `href="#frag"`) || !strings.Contains(got, `href="https://x.io/"`) {
		t.Errorf("anchor/external altered: %s", got)
	}
	// (6) Identical dirs: a no-op.
	if s := `<a href="../x/">y</a>`; rebaseLinks(s, "a/b", "a/b") != s {
		t.Error("same-dir rebase should be a no-op")
	}
}

func TestEmbedItemCards(t *testing.T) {
	docs := t.TempDir()
	browse := filepath.Join(docs, "Browse")
	leafDir := filepath.Join(browse, "feature", "ability", "censor", "level-1")
	classDir := filepath.Join(browse, "class")
	for _, d := range []string{leafDir, classDir} {
		if err := os.MkdirAll(d, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// The leaf is at Browse/feature/ability/censor/level-1/repent (URL dir depth
	// 6); its link is relative to that. Spliced into Browse/class/censor (depth
	// 3) the link must be rebased.
	leaf := "---\nname: Repent\nscc: x/feature.ability.censor.level-1/repent\ntype: ability\n---\n\n# Repent\n\n---\n\n<article class=\"sc-ability\"><a href=\"../../../../../rule/combat/ranged/\">ranged</a> REPENT-CARD</article>\n"
	if err := os.WriteFile(filepath.Join(leafDir, "repent.md"), []byte(leaf), 0644); err != nil {
		t.Fatal(err)
	}

	class := strings.Join([]string{
		"---", "name: Censor", "scc: x/class.censor", "type: class", "---", "",
		"# Censor", "", "---", "",
		"## 1st-Level Features", "",
		`### Repent {data-scc="x/feature.ability.censor.level-1/repent"}`, "",
		"repent inlined markdown body", "",
	}, "\n")
	classPath := filepath.Join(classDir, "censor.md")
	if err := os.WriteFile(classPath, []byte(class), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := &Config{DocsDir: docs}
	count, errs := embedItemCards(cfg)
	if len(errs) != 0 {
		t.Fatalf("errs: %v", errs)
	}
	if count != 1 {
		t.Fatalf("rewrote %d container pages, want 1", count)
	}

	got, _ := os.ReadFile(classPath)
	gs := string(got)
	if !strings.Contains(gs, "REPENT-CARD") {
		t.Error("class page should contain the transcluded card")
	}
	// Link rebased from the leaf's depth-6 URL dir to the container's depth-3:
	// both resolve to Browse/rule/combat/ranged, so from class/censor it is ../../rule/...
	if !strings.Contains(gs, `href="../../rule/combat/ranged/"`) {
		t.Errorf("transcluded link not rebased to container depth:\n%s", gs)
	}
	if strings.Contains(gs, "repent inlined markdown body") {
		t.Error("inlined markdown should have been replaced")
	}
	if !strings.Contains(gs, `### Repent {data-scc="x/feature.ability.censor.level-1/repent"}`) {
		t.Error("item heading should be kept")
	}
	if !strings.HasPrefix(gs, "---\nname: Censor") {
		t.Error("frontmatter must be preserved")
	}

	// The leaf page itself is not a container and is left byte-for-byte.
	gotLeaf, _ := os.ReadFile(filepath.Join(leafDir, "repent.md"))
	if string(gotLeaf) != leaf {
		t.Error("leaf page should be untouched")
	}
}
