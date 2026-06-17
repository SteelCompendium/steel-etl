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

	got, n := spliceCards(body, "x/class.censor", cards)
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

	leaf := "---\nname: Repent\nscc: x/feature.ability.censor.level-1/repent\ntype: ability\n---\n\n# Repent\n\n---\n\n<article class=\"sc-ability\">REPENT-CARD</article>\n"
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
