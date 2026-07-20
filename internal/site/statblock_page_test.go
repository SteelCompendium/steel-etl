package site

import (
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
> | **Magic, Ranged** |           **[Triggered action](../../rule/combat/triggered-action.md)** |
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
	if isl.Eyebrow != "Devil, Infernal" {
		t.Errorf("eyebrow = %q, want %q", isl.Eyebrow, "Devil, Infernal")
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
	// The high judge is a Leader (no captain bonus in the grid, no with_captain
	// frontmatter) — statblockMeta4 drops the cell rather than showing a blank
	// "With Captain" (FOLLOWUPS #7 piece 2).
	if isl.Meta.Captain != (sbCaptain{}) {
		t.Errorf("captain = %+v, want zero value (no captain bonus)", isl.Meta.Captain)
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
	// A linked usage cell keeps its raw markdown link (same treatment as
	// distance/target) — not stripped to text; richSb resolves it at render.
	if want := "[Triggered action](../../rule/combat/triggered-action.md)"; sug.Usage != want {
		t.Errorf("suggestion usage = %q, want %q", sug.Usage, want)
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

func TestBuildStatblockIslandPage_EmitsCard(t *testing.T) {
	out, ok := buildStatblockIslandPage([]byte(devilHighJudgePage))
	if !ok {
		t.Fatal("expected statblock page to be rewritten")
	}
	s := string(out)
	// Build-time .sb-wrap card, no JSON island.
	if !strings.Contains(s, `<div class="sb-wrap" data-role="leader" data-creature="devil-high-judge">`) {
		t.Fatal("sb-wrap card missing")
	}
	if strings.Contains(s, "sc-statblock-mount") || strings.Contains(s, "sc-statblock-data") {
		t.Error("JSON island markup should be gone")
	}
	// Frontmatter preserved.
	if !strings.HasPrefix(s, "---\n") || !strings.Contains(s, "type: statblock") {
		t.Error("frontmatter not preserved")
	}
	// A representative rendered feature is present.
	if !strings.Contains(s, "Infernal Decree") || !strings.Contains(s, `class="sb__features"`) {
		t.Error("rendered features missing")
	}

	// Non-statblock pages pass through untouched.
	if _, ok := buildStatblockIslandPage([]byte("---\ntype: ability\nname: X\n---\n\nbody")); ok {
		t.Error("non-statblock page should not be rewritten")
	}
}

// linkedFieldsPage exercises every feature field that can carry a source link:
// the title's parenthetical (Signature / Malice cost / Villain Action — with the
// link's own "(…)" nested inside the group), the enhancement cost, a section
// label, and a feature name that is itself a link. Links arrive as relative ".md"
// links (the gen link-sweep) and are stored RAW in the island; richSb resolves
// them to directory URLs at render (asserted at the end of the test).
const linkedFieldsPage = `---
name: Link Test Monster
organization: ""
role: Brute
level: 1
type: statblock
---

> 🏹 **Accursed Bite ([Signature Ability](signature-ability.md))**
>
> | **Magic** |        **Main action** |
> |-----------|-----------------------:|
> | **📏 Melee 1** | **🎯 One creature** |
>
> **Power Roll + 3:**
>
> - **≤11:** 5 damage
> - **12-16:** 8 damage
> - **17+:** 11 damage
>
> **2 [Malice](malice.md):** The target is weakened.
>
> **[End Effect](end-effect.md):** The effect ends.

> ☠️ **Final Judgment ([Villain Action](villain-action.md) 3)**
>
> | **Magic** |   **-** |
> |-----------|--------:|
> | **📏 5 burst** | **🎯 Each enemy** |
>
> **Effect:** Massive damage.

> ⭐️ **[Solo](solo.md) Monster**
>
> This creature acts alone.

> ⭐️ **[End Effect](end-effect.md)**
>
> At the end of each turn, the creature can take damage to end an effect.
`

func TestBuildStatblockIsland_PreservesRawLinksInAllFields(t *testing.T) {
	fm, body := splitFrontmatter(linkedFieldsPage)
	isl := buildStatblockIsland(fm, body)

	// ── name parenthetical: Signature → cost "Signature", name cleaned ──
	bite := featureByName(isl.Features, "Accursed Bite")
	if bite == nil {
		t.Fatal("Accursed Bite missing (linked signature paren broke title split)")
	}
	if bite.Cost != "Signature" {
		t.Errorf("bite cost = %q, want Signature", bite.Cost)
	}
	// ── enhancement cost: "2 [Malice](…)" link kept raw, not stripped ──
	if len(bite.Enhancements) != 1 {
		t.Fatalf("bite enhancements = %+v", bite.Enhancements)
	}
	if want := "2 [Malice](malice.md)"; bite.Enhancements[0].Cost != want {
		t.Errorf("enhancement cost = %q, want %q", bite.Enhancements[0].Cost, want)
	}
	// ── section label: "[End Effect](…)" link kept raw ──
	var endEff *sbSection
	for i := range bite.Sections {
		if strings.Contains(bite.Sections[i].Label, "End Effect") {
			endEff = &bite.Sections[i]
		}
	}
	if endEff == nil {
		t.Fatalf("End Effect section missing; sections = %+v", bite.Sections)
	}
	if want := "[End Effect](end-effect.md)"; endEff.Label != want {
		t.Errorf("section label = %q, want %q", endEff.Label, want)
	}

	// ── villain action: linked "[Villain Action](…) 3" still classifies villain,
	//    cost keeps the raw link, name cleaned ──
	vj := featureByName(isl.Features, "Final Judgment")
	if vj == nil {
		t.Fatal("Final Judgment missing (linked villain paren broke title split)")
	}
	if vj.Kind != "villain" || vj.Action != "villain" {
		t.Errorf("villain kind/action = %q/%q", vj.Kind, vj.Action)
	}
	if want := "[Villain Action](villain-action.md) 3"; vj.Cost != want {
		t.Errorf("villain cost = %q, want %q", vj.Cost, want)
	}

	// ── feature name that is itself a link (no paren) is kept raw ──
	solo := featureByName(isl.Features, "[Solo](solo.md) Monster")
	if solo == nil {
		t.Fatalf("link-named trait missing; names = %v", featureNames(isl.Features))
	}

	// ── feature whose ENTIRE title is a link: the link's own "(url)" must not be
	//    mistaken for a trailing cost parenthetical (would yield name "[End Effect]"
	//    + a bare-URL cost). Name stays the raw link; no cost. ──
	ee := featureByName(isl.Features, "[End Effect](end-effect.md)")
	if ee == nil {
		t.Fatalf("link-only-titled trait missing; names = %v", featureNames(isl.Features))
	}
	if ee.Cost != "" {
		t.Errorf("link-only title produced spurious cost = %q", ee.Cost)
	}

	// ── render resolves the raw targets to served directory URLs (richSb →
	//    cardHref), the single resolve point that replaced the old parse-time
	//    resolveSbLinks pre-pass. ──
	card := renderStatblockCard(isl)
	for _, want := range []string{
		`href="../malice/"`,
		`href="../end-effect/"`,
		`href="../villain-action/"`,
		`href="../solo/"`,
	} {
		if !strings.Contains(card, want) {
			t.Errorf("rendered card missing resolved link %s", want)
		}
	}
	if strings.Contains(card, ".md\"") {
		t.Error("rendered card still contains an unresolved .md href")
	}
}

func featureNames(feats []sbFeature) []string {
	var out []string
	for _, f := range feats {
		out = append(out, f.Name)
	}
	return out
}

func TestBuildStatblockIsland_ProvenanceEyebrowOverridesKeywords(t *testing.T) {
	// A rival summoner minion: keywords say "—", but the scc carries echelon +
	// rival context, so the Eyebrow field must be the derived provenance.
	fm := "name: Zombie Titan\n" +
		"organization: Minion\n" +
		"role: Defender\n" +
		"keywords:\n    - —\n" +
		"scc: mcdm.summoner.v1/monster.rival.4th-echelon.summoner.minion/zombie-titan\n"
	got := buildStatblockIsland(fm, "")
	if got.Eyebrow != "Rival Summoner Summon · Echelon 4" {
		t.Errorf("Eyebrow = %q, want %q", got.Eyebrow, "Rival Summoner Summon · Echelon 4")
	}
}

func TestBuildStatblockIsland_ElementalDomainsAppendedToEyebrow(t *testing.T) {
	// A summoner elemental minion: keywords arrive distributed per-domain
	// ("Elemental (Air)", "Elemental (Earth)"). The provenance eyebrow stays,
	// with the book-faithful domain parenthetical appended so the head reads
	// like the sourcebook ("Summoner Minion · Elemental (Air, Earth)").
	fm := "name: Desolation of Sand\n" +
		"organization: Minion\n" +
		"role: Hexer\n" +
		"keywords:\n    - Elemental (Air)\n    - Elemental (Earth)\n" +
		"scc: mcdm.summoner.v1/monster.minion.summoner.elemental.statblock/desolation-of-sand\n"
	got := buildStatblockIsland(fm, "")
	if got.Eyebrow != "Summoner Minion · Elemental (Air, Earth)" {
		t.Errorf("Eyebrow = %q, want %q", got.Eyebrow, "Summoner Minion · Elemental (Air, Earth)")
	}
}

func TestCollapseKeywords(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want string
	}{
		{"distributed domains recollapse", []string{"Elemental (Air)", "Elemental (Earth)"}, "Elemental (Air, Earth)"},
		{"single domain", []string{"Elemental (Fire)"}, "Elemental (Fire)"},
		{"plain multi keyword", []string{"Humanoid", "Goblin"}, "Humanoid, Goblin"},
		{"bare keyword", []string{"Undead"}, "Undead"},
		{"base and qualified mix preserves order", []string{"Dragon", "Elemental (Fire)", "Elemental (Air)"}, "Dragon, Elemental (Fire, Air)"},
		{"empty", nil, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := collapseKeywords(tc.in); got != tc.want {
				t.Errorf("collapseKeywords(%v) = %q, want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestBuildStatblockIsland_NonSummonerKeepsKeywords(t *testing.T) {
	// A Monsters-book statblock keeps its real keyword-derived ancestry.
	fm := "name: Goblin Warrior\n" +
		"organization: Minion\n" +
		"role: Harrier\n" +
		"keywords:\n    - Humanoid\n    - Goblin\n" +
		"scc: mcdm.monsters.v1/monster.goblins.statblock/goblin-warrior\n"
	got := buildStatblockIsland(fm, "")
	if got.Eyebrow != "Humanoid, Goblin" {
		t.Errorf("Eyebrow = %q, want %q", got.Eyebrow, "Humanoid, Goblin")
	}
}

// TestStatblockMeta4 locks the fix for FOLLOWUPS #7 piece 2: the 4th .sb__meta
// cell is context-driven instead of an always-"With Captain" label that
// previously never picked up a real captain bonus (buildStatblockIsland used
// to derive it from a body-prose regex that only ever matched the corpus's
// one illustrative @classify:false example — real grid values live in the
// `with_captain` frontmatter field ParseStatblockFields already parses).
func TestStatblockMeta4(t *testing.T) {
	tests := []struct {
		name      string
		fm        string
		body      string
		wantLabel string
		wantValue string
	}{
		{
			name:      "Monsters-book minion with a real captain bonus (goblin-sniper shape)",
			fm:        "scc: mcdm.monsters.v1/monster.goblin.statblock/goblin-sniper\nwith_captain: +5 bonus to ranged distance\n",
			wantLabel: "With Captain",
			wantValue: "+5 bonus to ranged distance",
		},
		{
			name:      "Monsters-book leader/solo with no captain bonus drops the cell",
			fm:        "scc: mcdm.monsters.v1/monster.devil/devil-legate\n",
			wantLabel: "",
			wantValue: "",
		},
		{
			name:      "Summoner-book statblock always shows Free Strike Damage Type",
			fm:        "scc: mcdm.summoner.v1/monster.minion.summoner.demon.statblock/rasquine\nfree_strike_damage_type: Poison\n",
			wantLabel: "Free Strike Damage Type",
			wantValue: "Poison",
		},
		{
			name:      "Summoner-book statblock with no damage type still shows the cell, dashed",
			fm:        "scc: mcdm.summoner.v1/monster.minion.summoner.demon.statblock/ensnarer\n",
			wantLabel: "Free Strike Damage Type",
			wantValue: "—",
		},
		{
			name:      "the one illustrative @classify:false inline example keeps its body-prose convention",
			fm:        "",
			body:      "With Captain: +2 bonus to speed\n",
			wantLabel: "With Captain",
			wantValue: "+2 bonus to speed",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := statblockMeta4(tc.fm, tc.body)
			if got.Label != tc.wantLabel || got.Value != tc.wantValue {
				t.Errorf("statblockMeta4() = %+v, want {%q %q}", got, tc.wantLabel, tc.wantValue)
			}
		})
	}
}
