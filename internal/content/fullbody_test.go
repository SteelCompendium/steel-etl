package content

import (
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// Tests that verify content parsers include unannotated sub-heading content.
// These guard against the regression where sections with unannotated child
// headings (e.g. tables under a feature) lost their sub-heading content.

func TestFeatureParser_IncludesUnannotatedSubheadingTable(t *testing.T) {
	// Simulates: #### Growing Ferocity → ###### Berserker Growing Ferocity Table
	tableBody := "| Ferocity | Benefit |\n|----------|----------|\n| 2 | Knockback bonus equal to Might score. |\n| 4 | First push grants 1 surge. |"
	section := &parser.Section{
		Heading:      "Growing Ferocity",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "You gain benefits based on ferocity amount.",
		Children: []*parser.Section{
			{
				Heading:      "Berserker Growing Ferocity Table",
				HeadingLevel: 6,
				BodySource:   tableBody,
			},
			{
				Heading:      "Reaver Growing Ferocity Table",
				HeadingLevel: 6,
				BodySource:   "| Ferocity | Benefit |\n| 2 | Agility bonus. |",
			},
		},
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "fury"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "level": "1"})

	result, err := (&FeatureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Body must contain the unannotated sub-heading tables
	if !strings.Contains(result.Body, "Berserker Growing Ferocity Table") {
		t.Error("body should contain unannotated sub-heading 'Berserker Growing Ferocity Table'")
	}
	if !strings.Contains(result.Body, "Knockback bonus equal to Might score.") {
		t.Error("body should contain table content from unannotated child")
	}
	if !strings.Contains(result.Body, "Reaver Growing Ferocity Table") {
		t.Error("body should contain second unannotated sub-heading")
	}
	if !strings.Contains(result.Body, "Agility bonus.") {
		t.Error("body should contain second table content")
	}

	// Also verify the parent's own content is still there
	if !strings.Contains(result.Body, "You gain benefits based on ferocity amount.") {
		t.Error("body should still contain the parent's own content")
	}
}

func TestFeatureParser_MultiAbilityContainerEmbedsAllInline(t *testing.T) {
	// Simulates a "Class Abilities" container feature (e.g. "Censor Abilities",
	// "Fury Abilities"): an intro, then unannotated sub-headings each holding
	// multiple @type: ability children. All abilities must render inline, in
	// document order, under their sub-headings -- not dropped, and not collapsed
	// to just the first ability appended at the end.
	section := &parser.Section{
		Heading:      "Censor Abilities",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "You use a blend of martial techniques and divine magic.",
		Children: []*parser.Section{
			{
				Heading:      "Signature Ability",
				HeadingLevel: 5,
				BodySource:   "Choose one signature ability from the following options.",
				Children: []*parser.Section{
					{
						Heading:      "Back Blasphemer!",
						HeadingLevel: 6,
						Annotation:   map[string]string{"type": "ability", "subtype": "signature"},
						BodySource:   "> *You channel power through your weapon.*",
					},
					{
						Heading:      "Halt Miscreant!",
						HeadingLevel: 6,
						Annotation:   map[string]string{"type": "ability", "subtype": "signature"},
						BodySource:   "> *You infuse your weapon with holy magic.*",
					},
				},
			},
			{
				Heading:      "Heroic Abilities",
				HeadingLevel: 5,
				BodySource:   "You call upon a number of heroic abilities.",
				Children: []*parser.Section{
					{
						Heading:      "Repent!",
						HeadingLevel: 6,
						Annotation:   map[string]string{"type": "ability", "cost": "3 Wrath"},
						BodySource:   "> *You demand penance.*",
					},
				},
			},
		},
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "censor"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "level": "1"})

	result, err := (&FeatureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// All sub-headings present
	for _, want := range []string{"Signature Ability", "Heroic Abilities"} {
		if !strings.Contains(result.Body, want) {
			t.Errorf("body should contain sub-heading %q", want)
		}
	}
	// ALL abilities present (not just the first)
	for _, want := range []string{"Back Blasphemer!", "Halt Miscreant!", "Repent!"} {
		if !strings.Contains(result.Body, want) {
			t.Errorf("body should contain ability %q", want)
		}
	}
	// Ability bodies present
	if !strings.Contains(result.Body, "You demand penance.") {
		t.Error("body should contain the last ability's body content")
	}
	// Document order: Back Blasphemer before Halt Miscreant before Repent
	iBack := strings.Index(result.Body, "Back Blasphemer!")
	iHalt := strings.Index(result.Body, "Halt Miscreant!")
	iRepent := strings.Index(result.Body, "Repent!")
	if !(iBack < iHalt && iHalt < iRepent) {
		t.Errorf("abilities out of document order: Back=%d Halt=%d Repent=%d", iBack, iHalt, iRepent)
	}
	// A multi-ability container must NOT embed a single arbitrary ability into
	// the SDK trait schema's singular `ability` field.
	if result.Children != nil {
		if _, ok := result.Children["ability"]; ok {
			t.Error("multi-ability container should not set Children[\"ability\"] (singular embed)")
		}
	}
}

func TestFeatureParser_SingleAbilityTraitStillEmbeds(t *testing.T) {
	// "Faithful Friend" pattern: a trait that grants exactly one ability. It must
	// keep embedding that ability as a structured child (SDK trait schema) and
	// render it inline in the body.
	section := &parser.Section{
		Heading:      "Faithful Friend",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "You have the following ability.",
		Children: []*parser.Section{
			{
				Heading:      "Faithful Friend",
				HeadingLevel: 6,
				Annotation:   map[string]string{"type": "ability"},
				BodySource:   "> *An animal spirit is drawn to you.*",
			},
		},
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "censor"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "level": "1"})

	result, err := (&FeatureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if result.Children == nil {
		t.Fatal("single-ability trait should embed its ability in Children")
	}
	if _, ok := result.Children["ability"]; !ok {
		t.Error("single-ability trait should set Children[\"ability\"]")
	}
	if !strings.Contains(result.Body, "An animal spirit is drawn to you.") {
		t.Error("single-ability trait body should include the embedded ability inline")
	}
	if !strings.Contains(result.Body, "You have the following ability.") {
		t.Error("single-ability trait body should keep its intro prose")
	}
}

func TestFeatureParser_ExcludesAnnotatedSiblings(t *testing.T) {
	// Ensure annotated children are NOT folded into the parent
	section := &parser.Section{
		Heading:      "1st-Level Aspect Features",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "Your primordial aspect grants you two features.",
		Children: []*parser.Section{
			{
				// Unannotated table — should be included
				Heading:      "1st-Level Aspect Features Table",
				HeadingLevel: 6,
				BodySource:   "| Aspect | Feature |\n| Berserker | Kit |",
			},
			{
				// Annotated child — should NOT be included
				Heading:      "Beast Shape",
				HeadingLevel: 5,
				Annotation:   map[string]string{"type": "feature"},
				BodySource:   "You can use a stormwight kit.",
			},
		},
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "fury"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "level": "1"})

	result, err := (&FeatureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(result.Body, "1st-Level Aspect Features Table") {
		t.Error("body should include unannotated table heading")
	}
	if !strings.Contains(result.Body, "| Berserker | Kit |") {
		t.Error("body should include unannotated table content")
	}
	if strings.Contains(result.Body, "Beast Shape") {
		t.Error("body should NOT include annotated child 'Beast Shape'")
	}
	if strings.Contains(result.Body, "stormwight kit") {
		t.Error("body should NOT include annotated child's body content")
	}
}

func TestClassParser_IncludesUnannotatedBasicsSection(t *testing.T) {
	// Class sections often have unannotated sub-headings like ### Basics
	// and ###### Advancement Table that should be included
	section := &parser.Section{
		Heading:      "Fury",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "fury"},
		BodySource:   "The fury is a primal warrior.\n\n**Heroic Resource: Ferocity**",
		Children: []*parser.Section{
			{
				// Unannotated Basics heading
				Heading:      "Basics",
				HeadingLevel: 3,
				BodySource:   "**Starting Characteristics:** Might 2, Agility 2",
				Children: []*parser.Section{
					{
						// Unannotated table under Basics
						Heading:      "Fury Advancement Table",
						HeadingLevel: 6,
						BodySource:   "| Level | Features |\n| 1st | Ferocity |",
					},
				},
			},
			{
				// Annotated feature-group — should NOT be included
				Heading:      "1st-Level Features",
				HeadingLevel: 3,
				Annotation:   map[string]string{"type": "feature-group", "level": "1"},
				BodySource:   "As a 1st-level fury...",
			},
		},
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	result, err := (&ClassParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(result.Body, "### Basics") {
		t.Error("body should include unannotated Basics heading")
	}
	if !strings.Contains(result.Body, "Starting Characteristics") {
		t.Error("body should include Basics content")
	}
	if !strings.Contains(result.Body, "Fury Advancement Table") {
		t.Error("body should include nested unannotated Advancement Table")
	}
	if !strings.Contains(result.Body, "| 1st | Ferocity |") {
		t.Error("body should include Advancement Table content")
	}
	if strings.Contains(result.Body, "1st-Level Features") {
		t.Error("body should NOT include annotated feature-group")
	}

	// Heroic resource should still be extracted from own body
	if result.Frontmatter["heroic_resource"] != "Ferocity" {
		t.Errorf("heroic_resource = %v, want Ferocity", result.Frontmatter["heroic_resource"])
	}
}

func TestAbilityParser_IncludesUnannotatedSubheadings(t *testing.T) {
	// Edge case: an ability with an unannotated sub-section (uncommon but possible)
	body := `*A devastating attack.*

| **Melee, Strike, Weapon** | **Main action** |
| --- | ---: |
| **Melee 1** | **One creature** |

**Power Roll + Might:**
- **≤11:** 4 + M damage
- **12-16:** 7 + M damage
- **17+:** 10 + M damage`

	section := &parser.Section{
		Heading:      "Raging Blow",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability"},
		BodySource:   body,
		Children: []*parser.Section{
			{
				Heading:      "Raging Blow Enhancement Table",
				HeadingLevel: 6,
				BodySource:   "| Level | Bonus |\n| 5 | +2 damage |",
			},
		},
	}

	ctx := context.NewContextStack(context.Metadata{})
	ctx.Push(2, context.Metadata{"type": "class", "id": "fury"})

	result, err := (&AbilityParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(result.Body, "Raging Blow Enhancement Table") {
		t.Error("ability body should include unannotated sub-heading")
	}
	if !strings.Contains(result.Body, "+2 damage") {
		t.Error("ability body should include unannotated sub-heading content")
	}

	// Core ability extraction should still work
	if result.Frontmatter["power_roll_characteristic"] != "Might" {
		t.Errorf("power_roll_characteristic = %v, want Might", result.Frontmatter["power_roll_characteristic"])
	}
}

func TestKitParser_IncludesUnannotatedSubheadings(t *testing.T) {
	section := &parser.Section{
		Heading:      "Panther Kit",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "kit"},
		BodySource:   "A swift melee kit.\n\n**Stamina Bonus:** +3\n\n**Speed Bonus:** +2",
		Children: []*parser.Section{
			{
				Heading:      "Panther Kit Bonuses",
				HeadingLevel: 5,
				BodySource:   "Additional movement benefits at higher levels.",
			},
		},
	}

	ctx := context.NewContextStack(nil)
	result, err := (&KitParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(result.Body, "Panther Kit Bonuses") {
		t.Error("kit body should include unannotated sub-heading")
	}
	if !strings.Contains(result.Body, "Additional movement benefits") {
		t.Error("kit body should include unannotated sub-heading content")
	}

	// Individual bonus field extraction should still work from own body
	if result.Frontmatter["stamina_bonus"] != "+3" {
		t.Errorf("stamina_bonus = %v, want +3", result.Frontmatter["stamina_bonus"])
	}
}

func TestChapterParser_IncludesUnannotatedSubheadings(t *testing.T) {
	section := &parser.Section{
		Heading:      "Introduction",
		HeadingLevel: 1,
		Annotation:   map[string]string{"type": "chapter", "id": "intro"},
		BodySource:   "Welcome to Draw Steel.",
		Children: []*parser.Section{
			{
				Heading:      "How to Use This Book",
				HeadingLevel: 2,
				BodySource:   "Read the chapters in order.",
			},
			{
				// Annotated child should be excluded
				Heading:      "Classes",
				HeadingLevel: 2,
				Annotation:   map[string]string{"type": "chapter", "id": "classes"},
				BodySource:   "A hero's class...",
			},
		},
	}

	ctx := context.NewContextStack(nil)
	result, err := (&ChapterParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(result.Body, "How to Use This Book") {
		t.Error("chapter body should include unannotated sub-heading")
	}
	if strings.Contains(result.Body, "A hero's class") {
		t.Error("chapter body should NOT include annotated child")
	}
}

func TestComplicationParser_IncludesUnannotatedSubheadings(t *testing.T) {
	section := &parser.Section{
		Heading:      "Haunted",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "complication"},
		BodySource:   "A spirit follows you.",
		Children: []*parser.Section{
			{
				Heading:      "Haunting Effects Table",
				HeadingLevel: 6,
				BodySource:   "| Roll | Effect |\n| 1 | Chills |",
			},
		},
	}

	ctx := context.NewContextStack(nil)
	result, err := (&ComplicationParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(result.Body, "Haunting Effects Table") {
		t.Error("complication body should include unannotated sub-heading")
	}
}

func TestConditionParser_IncludesUnannotatedSubheadings(t *testing.T) {
	section := &parser.Section{
		Heading:      "Burning",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "condition"},
		BodySource:   "The target is on fire.",
		Children: []*parser.Section{
			{
				Heading:      "Burning Severity",
				HeadingLevel: 6,
				BodySource:   "| Severity | Damage |\n| Minor | 2 fire |",
			},
		},
	}

	ctx := context.NewContextStack(nil)
	result, err := (&ConditionParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	if !strings.Contains(result.Body, "Burning Severity") {
		t.Error("condition body should include unannotated sub-heading")
	}
}
