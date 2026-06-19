package content

import (
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// newSection builds a Section with the given heading, level, annotation, and body.
func newSection(heading string, level int, ann map[string]string, body string) *parser.Section {
	return &parser.Section{
		Heading:      heading,
		HeadingLevel: level,
		Annotation:   ann,
		BodySource:   body,
	}
}

func TestStatblockParser(t *testing.T) {
	body := cursespitterGrid + "\n" + cursespitterFeatures
	sec := newSection("Goblin Cursespitter", 7, map[string]string{"type": "statblock"}, body)

	ctx := context.NewContextStack(nil)
	ctx.Push(2, map[string]string{"category": "goblin"})

	p := &StatblockParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}

	if got.ItemID != "goblin-cursespitter" {
		t.Errorf("ItemID: got %q", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "monster/goblin/statblock" {
		t.Errorf("TypePath: got %v, want [monster goblin statblock]", got.TypePath)
	}
	if got.Frontmatter["type"] != "statblock" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
	if got.Frontmatter["level"] != 1 {
		t.Errorf("level: got %v", got.Frontmatter["level"])
	}
	if got.Frontmatter["role"] != "Hexer" || got.Frontmatter["organization"] != "Horde" {
		t.Errorf("role/org: got %v / %v", got.Frontmatter["role"], got.Frontmatter["organization"])
	}
	if got.Frontmatter["ev"] != "3" {
		t.Errorf("ev: got %v", got.Frontmatter["ev"])
	}
	if got.Frontmatter["might"] != -2 || got.Frontmatter["intuition"] != 2 {
		t.Errorf("scores: got might=%v int=%v", got.Frontmatter["might"], got.Frontmatter["intuition"])
	}
	if got.Frontmatter["movement"] != "Climb" {
		t.Errorf("movement: got %v", got.Frontmatter["movement"])
	}
}

func TestMonsterParser(t *testing.T) {
	sec := newSection("Goblins", 2, map[string]string{
		"type": "monster", "category": "goblin",
	}, "Goblins are small and crafty...")

	p := &MonsterParser{}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ItemID != "goblin" {
		t.Errorf("ItemID: got %q", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "monster/group" {
		t.Errorf("TypePath: got %v, want [monster group]", got.TypePath)
	}
	if got.Frontmatter["type"] != "monster" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
}

func TestFeatureblockParser(t *testing.T) {
	body := "" +
		"At the start of any goblin's turn, you can spend Malice...\n\n" +
		"> ⭐️ **Goblin Mode (3 Malice)**\n>\n> Each goblin gains +2 speed.\n"
	sec := newSection("Goblin Malice (Malice Features)", 9,
		map[string]string{"type": "featureblock"}, body)

	ctx := context.NewContextStack(nil)
	ctx.Push(2, map[string]string{"category": "goblin"})

	p := &FeatureblockParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.ItemID != "goblin-malice" {
		t.Errorf("ItemID: got %q (want goblin-malice)", got.ItemID)
	}
	if strings.Join(got.TypePath, "/") != "monster/goblin" {
		t.Errorf("TypePath: got %v, want [monster goblin]", got.TypePath)
	}
	if got.Frontmatter["type"] != "featureblock" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
}

func TestStatblockParser_Fixture(t *testing.T) {
	body := "*Hazard Support*\n\n" +
		"| **Stamina:** 20 + your level | **Size:** 2 |\n" +
		"|------------------------------|------------:|\n\n" +
		"> ⭐️ **Hunger Thrush**\n>\n> Each enemy that starts their turn within 3 squares is taunted.\n"

	sec := newSection("The Boil", 7, map[string]string{"type": "statblock"}, body)
	ctx := context.NewContextStack(nil)
	ctx.Push(5, map[string]string{"domain": "fixture", "category": "demon"})

	p := &StatblockParser{}
	got, err := p.Parse(ctx, sec)
	if err != nil {
		t.Fatal(err)
	}
	fm := got.Frontmatter

	// Plan 5c: fixtures are now featureblocks, not statblocks.
	if fm["type"] != "featureblock" {
		t.Errorf("type = %v, want featureblock", fm["type"])
	}
	if _, ok := fm["statblock_kind"]; ok {
		t.Errorf("statblock_kind should be absent for fixture featureblocks")
	}
	// stamina/size are promoted into stats[]; top-level keys removed.
	if _, ok := fm["stamina"]; ok {
		t.Errorf("stamina should be absent (moved to stats[])")
	}
	if _, ok := fm["size"]; ok {
		t.Errorf("size should be absent (moved to stats[])")
	}
	stats, ok := fm["stats"].([]map[string]any)
	if !ok || len(stats) != 2 {
		t.Fatalf("stats = %v, want [{Stamina,...},{Size,...}]", fm["stats"])
	}
	if stats[0]["name"] != "Stamina" || stats[0]["value"] != "20 + your level" {
		t.Errorf("stats[0] = %v", stats[0])
	}
	if stats[1]["name"] != "Size" || stats[1]["value"] != "2" {
		t.Errorf("stats[1] = %v", stats[1])
	}
	if fm["terrain_type"] != "Hazard" {
		t.Errorf("terrain_type = %v", fm["terrain_type"])
	}
	if fm["role"] != "Support" {
		t.Errorf("role = %v", fm["role"])
	}
	if kw, ok := fm["keywords"]; ok {
		t.Errorf("keywords should be absent for fixtures, got %v", kw)
	}
	// Plan 5c TypePath: monster.fixture.demon.featureblock
	if strings.Join(got.TypePath, "/") != "monster/fixture/demon/featureblock" {
		t.Errorf("TypePath = %v", got.TypePath)
	}
}

func TestFeatureblockParser_Metadata(t *testing.T) {
	tests := []struct {
		heading   string
		wantKind  string
		wantLevel int // 0 = absent
		wantName  string
	}{
		{"Basilisk Malice (Malice Features)", "malice", 0, "Basilisk Malice"},
		{"War Dog Malice (Level 4+ Malice Features)", "malice", 4, "War Dog Malice (Level 4+ Malice Features)"},
		{"Tactical Stance (Ajax Feature)", "feature", 0, "Tactical Stance"},
		{"Basic Malice", "malice", 0, "Basic Malice"},
	}
	body := "At the start of any basilisk's turn, you can spend Malice to activate one of the following features.\n\n" +
		"> 🔳 **Walleye (7 Malice)**\n>\n> A basilisk spews reflective spittle.\n"

	for _, tt := range tests {
		t.Run(tt.heading, func(t *testing.T) {
			sec := newSection(tt.heading, 9, map[string]string{"type": "featureblock"}, body)
			ctx := context.NewContextStack(nil)
			ctx.Push(2, map[string]string{"category": "basilisk"})

			p := &FeatureblockParser{}
			got, err := p.Parse(ctx, sec)
			if err != nil {
				t.Fatal(err)
			}
			if got.Frontmatter["kind"] != tt.wantKind {
				t.Errorf("kind = %v, want %q", got.Frontmatter["kind"], tt.wantKind)
			}
			if tt.wantLevel > 0 {
				if got.Frontmatter["level"] != tt.wantLevel {
					t.Errorf("level = %v, want %d", got.Frontmatter["level"], tt.wantLevel)
				}
			} else if _, ok := got.Frontmatter["level"]; ok {
				t.Errorf("level should be absent, got %v", got.Frontmatter["level"])
			}
			if got.Frontmatter["name"] != tt.wantName {
				t.Errorf("name = %v, want %q", got.Frontmatter["name"], tt.wantName)
			}
			flavor, _ := got.Frontmatter["flavor"].(string)
			if !strings.HasPrefix(flavor, "At the start of any basilisk's turn") {
				t.Errorf("flavor = %q", flavor)
			}
			feats, ok := got.Frontmatter["features"].([]map[string]any)
			if !ok || len(feats) != 1 || feats[0]["name"] != "Walleye" {
				t.Errorf("features = %+v", got.Frontmatter["features"])
			}
		})
	}
}

func TestMonsterGroupParser(t *testing.T) {
	sec := newSection("Environmental Hazards", 3, map[string]string{
		"type": "monster-group", "domain": "dynamic-terrain", "category": "environmental-hazards",
	}, "intro prose")
	p := &MonsterGroupParser{}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got.TypePath != nil || got.ItemID != "" {
		t.Errorf("expected no classification, got TypePath=%v ItemID=%q", got.TypePath, got.ItemID)
	}
	if got.Frontmatter["type"] != "monster-group" {
		t.Errorf("type: got %v", got.Frontmatter["type"])
	}
}

func TestStatblockParser_SummonerMinionChampion(t *testing.T) {
	body := "| — | Demon | Minion Ambusher | - | 1 Malice |\n" +
		"|:-:|:-:|:-:|:-:|:-:|\n" +
		"| **1S**<br>Size | **4**<br>Speed | **3**<br>Stamina | **0**<br>Stability | **2**<br>Free Strike |\n"

	cases := []struct {
		name     string
		domain   string
		category string
		want     string
	}{
		{"Rasquine", "minion", "demon", "monster/minion/summoner/demon/statblock"},
		{"Demon Lord's Aspect", "champion", "demon", "monster/champion/summoner/demon/statblock"},
	}
	for _, tc := range cases {
		t.Run(tc.domain, func(t *testing.T) {
			sec := newSection(tc.name, 7, map[string]string{"type": "statblock"}, body)
			ctx := context.NewContextStack(nil)
			ctx.Push(5, map[string]string{"domain": tc.domain, "category": tc.category})

			got, err := (&StatblockParser{}).Parse(ctx, sec)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(got.TypePath, "/") != tc.want {
				t.Errorf("TypePath = %v, want %s", got.TypePath, tc.want)
			}
			if got.Frontmatter["type"] != "statblock" {
				t.Errorf("type = %v, want statblock", got.Frontmatter["type"])
			}
		})
	}
}

func TestStatblockParser_Retainer(t *testing.T) {
	ctx := context.NewContextStack(nil)
	ctx.Push(4, map[string]string{"domain": "retainer"}) // mirrors `#### Retainer Statblocks`
	sec := &parser.Section{Heading: "Angulotl Hopper", HeadingLevel: 6,
		BodySource: "|  Angulotl, Humanoid | - | Level 1 | Harrier Retainer | EV - |\n\n> 🗡 **Leapfrog (Signature Ability)**\n>\n> **Effect:** Jump."}
	got, err := (&StatblockParser{}).Parse(ctx, sec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.TypePath, "/") != "monster/retainer/statblock" {
		t.Errorf("TypePath = %v, want [monster retainer statblock]", got.TypePath)
	}
	if got.ItemID != "angulotl-hopper" {
		t.Errorf("ItemID = %q, want angulotl-hopper", got.ItemID)
	}
}

func TestStatblockParser_SummonerRetainerUnchanged(t *testing.T) {
	// Summoner-book retainers carry @category: summoner and are OUT of Plan 6 scope:
	// they must stay retainer.summoner.statblock, NOT monster.retainer.summoner.statblock.
	ctx := context.NewContextStack(nil)
	ctx.Push(4, map[string]string{"domain": "retainer", "category": "summoner"})
	sec := &parser.Section{Heading: "Devil Detective", HeadingLevel: 6,
		BodySource: "|  Devil, Fiend | - | Level 1 | Controller Retainer | EV - |\n\n> 🗡 **Interrogate (Signature Ability)**\n>\n> **Effect:** Question."}
	got, err := (&StatblockParser{}).Parse(ctx, sec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.TypePath, "/") != "retainer/summoner/statblock" {
		t.Errorf("TypePath = %v, want [retainer summoner statblock]", got.TypePath)
	}
}

func TestFeatureblockParser_RetainerAdvancement(t *testing.T) {
	ctx := context.NewContextStack(nil)
	ctx.Push(4, map[string]string{"domain": "retainer"})
	// Level label must be inside a blockquote for fbLevelLabelRe to match via ParseRichFeatures.
	body := "> **Level 4 Retainer Advancement Ability**\n>\n" +
		"> 🗡 **Leaping Attack (Encounter)**\n>\n> **Effect:** Jump and strike."
	sec := &parser.Section{Heading: "Angulotl Hopper Advancement Features", HeadingLevel: 6,
		Annotation: map[string]string{"id": "angulotl-hopper"}, BodySource: body}
	got, err := (&FeatureblockParser{}).Parse(ctx, sec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.TypePath, "/") != "monster/retainer/advancement-features" {
		t.Errorf("TypePath = %v, want [monster retainer advancement-features]", got.TypePath)
	}
	if got.ItemID != "angulotl-hopper" {
		t.Errorf("ItemID = %q, want angulotl-hopper", got.ItemID)
	}
	feats, _ := got.Frontmatter["features"].([]map[string]any)
	if len(feats) == 0 {
		t.Fatalf("expected inline features, got %v", got.Frontmatter["features"])
	}
	if lv, _ := feats[0]["level"].(int); lv != 4 {
		t.Errorf("member level = %v, want 4 (fbLevelLabelRe must attach it)", feats[0]["level"])
	}
}

func TestFeatureblockParser_RoleAdvancement(t *testing.T) {
	ctx := context.NewContextStack(nil)
	ctx.Push(4, map[string]string{"domain": "retainer", "category": "role-advancement"})
	sec := &parser.Section{Heading: "Ambusher Abilities", HeadingLevel: 5,
		Annotation: map[string]string{"id": "ambusher"},
		BodySource: "> **Level 4 Role Advancement Ability**\n>\n" +
			"> 🗡 **Go for the Jugular (Encounter)**\n>\n> **Effect:** Bleed."}
	got, err := (&FeatureblockParser{}).Parse(ctx, sec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.TypePath, "/") != "monster/retainer/role-advancement" {
		t.Errorf("TypePath = %v, want [monster retainer role-advancement]", got.TypePath)
	}
	if got.ItemID != "ambusher" {
		t.Errorf("ItemID = %q, want ambusher", got.ItemID)
	}
}

func TestStatblockParser_SummonerRival(t *testing.T) {
	npcBody := "| — | Humanoid, Rival | Level 2 Elite Controller | - | EV 16 |\n" +
		"|:-:|:-:|:-:|:-:|:-:|\n" +
		"| **1M**<br>Size | **5**<br>Speed | **80**<br>Stamina | **0**<br>Stability | **3**<br>Free Strike |\n"
	summonBody := "| — | Undead | Signature Minion Harrier | - | 1 Malice |\n" +
		"|:-:|:-:|:-:|:-:|:-:|\n" +
		"| **1S**<br>Size | **6**<br>Speed | **3**<br>Stamina | **0**<br>Stability | **1**<br>Free Strike |\n"

	cases := []struct {
		name, heading, body, want string
	}{
		{"npc", "Rival Summoner", npcBody, "monster/rival/2nd-echelon/statblock"},
		{"summon", "Skeleton", summonBody, "monster/rival/2nd-echelon/summoner/minion"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			sec := newSection(tc.heading, 7, map[string]string{"type": "statblock"}, tc.body)
			ctx := context.NewContextStack(nil)
			ctx.Push(5, map[string]string{
				"domain": "rival", "category": "summoner", "subcategory": "2nd-echelon",
			})

			got, err := (&StatblockParser{}).Parse(ctx, sec)
			if err != nil {
				t.Fatal(err)
			}
			if strings.Join(got.TypePath, "/") != tc.want {
				t.Errorf("TypePath = %v, want %s (org=%v)", got.TypePath, tc.want, got.Frontmatter["organization"])
			}
		})
	}
}
