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

	got := RenderSubtree(class, nil)

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
	got := RenderSubtree(ability, nil)
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

func TestRenderSubtree_DemotesOverflowHeadings(t *testing.T) {
	// Retainer statblocks carry H8 "Level N … Advancement Ability" sub-labels,
	// which are NOT collected as sections and would otherwise leak as literal
	// "########" text. They must demote to bold body labels.
	sb := &parser.Section{
		Heading:      "Devil Detective",
		HeadingLevel: 6,
		Annotation:   map[string]string{"type": "statblock"},
		BodySource:   "> ⭐️ **Soulsight**\n\n######## Level 4 Retainer Advancement Ability\n\n> 🏹 **Soul Sleuth**",
	}
	got := RenderSubtree(sb, nil)
	if strings.Contains(got, "#######") {
		t.Errorf("7+ hash heading must not leak as literal hashes:\n%s", got)
	}
	if !strings.Contains(got, "**Level 4 Retainer Advancement Ability**") {
		t.Errorf("overflow heading should demote to bold:\n%s", got)
	}
	// A genuine blockquote in the same body must be untouched.
	if !strings.Contains(got, "> ⭐️ **Soulsight**") {
		t.Errorf("statblock blockquotes must be preserved:\n%s", got)
	}
}

func TestRenderSubtree_ClampsShallowChildToH1(t *testing.T) {
	// A child whose HeadingLevel is shallower than its parent's would yield a
	// negative computed level; it must clamp to H1 rather than panic on a
	// negative strings.Repeat count.
	root := &parser.Section{
		Heading:      "Deep Section",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "intro",
		Children: []*parser.Section{
			{Heading: "Shallow Child", HeadingLevel: 2, BodySource: "child body"},
		},
	}
	got := RenderSubtree(root, nil) // must not panic
	if !strings.Contains(got, "# Shallow Child") {
		t.Error("shallow child should clamp to a valid heading (H1)")
	}
	if strings.Contains(got, "## Shallow Child") || strings.Contains(got, "### Shallow Child") {
		t.Error("shallow child should not produce a deeper heading than H1")
	}
}

func TestRenderSubtree_CapsDeepNestingAtH6(t *testing.T) {
	// Descendants deeper than 6 levels below the root must cap at H6.
	root := &parser.Section{
		Heading:      "Root",
		HeadingLevel: 1,
		Annotation:   map[string]string{"type": "chapter"},
		BodySource:   "root body",
		Children: []*parser.Section{
			{Heading: "L8", HeadingLevel: 9, BodySource: "deep body"},
		},
	}
	got := RenderSubtree(root, nil)
	if !strings.Contains(got, "###### L8") {
		t.Error("deeply nested child should cap at H6")
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
	got := RenderSubtree(chapter, nil)
	if !strings.Contains(got, "## Censor") {
		t.Error("class under chapter should be H2")
	}
	if !strings.Contains(got, "How classes work.") {
		t.Error("chapter own body missing")
	}
}

func TestRenderSubtree_EmitsDataSCCOnCodedHeadings(t *testing.T) {
	ability := &parser.Section{Heading: "Gouge", HeadingLevel: 3, Annotation: map[string]string{"type": "ability"}, BodySource: "Stab them."}
	structural := &parser.Section{Heading: "Heroic Resource", HeadingLevel: 3, BodySource: "You have Ferocity."}
	class := &parser.Section{
		Heading:      "Fury",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "fury"},
		BodySource:   "You rage.",
		Children:     []*parser.Section{ability, structural},
	}

	scc := map[*parser.Section]string{
		ability: "mcdm.heroes.v1/feature.ability.fury.level-1/gouge",
		// structural intentionally absent: no SCC code
	}

	got := RenderSubtree(class, scc)

	if !strings.Contains(got, `## Gouge {data-scc="mcdm.heroes.v1/feature.ability.fury.level-1/gouge"}`) {
		t.Errorf("coded heading missing data-scc marker:\n%s", got)
	}
	if strings.Contains(got, "Heroic Resource {data-scc") {
		t.Error("structural heading (no code) must not get a data-scc marker")
	}
	if !strings.Contains(got, "## Heroic Resource\n") {
		t.Errorf("structural heading should render as a plain heading:\n%s", got)
	}
}

func TestRenderSubtree_NilMapEmitsNoMarkers(t *testing.T) {
	class := &parser.Section{
		Heading:      "Fury",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "fury"},
		BodySource:   "You rage.",
		Children:     []*parser.Section{{Heading: "Gouge", HeadingLevel: 3, Annotation: map[string]string{"type": "ability"}, BodySource: "Stab."}},
	}
	got := RenderSubtree(class, nil)
	if strings.Contains(got, "data-scc") {
		t.Errorf("nil map must emit no data-scc markers:\n%s", got)
	}
	if !strings.Contains(got, "## Gouge") {
		t.Error("heading should still render with a nil map")
	}
}

func TestRenderSubtree_CalloutSuppression(t *testing.T) {
	loose := "<!-- @type: callout | @owner: loose -->\n> **Incidental**\n> just landed here"
	self := "<!-- @type: callout | @owner: self -->\n> **Alt Rule**\n> use this instead"

	t.Run("loose callout stripped from root body", func(t *testing.T) {
		sec := &parser.Section{
			Heading: "Leader Formation", HeadingLevel: 4,
			Annotation: map[string]string{"type": "feature"},
			BodySource: "Feature text.\n\n" + loose,
		}
		got := RenderSubtree(sec, nil)
		if strings.Contains(got, "Incidental") {
			t.Errorf("loose callout should be stripped from root body, got:\n%s", got)
		}
		if !strings.Contains(got, "Feature text.") {
			t.Errorf("feature text missing, got:\n%s", got)
		}
	})

	t.Run("loose callout kept in descendant body", func(t *testing.T) {
		parent := &parser.Section{
			Heading: "Summoner", HeadingLevel: 2,
			Annotation: map[string]string{"type": "class"},
			BodySource: "Class intro.",
			Children: []*parser.Section{
				{
					Heading: "Leader Formation", HeadingLevel: 4,
					Annotation: map[string]string{"type": "feature"},
					BodySource: "Feature text.\n\n" + loose,
				},
			},
		}
		got := RenderSubtree(parent, nil)
		if !strings.Contains(got, "Incidental") {
			t.Errorf("loose callout should survive in descendant body, got:\n%s", got)
		}
	})

	t.Run("self callout kept in root body", func(t *testing.T) {
		sec := &parser.Section{
			Heading: "Some Rule", HeadingLevel: 4,
			Annotation: map[string]string{"type": "feature"},
			BodySource: "Rule text.\n\n" + self,
		}
		got := RenderSubtree(sec, nil)
		if !strings.Contains(got, "Alt Rule") {
			t.Errorf("self callout should be kept on its own page, got:\n%s", got)
		}
	})
}
