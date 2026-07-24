package content

import (
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// effectsList pulls the parsed, ordered named-effect list off ability frontmatter.
func effectsList(t *testing.T, fm map[string]any) []map[string]any {
	t.Helper()
	raw, ok := fm["effects"].([]map[string]any)
	if !ok {
		t.Fatalf("expected fm[\"effects\"] to be []map[string]any, got %T (%v)", fm["effects"], fm["effects"])
	}
	return raw
}

func parseAbility(t *testing.T, body string) map[string]any {
	t.Helper()
	section := &parser.Section{
		Heading:      "Test Ability",
		HeadingLevel: 4,
		Annotation:   map[string]string{"type": "ability"},
		BodySource:   body,
	}
	ctx := context.NewContextStack(context.Metadata{"book": "mcdm.heroes.v1"})
	ctx.Push(2, context.Metadata{"type": "class", "id": "talent"})
	result, err := (&AbilityParser{}).Parse(ctx, section)
	if err != nil {
		t.Fatalf("AbilityParser.Parse failed: %v", err)
	}
	return result.Frontmatter
}

// Minor Telekinesis has two "Spend X" effects; both must survive (bug: only the
// first was kept).
func TestAbilityParser_MultipleSpends(t *testing.T) {
	body := `*Wisps of psychic energy ripple visibly from your brain.*

| **Psionic, Ranged** | **Maneuver** |
| --- | ---: |
| **Ranged 10** | **Self or one creature or object** |

**Effect:** You slide the target up to a number of squares equal to your Reason score.

**Spend 2+ Clarity:** The size of the creature or object you can target increases by 1 for every 2 clarity spent.

**Spend 3 Clarity:** You can vertical slide the target.`

	effects := effectsList(t, parseAbility(t, body))
	if len(effects) != 3 {
		t.Fatalf("expected 3 effects (Effect + 2 Spends), got %d: %v", len(effects), effects)
	}
	if effects[0]["name"] != "Effect" {
		t.Errorf("effects[0] name: got %v", effects[0]["name"])
	}
	if effects[1]["cost"] != "Spend 2+ Clarity" {
		t.Errorf("effects[1] cost: got %v", effects[1]["cost"])
	}
	if effects[2]["cost"] != "Spend 3 Clarity" {
		t.Errorf("effects[2] cost: got %v", effects[2]["cost"])
	}
	if effects[2]["effect"] != "You can vertical slide the target." {
		t.Errorf("effects[2] effect: got %v", effects[2]["effect"])
	}
}

// Conflagration's only rider is "Persistent 2" — a named effect that is neither
// "Effect" nor "Spend X". It must be captured (bug: dropped entirely).
func TestAbilityParser_NamedEffectPersistent(t *testing.T) {
	body := `*A storm of fire descends upon your enemies.*

**Power Roll + Reason:**
- **≤11:** 4 fire damage
- **12-16:** 6 fire damage
- **17+:** 10 fire damage

**Persistent 2:** At the start of your turn, you can use a maneuver to use this ability again without spending essence.`

	effects := effectsList(t, parseAbility(t, body))
	if len(effects) != 2 {
		t.Fatalf("expected 2 effects (roll then Persistent 2), got %d: %v", len(effects), effects)
	}
	if effects[0]["roll"] != "Power Roll + Reason" {
		t.Errorf("effects[0] should be the power roll, got %v", effects[0])
	}
	if effects[1]["name"] != "Persistent 2" {
		t.Errorf("effects[1] name: got %v", effects[1]["name"])
	}
	if effects[1]["effect"] != "At the start of your turn, you can use a maneuver to use this ability again without spending essence." {
		t.Errorf("effects[1] effect: got %v", effects[1]["effect"])
	}
}

// Hoarfrost's only rider is "Strained" — again neither "Effect" nor "Spend X".
func TestAbilityParser_NamedEffectStrained(t *testing.T) {
	body := `*You blast a foe with a pulse of cold energy.*

**Power Roll + Reason:**
- **≤11:** 2 + R cold damage
- **12-16:** 4 + R cold damage
- **17+:** 6 + R cold damage

**Strained:** You are slowed until the end of your next turn.`

	effects := effectsList(t, parseAbility(t, body))
	if len(effects) != 2 {
		t.Fatalf("expected 2 effects (roll then Strained), got %d: %v", len(effects), effects)
	}
	if effects[0]["roll"] != "Power Roll + Reason" {
		t.Errorf("effects[0] should be the power roll, got %v", effects[0])
	}
	if effects[1]["name"] != "Strained" {
		t.Errorf("effects[1] name: got %v", effects[1]["name"])
	}
}

// The effects array must mirror document order, including where the power roll
// sits. Instantaneous Excavation states its Effect BEFORE the power roll, so the
// array must be [Effect, roll], not [roll, Effect].
func TestAbilityParser_EffectBeforePowerRollOrder(t *testing.T) {
	body := `*The surface of the world opens up to swallow foes.*

| **Earth, Magic** | **Maneuver** |
| --- | ---: |
| **Ranged 10** | **Special** |

**Effect:** You open up two holes with 1-square openings.

**Power Roll + Reason:**
- **≤11:** The target shifts 1 square.
- **12-16:** The target falls in.
- **17+:** The target falls in and is restrained.`

	effects := effectsList(t, parseAbility(t, body))
	if len(effects) != 2 {
		t.Fatalf("expected 2 effects (Effect then roll), got %d: %v", len(effects), effects)
	}
	if effects[0]["name"] != "Effect" {
		t.Errorf("effects[0] should be the Effect (document order), got %v", effects[0])
	}
	if effects[1]["roll"] != "Power Roll + Reason" {
		t.Errorf("effects[1] should be the power roll, got %v", effects[1])
	}
	if effects[1]["tier1"] != "The target shifts 1 square." {
		t.Errorf("effects[1] tier1: got %v", effects[1]["tier1"])
	}
}

// The Trigger line stays a top-level field and must NOT be duplicated into the
// effects array.
func TestAbilityParser_TriggerNotInEffects(t *testing.T) {
	body := `*They aren't going anywhere, but you might!*

**Trigger:** The target takes damage or is force moved.

**Effect:** The target takes half the triggering damage.`

	fm := parseAbility(t, body)
	if fm["trigger"] != "The target takes damage or is force moved." {
		t.Errorf("trigger top-level: got %v", fm["trigger"])
	}
	effects := effectsList(t, fm)
	if len(effects) != 1 {
		t.Fatalf("expected 1 effect (Effect only, Trigger excluded), got %d: %v", len(effects), effects)
	}
	if effects[0]["name"] != "Effect" {
		t.Errorf("effects[0] name: got %v", effects[0]["name"])
	}
}
