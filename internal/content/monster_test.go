package content

import (
	"os"
	"reflect"
	"strings"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
	"github.com/SteelCompendium/steel-etl/internal/scc"
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

func TestStatblockParser_SummonerRetainerMonsterFamily(t *testing.T) {
	// Summoner-book retainers (@category: summoner) merge flat into the monster.*
	// retainer family — same type as the Monsters-book retainers (only the
	// mcdm.summoner.v1 source segment + the source-derived card eyebrow distinguish
	// them). The `summoner` category segment is dropped, so they become
	// monster.retainer.statblock (NOT retainer.summoner.statblock or
	// monster.retainer.summoner.statblock).
	ctx := context.NewContextStack(nil)
	ctx.Push(4, map[string]string{"domain": "retainer", "category": "summoner"})
	sec := &parser.Section{Heading: "Devil Detective", HeadingLevel: 6,
		BodySource: "|  Devil, Fiend | - | Level 1 | Controller Retainer | EV - |\n\n> 🗡 **Interrogate (Signature Ability)**\n>\n> **Effect:** Question."}
	got, err := (&StatblockParser{}).Parse(ctx, sec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.TypePath, "/") != "monster/retainer/statblock" {
		t.Errorf("TypePath = %v, want [monster retainer statblock]", got.TypePath)
	}
	if got.ItemID != "devil-detective" {
		t.Errorf("ItemID = %q, want devil-detective", got.ItemID)
	}
}

func TestStatblockParser_SummonerRetainerMinionNests(t *testing.T) {
	// A summoner-book retainer-group statblock whose organization is "Minion"
	// (the detective's summons) nests as monster.retainer.summoner.minion.statblock,
	// out of the flat retainer index. The detective itself (organization Retainer)
	// stays monster.retainer.statblock.
	cases := []struct{ name, body, want string }{
		{"detective",
			"| Devil, Fiend | - | Level 1 | Controller Retainer | - |\n\n> ⭐️ **X**\n>\n> Y.",
			"monster/retainer/statblock"},
		{"minion",
			"| Abyssal, Demon | - | - | Signature Minion Harrier | - |\n\n> ⭐️ **X**\n>\n> Y.",
			"monster/retainer/summoner/minion/statblock"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.NewContextStack(nil)
			ctx.Push(4, map[string]string{"domain": "retainer", "category": "summoner"})
			sec := &parser.Section{Heading: "Thing", HeadingLevel: 6, BodySource: tc.body}
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

func TestFeatureblockParser_SummonerRetainerAdvancement(t *testing.T) {
	// A summoner-book retainer advancement featureblock now mints
	// monster.retainer.advancement-features/<id> (the category != "summoner"
	// guard was lifted), same as the Monsters-book retainers.
	ctx := context.NewContextStack(nil)
	ctx.Push(4, map[string]string{"domain": "retainer", "category": "summoner"})
	body := "> **Level 4 Retainer Advancement Ability**\n>\n" +
		"> 🏹 **Soul Sleuth (Encounter)**\n>\n> **Effect:** Reveal."
	sec := &parser.Section{Heading: "Devil Detective Advancement Features", HeadingLevel: 6,
		Annotation: map[string]string{"id": "devil-detective"}, BodySource: body}
	got, err := (&FeatureblockParser{}).Parse(ctx, sec)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Join(got.TypePath, "/") != "monster/retainer/advancement-features" {
		t.Errorf("TypePath = %v, want [monster retainer advancement-features]", got.TypePath)
	}
	if got.ItemID != "devil-detective" {
		t.Errorf("ItemID = %q, want devil-detective", got.ItemID)
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
		{"summon", "Skeleton", summonBody, "monster/rival/2nd-echelon/summoner/minion/statblock"},
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

func TestFeatureblockParser_FixtureAdvancementCodedChildren(t *testing.T) {
	// The advancement featureblock body: two leveled bands, three members, each
	// member preceded by its inline annotation (the source form, Task 3).
	body := strings.Join([]string{
		"> **Level 5 Fixture Advancement Feature**",
		"",
		"<!-- @type: feature | @id: soul-rancor | @level: 5 -->",
		"> ⭐️ **Soul Rancor**",
		">",
		"> You gain a surge the first time your demon minions deal 3+ damage.",
		"",
		"> **Level 9 Fixture Advancement Feature**",
		"",
		"<!-- @type: feature | @id: size-increase | @level: 9 -->",
		"> ⭐️ **Size Increase**",
		">",
		"> The boil is now size 3.",
		"",
		"<!-- @type: feature | @id: fester-field | @level: 9 -->",
		"> ⭐️ **Fester Field**",
		">",
		"> Each non-abyssal enemy within 3 squares takes 5 corruption damage.",
	}, "\n")
	fb := newSection("The Boil Advancement Features", 6,
		map[string]string{"type": "featureblock", "id": "the-boil"}, body)

	ctx := context.NewContextStack(nil)
	ctx.Push(3, map[string]string{"domain": "fixture", "category": "demon"})

	p := &FeatureblockParser{}
	got, err := p.Parse(ctx, fb)
	if err != nil {
		t.Fatal(err)
	}
	// Container code unchanged (Plan 5c).
	if gotPath := strings.Join(got.TypePath, "/"); gotPath != "monster/fixture/demon/advancement-features" {
		t.Errorf("container TypePath = %q, want monster/fixture/demon/advancement-features", gotPath)
	}
	if got.ItemID != "the-boil" {
		t.Errorf("container ItemID = %q, want the-boil", got.ItemID)
	}
	// The card still gets its inline features[].
	if feats, ok := got.Frontmatter["features"].([]map[string]any); !ok || len(feats) != 3 {
		t.Fatalf("features = %v, want 3 inline members", got.Frontmatter["features"])
	}
	// And each member is now a coded child with a base-inclusive, leveled path.
	if len(got.CodedChildren) != 3 {
		t.Fatalf("CodedChildren = %d, want 3", len(got.CodedChildren))
	}
	want := []struct {
		path, id string
	}{
		{"feature/fixture/demon/the-boil/level-5", "soul-rancor"},
		{"feature/fixture/demon/the-boil/level-9", "size-increase"},
		{"feature/fixture/demon/the-boil/level-9", "fester-field"},
	}
	for i, w := range want {
		c := got.CodedChildren[i]
		if gotPath := strings.Join(c.TypePath, "/"); gotPath != w.path {
			t.Errorf("child[%d] TypePath = %q, want %q", i, gotPath, w.path)
		}
		if c.ItemID != w.id {
			t.Errorf("child[%d] ItemID = %q, want %q", i, c.ItemID, w.id)
		}
		if c.Frontmatter["type"] != "feature" {
			t.Errorf("child[%d] type = %v, want feature", i, c.Frontmatter["type"])
		}
		if strings.TrimSpace(c.Body) == "" {
			t.Errorf("child[%d] Body is empty, want the member prose", i)
		}
	}
}

func TestFixtureAdvancementCodedChildren_ViaParseDocument(t *testing.T) {
	src := []byte(strings.Join([]string{
		"<!-- @type: monster-group | @domain: fixture | @category: demon -->",
		"##### Demon Portfolio Fixture", "", "Lore.", "",
		"<!-- @type: statblock -->",
		"####### The Boil", "", "*Hazard Support*", "",
		"| **Stamina:** 20 + your level | **Size:** 2 |",
		"|------------------------------|------------:|", "",
		"> ⭐️ **Hunger Thrush**", ">", "> Inline base ability.", "",
		"<!-- @type: featureblock | @id: the-boil -->",
		"####### The Boil Advancement Features", "",
		"> **Level 5 Fixture Advancement Feature**", ">",
		"<!-- @type: feature | @id: soul-rancor | @level: 5 -->",
		"> ⭐️ **Soul Rancor**", ">", "> Surge body.", "",
		"> **Level 9 Fixture Advancement Feature**", ">",
		"<!-- @type: feature | @id: fester-field | @level: 9 -->",
		"> ⭐️ **Fester Field**", ">", "> Corruption body.", "",
	}, "\n"))

	doc, err := parser.ParseDocument(src)
	if err != nil {
		t.Fatal(err)
	}
	// Find the advancement featureblock section in the parsed tree.
	var fb *parser.Section
	var walk func(ss []*parser.Section)
	walk = func(ss []*parser.Section) {
		for _, s := range ss {
			if s.Type() == "featureblock" && s.ID() == "the-boil" {
				fb = s
			}
			walk(s.Children)
		}
	}
	walk(doc.Sections)
	if fb == nil {
		t.Fatal("advancement featureblock @id:the-boil not found in parsed tree")
	}

	ctx := context.NewContextStack(nil)
	ctx.Push(3, map[string]string{"domain": "fixture", "category": "demon"})
	got, err := (&FeatureblockParser{}).Parse(ctx, fb)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.CodedChildren) != 2 {
		t.Fatalf("CodedChildren = %d, want 2", len(got.CodedChildren))
	}
	if strings.Join(got.CodedChildren[0].TypePath, "/") != "feature/fixture/demon/the-boil/level-5" ||
		got.CodedChildren[0].ItemID != "soul-rancor" {
		t.Errorf("child[0] = %v/%q, want feature/fixture/demon/the-boil/level-5/soul-rancor",
			got.CodedChildren[0].TypePath, got.CodedChildren[0].ItemID)
	}
}

// TestCollectChildFeatures_PopulatesEffects locks SC-308 review r3's C-2 fix:
// collectChildFeatures builds a RichFeature by hand (name + prose body) rather
// than through parseRichFeature, so it never got an Effects entry — and
// renderFbFeat now walks Effects exclusively, meaning these beastheart
// companion advancement features (Foes Forever Frozen, Rock Smasher, …)
// rendered as a bare heading with no body at all. ToMap must carry `effects`
// alongside the flat `body`.
func TestCollectChildFeatures_PopulatesEffects(t *testing.T) {
	child := newSection("Foes Forever Frozen", 4, map[string]string{"type": "feature", "level": "3"},
		"The basilisk's gaze turns a foe to stone.")
	parent := &parser.Section{Heading: "Basilisk Advancement Features", HeadingLevel: 3, Children: []*parser.Section{child}}

	feats := collectChildFeatures(parent)
	if len(feats) != 1 {
		t.Fatalf("got %d features, want 1", len(feats))
	}
	f := feats[0]
	if f.Name != "Foes Forever Frozen" || f.Level != 3 {
		t.Errorf("name/level = %q/%d", f.Name, f.Level)
	}
	if f.Body != "The basilisk's gaze turns a foe to stone." {
		t.Errorf("Body = %q", f.Body)
	}
	if len(f.Effects) != 1 || f.Effects[0].Effect != f.Body {
		t.Fatalf("Effects = %+v, want 1 entry mirroring Body", f.Effects)
	}

	m := f.ToMap()
	effs, ok := m["effects"].([]map[string]any)
	if !ok || len(effs) != 1 || effs[0]["effect"] != f.Body {
		t.Errorf("ToMap effects = %+v, want 1 entry mirroring body", m["effects"])
	}
	if m["body"] != f.Body {
		t.Errorf("ToMap body = %v, want the flat body kept alongside effects", m["body"])
	}
}

// companionWant is the expected parse of one beastheart companion's stat grid.
type companionWant struct {
	keywords   []string
	size       string
	speed      int
	stability  int
	movement   string // "" = none (em-dash in the source)
	immunities []string
	skills     []string
	chars      [5]int // might, agility, reason, intuition, presence
}

// TestStatblockParser_CompanionGrids_AllSpecies parses the 14 real companion
// species from the Beastheart book source through StatblockParser and checks every
// grid shape: 1S/1M/1L sizes, em-dash movement/immunity (omitted), the drake's
// linked immunity (link text kept, link stripped), Skills link-stripping, and the
// constants every companion shares (SC-390). It also pins the SCC identity: the
// entity moved from feature-group to statblock WITHOUT changing its code.
func TestStatblockParser_CompanionGrids_AllSpecies(t *testing.T) {
	data, err := os.ReadFile("../../input/beastheart/Draw Steel Beastheart.md")
	if err != nil {
		t.Skipf("skipping book-source test: %v", err)
	}
	doc, err := parser.ParseDocument(data)
	if err != nil {
		t.Fatalf("ParseDocument: %v", err)
	}
	want := map[string]companionWant{
		"basilisk":        {[]string{"Beast", "Companion"}, "1L", 5, 2, "", []string{"Poison 3"}, []string{"Alertness"}, [5]int{2, 1, -1, 2, 2}},
		"bear":            {[]string{"Animal", "Companion"}, "1L", 5, 2, "Climb", nil, []string{"Intimidate"}, [5]int{2, 1, -1, 2, 2}},
		"boar":            {[]string{"Animal", "Companion"}, "1M", 5, 2, "", nil, []string{"Search"}, [5]int{2, 1, -1, 2, 2}},
		"condor":          {[]string{"Animal", "Companion"}, "1M", 7, 0, "Fly", nil, []string{"Alertness"}, [5]int{2, 1, -1, 2, 1}},
		"deinonychus":     {[]string{"Animal", "Companion"}, "1M", 7, 1, "", nil, []string{"Track"}, [5]int{2, 2, -1, 2, 1}},
		"drake":           {[]string{"Companion", "Dragon"}, "1M", 5, 1, "Fly", []string{"Attuned damage type 3 (see Elementally Attuned)"}, []string{"Intimidate"}, [5]int{2, 1, -1, 2, 2}},
		"elemental-spark": {[]string{"Companion", "Elemental"}, "1M", 7, 1, "", []string{"Lightning 3"}, []string{"Magic"}, [5]int{2, 2, -1, 2, 1}},
		"gummy-ball":      {[]string{"Companion", "Ooze"}, "1L", 5, 2, "", []string{"Acid 3"}, []string{"Sneak"}, [5]int{2, 2, -1, 2, 1}},
		"hellhound":       {[]string{"Companion", "Infernal"}, "1M", 7, 1, "", []string{"Fire 3"}, []string{"Intimidate"}, [5]int{2, 2, -1, 2, 1}},
		"lightbender":     {[]string{"Beast", "Companion"}, "1L", 7, 2, "", nil, []string{"Hide"}, [5]int{2, 1, -1, 2, 2}},
		"panther":         {[]string{"Animal", "Companion"}, "1M", 7, 1, "Climb", nil, []string{"Sneak"}, [5]int{2, 2, -1, 2, 1}},
		"spider":          {[]string{"Animal", "Companion"}, "1M", 5, 1, "Climb", nil, []string{"Sneak"}, [5]int{2, 2, -1, 2, 1}},
		"sporeling":       {[]string{"Beast", "Companion"}, "1S", 5, 0, "", []string{"Poison 3"}, []string{"Track"}, [5]int{2, 2, -1, 2, 1}},
		"wolf":            {[]string{"Animal", "Companion"}, "1M", 7, 1, "", nil, []string{"Track"}, [5]int{2, 2, -1, 2, 1}},
	}

	seen := 0
	for _, top := range doc.Sections {
		for _, sec := range top.AllSections() {
			species := sec.Annotation["companion"]
			if species == "" {
				continue
			}
			seen++
			if sec.Type() != "statblock" {
				t.Errorf("%s: annotated @type %q, want statblock", species, sec.Type())
			}
			w, ok := want[species]
			if !ok {
				t.Errorf("unexpected companion %q", species)
				continue
			}
			ctx := context.NewContextStack(context.Metadata{"book": "mcdm.beastheart.v1"})
			ctx.Push(2, context.Metadata{"type": "class", "id": "beastheart"})
			got, err := (&StatblockParser{}).Parse(ctx, sec)
			if err != nil {
				t.Fatalf("%s: parse: %v", species, err)
			}
			fm := got.Frontmatter
			if code := scc.Classify("mcdm.beastheart.v1", got.TypePath, got.ItemID); code != "mcdm.beastheart.v1/monster.companion.beastheart.statblock/"+species {
				t.Errorf("%s: scc = %s (code must not change)", species, code)
			}
			if fm["type"] != "statblock" || fm["organization"] != "Companion" || fm["companion"] != species {
				t.Errorf("%s: type/org/companion = %v/%v/%v", species, fm["type"], fm["organization"], fm["companion"])
			}
			// Shared by all 14: level, "= yours" stamina, "1 + M" free strike (text).
			if fm["level"] != 1 || fm["stamina"] != "= yours" || fm["free_strike"] != "1 + M" {
				t.Errorf("%s: level/stamina/free_strike = %v/%v/%v", species, fm["level"], fm["stamina"], fm["free_strike"])
			}
			if _, ok := fm["role"]; ok {
				t.Errorf("%s: companions have no role, got %v", species, fm["role"])
			}
			if _, ok := fm["ev"]; ok {
				t.Errorf("%s: companions have no EV, got %v", species, fm["ev"])
			}
			if !reflect.DeepEqual(fm["keywords"], w.keywords) || fm["size"] != w.size || fm["speed"] != w.speed || fm["stability"] != w.stability {
				t.Errorf("%s: keywords/size/speed/stability = %v/%v/%v/%v", species, fm["keywords"], fm["size"], fm["speed"], fm["stability"])
			}
			if mv, _ := fm["movement"].(string); mv != w.movement {
				t.Errorf("%s: movement = %q, want %q (em-dash is omitted)", species, mv, w.movement)
			}
			imm, hasImm := fm["immunities"]
			if (w.immunities == nil) == hasImm || (hasImm && !reflect.DeepEqual(imm, w.immunities)) {
				t.Errorf("%s: immunities = %v, want %v", species, imm, w.immunities)
			}
			if !reflect.DeepEqual(fm["skills"], w.skills) {
				t.Errorf("%s: skills = %v, want %v", species, fm["skills"], w.skills)
			}
			chars := [5]int{}
			for i, k := range []string{"might", "agility", "reason", "intuition", "presence"} {
				chars[i], _ = fm[k].(int)
			}
			if chars != w.chars {
				t.Errorf("%s: characteristics = %v, want %v", species, chars, w.chars)
			}
		}
	}
	if seen != len(want) {
		t.Errorf("found %d companion sections, want %d", seen, len(want))
	}
}

// TestParseStatblockFields_FreeStrikeTextAndSkills covers the two shared-parser
// extensions (SC-390): a non-integer Free Strike is kept as text ("1 + M") instead
// of being dropped, an integer one stays an int, and Skills become a link-free list.
func TestParseStatblockFields_FreeStrikeTextAndSkills(t *testing.T) {
	grid := "" +
		"| Animal, Companion | - | Level 1 | - | - |\n" +
		"|:-:|:-:|:-:|:-:|:-:|\n" +
		"| **1M**<br>Size | **7**<br>Speed | **= yours**<br>Stamina | **1**<br>Stability | **1 + M**<br>Free Strike |\n" +
		"| **—**<br>Immunity | **Climb**<br>Movement | **[Sneak](scc.v1:x/skill/sneak), [Hide](scc.v1:x/skill/hide)**<br>Skills | | |\n"
	fm := ParseStatblockFields("Panther", grid)
	if fm["free_strike"] != "1 + M" {
		t.Errorf("free_strike = %#v, want text \"1 + M\"", fm["free_strike"])
	}
	if !reflect.DeepEqual(fm["skills"], []string{"Sneak", "Hide"}) {
		t.Errorf("skills = %#v, want [Sneak Hide]", fm["skills"])
	}
	// A plain integer Free Strike is unchanged (the monsters' shape).
	fm = ParseStatblockFields("Goblin", cursespitterGrid)
	if fm["free_strike"] != 1 {
		t.Errorf("integer free_strike = %#v, want int 1", fm["free_strike"])
	}
	if _, ok := fm["skills"]; ok {
		t.Errorf("skills must be absent when the grid has no Skills cell, got %v", fm["skills"])
	}
}
