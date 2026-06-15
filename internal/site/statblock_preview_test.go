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
