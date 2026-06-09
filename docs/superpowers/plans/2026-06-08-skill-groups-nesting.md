# Skill Groups Nesting — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Nest the 57 hero skills under their five rulebook skill groups, giving each skill a grouped SCC code `skill.<group>/<item>`, each group a linkable landing page `skill.<group>/<group>`, and rendering both index levels in the redesign `.sc-folder` / `.sc-card` UI.

**Architecture:** Mirror two existing precedents exactly — `rule.<group>/<term>` (grouped type via an `@group` annotation) for the leaves, and `monster.<category>/<category>` (self-named-leaf container) for the linkable group page. Source annotations + an in-prose link sweep drive the data; two small site-builder extensions drive the UI. No classifier or permalink-generator changes are needed.

**Tech Stack:** Go (steel-etl pipeline + site builder), annotated markdown (`input/heroes/Draw Steel Heroes.md`), MkDocs Material (v2 site). Run Go via devbox: `devbox run -- bash -c 'cd steel-etl && go …'`.

**Spec:** `steel-etl/docs/superpowers/specs/2026-06-08-skill-groups-nesting-design.md`

---

## Reference: skill-id → group map (authoritative)

Every `@type: skill` id maps to exactly one group. Used by the annotation pass (Task 3) and the link sweep (Task 4). Verified complete: all 57 ids, and every in-prose `scc:.../skill/<id>` link id, are covered.

```
crafting:      alchemy architecture blacksmithing carpentry cooking fletching forgery jewelry mechanics tailoring
exploration:   climb drive endurance gymnastics heal jump lift navigate ride swim
interpersonal: brag empathize flirt gamble handle-animals interrogate intimidate lead lie music perform persuade read-person
intrigue:      alertness conceal-object disguise eavesdrop escape-artist hide pick-lock pick-pocket sabotage search sneak track
lore:          criminal-underworld culture history magic monsters nature psionics religion rumors society strategy timescape
```

---

## Task 1: `SkillParser` reads `@group` → grouped TypePath

**Files:**
- Modify: `steel-etl/internal/content/skill.go`
- Test: `steel-etl/internal/content/skill_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/content/skill_test.go`:

```go
package content

import (
	"slices"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestSkillParserGroupedTypePath(t *testing.T) {
	p := &SkillParser{}
	sec := &parser.Section{
		Heading:    "Alchemy",
		Annotation: map[string]string{"type": "skill", "id": "alchemy", "group": "crafting"},
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if want := []string{"skill", "crafting"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
	if got.ItemID != "alchemy" {
		t.Errorf("ItemID = %q, want alchemy", got.ItemID)
	}
	if got.Frontmatter["type"] != "skill" {
		t.Errorf("type = %v, want skill", got.Frontmatter["type"])
	}
}

func TestSkillParserFlatWhenNoGroup(t *testing.T) {
	p := &SkillParser{}
	sec := &parser.Section{
		Heading:    "Alchemy",
		Annotation: map[string]string{"type": "skill", "id": "alchemy"},
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if want := []string{"skill"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v (flat fallback)", got.TypePath, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestSkillParser -v'`
Expected: FAIL — `TestSkillParserGroupedTypePath` gets `TypePath = [skill]`, want `[skill crafting]`.

- [ ] **Step 3: Implement `@group` handling**

Replace the body of `Parse` in `steel-etl/internal/content/skill.go` so the TypePath picks up the group (mirror of `RuleParser`):

```go
func (p *SkillParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	typePath := []string{"skill"}
	if group, ok := section.Annotation["group"]; ok && group != "" {
		typePath = []string{"skill", group}
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": section.Heading,
			"type": "skill",
		},
		Body:     section.FullBodySource(),
		TypePath: typePath,
		ItemID:   id,
	}, nil
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestSkillParser -v'`
Expected: PASS (both tests).

- [ ] **Step 5: Commit**

```bash
cd steel-etl && git add internal/content/skill.go internal/content/skill_test.go
git commit -m "feat(skill): SkillParser nests under @group (skill.<group>/<item>)"
```

---

## Task 2: `skill-group` self-named-leaf container parser

**Files:**
- Create: `steel-etl/internal/content/skill_group.go`
- Modify: `steel-etl/internal/content/registry.go:23` (register after `SkillParser`)
- Test: `steel-etl/internal/content/skill_group_test.go` (create)

The container emits the group's intro prose (its own `BodySource` plus the unannotated "<Group> Skills Table" child — `FullBodySource` already skips the annotated per-skill children) as a leaf page `skill.<group>/<group>`. It pushes **no** path context, so child skills get their group solely from their own `@group` annotation.

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/content/skill_group_test.go`:

```go
package content

import (
	"slices"
	"testing"

	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

func TestSkillGroupParserSelfNamedLeaf(t *testing.T) {
	p := &SkillGroupParser{}
	if p.Type() != "skill-group" {
		t.Fatalf("Type() = %q, want skill-group", p.Type())
	}
	sec := &parser.Section{
		Heading:    "Crafting Skills",
		Annotation: map[string]string{"type": "skill-group", "id": "crafting"},
		BodySource: "Skills from the crafting skill group are used in creation.",
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if want := []string{"skill", "crafting"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
	if got.ItemID != "crafting" {
		t.Errorf("ItemID = %q, want crafting", got.ItemID)
	}
	if got.Frontmatter["type"] != "skill-group" {
		t.Errorf("type = %v, want skill-group", got.Frontmatter["type"])
	}
	if got.Frontmatter["name"] != "Crafting Skills" {
		t.Errorf("name = %v, want \"Crafting Skills\"", got.Frontmatter["name"])
	}
}

func TestSkillGroupParserDerivesIDFromHeading(t *testing.T) {
	p := &SkillGroupParser{}
	sec := &parser.Section{
		Heading:    "Lore Skills",
		Annotation: map[string]string{"type": "skill-group"},
	}
	got, err := p.Parse(context.NewContextStack(nil), sec)
	if err != nil {
		t.Fatalf("Parse error: %v", err)
	}
	if got.ItemID != "lore-skills" {
		t.Errorf("ItemID = %q, want lore-skills (slug of heading when @id absent)", got.ItemID)
	}
	if want := []string{"skill", "lore-skills"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestSkillGroupParser -v'`
Expected: FAIL — `undefined: SkillGroupParser` (compile error).

- [ ] **Step 3: Implement the parser**

Create `steel-etl/internal/content/skill_group.go`:

```go
package content

import (
	"github.com/SteelCompendium/steel-etl/internal/context"
	"github.com/SteelCompendium/steel-etl/internal/parser"
)

// SkillGroupParser handles @type: skill-group sections — the five skill-group
// overview sections (Crafting, Exploration, Interpersonal, Intrigue, Lore).
// Each emits a self-named leaf page skill.<group>/<group> (mirroring the
// monster-group container monster.<category>/<category>) so prose can link to
// "the <group> skill group". The container pushes NO path context: child skills
// derive their group from their own @group annotation (see SkillParser), so the
// intro page and the leaf skills stay decoupled. FullBodySource carries the
// intro prose + the unannotated skills table; the annotated per-skill children
// are skipped (they become their own pages).
type SkillGroupParser struct{}

func (p *SkillGroupParser) Type() string { return "skill-group" }

func (p *SkillGroupParser) Parse(ctx *context.ContextStack, section *parser.Section) (*ParsedContent, error) {
	id := section.ID()
	if id == "" {
		id = Slugify(section.Heading)
	}

	return &ParsedContent{
		Frontmatter: map[string]any{
			"name": section.Heading,
			"type": "skill-group",
		},
		Body:     section.FullBodySource(),
		TypePath: []string{"skill", id},
		ItemID:   id,
	}, nil
}
```

- [ ] **Step 4: Register the parser**

In `steel-etl/internal/content/registry.go`, add the registration immediately after the `SkillParser` line (currently line 23):

```go
	r.Register(&SkillParser{})
	r.Register(&SkillGroupParser{})
```

- [ ] **Step 5: Run test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestSkillGroupParser -v'`
Expected: PASS (both tests).

- [ ] **Step 6: Commit**

```bash
cd steel-etl && git add internal/content/skill_group.go internal/content/skill_group_test.go internal/content/registry.go
git commit -m "feat(skill): add skill-group self-named-leaf container parser"
```

---

## Task 3: Annotate the source — `@group` on 57 skills + 5 `skill-group` containers

**Files:**
- Modify: `steel-etl/input/heroes/Draw Steel Heroes.md`

The skill definitions live under `#### Skill Groups` (≈ line 20682). Five H5 headings `##### <Group> Skills` each lead a block of `<!-- @type: skill | @id: x -->` comments.

- [ ] **Step 1: Add `@group` to each skill annotation and `skill-group` container to each H5**

Run this Python edit (operates only within the `#### Skill Groups` region so it never touches `@type: skill` outside it — there are none, but the region guard keeps it safe):

```bash
cd /home/scott/code/steelCompendium/workspace
python3 - <<'PY'
import re
path="steel-etl/input/heroes/Draw Steel Heroes.md"
lines=open(path).read().split("\n")
groups={"Crafting":"crafting","Exploration":"exploration","Interpersonal":"interpersonal","Intrigue":"intrigue","Lore":"lore"}
group=None
out=[]
for ln in lines:
    h=re.match(r'^##### (Crafting|Exploration|Interpersonal|Intrigue|Lore) Skills\s*$', ln)
    if h:
        group=groups[h.group(1)]
        out.append(f"<!-- @type: skill-group | @id: {group} -->")
        out.append(ln)
        continue
    a=re.match(r'^<!-- @type: skill \| @id: ([a-z0-9-]+) -->\s*$', ln)
    if a and group:
        out.append(f"<!-- @type: skill | @id: {a.group(1)} | @group: {group} -->")
        continue
    out.append(ln)
open(path,"w").write("\n".join(out))
print("skill-group containers added:", sum(1 for l in out if "@type: skill-group" in l))
print("grouped skill annotations:", sum(1 for l in out if re.search(r'@type: skill \| @id: .* \| @group:', l)))
PY
```

Expected output: `skill-group containers added: 5` and `grouped skill annotations: 57`.

- [ ] **Step 2: Verify build + classification**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl classify --diff 2>&1 | grep -E "skill\.(crafting|exploration|interpersonal|intrigue|lore)" | head'`
Expected: shows new codes like `skill.crafting/alchemy`, `skill.crafting/crafting`, etc. (registry diff — codes moved from `skill/*` to `skill.<group>/*`).

- [ ] **Step 3: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace
git -C steel-etl add input/heroes/"Draw Steel Heroes.md"
git -C steel-etl commit -m "feat(heroes): annotate skills with @group + skill-group containers"
```

---

## Task 4: Rewrite in-prose skill links → `skill.<group>/<item>`

**Files:**
- Modify: `steel-etl/input/heroes/Draw Steel Heroes.md`

All 177 `scc:mcdm.heroes.v1/skill/<id>` references become `scc:mcdm.heroes.v1/skill.<group>/<id>`. Fully deterministic via the id→group map.

- [ ] **Step 1: Run the link rewrite**

```bash
cd /home/scott/code/steelCompendium/workspace
python3 - <<'PY'
import re
path="steel-etl/input/heroes/Draw Steel Heroes.md"
g={}
for grp,ids in {
 "crafting":"alchemy architecture blacksmithing carpentry cooking fletching forgery jewelry mechanics tailoring",
 "exploration":"climb drive endurance gymnastics heal jump lift navigate ride swim",
 "interpersonal":"brag empathize flirt gamble handle-animals interrogate intimidate lead lie music perform persuade read-person",
 "intrigue":"alertness conceal-object disguise eavesdrop escape-artist hide pick-lock pick-pocket sabotage search sneak track",
 "lore":"criminal-underworld culture history magic monsters nature psionics religion rumors society strategy timescape",
}.items():
    for i in ids.split(): g[i]=grp
txt=open(path).read()
def repl(m):
    sid=m.group(1)
    if sid not in g:
        raise SystemExit(f"unmapped skill id in link: {sid}")
    return f"scc:mcdm.heroes.v1/skill.{g[sid]}/{sid}"
new,n=re.subn(r'scc:mcdm\.heroes\.v1/skill/([a-z0-9-]+)', repl, txt)
open(path,"w").write(new)
print("rewrote", n, "skill links")
PY
```

Expected: `rewrote 177 skill links`.

- [ ] **Step 2: Verify zero flat skill links remain**

Run: `grep -c "scc:mcdm.heroes.v1/skill/[a-z]" "steel-etl/input/heroes/Draw Steel Heroes.md"`
Expected: `0`.

- [ ] **Step 3: Verify links resolve (no broken SCC references)**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate 2>&1 | tail -20'`
Expected: no increase in broken/unknown links; the validate summary reports the skill codes as known. (If validate flags broken `skill.<group>/<id>` links, the group in a link doesn't match the skill's annotated group — re-check Task 3.)

- [ ] **Step 4: Commit**

```bash
git -C steel-etl add input/heroes/"Draw Steel Heroes.md"
git -C steel-etl commit -m "refactor(heroes): repoint 177 in-prose skill links to skill.<group>/<item>"
```

---

## Task 5: Link the `<group> skill group` phrases → `skill.<group>/<group>`

**Files:**
- Modify: `steel-etl/input/heroes/Draw Steel Heroes.md`

Wrap each singular `<group> skill group` phrase in a link to the group's landing page, **except** inside the Skill Groups definition region (self-reference) and except phrases already inside a markdown link.

- [ ] **Step 1: Run the phrase sweep**

```bash
cd /home/scott/code/steelCompendium/workspace
python3 - <<'PY'
import re
path="steel-etl/input/heroes/Draw Steel Heroes.md"
lines=open(path).read().split("\n")

# Determine the Skill Groups definition region [start, end): from "#### Skill Groups"
# to the next heading of level <= 4 (#### or higher). Phrases there are self-reference.
start=end=None
for i,ln in enumerate(lines):
    if re.match(r'^#### Skill Groups\s*$', ln):
        start=i; continue
    if start is not None and re.match(r'^#{1,4} ', ln) and i>start:
        end=i; break
if start is None:
    raise SystemExit("could not locate '#### Skill Groups' region")
if end is None:
    end=len(lines)

groups=("crafting","exploration","interpersonal","intrigue","lore")
phrase_re=re.compile(r'(?<!\[)\b(crafting|exploration|interpersonal|intrigue|lore) skill group\b(?!\])', re.IGNORECASE)

def link_line(ln):
    def repl(m):
        word=m.group(1).lower()
        return f"[{m.group(0)}](scc:mcdm.heroes.v1/skill.{word}/{word})"
    return phrase_re.sub(repl, ln)

count=0
for i,ln in enumerate(lines):
    if start<=i<end:           # skip the definition region (self-reference)
        continue
    # skip lines where the phrase is already inside an existing link target
    if "](scc:" in ln and "skill group" in ln.lower() and phrase_re.search(ln) is None:
        continue
    new=link_line(ln)
    if new!=ln:
        count+=phrase_re.findall(ln).__len__()
        lines[i]=new
open(path,"w").write("\n".join(lines))
print("linked", count, "group phrases (region", start, "-", end, "skipped)")
PY
```

Expected: prints a count in the ~90–110 range (the ~118 total minus the self-reference occurrences inside the definition region). Note the exact number printed for the commit message.

- [ ] **Step 2: Spot-check a few links**

Run: `grep -nE "\[(crafting|interpersonal) skill group\]\(scc:mcdm.heroes.v1/skill\.(crafting|interpersonal)/(crafting|interpersonal)\)" "steel-etl/input/heroes/Draw Steel Heroes.md" | head`
Expected: several matches showing well-formed links (e.g. `[crafting skill group](scc:mcdm.heroes.v1/skill.crafting/crafting)`).

- [ ] **Step 3: Verify no double-wrapped links**

Run: `grep -cE "\]\(scc:mcdm.heroes.v1/skill\.[a-z]+/[a-z]+\)\]\(" "steel-etl/input/heroes/Draw Steel Heroes.md"`
Expected: `0` (no nested/double links produced).

- [ ] **Step 4: Validate + commit**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate 2>&1 | tail -5'`
Expected: clean (no new broken links).

```bash
git -C steel-etl add input/heroes/"Draw Steel Heroes.md"
git -C steel-etl commit -m "feat(heroes): link <group> skill group phrases to group landing pages"
```

---

## Task 6: Site — per-group skill card grid (`buildCardsContent`)

**Files:**
- Modify: `steel-etl/internal/site/cards.go:55-92` (`buildCardsContent`)
- Test: `steel-etl/internal/site/cards_test.go`

A nested skill leaf dir is `Browse/skill/<group>/` (files only, no subdirs). Render its skills as `skill` cards (like the treasure-leaf branch), and **exclude the self-named `<group>.md` container file** from the grid.

- [ ] **Step 1: Write the failing test**

Add to `steel-etl/internal/site/cards_test.go`:

```go
func TestBuildCardsContentNestedSkillLeaf(t *testing.T) {
	dir := t.TempDir()
	// the self-named container page (must be excluded from the card grid)
	os.WriteFile(filepath.Join(dir, "crafting.md"),
		[]byte("---\nname: Crafting Skills\ntype: skill-group\n---\n\nOverview.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "alchemy.md"),
		[]byte("---\nname: Alchemy\ntype: skill\n---\n\nMake bombs and potions.\n"), 0o644)
	os.WriteFile(filepath.Join(dir, "carpentry.md"),
		[]byte("---\nname: Carpentry\ntype: skill\n---\n\nCreate items out of wood.\n"), 0o644)

	// dir basename is "crafting", parent path contains "skill"
	skillDir := filepath.Join(dir) // emulate .../skill/crafting
	content, ok := buildCardsContent(skillDir, "crafting",
		[]string{"crafting.md", "alchemy.md", "carpentry.md"}, nil)
	if !ok {
		t.Fatalf("buildCardsContent ok=false, want true for nested skill leaf")
	}
	if !strings.Contains(content, ">Alchemy<") || !strings.Contains(content, ">Carpentry<") {
		t.Errorf("expected skill cards for Alchemy and Carpentry; got:\n%s", content)
	}
	if strings.Contains(content, "Crafting Skills") || strings.Contains(content, "crafting/\"") {
		t.Errorf("self-named container card should be excluded; got:\n%s", content)
	}
}
```

The test relies on `pathHasSegment` matching the `skill` segment. Because `t.TempDir()` won't contain a `skill` path segment, **adjust the test dir** to include one: replace the `skillDir` line with a real nested path under the temp dir:

```go
	skillDir := filepath.Join(dir, "skill", "crafting")
	os.MkdirAll(skillDir, 0o755)
	for _, f := range []string{"crafting.md", "alchemy.md", "carpentry.md"} {
		data, _ := os.ReadFile(filepath.Join(dir, f))
		os.WriteFile(filepath.Join(skillDir, f), data, 0o644)
	}
```

(Place those lines before the `buildCardsContent` call and keep the call using `skillDir`.)

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildCardsContentNestedSkillLeaf -v'`
Expected: FAIL — `buildCardsContent ok=false` (dirName "crafting" isn't a rich card type and the skill branch doesn't exist yet).

- [ ] **Step 3: Add the nested-skill branch + container exclusion**

In `steel-etl/internal/site/cards.go`, extend the type-resolution block in `buildCardsContent`. Replace:

```go
	cardType := dirName
	if !richCardTypes[dirName] {
		// Treasure leaves are nested (treasure/<tier>/<category>); render their
		// items as treasure cards even though the leaf dirName isn't "treasure".
		if len(subdirs) == 0 && len(files) > 0 && pathHasSegment(dir, "treasure") {
			cardType = "treasure"
		} else {
			return "", false
		}
	}
```

with:

```go
	cardType := dirName
	if !richCardTypes[dirName] {
		switch {
		// Treasure leaves are nested (treasure/<tier>/<category>); render their
		// items as treasure cards even though the leaf dirName isn't "treasure".
		case len(subdirs) == 0 && len(files) > 0 && pathHasSegment(dir, "treasure"):
			cardType = "treasure"
		// Skill leaves are nested (skill/<group>/<item>); render their items as
		// skill cards. The self-named <group>.md container page is dropped below.
		case len(subdirs) == 0 && len(files) > 0 && pathHasSegment(dir, "skill"):
			cardType = "skill"
			files = dropSelfNamed(files, dirName)
		default:
			return "", false
		}
	}
```

Then add the helper near `pathHasSegment` in the same file:

```go
// dropSelfNamed removes the self-named container page (<dirName>.md) from a leaf
// directory's file list, so a skill-group's landing page doesn't appear as a
// card inside its own group grid.
func dropSelfNamed(files []string, dirName string) []string {
	self := dirName + ".md"
	out := files[:0:0]
	for _, f := range files {
		if f != self {
			out = append(out, f)
		}
	}
	return out
}
```

- [ ] **Step 4: Run test to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildCardsContentNestedSkillLeaf -v'`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
cd steel-etl && git add internal/site/cards.go internal/site/cards_test.go
git commit -m "feat(site): render nested skill-group leaves as skill cards"
```

---

## Task 7: Site — folder index for the skill root + group crest/count

**Files:**
- Modify: `steel-etl/internal/site/feature_index.go` (`underFeatureOrTreasure` → generalize; `folderCrestIcon`; `countLeafFiles`)
- Test: `steel-etl/internal/site/feature_index_test.go`

The `Browse/skill/` root (subdirs = the 5 groups, no files) must render `.sc-folder` cards. Each folder shows the skill icon and a count that **excludes the self-named container** so it reads "10" not "11".

- [ ] **Step 1: Write the failing test**

Add to `steel-etl/internal/site/feature_index_test.go`:

```go
func TestBuildFeatureIndexContentSkillRoot(t *testing.T) {
	root := t.TempDir()
	skillRoot := filepath.Join(root, "skill")
	craft := filepath.Join(skillRoot, "crafting")
	os.MkdirAll(craft, 0o755)
	// self-named container + two skills → count should be 2, not 3
	for _, f := range []string{"crafting.md", "alchemy.md", "carpentry.md"} {
		os.WriteFile(filepath.Join(craft, f), []byte("---\nname: X\n---\n"), 0o644)
	}
	content, ok := buildFeatureIndexContent(skillRoot, "skill", nil, []string{"crafting"})
	if !ok {
		t.Fatalf("buildFeatureIndexContent ok=false, want true for skill root")
	}
	if !strings.Contains(content, `<a class="sc-folder" href="crafting/">`) ||
		!strings.Contains(content, `<h3 class="sc-folder__name">Crafting</h3>`) {
		t.Errorf("expected a Crafting folder card; got:\n%s", content)
	}
	if !strings.Contains(content, `<span class="sc-folder__count">2</span>`) {
		t.Errorf("expected count 2 (container excluded); got:\n%s", content)
	}
	if !strings.Contains(content, iconPaths["skill"]) {
		t.Errorf("expected the skill crest glyph on group folder cards; got:\n%s", content)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildFeatureIndexContentSkillRoot -v'`
Expected: FAIL — `ok=false` (the folder-index gate only matches feature/treasure).

- [ ] **Step 3: Generalize the folder-index gate**

In `steel-etl/internal/site/feature_index.go`, rename and extend `underFeatureOrTreasure`. Replace:

```go
// underFeatureOrTreasure reports whether dir is the feature/ or treasure/ node
// itself, or any node beneath them.
func underFeatureOrTreasure(dir string) bool {
	for _, p := range strings.Split(filepath.ToSlash(dir), "/") {
		if p == "feature" || p == "treasure" {
			return true
		}
	}
	return false
}
```

with:

```go
// usesFolderIndex reports whether dir is one of the grouped Browse trees
// (feature/, treasure/, skill/) — the index-of-indexes nodes that render as
// .sc-folder cards. Other sections keep the default browse-index list.
func usesFolderIndex(dir string) bool {
	for _, p := range strings.Split(filepath.ToSlash(dir), "/") {
		if p == "feature" || p == "treasure" || p == "skill" {
			return true
		}
	}
	return false
}
```

Update the call site at the top of `buildFeatureIndexContent`:

```go
	if len(subdirs) > 0 && len(files) == 0 && usesFolderIndex(dir) {
		return buildFolderIndex(dir, dirName, subdirs), true
	}
```

Update the existing gate test in `feature_index_test.go` (around line 350) that calls `underFeatureOrTreasure` — rename it to `usesFolderIndex` and add a `skill` true-case:

```go
	cases := map[string]bool{
		"docs/Browse/feature":           true,
		"docs/Browse/feature/ability":   true,
		"docs/Browse/treasure":          true,
		"docs/Browse/treasure/leveled":  true,
		"docs/Browse/skill":             true,
		"docs/Browse/skill/crafting":    true,
		"docs/Browse/condition":         false,
	}
	for in, want := range cases {
		if got := usesFolderIndex(in); got != want {
			t.Errorf("usesFolderIndex(%q)=%v want %v", in, got, want)
		}
	}
```

(If the existing test body differs, just swap the function name and add the two `skill` rows.)

- [ ] **Step 4: Give skill-group folders the skill crest**

In `folderCrestIcon`, add a `skill` case to the path-based `switch` (before `default`):

```go
	case strings.Contains(slash, "/skill"):
		return "skill"
```

- [ ] **Step 5: Exclude self-named containers from the folder count**

In `countLeafFiles`, skip a file whose basename matches its parent directory (the self-named container). Replace the inner predicate:

```go
		if strings.HasSuffix(base, ".md") && base != "index.md" && base != "_Index.md" {
			n++
		}
```

with:

```go
		if strings.HasSuffix(base, ".md") && base != "index.md" && base != "_Index.md" &&
			strings.TrimSuffix(base, ".md") != filepath.Base(filepath.Dir(path)) {
			n++
		}
```

- [ ] **Step 6: Run tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run "TestBuildFeatureIndexContentSkillRoot|usesFolderIndex|TestUnderFeature" -v'`
Expected: PASS. Then run the whole site package: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/'` — expected PASS (no regression from the count/gate rename).

- [ ] **Step 7: Commit**

```bash
cd steel-etl && git add internal/site/feature_index.go internal/site/feature_index_test.go
git commit -m "feat(site): skill root renders group folder cards (skill crest, container-excluded count)"
```

---

## Task 8: Update docs

**Files:**
- Modify: `steel-etl/docs/linking-reference.md`
- Modify: `steel-etl/docs/linking-guide.md`
- Modify: `steel-etl/CLAUDE.md`
- Modify: `CLAUDE.md` (workspace root)

- [ ] **Step 1: Update `linking-reference.md`**

Re-point the 57 skill entries from `skill/<id>` to `skill.<group>/<id>` (use the id→group map). Add 5 group terms:

```
| crafting skill group | skill.crafting/crafting |
| exploration skill group | skill.exploration/exploration |
| interpersonal skill group | skill.interpersonal/interpersonal |
| intrigue skill group | skill.intrigue/intrigue |
| lore skill group | skill.lore/lore |
```

(Match the file's existing column/format; if it lists a term count in a header, bump it by 5.)

- [ ] **Step 2: Update `linking-guide.md`**

Add a short note under the skills section: skills now classify as `skill.<group>/<item>`; the five `<group> skill group` phrases link to `skill.<group>/<group>`; skip the phrase inside the Skill Groups definition region (self-reference).

- [ ] **Step 3: Update `steel-etl/CLAUDE.md`**

In the parser/type discussion, note the new `skill-group` container type and the `skill.<group>/<item>` shape (alongside the existing `rule.<group>/<term>` and monster-group references).

- [ ] **Step 4: Update workspace `CLAUDE.md`**

In the SCC-registry paragraph, note that skills were nested under their five groups on 2026-06-08 (`skill.<group>/<item>`), each group is a linkable self-named-leaf landing page (`skill.<group>/<group>`) via the new `skill-group` type, and refresh the heroes-doc link-count figure (it grew by the ~90–110 group-phrase links from Task 5; use the actual number).

- [ ] **Step 5: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace
git -C steel-etl add docs/linking-reference.md docs/linking-guide.md CLAUDE.md
git -C steel-etl commit -m "docs: skill-group nesting (linking refs/guide, steel-etl CLAUDE)"
git add CLAUDE.md
git commit -m "docs: note skill-group nesting in workspace CLAUDE.md"
```

---

## Task 9: Full regen + validate + visual verification

**Files:** none (generates output under `v2/docs/Browse/skill/` and `data/`)

- [ ] **Step 1: Run the full pipeline + site build**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml && go run ./cmd/steel-etl site --config ../v2/site.yaml'`
Expected: completes without error.

- [ ] **Step 2: Verify the nested directory structure**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
find v2/docs/Browse/skill -maxdepth 2 -type d
ls v2/docs/Browse/skill/crafting/
```
Expected: five group dirs (`crafting`, `exploration`, `interpersonal`, `intrigue`, `lore`); `crafting/` contains `index.md`, `crafting.md` (container), and the 10 skill files. **No** flat `v2/docs/Browse/skill/<skill>.md` leaves remain.

- [ ] **Step 3: Verify codes on generated leaves**

Run:
```bash
head -5 v2/docs/Browse/skill/crafting/alchemy.md
head -5 v2/docs/Browse/skill/crafting/crafting.md
```
Expected: `scc: mcdm.heroes.v1/skill.crafting/alchemy` and `scc: mcdm.heroes.v1/skill.crafting/crafting` respectively.

- [ ] **Step 4: Verify the redesigned indexes**

Run:
```bash
grep -c "sc-folder" v2/docs/Browse/skill/index.md
grep -c "sc-card\b" v2/docs/Browse/skill/crafting/index.md
grep "sc-folder__count" v2/docs/Browse/skill/index.md
```
Expected: root `index.md` has 5 `sc-folder` cards with counts (10/10/13/12/12); `crafting/index.md` is a `sc-card` grid of 10 skills (no Crafting-Skills container card).

- [ ] **Step 5: Verify permalink stub for a group page**

Run: `ls v2/site 2>/dev/null && find v2 -path "*scc*skill.crafting*" -maxdepth 6 2>/dev/null | head` — or, if the stub generation runs during `site`, grep the build output. Expected: a redirect stub exists for `mcdm.heroes.v1/skill.crafting/crafting`. (If stubs are only emitted by `deploy`, note that and defer the check to deploy.)

- [ ] **Step 6: Confirm no stragglers**

Run:
```bash
grep -rn "scc:mcdm.heroes.v1/skill/[a-z]" "steel-etl/input/heroes/Draw Steel Heroes.md" | wc -l
grep -rln "mcdm.heroes.v1/skill/[a-z]" v2/docs/Browse/skill/ | grep -v "skill\." | head
```
Expected: `0` and no output (no flat-code references anywhere).

- [ ] **Step 7: Run the full Go test suite**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./...'`
Expected: all PASS.

- [ ] **Step 8: Commit generated output**

```bash
cd /home/scott/code/steelCompendium/workspace
git -C steel-etl add -A && git -C steel-etl commit -m "chore: regen skill-group nested output" || true
# v2 + data repos are separate working trees; stage their changes too:
git add v2/docs/Browse/skill 2>/dev/null; git commit -m "chore(v2): regen nested skill Browse pages" || true
```

(Adjust to whichever repos own `v2/docs` and `data/` — match the existing deploy-commit pattern; do **not** push unless asked.)

- [ ] **Step 9: Hand to Scott for visual review**

Per Scott's working style, he reviews the rendered v2 site. Surface: the skill root (5 folder cards), one per-group grid, and a group landing page (`skill.crafting/crafting`) — confirm the redesign UI renders and the group-phrase links resolve before any deploy.

---

## Self-review notes

- **Spec coverage:** §1 source annotations → Tasks 3; §2 parsers → Tasks 1–2; §3 index redesign → Tasks 6–7; §4 linkable group page → Task 2 (page) + Task 5 (phrase links) + Task 9 step 5 (stub); §5 link sweep + docs → Tasks 4, 5, 8; §6 verification → Task 9. All covered.
- **Container-in-grid exclusion:** handled in Task 6 (`dropSelfNamed`) and Task 7 (count predicate) — both call out the self-named `<group>.md`.
- **Type names are consistent:** `SkillGroupParser` / `@type: skill-group` / `skill-group` registry key used identically across Tasks 2, 3, 8.
- **Deferred decisions:** the exact group-phrase link count (Task 5) and final link-count figure (Task 8 step 4) are read from command output at run time — recorded, not guessed.
