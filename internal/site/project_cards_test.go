package site

import (
	"os"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/content"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestProjectCardsCorpusNesting(t *testing.T) {
	raw, err := os.ReadFile("../../input/heroes/Draw Steel Heroes.md")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := parser.ParseDocument(raw)
	if err != nil {
		t.Fatal(err)
	}
	var imbue *parser.Section
	codes := map[*parser.Section]string{}
	for _, root := range doc.Sections {
		for _, s := range root.AllSections() {
			if s.Heading == "Imbue Treasure" {
				imbue = s
			}
			if s.Heading == "Dragon's Fire" {
				codes[s] = "mcdm.heroes.v1/feature.ability.common/dragons-fire"
			}
		}
	}
	if imbue == nil {
		t.Fatal("missing corpus project")
	}
	body := content.RenderSubtree(imbue, codes)
	data, ok := buildProjectCardPage([]byte("---\nname: Imbue Treasure\ntype: project\n---\n\n" + body))
	if !ok {
		t.Fatal("not carded")
	}
	got := string(data)
	// 85 named enhancements; the earlier probe counted Dragon Soul's inline
	// Power Roll label as an 86th enhancement.
	if n := strings.Count(got, `class="pj__item"`); n != 85 {
		t.Errorf("enhancements=%d, want 85", n)
	}
	if n := strings.Count(got, `class="pj__grant"`); n != 3 {
		t.Errorf("grants=%d, want 3", n)
	}
	for _, pair := range [][2]string{{"Chargebreaker", "Stop Right There"}, {"Nova", "Nova"}, {"Dragon Soul II", "Dragon&#39;s Fire"}} {
		start := strings.Index(got, `class="pj__item-name">`+pair[0])
		if start < 0 {
			t.Errorf("missing %s", pair[0])
			continue
		}
		end := strings.Index(got[start:], "</section>")
		if end < 0 {
			t.Fatal("unterminated item")
		}
		item := got[start : start+end]
		if !strings.Contains(item, `class="pj__grant"`) || !strings.Contains(item, pair[1]) {
			t.Errorf("%s did not contain %s", pair[0], pair[1])
		}
	}
	if !strings.Contains(got, ">Chilling II</span>") {
		t.Fatal("following enhancement lost its own panel")
	}
	if !strings.Contains(got, "<table>") {
		t.Fatal("enhancement tables lost")
	}
	if !strings.Contains(got, "Free Triggered Action") {
		t.Fatal("grant lost its free action qualifier")
	}
	if strings.Contains(got, "&gt; ######") {
		t.Fatal("raw ability heading leaked")
	}
}

func TestProjectCardPlainAndDelegating(t *testing.T) {
	body := "**Item Prerequisite:** None\n\n**Project Source:** A manual\n\n**Project Roll Characteristic:** Reason\n\n**Project Goal:** 45 (per mile)\n\nKeep **all** the prose.\n\n## Results\n\n| Roll | Result |\n|---|---|\n| 1 | Done |"
	got, ok := buildProjectCardPage([]byte("---\nname: Road\ntype: project\n---\n\n" + body))
	if !ok {
		t.Fatal("not carded")
	}
	for _, want := range []string{`class="pj"`, `<dl class="pj__ledger">`, "45 (per mile)", "<strong>all</strong>", "<table>", "<h2"} {
		if !strings.Contains(string(got), want) {
			t.Errorf("missing %s", want)
		}
	}
	if strings.Count(string(got), "A manual") != 1 {
		t.Fatal("duplicated ledger value")
	}
	input := []byte("---\nname: Ordinary\ntype: rule\n---\n\nUnrelated prose.")
	if _, ok := buildProjectCardPage(input); ok {
		t.Fatal("unrelated rule was carded")
	}
}
