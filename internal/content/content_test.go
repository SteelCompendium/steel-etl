package content

import (
	"os"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
	"github.com/SteelCompendium/steel-etl/internal/scc"
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
	// fury is a class → plain feature: feature.fury.level-1 (no trait segment)
	expected := []string{"feature", "fury", "level-1"}
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

func TestFeatureParser_TaxonomyPaths(t *testing.T) {
	cases := []struct {
		name     string
		homeType string // "class" | "ancestry" | "companion"
		homeID   string
		wantType string   // fm["type"]
		wantPath []string // full TypePath
	}{
		{"class feature is plain feature", "class", "shadow", "feature", []string{"feature", "shadow"}},
		{"ancestry feature is trait", "ancestry", "dwarf", "trait", []string{"feature", "trait", "dwarf"}},
		{"companion feature carries class segment", "companion", "wolf", "feature", []string{"feature", "companion", "beastheart", "wolf"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
			if tc.homeType == "companion" {
				ctx.Push(1, context.Metadata{"type": "class", "id": "beastheart"})
				ctx.Push(2, context.Metadata{"type": "feature-group", "companion": tc.homeID})
			} else {
				ctx.Push(2, context.Metadata{"type": tc.homeType, "id": tc.homeID})
			}
			section := &parser.Section{
				Heading:      "Sample Feature",
				HeadingLevel: 3,
				Annotation:   map[string]string{"type": "feature"},
			}
			p := &FeatureParser{}
			got, err := p.Parse(ctx, section)
			if err != nil {
				t.Fatalf("Parse failed: %v", err)
			}
			if got.Frontmatter["type"] != tc.wantType {
				t.Errorf("type = %v, want %v", got.Frontmatter["type"], tc.wantType)
			}
			if len(got.TypePath) != len(tc.wantPath) {
				t.Fatalf("TypePath = %v, want %v", got.TypePath, tc.wantPath)
			}
			for i, seg := range tc.wantPath {
				if got.TypePath[i] != seg {
					t.Errorf("TypePath = %v, want %v", got.TypePath, tc.wantPath)
					break
				}
			}
		})
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

// The SCC linking sweep wraps the power-roll header and characteristics in
// scc: links ("**[Power Roll](scc:…) + [Might](scc:…) or [Agility](scc:…):**").
// extractPowerRoll must still detect the header (so tier1/2/3 are populated) and
// capture the multi-characteristic expression verbatim (links kept, like the
// sibling effect/distance fields). Without this, buildAbilityEffects silently
// drops the entire power-roll effect from JSON/YAML/DSE output.
func TestAbilityParserLinkedPowerRoll(t *testing.T) {
	body := `*The strength of your assault makes it impossible to ignore you.*

**[Power Roll](scc:mcdm.heroes.v1/rule.dice/power-roll) + [Might](scc:mcdm.heroes.v1/rule.character/might) or [Agility](scc:mcdm.heroes.v1/rule.character/agility):**
- **≤11:** 5 + M or A damage
- **12-16:** 8 + M or A damage
- **17+:** 11 + M or A damage

**Effect:** The target is taunted.`

	section := &parser.Section{
		Heading:      "Protective Attack",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability", "subtype": "signature"},
		BodySource:   body,
	}

	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "tactician"})

	p := &AbilityParser{}
	result, err := p.Parse(ctx, section)
	if err != nil {
		t.Fatalf("AbilityParser.Parse failed: %v", err)
	}
	fm := result.Frontmatter

	// Characteristic captured verbatim (multi-char + links kept, matching effect/distance).
	wantChar := "[Might](scc:mcdm.heroes.v1/rule.character/might) or [Agility](scc:mcdm.heroes.v1/rule.character/agility)"
	if fm["power_roll_characteristic"] != wantChar {
		t.Errorf("power_roll_characteristic: got %v, want %q", fm["power_roll_characteristic"], wantChar)
	}
	if fm["tier1"] != "5 + M or A damage" {
		t.Errorf("tier1: got %v", fm["tier1"])
	}
	if fm["tier2"] != "8 + M or A damage" {
		t.Errorf("tier2: got %v", fm["tier2"])
	}
	if fm["tier3"] != "11 + M or A damage" {
		t.Errorf("tier3: got %v", fm["tier3"])
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

func TestAbilityCompanionTypePath(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.beastheart.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "beastheart"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "companion": "wolf", "level": "1"})

	section := &parser.Section{
		Heading:      "Clamping Jaws",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability", "subtype": "signature", "id": "clamping-jaws"},
	}
	parsed, err := (&AbilityParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := scc.Classify("mcdm.beastheart.v1", parsed.TypePath, parsed.ItemID)
	want := "mcdm.beastheart.v1/feature.ability.companion.beastheart.wolf.level-1/clamping-jaws"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
	if parsed.Frontmatter["companion"] != "wolf" {
		t.Errorf("companion frontmatter = %v, want wolf", parsed.Frontmatter["companion"])
	}
}

func TestFeatureCompanionTypePath(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.beastheart.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "beastheart"})
	ctx.Push(3, context.Metadata{"type": "feature-group", "companion": "wolf", "level": "1"})
	ctx.Push(4, context.Metadata{"type": "feature-group", "level": "3"})

	section := &parser.Section{
		Heading:      "My, What Big Teeth You Have",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "feature", "id": "my-what-big-teeth-you-have"},
	}
	parsed, err := (&FeatureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := scc.Classify("mcdm.beastheart.v1", parsed.TypePath, parsed.ItemID)
	want := "mcdm.beastheart.v1/feature.companion.beastheart.wolf.level-3/my-what-big-teeth-you-have"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFeatureGroupCompanionClassified(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.beastheart.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "beastheart"})

	section := &parser.Section{
		Heading:      "Wolf",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature-group", "companion": "wolf", "level": "1"},
	}
	parsed, err := (&FeatureGroupParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got := scc.Classify("mcdm.beastheart.v1", parsed.TypePath, parsed.ItemID)
	want := "mcdm.beastheart.v1/monster.companion.beastheart.statblock/wolf"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFeatureGroupPlainUnclassified(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{})
	section := &parser.Section{
		Heading:      "1st-Level Features",
		HeadingLevel: 3,
		Annotation:   map[string]string{"type": "feature-group", "level": "1"},
	}
	parsed, err := (&FeatureGroupParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if len(parsed.TypePath) != 0 || parsed.ItemID != "" {
		t.Errorf("plain feature-group should be unclassified, got path=%v id=%q", parsed.TypePath, parsed.ItemID)
	}
}

// Subclass is surfaced as a frontmatter field only; it never changes the SCC path
// (the SCC code is a stable reference identifier). Single subclass -> string,
// comma-separated -> list, absent -> field omitted.
func TestAbilitySubclassFrontmatter(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.beastheart.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "beastheart"})

	section := &parser.Section{
		Heading:      "Sic 'Em!",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability", "id": "sic-em", "level": "6", "cost": "9 Ferocity", "subclass": "guardian"},
	}
	ctx.Push(section.HeadingLevel, context.Metadata(section.Annotation)) // mirror collect.go
	parsed, err := (&AbilityParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Frontmatter["subclass"] != "guardian" {
		t.Errorf("subclass frontmatter = %v, want guardian", parsed.Frontmatter["subclass"])
	}
	// Path must NOT contain the subclass — stable reference, level-gated only.
	got := scc.Classify("mcdm.beastheart.v1", parsed.TypePath, parsed.ItemID)
	want := "mcdm.beastheart.v1/feature.ability.beastheart.level-6/sic-em"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestFeatureSubclassFrontmatter(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.beastheart.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "beastheart"})

	section := &parser.Section{
		Heading:      "Stormheart",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "feature", "id": "stormheart", "level": "2", "subclass": "spark"},
	}
	ctx.Push(section.HeadingLevel, context.Metadata(section.Annotation))
	parsed, err := (&FeatureParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if parsed.Frontmatter["subclass"] != "spark" {
		t.Errorf("subclass frontmatter = %v, want spark", parsed.Frontmatter["subclass"])
	}
	got := scc.Classify("mcdm.beastheart.v1", parsed.TypePath, parsed.ItemID)
	want := "mcdm.beastheart.v1/feature.beastheart.level-2/stormheart"
	if got != want {
		t.Errorf("got %q, want %q", got, want)
	}
}

func TestAbilityMultiSubclass(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.beastheart.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "beastheart"})

	section := &parser.Section{
		Heading:      "Hypothetical Multi",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability", "id": "hypothetical-multi", "level": "1", "subclass": "guardian, prowler"},
	}
	ctx.Push(section.HeadingLevel, context.Metadata(section.Annotation))
	parsed, err := (&AbilityParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	got, ok := parsed.Frontmatter["subclass"].([]string)
	if !ok {
		t.Fatalf("subclass = %T %v, want []string", parsed.Frontmatter["subclass"], parsed.Frontmatter["subclass"])
	}
	if len(got) != 2 || got[0] != "guardian" || got[1] != "prowler" {
		t.Errorf("subclass = %v, want [guardian prowler]", got)
	}
}

func TestAbilityNoSubclass(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.beastheart.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "beastheart"})

	section := &parser.Section{
		Heading:      "Bodyswap",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability", "subtype": "signature", "id": "bodyswap"},
	}
	ctx.Push(section.HeadingLevel, context.Metadata(section.Annotation))
	parsed, err := (&AbilityParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if _, exists := parsed.Frontmatter["subclass"]; exists {
		t.Errorf("subclass should be absent, got %v", parsed.Frontmatter["subclass"])
	}
}

func TestMonsterParsersRegistered(t *testing.T) {
	r := NewRegistry()
	for _, typeName := range []string{"monster", "monster-group", "statblock", "featureblock", "dynamic-terrain"} {
		if !r.Has(typeName) {
			t.Errorf("parser %q not registered", typeName)
		}
	}
}

func TestFeatureblockCompanionAdvancement(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.beastheart.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "beastheart"})
	ctx.Push(4, context.Metadata{"type": "feature-group", "companion": "wolf"})

	adv := &parser.Section{
		Heading:      "Wolf Advancement Features",
		HeadingLevel: 5,
		Annotation:   map[string]string{"type": "featureblock"},
		Children: []*parser.Section{
			{Heading: "My, What Big Teeth You Have", HeadingLevel: 6,
				Annotation: map[string]string{"type": "feature", "id": "my-what-big-teeth-you-have", "level": "3"},
				BodySource: "Whenever the wolf makes a strike..."},
			{Heading: "Dire Wolf", HeadingLevel: 6,
				Annotation: map[string]string{"type": "feature", "id": "dire-wolf", "level": "10"},
				BodySource: "While the wolf is rampaging..."},
		},
	}
	got, err := (&FeatureblockParser{}).Parse(ctx, adv)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	code := scc.Classify("mcdm.beastheart.v1", got.TypePath, got.ItemID)
	if want := "mcdm.beastheart.v1/monster.companion.beastheart.advancement-features/wolf"; code != want {
		t.Errorf("code = %q, want %q", code, want)
	}
	if got.Frontmatter["type"] != "featureblock" {
		t.Errorf("type = %v, want featureblock", got.Frontmatter["type"])
	}
	feats, ok := got.Frontmatter["features"].([]map[string]any)
	if !ok || len(feats) != 2 {
		t.Fatalf("features = %v", got.Frontmatter["features"])
	}
	if feats[0]["name"] != "My, What Big Teeth You Have" || feats[0]["level"] != 3 {
		t.Errorf("feat[0] = %v", feats[0])
	}
	if feats[1]["level"] != 10 {
		t.Errorf("feat[1] level = %v, want 10", feats[1]["level"])
	}
}

// Task 1: fixture statblock → monster.fixture.<element>.featureblock
// Context mirrors the real source: H5 monster-group with domain=fixture/category=demon
// wraps an H7 statblock (capped to level 6 by the parser).
func TestStatblockFixtureFeatureblock(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.summoner.v1"})
	// H5 monster-group sets fixture domain + demon category (level 5)
	ctx.Push(5, context.Metadata{"type": "monster-group", "domain": "fixture", "category": "demon"})

	// The boil body: italic role, 2-col stamina/size grid, two base features
	body := `*Hazard Support*

| **Stamina:** 20 + your level | **Size:** 2 |
|------------------------------|------------:|

> ⭐️ **Hunger Thrush**
>
> Each enemy that starts their turn within 3 squares of the boil is taunted (EoT).

> ⭐️ **Oh, It Pops**
>
> When the boil is destroyed, each enemy within 3 squares takes acid damage.`

	section := &parser.Section{
		Heading:      "The Boil",
		HeadingLevel: 6, // H7 capped to 6
		Annotation:   map[string]string{"type": "statblock"},
		BodySource:   body,
	}

	got, err := (&StatblockParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Must classify as monster.fixture.demon.featureblock/the-boil
	code := scc.Classify("mcdm.summoner.v1", got.TypePath, got.ItemID)
	want := "mcdm.summoner.v1/monster.fixture.demon.featureblock/the-boil"
	if code != want {
		t.Errorf("code = %q, want %q", code, want)
	}

	if got.Frontmatter["type"] != "featureblock" {
		t.Errorf("type = %v, want featureblock", got.Frontmatter["type"])
	}

	// statblock_kind must not be emitted
	if _, ok := got.Frontmatter["statblock_kind"]; ok {
		t.Errorf("statblock_kind should not be present for fixture featureblock")
	}

	// stats[] must contain Stamina and Size
	stats, ok := got.Frontmatter["stats"].([]map[string]any)
	if !ok || len(stats) == 0 {
		t.Fatalf("stats = %v, want [{name:Stamina,...},{name:Size,...}]", got.Frontmatter["stats"])
	}
	foundStamina, foundSize := false, false
	for _, s := range stats {
		switch s["name"] {
		case "Stamina":
			foundStamina = true
		case "Size":
			foundSize = true
		}
	}
	if !foundStamina {
		t.Error("stats[] missing Stamina entry")
	}
	if !foundSize {
		t.Error("stats[] missing Size entry")
	}

	// features[] must contain the base (Level-0) features
	feats, ok := got.Frontmatter["features"].([]map[string]any)
	if !ok || len(feats) == 0 {
		t.Fatalf("features = %v, want at least one base feature", got.Frontmatter["features"])
	}
	foundBase := false
	for _, f := range feats {
		if f["name"] == "Hunger Thrush" {
			foundBase = true
		}
	}
	if !foundBase {
		t.Error("features[] missing base feature 'Hunger Thrush'")
	}
}

// Task 3: fixture advancement-features featureblock
// The sibling section (post source-split) carries Level-5/9 advancement features.
func TestFeatureblockFixtureAdvancement(t *testing.T) {
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.summoner.v1"})
	// H5 monster-group sets fixture domain + demon category (level 5)
	ctx.Push(5, context.Metadata{"type": "monster-group", "domain": "fixture", "category": "demon"})

	body := `> **Level 5 Fixture Advancement Feature**
>
> ⭐️ **Soul Rancor**
>
> You gain a surge the first time in a round that your demon minions deal 3 or more damage.

> **Level 9 Fixture Advancement Feature**
>
> ⭐️ **Size Increase**
>
> The boil is now size 3.
>
> ⭐️ **Fester Field**
>
> Each non-abyssal enemy that starts their turn within 3 squares takes 5 corruption damage.`

	section := &parser.Section{
		Heading:      "The Boil Advancement Features",
		HeadingLevel: 6, // H7 capped to 6, sibling of the base statblock
		Annotation:   map[string]string{"type": "featureblock", "id": "the-boil"},
		BodySource:   body,
	}

	got, err := (&FeatureblockParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}

	// Must classify as monster.fixture.demon.advancement-features/the-boil
	code := scc.Classify("mcdm.summoner.v1", got.TypePath, got.ItemID)
	want := "mcdm.summoner.v1/monster.fixture.demon.advancement-features/the-boil"
	if code != want {
		t.Errorf("code = %q, want %q", code, want)
	}

	if got.Frontmatter["type"] != "featureblock" {
		t.Errorf("type = %v, want featureblock", got.Frontmatter["type"])
	}

	// features[] must carry Level-5 and Level-9 features
	feats, ok := got.Frontmatter["features"].([]map[string]any)
	if !ok || len(feats) == 0 {
		t.Fatalf("features = %v, want advancement features", got.Frontmatter["features"])
	}

	foundLevel5, foundLevel9 := false, false
	for _, f := range feats {
		switch f["level"] {
		case 5:
			foundLevel5 = true
		case 9:
			foundLevel9 = true
		}
	}
	if !foundLevel5 {
		t.Error("features[] missing Level-5 advancement feature")
	}
	if !foundLevel9 {
		t.Error("features[] missing Level-9 advancement feature")
	}
}
