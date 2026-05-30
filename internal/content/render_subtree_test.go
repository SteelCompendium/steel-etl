package content

import (
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestRenderSubtree_NormalizesHeadingsAndOrder(t *testing.T) {
	// Class (H2) -> feature-group (H3) -> trait (H4) -> subheading (H5) -> ability (H6)
	class := &parser.Section{
		Heading:      "Censor",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "censor"},
		BodySource:   "Demons fear you.\n\n> \"We FIGHT!\"\n> **Sir V**",
		Children: []*parser.Section{
			{
				Heading:      "1st-Level Features",
				HeadingLevel: 3,
				Annotation:   map[string]string{"type": "feature-group", "level": "1"},
				BodySource:   "As a 1st-level censor...",
				Children: []*parser.Section{
					{
						Heading:      "Censor Abilities",
						HeadingLevel: 4,
						Annotation:   map[string]string{"type": "feature"},
						BodySource:   "You use a blend.",
						Children: []*parser.Section{
							{
								Heading:      "Signature Ability",
								HeadingLevel: 5,
								BodySource:   "Choose one.",
								Children: []*parser.Section{
									{Heading: "Back Blasphemer!", HeadingLevel: 6, Annotation: map[string]string{"type": "ability"}, BodySource: "> *flavor1*\n> \n> | t | t |"},
									{Heading: "Halt Miscreant!", HeadingLevel: 6, Annotation: map[string]string{"type": "ability"}, BodySource: "> *flavor2*"},
								},
							},
						},
					},
				},
			},
		},
	}

	got := RenderSubtree(class)

	if !strings.Contains(got, "Demons fear you.") {
		t.Error("own body missing")
	}
	if !strings.Contains(got, "> \"We FIGHT!\"") {
		t.Error("genuine flavor blockquote must stay blockquoted")
	}
	if !strings.Contains(got, "## 1st-Level Features") {
		t.Error("feature-group should normalize to H2")
	}
	if !strings.Contains(got, "### Censor Abilities") {
		t.Error("trait should normalize to H3")
	}
	if !strings.Contains(got, "#### Signature Ability") {
		t.Error("subheading should normalize to H4")
	}
	if !strings.Contains(got, "##### Back Blasphemer!") {
		t.Error("ability should normalize to H5")
	}
	if strings.Contains(got, "> *flavor1*") {
		t.Error("ability body must be un-blockquoted")
	}
	if !strings.Contains(got, "*flavor1*") {
		t.Error("ability flavor text should still be present (un-blockquoted)")
	}
	iBack := strings.Index(got, "Back Blasphemer!")
	iHalt := strings.Index(got, "Halt Miscreant!")
	if !(iBack >= 0 && iBack < iHalt) {
		t.Errorf("abilities out of order: back=%d halt=%d", iBack, iHalt)
	}
}

func TestRenderSubtree_LeafEqualsOwnBody(t *testing.T) {
	ability := &parser.Section{
		Heading:      "Back Blasphemer!",
		HeadingLevel: 6,
		Annotation:   map[string]string{"type": "ability"},
		BodySource:   "> *You channel power.*\n> \n> **Power Roll + Presence:**",
	}
	got := RenderSubtree(ability)
	if strings.Contains(got, "> *You channel") {
		t.Error("leaf ability should be un-blockquoted")
	}
	if !strings.Contains(got, "*You channel power.*") {
		t.Error("leaf ability content missing")
	}
	if strings.Contains(got, "#") {
		t.Error("leaf with no children should add no headings")
	}
}

func TestRenderSubtree_ChapterPreservesSourceLevels(t *testing.T) {
	chapter := &parser.Section{
		Heading:      "Classes",
		HeadingLevel: 1,
		Annotation:   map[string]string{"type": "chapter", "id": "classes"},
		BodySource:   "How classes work.",
		Children: []*parser.Section{
			{Heading: "Censor", HeadingLevel: 2, Annotation: map[string]string{"type": "class", "id": "censor"}, BodySource: "Demons fear you."},
		},
	}
	got := RenderSubtree(chapter)
	if !strings.Contains(got, "## Censor") {
		t.Error("class under chapter should be H2")
	}
	if !strings.Contains(got, "How classes work.") {
		t.Error("chapter own body missing")
	}
}
