package content

import (
	"os"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestChapterParser(t *testing.T) {
	section := &parser.Section{
		Heading:      "Classes",
		HeadingLevel: 1,
		Annotation:   map[string]string{"type": "chapter", "id": "classes"},
		BodySource:   "A hero's class determines their role in combat.",
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	p := &ChapterParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("ChapterParser.Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Classes" {
		t.Errorf("expected name=Classes, got %v", result.Frontmatter["name"])
	}
	if result.ItemID != "classes" {
		t.Errorf("expected itemID=classes, got %s", result.ItemID)
	}
	if len(result.TypePath) != 1 || result.TypePath[0] != "chapter" {
		t.Errorf("expected TypePath=[chapter], got %v", result.TypePath)
	}
}

func TestChapterParserSlugifiedID(t *testing.T) {
	section := &parser.Section{
		Heading:      "Character Creation",
		HeadingLevel: 1,
		Annotation:   map[string]string{"type": "chapter"},
		BodySource:   "How to create a character.",
	}

	ctx := context.NewContextStack(context.Metadata{})
	p := &ChapterParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}
	if result.ItemID != "character-creation" {
		t.Errorf("expected itemID=character-creation, got %s", result.ItemID)
	}
}

func TestClassParser(t *testing.T) {
	section := &parser.Section{
		Heading:      "Fury",
		HeadingLevel: 2,
		Annotation:   map[string]string{"type": "class", "id": "fury"},
		BodySource:   "The fury is a primal warrior.\n\n**Heroic Resource: Ferocity**",
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	p := &ClassParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("ClassParser.Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Fury" {
		t.Errorf("expected name=Fury, got %v", result.Frontmatter["name"])
	}
	if result.Frontmatter["heroic_resource"] != "Ferocity" {
		t.Errorf("expected heroic_resource=Ferocity, got %v", result.Frontmatter["heroic_resource"])
	}
	if result.ItemID != "fury" {
		t.Errorf("expected itemID=fury, got %s", result.ItemID)
	}
	if len(result.TypePath) != 1 || result.TypePath[0] != "class" {
		t.Errorf("expected TypePath=[class], got %v", result.TypePath)
	}
}

func TestFeatureGroupParser(t *testing.T) {
	section := &parser.Section{
		Heading:      "1st-Level Features",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "feature-group", "level": "1"},
		BodySource:   "As a 1st-level fury, you gain the following features.",
	}

	ctx := context.NewContextStack(context.Metadata{})
	p := &FeatureGroupParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("FeatureGroupParser.Parse failed: %v", err)
	}

	if result.Frontmatter["level"] != "1" {
		t.Errorf("expected level=1, got %v", result.Frontmatter["level"])
	}
	// feature-group has no SCC
	if result.TypePath != nil {
		t.Errorf("expected nil TypePath for feature-group, got %v", result.TypePath)
	}
}

func TestFeatureParser(t *testing.T) {
	section := &parser.Section{
		Heading:      "Growing Ferocity",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature"},
		BodySource:   "At the start of each of your turns during combat, you gain 1d3 ferocity.",
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "fury"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "level": "1"})

	p := &FeatureParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("FeatureParser.Parse failed: %v", err)
	}

	if result.Frontmatter["name"] != "Growing Ferocity" {
		t.Errorf("expected name=Growing Ferocity, got %v", result.Frontmatter["name"])
	}
	if result.Frontmatter["class"] != "fury" {
		t.Errorf("expected class=fury, got %v", result.Frontmatter["class"])
	}
	if result.ItemID != "growing-ferocity" {
		t.Errorf("expected itemID=growing-ferocity, got %s", result.ItemID)
	}
	// feature.trait.fury.level-1
	expected := []string{"feature", "trait", "fury", "level-1"}
	if len(result.TypePath) != len(expected) {
		t.Errorf("expected TypePath=%v, got %v", expected, result.TypePath)
	} else {
		for i, v := range expected {
			if result.TypePath[i] != v {
				t.Errorf("expected TypePath=%v, got %v", expected, result.TypePath)
				break
			}
		}
	}
}

func TestAbilityParserBasic(t *testing.T) {
	body := `*You slam your weapon into a foe with awesome might.*

| **Melee, Strike, Weapon** | **Main action** |
| --- | ---: |
| **Melee 1** | **One creature** |

**Power Roll + Might:**
- **≤11:** 4 + M damage
- **12-16:** 7 + M damage; push 1
- **17+:** 10 + M damage; push 3

**Effect:** You can shift 1 after this attack.`

	section := &parser.Section{
		Heading:      "Brutal Slam",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability", "subtype": "signature"},
		BodySource:   body,
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "fury"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "level": "1"})

	p := &AbilityParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("AbilityParser.Parse failed: %v", err)
	}

	fm := result.Frontmatter
	if fm["name"] != "Brutal Slam" {
		t.Errorf("name: got %v", fm["name"])
	}
	if fm["subtype"] != "signature" {
		t.Errorf("subtype: got %v", fm["subtype"])
	}
	if fm["flavor"] != "You slam your weapon into a foe with awesome might." {
		t.Errorf("flavor: got %v", fm["flavor"])
	}
	if fm["action_type"] != "Main action" {
		t.Errorf("action_type: got %v", fm["action_type"])
	}
	if fm["distance"] != "Melee 1" {
		t.Errorf("distance: got %v", fm["distance"])
	}
	if fm["target"] != "One creature" {
		t.Errorf("target: got %v", fm["target"])
	}
	if fm["power_roll_characteristic"] != "Might" {
		t.Errorf("power_roll_characteristic: got %v", fm["power_roll_characteristic"])
	}
	if fm["tier1"] != "4 + M damage" {
		t.Errorf("tier1: got %v", fm["tier1"])
	}
	if fm["tier2"] != "7 + M damage; push 1" {
		t.Errorf("tier2: got %v", fm["tier2"])
	}
	if fm["tier3"] != "10 + M damage; push 3" {
		t.Errorf("tier3: got %v", fm["tier3"])
	}
	if fm["effect"] != "You can shift 1 after this attack." {
		t.Errorf("effect: got %v", fm["effect"])
	}
	if fm["class"] != "fury" {
		t.Errorf("class: got %v", fm["class"])
	}
	if fm["level"] != "1" {
		t.Errorf("level: got %v", fm["level"])
	}

	kw, ok := fm["keywords"].([]string)
	if !ok || len(kw) != 3 || kw[0] != "Melee" || kw[1] != "Strike" || kw[2] != "Weapon" {
		t.Errorf("keywords: got %v", fm["keywords"])
	}

	if result.ItemID != "brutal-slam" {
		t.Errorf("itemID: got %s", result.ItemID)
	}
	// feature.ability.fury.level-1
	expectedTP := []string{"feature", "ability", "fury", "level-1"}
	if len(result.TypePath) != len(expectedTP) {
		t.Errorf("TypePath: expected %v, got %v", expectedTP, result.TypePath)
	} else {
		for i, v := range expectedTP {
			if result.TypePath[i] != v {
				t.Errorf("TypePath: expected %v, got %v", expectedTP, result.TypePath)
				break
			}
		}
	}
}

func TestAbilityParserBlockquoteBody(t *testing.T) {
	// Test with blockquote-prefixed body (as it appears in real data)
	body := `> *You channel power through your weapon to repel foes.*
>
> | **Area, Magic, Melee, Weapon** |               **Main action** |
> |--------------------------------|------------------------------:|
> | **📏 2 cube within 1**         | **🎯 Each enemy in the area** |
>
> **Power Roll + Presence:**
>
> - **≤11:** 2 holy damage; push 1
> - **12-16:** 4 holy damage; push 2
> - **17+:** 6 holy damage; push 3`

	section := &parser.Section{
		Heading:      "Back Blasphemer!",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability", "subtype": "signature", "id": "back-blasphemer"},
		BodySource:   body,
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "censor"})

	p := &AbilityParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("AbilityParser.Parse failed: %v", err)
	}

	fm := result.Frontmatter
	kw, ok := fm["keywords"].([]string)
	if !ok || len(kw) != 4 {
		t.Errorf("keywords: expected 4, got %v", fm["keywords"])
	}
	if fm["distance"] != "2 cube within 1" {
		t.Errorf("distance: got %v", fm["distance"])
	}
	if fm["target"] != "Each enemy in the area" {
		t.Errorf("target: got %v", fm["target"])
	}
	if fm["power_roll_characteristic"] != "Presence" {
		t.Errorf("power_roll: got %v", fm["power_roll_characteristic"])
	}
	if result.ItemID != "back-blasphemer" {
		t.Errorf("itemID: got %s", result.ItemID)
	}
}

func TestAbilityParserWithCost(t *testing.T) {
	body := `*Your sharp claws tear into your foe.*

| **Melee, Strike, Weapon** | **Main action** |
| --- | ---: |
| **Melee 1** | **One creature** |

**Power Roll + Might:**
- **≤11:** 4 + M damage
- **12-16:** 7 + M damage
- **17+:** 10 + M damage`

	section := &parser.Section{
		Heading:      "Gouge",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability", "cost": "3 Ferocity"},
		BodySource:   body,
	}

	ctx := context.NewContextStack(context.Metadata{})
	ctx.Push(2, context.Metadata{"type": "class", "id": "fury"})

	p := &AbilityParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("AbilityParser.Parse failed: %v", err)
	}

	if result.Frontmatter["cost"] != "3 Ferocity" {
		t.Errorf("cost: got %v", result.Frontmatter["cost"])
	}
}

func TestAbilityParserCommonAbility(t *testing.T) {
	// An ability without a class parent gets abilities.common
	section := &parser.Section{
		Heading:      "Grab",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability"},
		BodySource:   "Some body",
	}

	ctx := context.NewContextStack(context.Metadata{})
	// No class in context

	p := &AbilityParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("AbilityParser.Parse failed: %v", err)
	}

	// feature.ability.common (no level in context)
	if len(result.TypePath) != 3 || result.TypePath[0] != "feature" || result.TypePath[1] != "ability" || result.TypePath[2] != "common" {
		t.Errorf("expected TypePath=[feature, ability, common], got %v", result.TypePath)
	}
}

func TestRegistryGetAndHas(t *testing.T) {
	r := NewRegistry()

	if !r.Has("ability") {
		t.Error("expected registry to have 'ability'")
	}
	if !r.Has("class") {
		t.Error("expected registry to have 'class'")
	}
	if !r.Has("chapter") {
		t.Error("expected registry to have 'chapter'")
	}
	if !r.Has("feature") {
		t.Error("expected registry to have 'feature'")
	}
	if !r.Has("feature-group") {
		t.Error("expected registry to have 'feature-group'")
	}
	if r.Has("nonexistent") {
		t.Error("expected registry to NOT have 'nonexistent'")
	}

	_, err := r.Get("nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent type")
	}
}

func TestSmokeContentParsersOnRealDocument(t *testing.T) {
	path := "../../input/heroes/Draw Steel Heroes.md"
	data, err := os.ReadFile(path)
	if err != nil {
		t.Skipf("skipping smoke test: %v", err)
	}

	doc, err := parser.ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument failed: %v", err)
	}

	registry := NewRegistry()
	ctx := context.NewContextStack(context.Metadata{})
	if book, ok := doc.Frontmatter["book"]; ok {
		if bookStr, ok := book.(string); ok {
			ctx = context.NewContextStack(context.Metadata{"book": bookStr})
		}
	}

	var parsed, skipped, failed int
	var failedSections []string

	var walk func(sections []*parser.Section)
	walk = func(sections []*parser.Section) {
		for _, s := range sections {
			if s.Annotation != nil {
				typeName := s.Type()
				if typeName != "" {
					// Update context stack
					ctx.Push(s.HeadingLevel, context.Metadata(s.Annotation))

					if registry.Has(typeName) {
						p, _ := registry.Get(typeName)
						_, err := p.Parse(ctx, s)
						if err != nil {
							failed++
							if len(failedSections) < 10 {
								failedSections = append(failedSections, s.Heading+": "+err.Error())
							}
						} else {
							parsed++
						}
					} else {
						skipped++
					}
				}
			}
			walk(s.Children)
		}
	}
	walk(doc.Sections)

	t.Logf("Parsed: %d, Skipped (no parser): %d, Failed: %d", parsed, skipped, failed)
	for _, f := range failedSections {
		t.Logf("  FAILED: %s", f)
	}

	if failed > 0 {
		t.Errorf("%d sections failed to parse", failed)
	}
	if parsed < 1000 {
		t.Errorf("expected >1000 parsed sections, got %d", parsed)
	}
}
