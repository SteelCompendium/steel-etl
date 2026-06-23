package site

import (
	"strings"
	"testing"
)

func TestRenderCardHead_AllSlots(t *testing.T) {
	got := renderCardHead(cardHeadSlots{
		Crest:        `<span class="crest"></span>`,
		RoleKey:      "brute",
		NameTag:      "h2",
		LeftEyebrow:  hLine("Monster"),
		LeftPrimary:  hLine("Goblin Cutter"),
		LeftDeck:     hLine("Goblin, Humanoid"),
		RightEyebrow: hChip("Level 1"),
		RightPrimary: hMini("Minion Harrier"),
		RightDeck:    hChip("EV 4"),
	})
	for _, want := range []string{
		`<header class="sc-head">`,
		`<span class="crest"></span>`,
		`class="sc-head__slot sc-head__left-eyebrow sc-head__slot--line">Monster</div>`,
		`<h2 class="sc-head__slot sc-head__left-primary sc-head__slot--line">Goblin Cutter</h2>`,
		`class="sc-head__slot sc-head__left-deck sc-head__slot--line">Goblin, Humanoid</div>`,
		`class="sc-head__slot sc-head__right-eyebrow sc-head__slot--chip">Level 1</div>`,
		`class="sc-head__slot sc-head__right-primary sc-head__slot--mini" data-role="brute">Minion Harrier</div>`,
		`class="sc-head__slot sc-head__right-deck sc-head__slot--chip">EV 4</div>`,
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
}

func TestRenderCardHead_OmitsEmptySlotsAndDefaultsNameTag(t *testing.T) {
	got := renderCardHead(cardHeadSlots{
		LeftPrimary:  hLine("Cleave"),
		RightPrimary: hMini("Signature"),
		RightDeck:    hChip("Main Action"),
	})
	if strings.Contains(got, "left-eyebrow") || strings.Contains(got, "left-deck") || strings.Contains(got, "right-eyebrow") {
		t.Errorf("empty slots should be omitted:\n%s", got)
	}
	if !strings.Contains(got, `<h3 class="sc-head__slot sc-head__left-primary sc-head__slot--line">Cleave</h3>`) {
		t.Errorf("NameTag should default to h3:\n%s", got)
	}
	if strings.Contains(got, "data-role=") {
		t.Errorf("no RoleKey set, should emit no data-role:\n%s", got)
	}
}
