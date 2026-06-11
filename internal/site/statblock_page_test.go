package site

import (
	"encoding/json"
	"strings"
	"testing"
)

// A representative md-linked statblock page (verbatim shape from
// data-bestiary/en/md-linked): Signature ability w/ labeled power roll +
// Malice enhancement, a test-result ability (no power-roll header), passive
// traits, and villain actions.
const devilHighJudgePage = `---
agility: 3
ev: "32"
free_strike: 6
immunities:
    - Fire 5
intuition: 1
keywords:
    - Devil
    - Infernal
level: 6
might: 1
movement: Fly
name: Devil High Judge
organization: Leader
presence: 2
reason: 0
scc: mcdm.monsters.v1/monster.devils.statblock/devil-high-judge
size: 1M
speed: 7
stability: 2
stamina: "181"
type: statblock
---

|    Devil, Infernal     |          -          |      Level 6       |        Leader         |        EV 32         |
|:----------------------:|:-------------------:|:------------------:|:---------------------:|:--------------------:|
|     **1M**<br>Size     |   **7**<br>Speed    | **181**<br>Stamina |  **2**<br>Stability   | **6**<br>Free Strike |
| **Fire 5**<br>Immunity | **Fly**<br>Movement |         -          | **-**<br>With Captain |  **-**<br>Weakness   |
|    **+1**<br>Might     |  **+3**<br>Agility  |  **+0**<br>Reason  |  **+1**<br>Intuition  |  **+2**<br>Presence  |

> 🏹 **Infernal Decree (Signature Ability)**
>
> | **Magic, Ranged, Strike** |                   **Main action** |
> |---------------------------|----------------------------------:|
> | **📏 Ranged 12**          | **🎯 Three creatures or objects** |
>
> **Power Roll + 4:**
>
> - **≤11:** 10 damage; P < 2 the target can't hide (save ends)
> - **12-16:** 15 damage; P < 3 the target can't hide (save ends)
> - **17+:** 19 damage; P < 4 the target can't hide (save ends)
>
> **2 Malice:** While a target is unable to hide this way, any strike against them made by a devil gains an edge.

> ❗️ **Devilish Suggestion (2 Malice)**
>
> | **Magic, Ranged** |           **Triggered action** |
> |-------------------|-------------------------------:|
> | **📏 Ranged 5**   | **🎯 The triggering creature** |
>
> **Trigger:** A creature targets the high judge with a strike.
>
> **Effect:** The target makes a **Presence test**.
>
> - **≤11:** The target is charmed (save ends).
> - **12-16:** The high judge chooses a new target for the strike.
> - **17+:** The target takes a bane on the strike.
>
> While charmed this way, a creature treats the high judge as an ally.

> ⭐️ **True Name**
>
> If a creature within 10 squares speaks the high judge's true name, the high judge loses their damage immunities until the end of the encounter.

> ☠️ **All Rise (Villain Action 1)**
>
> | **Area, Magic** |                         **-** |
> |-----------------|------------------------------:|
> | **📏 3 burst**  | **🎯 Each enemy in the area** |
>
> **Effect:** The target makes a **Presence test**.
>
> - **≤11:** 15 psychic damage; the target is charmed (save ends)
> - **12-16:** 12 psychic damage; the target is charmed (save ends)
> - **17+:** 7 psychic damage
`

func featureByName(feats []sbFeature, name string) *sbFeature {
	for i := range feats {
		if feats[i].Name == name {
			return &feats[i]
		}
	}
	return nil
}

func TestBuildStatblockIsland_DevilHighJudge(t *testing.T) {
	fm, body := splitFrontmatter(devilHighJudgePage)
	isl := buildStatblockIsland(fm, body)

	if isl.Name != "Devil High Judge" {
		t.Errorf("name = %q", isl.Name)
	}
	if isl.Ancestry != "Devil, Infernal" {
		t.Errorf("ancestry = %q, want %q", isl.Ancestry, "Devil, Infernal")
	}
	if isl.Role != "Leader" || isl.RoleKey != "leader" {
		t.Errorf("role/roleKey = %q/%q, want Leader/leader", isl.Role, isl.RoleKey)
	}
	if isl.EV != "32" || isl.Level != "6" {
		t.Errorf("ev/level = %q/%q", isl.EV, isl.Level)
	}
	if isl.ID != "devil-high-judge" {
		t.Errorf("id = %q", isl.ID)
	}

	// Defenses (5) + characteristics (5, signed).
	if got := len(isl.Defenses); got != 5 {
		t.Fatalf("defenses len = %d", got)
	}
	if isl.Defenses[2] != (sbLV{L: "Stamina", V: "181"}) {
		t.Errorf("stamina cell = %+v", isl.Defenses[2])
	}
	wantChars := map[string]string{"Might": "+1", "Agility": "+3", "Reason": "+0", "Intuition": "+1", "Presence": "+2"}
	for _, c := range isl.Characteristics {
		if wantChars[c.L] != c.V {
			t.Errorf("char %s = %q, want %q", c.L, c.V, wantChars[c.L])
		}
	}

	// Meta 2×2.
	if isl.Meta.Immunity != "Fire 5" || isl.Meta.Movement != "Fly" || isl.Meta.Weakness != "—" {
		t.Errorf("meta = %+v", isl.Meta)
	}
	if isl.Meta.Captain.Label != "With Captain" {
		t.Errorf("captain label = %q", isl.Meta.Captain.Label)
	}

	// ── Signature ability: labeled power roll + Malice enhancement ──
	dec := featureByName(isl.Features, "Infernal Decree")
	if dec == nil {
		t.Fatal("Infernal Decree feature missing")
	}
	if dec.Kind != "ability" || dec.Action != "main" {
		t.Errorf("decree kind/action = %q/%q", dec.Kind, dec.Action)
	}
	if dec.Cost != "Signature" {
		t.Errorf("decree cost = %q, want Signature", dec.Cost)
	}
	if dec.Usage != "Main action" {
		t.Errorf("decree usage = %q", dec.Usage)
	}
	if want := []string{"Magic", "Ranged", "Strike"}; strings.Join(dec.Keywords, ",") != strings.Join(want, ",") {
		t.Errorf("decree keywords = %v", dec.Keywords)
	}
	if dec.Distance != "Ranged 12" || dec.Target != "Three creatures or objects" {
		t.Errorf("decree dist/target = %q / %q", dec.Distance, dec.Target)
	}
	if dec.PowerRoll == nil || dec.PowerRoll.Formula != "+ 4" {
		t.Fatalf("decree powerRoll = %+v", dec.PowerRoll)
	}
	if dec.PowerRoll.Tiers["low"] == "" || dec.PowerRoll.Tiers["high"] == "" {
		t.Errorf("decree tiers = %+v", dec.PowerRoll.Tiers)
	}
	if len(dec.Enhancements) != 1 || dec.Enhancements[0].Cost != "2 Malice" {
		t.Errorf("decree enhancements = %+v", dec.Enhancements)
	}

	// ── Test-result ability: no power-roll header → formula "" + sections + trailing ──
	sug := featureByName(isl.Features, "Devilish Suggestion")
	if sug == nil {
		t.Fatal("Devilish Suggestion missing")
	}
	if sug.Action != "triggered" {
		t.Errorf("suggestion action = %q", sug.Action)
	}
	if sug.Cost != "2 Malice" {
		t.Errorf("suggestion cost = %q", sug.Cost)
	}
	if sug.PowerRoll == nil || sug.PowerRoll.Formula != "" {
		t.Errorf("suggestion powerRoll (want test, formula \"\") = %+v", sug.PowerRoll)
	}
	if len(sug.Sections) != 2 || sug.Sections[0].Label != "Trigger" || sug.Sections[1].Label != "Effect" {
		t.Errorf("suggestion sections = %+v", sug.Sections)
	}
	if !strings.HasPrefix(sug.Trailing, "While charmed") {
		t.Errorf("suggestion trailing = %q", sug.Trailing)
	}

	// ── Passive trait: no table → kind passive, body set ──
	tn := featureByName(isl.Features, "True Name")
	if tn == nil {
		t.Fatal("True Name missing")
	}
	if tn.Kind != "passive" || tn.Action != "passive" {
		t.Errorf("true name kind/action = %q/%q", tn.Kind, tn.Action)
	}
	if !strings.Contains(tn.Body, "true name") || tn.PowerRoll != nil {
		t.Errorf("true name body = %q powerRoll=%v", tn.Body, tn.PowerRoll)
	}

	// ── Villain action: cost "Villain Action 1" → kind villain ──
	ar := featureByName(isl.Features, "All Rise")
	if ar == nil {
		t.Fatal("All Rise missing")
	}
	if ar.Kind != "villain" || ar.Action != "villain" {
		t.Errorf("all rise kind/action = %q/%q", ar.Kind, ar.Action)
	}
	if ar.Cost != "Villain Action 1" {
		t.Errorf("all rise cost = %q", ar.Cost)
	}
}

func TestBuildStatblockIslandPage_EmitsIsland(t *testing.T) {
	out, ok := buildStatblockIslandPage([]byte(devilHighJudgePage))
	if !ok {
		t.Fatal("expected statblock page to be rewritten")
	}
	s := string(out)
	// Island is wrapped in a .sc-statblock-mount container so the client can
	// locate it after navigation.instant strips the <script>'s attributes.
	if !strings.Contains(s, `<div class="sc-statblock-mount">`) {
		t.Fatal("sc-statblock-mount container missing")
	}
	if !strings.Contains(s, `<script type="application/json" class="sc-statblock-data">`) {
		t.Fatal("island script tag missing")
	}
	// Frontmatter preserved.
	if !strings.HasPrefix(s, "---\n") || !strings.Contains(s, "type: statblock") {
		t.Error("frontmatter not preserved")
	}
	// Island body is valid JSON.
	start := strings.Index(s, ">\n") + 2
	end := strings.LastIndex(s, "\n</script>")
	var isl sbIsland
	if err := json.Unmarshal([]byte(s[start:end]), &isl); err != nil {
		t.Fatalf("island JSON invalid: %v", err)
	}
	if isl.Name != "Devil High Judge" {
		t.Errorf("decoded name = %q", isl.Name)
	}

	// Non-statblock pages pass through untouched.
	if _, ok := buildStatblockIslandPage([]byte("---\ntype: ability\nname: X\n---\n\nbody")); ok {
		t.Error("non-statblock page should not be rewritten")
	}
}
