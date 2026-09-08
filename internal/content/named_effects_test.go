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
// sits. Instantaneous Excavation states its Effect BEFORE the power roll — per
// the SC-310 attachment rule (mirroring SC-308) the roll list attaches to the
// entry created by the paragraph immediately before it, so the array is ONE
// merged {name, effect, roll, tier1..3} entry, not [Effect, roll] (SC-308's
// landed model; this ability was the OLD extractOrderedEffects doc comment's
// worked example for keeping [Effect, roll] in that order — it is now nested
// rather than split, per SC-310).
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
	if len(effects) != 1 {
		t.Fatalf("expected 1 merged effect (Effect+roll nested), got %d: %v", len(effects), effects)
	}
	if effects[0]["name"] != "Effect" {
		t.Errorf("effects[0] should be the Effect (document order), got %v", effects[0])
	}
	if effects[0]["roll"] != "Power Roll + Reason" {
		t.Errorf("effects[0] should carry the attached power roll, got %v", effects[0])
	}
	if effects[0]["tier1"] != "The target shifts 1 square." {
		t.Errorf("effects[0] tier1: got %v", effects[0]["tier1"])
	}
}

// Divine Dragon (SC-310): two "Power Roll + Intuition" tier lists, each one
// sitting under its own bare-prose paragraph (not a labeled section) rather
// than a header. The attachment rule must produce 3 entries — Effect (no
// roll, since the paragraph right after it is bare prose, not a tier list),
// then each bare-prose paragraph fused with the roll immediately below it —
// in exact source order, both rolls intact (the pre-fix bug: only one
// tier1..3 triple ever survived, and neither bare-prose lead-in was captured
// at all).
func TestAbilityParser_DivineDragonTwoRolls(t *testing.T) {
	body := `*From nothing but divine will, you create a powerful ally.*

| **Magic, Ranged** | **Main action** |
| --- | ---: |
| **Ranged 10** | **Special** |

**Effect:** You conjure a size 4 dragon that appears in an unoccupied space.

On subsequent turns, you can use a main action to command the dragon to breathe magic fire. Make the following power roll targeting each enemy in the area.

**Power Roll + Intuition:**
- **≤11:** 5 fire damage
- **12-16:** 9 fire damage
- **17+:** 12 fire damage

Additionally, you can use a maneuver to move the dragon, or to make a melee weapon strike with their claw.

**Power Roll + Intuition:**
- **≤11:** 3 + I damage
- **12-16:** 5 + I damage
- **17+:** 8 + I damage`

	effects := effectsList(t, parseAbility(t, body))
	if len(effects) != 3 {
		t.Fatalf("expected 3 effects (Effect, prose+breath-roll, prose+claw-roll), got %d: %v", len(effects), effects)
	}
	if effects[0]["name"] != "Effect" || effects[0]["roll"] != nil {
		t.Errorf("effects[0] should be the bare Effect (no roll attached), got %v", effects[0])
	}
	if effects[1]["roll"] != "Power Roll + Intuition" || effects[1]["tier1"] != "5 fire damage" || effects[1]["tier3"] != "12 fire damage" {
		t.Errorf("effects[1] should be the breath prose+roll, got %v", effects[1])
	}
	if effects[1]["effect"] != "On subsequent turns, you can use a main action to command the dragon to breathe magic fire. Make the following power roll targeting each enemy in the area." {
		t.Errorf("effects[1] effect text: got %v", effects[1]["effect"])
	}
	if effects[2]["roll"] != "Power Roll + Intuition" || effects[2]["tier1"] != "3 + I damage" || effects[2]["tier3"] != "8 + I damage" {
		t.Errorf("effects[2] should be the claw prose+roll, got %v", effects[2])
	}
	// The FIRST list still wins the flat fm fields (SC-308's contract).
	fm := parseAbility(t, body)
	if fm["tier1"] != "5 fire damage" {
		t.Errorf("flat fm tier1 should keep the FIRST list, got %v", fm["tier1"])
	}
}

// A header-less tier list (no "**Power Roll + N:**" line) must attach with NO
// `roll` key at all — never a stored/synthesized label (the site derives the
// display head at render time instead).
func TestAbilityParser_HeaderlessListNoRollKey(t *testing.T) {
	body := `Make a Reason test:

- **≤11:** A false rumor.
- **12-16:** A likely rumor.
- **17+:** An obscure rumor.

**Effect:** You learn something.`

	effects := effectsList(t, parseAbility(t, body))
	if len(effects) != 2 {
		t.Fatalf("expected 2 effects (prose+headerless-roll, Effect), got %d: %v", len(effects), effects)
	}
	if _, ok := effects[0]["roll"]; ok {
		t.Errorf("header-less list must not store a `roll` key, got %v", effects[0])
	}
	if effects[0]["tier1"] != "A false rumor." {
		t.Errorf("effects[0] tier1: got %v", effects[0])
	}
	if effects[1]["name"] != "Effect" {
		t.Errorf("effects[1] should be the trailing Effect, got %v", effects[1])
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
