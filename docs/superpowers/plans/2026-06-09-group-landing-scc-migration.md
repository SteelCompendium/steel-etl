# Group-Landing SCC Migration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Migrate the two self-named-leaf group landings (`skill.<g>/<g>`, `monster.<cat>/<cat>`) to a unified `<type>.group/<member>` SCC scheme (`skill.group/crafting`, `monster.group/goblins`), rendering each landing AT its Browse group index.

**Architecture:** Two one-line parser changes set `TypePath` to `["skill","group"]` / `["monster","group"]` (code == canonical file path). A single general site-builder relocation rule maps any `<root>/group/<member>.md` source page to `<root>/<member>/index.md`, which (a) makes the landing the group index, (b) gives its `scc` to the permalink stub, and (c) makes the phantom `group/` folder vanish. A merge step folds the relocated lore above the generated card grid / browse list. A scoped prose repoint updates 164 heroes-doc links; monsters have none.

**Tech Stack:** Go (steel-etl pipeline + site builder), annotated markdown (`input/heroes/Draw Steel Heroes.md`), MkDocs Material (v2). Run Go via devbox from the **workspace root**: `devbox run -- bash -c 'cd steel-etl && go …'`.

**Spec:** `steel-etl/docs/superpowers/specs/2026-06-09-group-landing-scc-migration-design.md`

**Working branch:** `feat/group-landing-scc-migration` (already created; spec already committed).

---

## File map

| File | Change |
|---|---|
| `steel-etl/internal/content/skill_group.go` | `TypePath` → `["skill","group"]` (item stays the group id) |
| `steel-etl/internal/content/skill_group_test.go` | Update both tests to expect `["skill","group"]` |
| `steel-etl/internal/content/monster.go` | `MonsterParser.TypePath` → `["monster","group"]` (item stays category) |
| `steel-etl/internal/content/monster_test.go` | Update `TestMonsterParser`; keep statblock-context regression |
| `steel-etl/internal/site/build.go` | Add `groupLandingIndexDest`, wire into `buildSection`; add `mergeGroupLanding`/`stripTrailingTable`/`stripLeadingHeading`, wire into `generateIndexesRecursive` |
| `steel-etl/internal/site/build_test.go` (create) | Unit tests for the new site helpers |
| `steel-etl/internal/site/cards.go` | Remove the now-dead `dropSelfNamed` call + function |
| `steel-etl/input/heroes/Draw Steel Heroes.md` | Repoint 164 `skill.<g>/<g>` → `skill.group/<g>` |
| `steel-etl/CLAUDE.md`, workspace `CLAUDE.md` | Doc updates |

---

## Task 1: SkillGroupParser → `skill.group/<id>`

**Files:**
- Modify: `steel-etl/internal/content/skill_group.go:33`
- Test: `steel-etl/internal/content/skill_group_test.go:25-27,52-54`

- [ ] **Step 1: Update the failing tests**

In `steel-etl/internal/content/skill_group_test.go`, change the two `TypePath` expectations from `["skill", <id>]` to `["skill", "group"]` (the item id stays in `ItemID`). Replace lines 25-27:

```go
	if want := []string{"skill", "group"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
```

and replace lines 52-54:

```go
	if want := []string{"skill", "group"}; !slices.Equal(got.TypePath, want) {
		t.Errorf("TypePath = %v, want %v", got.TypePath, want)
	}
```

Also rename `TestSkillGroupParserSelfNamedLeaf` → `TestSkillGroupParserGroupLanding` (line 11) to reflect the new shape. `ItemID` expectations (`"crafting"`, `"lore-skills"`) are unchanged.

- [ ] **Step 2: Run the tests to verify they fail**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestSkillGroupParser -v'`
Expected: FAIL — `TypePath = [skill crafting], want [skill group]`.

- [ ] **Step 3: Change the parser**

In `steel-etl/internal/content/skill_group.go`, change line 33 from:

```go
		TypePath: []string{"skill", id},
```

to:

```go
		TypePath: []string{"skill", "group"},
```

(`ItemID: id` on the next line is unchanged — code becomes `skill.group/<id>`.) Update the doc comment at lines 9-12 to read:

```go
// Each emits a group-landing page skill.group/<group> (e.g. skill.group/crafting)
// so prose can link to "the <group> skill group". The container pushes NO path
// context: child skills derive their group from their own @group annotation (see
// SkillParser), so the landing page and the leaf skills stay decoupled.
```

- [ ] **Step 4: Run the tests to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestSkillGroupParser -v'`
Expected: PASS (both).

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add internal/content/skill_group.go internal/content/skill_group_test.go
git commit -m "feat(scc): skill-group landing → skill.group/<group>"
```

---

## Task 2: MonsterParser → `monster.group/<category>`

**Files:**
- Modify: `steel-etl/internal/content/monster.go:168-170`
- Test: `steel-etl/internal/content/monster_test.go:73-75`

- [ ] **Step 1: Update the failing test**

In `steel-etl/internal/content/monster_test.go`, change `TestMonsterParser`'s TypePath assertion (lines 73-75) from `monster/goblins` to `monster/group`:

```go
	if strings.Join(got.TypePath, "/") != "monster/group" {
		t.Errorf("TypePath: got %v, want [monster group]", got.TypePath)
	}
```

`ItemID` stays `"goblins"`. Do **not** touch `TestStatblockParser` / `TestFeatureblockParser` — they assert `monster/goblins/statblock` and `monster/goblins` from **context** and must stay green to prove descendants are unaffected.

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestMonsterParser -v'`
Expected: FAIL — `TypePath: got [monster goblins], want [monster group]`.

- [ ] **Step 3: Change the parser**

In `steel-etl/internal/content/monster.go`, change line 168 from:

```go
		TypePath:    []string{"monster", category},
```

to:

```go
		TypePath:    []string{"monster", "group"},
```

(`ItemID: category` is unchanged — code becomes `monster.group/<category>`.) Update the doc comment at lines 140-143 to read:

```go
// MonsterParser handles @type: monster sections — a monster group (e.g.
// "Goblins"). It produces a lore landing page at monster.group/{category}
// AND seeds the `category` (and optional `domain`) context the pipeline pushes
// for its descendant statblocks and featureblocks.
```

- [ ] **Step 4: Run content tests to verify pass + no descendant leak**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run "TestMonsterParser|TestStatblockParser|TestFeatureblockParser" -v'`
Expected: PASS (all three — statblock/featureblock prove context unaffected).

- [ ] **Step 5: Commit**

```bash
cd steel-etl
git add internal/content/monster.go internal/content/monster_test.go
git commit -m "feat(scc): monster-group landing → monster.group/<category>"
```

---

## Task 3: Site builder — relocate group landings to the group index

**Files:**
- Modify: `steel-etl/internal/site/build.go:186-196` (buildSection destRel decision)
- Modify: `steel-etl/internal/site/build.go` (add `groupLandingIndexDest` helper)
- Test: `steel-etl/internal/site/build_test.go` (create)

- [ ] **Step 1: Write the failing test**

Create `steel-etl/internal/site/build_test.go`:

```go
package site

import "testing"

func TestGroupLandingIndexDest(t *testing.T) {
	cases := []struct {
		in       string
		wantDest string
		wantOK   bool
	}{
		{"skill/group/crafting.md", "skill/crafting/index.md", true},
		{"monster/group/goblins.md", "monster/goblins/index.md", true},
		{"skill/crafting/cooking.md", "", false},        // leaf skill, not a landing
		{"monster/goblins/statblock/cutter.md", "", false}, // statblock
		{"feature/ability/fury/level-1/gouge.md", "", false},
	}
	for _, c := range cases {
		dest, ok := groupLandingIndexDest(c.in)
		if ok != c.wantOK || dest != c.wantDest {
			t.Errorf("groupLandingIndexDest(%q) = (%q,%v), want (%q,%v)",
				c.in, dest, ok, c.wantDest, c.wantOK)
		}
	}
}
```

- [ ] **Step 2: Run to verify it fails**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestGroupLandingIndexDest -v'`
Expected: FAIL — `undefined: groupLandingIndexDest`.

- [ ] **Step 3: Add the helper**

Add to `steel-etl/internal/site/build.go` (near `applyGroups`, after line ~468):

```go
// groupLandingIndexDest maps a unified group-landing source path to the group's
// index page:
//
//	<root>/group/<member>.md   ->   <root>/<member>/index.md
//
// So a skill.group/crafting page (file skill/group/crafting.md) renders AS the
// /Browse/skill/crafting/ index — carrying its scc to the permalink stub — and no
// phantom <root>/group/ subtree is ever created. ok=false for anything else.
func groupLandingIndexDest(relPath string) (string, bool) {
	parts := strings.Split(filepath.ToSlash(relPath), "/")
	if len(parts) == 3 && parts[1] == "group" && strings.HasSuffix(parts[2], ".md") {
		member := strings.TrimSuffix(parts[2], ".md")
		return parts[0] + "/" + member + "/index.md", true
	}
	return "", false
}
```

- [ ] **Step 4: Run to verify it passes**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestGroupLandingIndexDest -v'`
Expected: PASS.

- [ ] **Step 5: Wire it into buildSection**

In `steel-etl/internal/site/build.go`, replace the destRel decision block (lines 186-196) — currently:

```go
		var destRel, parentName string
		if section.GroupByBook {
			if srcBookFolder == "" {
				key := bookKeyFromSCC(parseFrontmatterField(fm, "scc"))
				errs = append(errs, fmt.Sprintf("no book config for scc prefix %q (%s)", key, entry.relPath))
				continue
			}
			destRel = filepath.ToSlash(filepath.Join(srcBookFolder, filepath.Base(entry.relPath)))
		} else {
			destRel, parentName = applyGroups(entry.relPath, section.Groups, entry.sourceDir)
		}
```

with (add the group-landing branch first):

```go
		var destRel, parentName string
		if dest, ok := groupLandingIndexDest(entry.relPath); ok {
			// Group landing (skill.group/* , monster.group/*) renders AS the
			// <root>/<member>/ index; mergeGroupLanding folds it above the listing.
			destRel = dest
		} else if section.GroupByBook {
			if srcBookFolder == "" {
				key := bookKeyFromSCC(parseFrontmatterField(fm, "scc"))
				errs = append(errs, fmt.Sprintf("no book config for scc prefix %q (%s)", key, entry.relPath))
				continue
			}
			destRel = filepath.ToSlash(filepath.Join(srcBookFolder, filepath.Base(entry.relPath)))
		} else {
			destRel, parentName = applyGroups(entry.relPath, section.Groups, entry.sourceDir)
		}
```

- [ ] **Step 6: Run the full site package to confirm no regressions yet**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/'`
Expected: PASS (the relocation is inert except for `group/` paths; the merge in Task 4 completes the behavior).

- [ ] **Step 7: Commit**

```bash
cd steel-etl
git add internal/site/build.go internal/site/build_test.go
git commit -m "feat(site): relocate <root>/group/<member> landing to the group index"
```

---

## Task 4: Site builder — merge relocated landing above the generated index

**Files:**
- Modify: `steel-etl/internal/site/build.go:907` (call `mergeGroupLanding`)
- Modify: `steel-etl/internal/site/build.go` (add `mergeGroupLanding`, `stripLeadingHeading`, `stripTrailingTable`)
- Modify: `steel-etl/internal/site/cards.go:71-75,107-119` (remove dead `dropSelfNamed`)
- Test: `steel-etl/internal/site/build_test.go`

- [ ] **Step 1: Write the failing tests**

Append to `steel-etl/internal/site/build_test.go`:

```go
import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestStripLeadingHeading(t *testing.T) {
	in := "# Crafting\n\n---\n\n<div class=\"sc-cards\">X</div>\n"
	want := "<div class=\"sc-cards\">X</div>\n"
	if got := stripLeadingHeading(in); got != want {
		t.Errorf("stripLeadingHeading = %q, want %q", got, want)
	}
	// No leading heading → unchanged.
	plain := "no heading here"
	if got := stripLeadingHeading(plain); got != plain {
		t.Errorf("stripLeadingHeading(plain) = %q, want unchanged", got)
	}
}

func TestStripTrailingTable(t *testing.T) {
	in := "# Crafting Skills\n\nIntro prose.\n\n| Skill | Desc |\n|---|---|\n| Cooking | food |\n"
	want := "# Crafting Skills\n\nIntro prose."
	if got := stripTrailingTable(in); got != want {
		t.Errorf("stripTrailingTable = %q, want %q", got, want)
	}
	// No trailing table → unchanged (trimmed).
	noTable := "# Goblins\n\nThey are crafty."
	if got := stripTrailingTable(noTable); got != noTable {
		t.Errorf("stripTrailingTable(noTable) = %q, want unchanged", got)
	}
}

func TestMergeGroupLanding(t *testing.T) {
	dir := t.TempDir()
	landing := "---\nname: Crafting Skills\nscc: mcdm.heroes.v1/skill.group/crafting\ntype: skill-group\n---\n# Crafting Skills\n\nThe crafting group makes things.\n\n| Skill | Desc |\n|---|---|\n| Cooking | food |\n"
	if err := os.WriteFile(filepath.Join(dir, "index.md"), []byte(landing), 0644); err != nil {
		t.Fatal(err)
	}
	generated := "# Crafting\n\n---\n\n<div class=\"sc-cards\">CARDS</div>\n"
	got := mergeGroupLanding(dir, generated)

	if !strings.Contains(got, "scc: mcdm.heroes.v1/skill.group/crafting") {
		t.Error("merged index lost the scc frontmatter")
	}
	if !strings.Contains(got, "The crafting group makes things.") {
		t.Error("merged index lost the lore")
	}
	if strings.Contains(got, "| Cooking | food |") {
		t.Error("merged index kept the redundant skills table")
	}
	if !strings.Contains(got, "<div class=\"sc-cards\">CARDS</div>") {
		t.Error("merged index lost the generated card grid")
	}
	if strings.Contains(got, "# Crafting\n\n---") {
		t.Error("merged index kept the generated duplicate H1")
	}
}

func TestMergeGroupLandingNoSCCPassthrough(t *testing.T) {
	dir := t.TempDir()
	// index.md without scc (a normal generated index) → generated returned as-is.
	os.WriteFile(filepath.Join(dir, "index.md"), []byte("# X\n\n---\n\nlist"), 0644)
	generated := "# X\n\n---\n\nlist"
	if got := mergeGroupLanding(dir, generated); got != generated {
		t.Errorf("mergeGroupLanding passthrough = %q, want unchanged", got)
	}
}
```

(If `build_test.go` already has an `import "testing"` from Task 3, merge the import blocks rather than duplicating.)

- [ ] **Step 2: Run to verify they fail**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run "TestStripLeadingHeading|TestStripTrailingTable|TestMergeGroupLanding" -v'`
Expected: FAIL — `undefined: stripLeadingHeading` (etc.).

- [ ] **Step 3: Add the helpers**

Add to `steel-etl/internal/site/build.go` (after `groupLandingIndexDest`):

```go
// mergeGroupLanding folds a relocated group-landing page (placed at dir/index.md
// by buildSection, carrying scc frontmatter + lore) into the generated index
// `generated` (card grid for skills, browse list for monsters). It preserves the
// landing's frontmatter — so the scc permalink stub targets THIS dir — and its
// lore, drops the generated listing's duplicate leading "# Title\n\n---\n\n", and
// strips any trailing table in the lore that the listing below supersedes. If
// dir/index.md is absent or has no scc, `generated` is returned unchanged.
func mergeGroupLanding(dir, generated string) string {
	data, err := os.ReadFile(filepath.Join(dir, "index.md"))
	if err != nil {
		return generated
	}
	fm, body := splitFrontmatter(string(data))
	if parseFrontmatterField(fm, "scc") == "" {
		return generated
	}
	lore := stripTrailingTable(strings.TrimRight(body, "\n"))
	listing := stripLeadingHeading(generated)

	var sb strings.Builder
	sb.WriteString("---\n")
	sb.WriteString(fm)
	sb.WriteString("\n---\n")
	sb.WriteString(strings.TrimLeft(lore, "\n"))
	sb.WriteString("\n\n---\n\n")
	sb.WriteString(listing)
	return sb.String()
}

// stripLeadingHeading drops the "# Title\n\n---\n\n" head that generated index
// content begins with, so a merged landing keeps only ITS own H1.
func stripLeadingHeading(s string) string {
	const sep = "\n---\n\n"
	if strings.HasPrefix(s, "# ") {
		if i := strings.Index(s, sep); i >= 0 {
			return s[i+len(sep):]
		}
	}
	return s
}

// stripTrailingTable removes a trailing GFM table (and its blank separator) from
// a group landing's lore — the index listing below already enumerates those rows.
func stripTrailingTable(body string) string {
	lines := strings.Split(body, "\n")
	end := len(lines)
	for end > 0 && strings.TrimSpace(lines[end-1]) == "" {
		end--
	}
	start := end
	for start > 0 && strings.HasPrefix(strings.TrimSpace(lines[start-1]), "|") {
		start--
	}
	if start == end { // no trailing table
		return body
	}
	for start > 0 && strings.TrimSpace(lines[start-1]) == "" {
		start-- // drop the blank line before the table
	}
	return strings.TrimRight(strings.Join(lines[:start], "\n"), "\n")
}
```

- [ ] **Step 4: Run to verify they pass**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run "TestStripLeadingHeading|TestStripTrailingTable|TestMergeGroupLanding" -v'`
Expected: PASS.

- [ ] **Step 5: Call the merge in the index walk**

In `steel-etl/internal/site/build.go`, in `generateIndexesRecursive`, change line 907 from:

```go
	content := buildIndexContent(dir, filepath.Base(dir), files, subdirs)
```

to:

```go
	content := buildIndexContent(dir, filepath.Base(dir), files, subdirs)
	content = mergeGroupLanding(dir, content)
```

- [ ] **Step 6: Remove the now-dead `dropSelfNamed`**

In `steel-etl/internal/site/cards.go`, the skill case no longer has a self-named `<group>.md` to drop (it moved to `index.md`, which the file walk already excludes). Change lines 71-75 from:

```go
	// Skill leaves are nested (skill/<group>/<item>); render their items as skill
	// cards. The self-named <group>.md container page is dropped below.
	case leaf && pathHasSegment(dir, "skill"):
		cardType = "skill"
		files = dropSelfNamed(files, dirName)
```

to:

```go
	// Skill leaves are nested (skill/<group>/<item>); render their items as skill
	// cards. The group landing lives in index.md (relocated from skill.group/*) and
	// is merged in by mergeGroupLanding — not listed as a card here.
	case leaf && pathHasSegment(dir, "skill"):
		cardType = "skill"
```

Then delete the `dropSelfNamed` function (cards.go lines 107-119) entirely.

- [ ] **Step 7: Run the full site package; fix any stale assertions**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./internal/site/'`
Expected: PASS. If `internal/site/cards_test.go` or `internal/site/feature_index_test.go` reference `dropSelfNamed` or assert a self-named `skill/<g>/<g>.md` card is dropped, update those assertions to the new behavior (landing is the index, not a card). Re-run until green.

- [ ] **Step 8: Commit**

```bash
cd steel-etl
git add internal/site/build.go internal/site/cards.go internal/site/build_test.go internal/site/*_test.go
git commit -m "feat(site): merge relocated group landing above the group index"
```

---

## Task 5: Repoint heroes-doc group links to `skill.group/<g>`

**Files:**
- Modify: `steel-etl/input/heroes/Draw Steel Heroes.md`

- [ ] **Step 1: Capture the before-counts**

Run (count occurrences — `grep -o | wc -l`, not `grep -c` which counts lines):
```bash
cd steel-etl
grep -oE 'skill\.(crafting|exploration|interpersonal|intrigue|lore)/(crafting|exploration|interpersonal|intrigue|lore)' "input/heroes/Draw Steel Heroes.md" | wc -l
```
Expected: `164` (the self-named group links). Note it.

- [ ] **Step 2: Repoint with a scoped, safe replace**

The replace only fires where the item equals its group (`skill.crafting/crafting`), never on leaf codes (`skill.crafting/cooking`), via a backreference + negative lookahead so no longer code is truncated:

```bash
cd steel-etl
perl -i -pe 's{skill\.(crafting|exploration|interpersonal|intrigue|lore)/\1(?![\w-])}{skill.group/$1}g' "input/heroes/Draw Steel Heroes.md"
```

- [ ] **Step 3: Verify the after-counts (guard)**

Run (all occurrence counts via `grep -o | wc -l`):
```bash
cd steel-etl
f="input/heroes/Draw Steel Heroes.md"
echo "self-named remaining (want 0):"
grep -oE 'skill\.(crafting|exploration|interpersonal|intrigue|lore)/(crafting|exploration|interpersonal|intrigue|lore)' "$f" | wc -l
echo "skill.group links (want 164):"
grep -oE 'skill\.group/(crafting|exploration|interpersonal|intrigue|lore)' "$f" | wc -l
echo "leaf links still matching old prefix (want 177):"
grep -oE 'skill\.(crafting|exploration|interpersonal|intrigue|lore)/[a-z-]+' "$f" | wc -l
```
Expected **after the repoint**: `0`, `164`, `177`.
- Line 1 = `0`: no self-named group link survived.
- Line 2 = `164`: every group link became `skill.group/<g>`.
- Line 3 = `177`: only the leaf links (`skill.<g>/<item>`) still carry the old `skill.<g>/` prefix; the 164 group links no longer match it because they are now `skill.group/`.
- Failure modes: line 3 still `341` → the perl replace did not run; line 3 below `177` → a leaf link was mangled — STOP and `git checkout` the file.

- [ ] **Step 4: Commit**

```bash
cd steel-etl
git add "input/heroes/Draw Steel Heroes.md"
git commit -m "docs(heroes): repoint 164 skill-group links to skill.group/<g>"
```

---

## Task 6: Regenerate registry + pipeline/site smoke check

**Files:** none edited; this validates the end-to-end result.

- [ ] **Step 1: Full unit test sweep**

Run: `devbox run -- bash -c 'cd steel-etl && go test ./...'`
Expected: PASS across all packages.

- [ ] **Step 2: Regenerate all books**

Run: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --all --config pipeline.yaml'`
Expected: success. (`gen` processes only the primary book without `--all`; the monster codes live in a `books:` entry, so `--all` is required — see `steel-etl/CLAUDE.md`.)

- [ ] **Step 3: Verify the regenerated registry holds exactly the intended codes**

`classification.json` is the generated SCC registry at the steel-etl root. Grep it for the code strings (robust to the exact JSON shape):

```bash
cd steel-etl
echo "skill.group landings (want 5):";   grep -oE 'skill\.group/[a-z-]+' classification.json | sort -u | wc -l
echo "monster.group landings (want 51):"; grep -oE 'monster\.group/[a-z0-9-]+' classification.json | sort -u | wc -l
echo "old self-named skill landings (want 0):";   grep -oE 'skill\.(crafting|exploration|interpersonal|intrigue|lore)/(crafting|exploration|interpersonal|intrigue|lore)"' classification.json | wc -l
echo "skill leaf intact, e.g. cooking (want >=1):"; grep -oE 'skill\.crafting/cooking' classification.json | wc -l
echo "statblock intact, e.g. goblin (want >=1):";   grep -oE 'monster\.goblins\.statblock/[a-z-]+' classification.json | wc -l
```
Expected: `5`, `51`, `0`, `≥1`, `≥1`. If old self-named codes survive or a leaf/statblock code is missing, STOP and investigate before building the site.

- [ ] **Step 4: Build the site and verify the Browse result**

Run:
```bash
cd /home/scott/code/steelCompendium/workspace
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl site --config ../v2/site.yaml'
echo "--- group index has lore + cards + scc, no doubled page ---"
test -f v2/docs/Browse/skill/crafting/index.md && echo "index OK"
test ! -e v2/docs/Browse/skill/crafting/crafting/ && echo "no doubled crafting/crafting OK"
test ! -e v2/docs/Browse/skill/group && echo "no phantom skill/group OK"
grep -q 'scc: mcdm.heroes.v1/skill.group/crafting' v2/docs/Browse/skill/crafting/index.md && echo "index scc OK"
grep -q 'sc-cards' v2/docs/Browse/skill/crafting/index.md && echo "card grid OK"
echo "--- permalink stub targets the group index ---"
test -f v2/docs/scc/mcdm.heroes.v1/skill.group/crafting/index.html && echo "skill stub OK"
test -f v2/docs/scc/mcdm.heroes.v1/monster.group/goblins/index.html && echo "monster stub OK"
echo "--- monster group lore folded into its Bestiary index ---"
test -f v2/docs/Bestiary/monster/goblins/index.md && echo "monster index OK"
```
Expected: every line prints its `OK`. (If `v2/docs/Browse` etc. are gitignored generated dirs, that's fine — this is a smoke check, not a commit.)

- [ ] **Step 5: Commit the regenerated registry if tracked**

Run: `cd steel-etl && git status --short classification.json`
- If `classification.json` is gitignored (per `steel-etl/CLAUDE.md` it is) → nothing to commit; skip.
- Otherwise: `git add classification.json && git commit -m "chore(scc): regenerate registry for group-landing migration"`.

---

## Task 7: Update the docs

**Files:**
- Modify: `steel-etl/CLAUDE.md` ("Grouped types (rule / skill)" + "Monsters book" sections)
- Modify: `/home/scott/code/steelCompendium/workspace/CLAUDE.md` (SCC registry paragraph)

- [ ] **Step 1: Update `steel-etl/CLAUDE.md` — Grouped types**

In the "### Grouped types (rule / skill)" section, replace the sentence describing the skill-group landing. Change:

> Each skill group also has a self-named-leaf landing page `skill.<group>/<group>` emitted by the **`skill-group`** container parser (`internal/content/skill_group.go`) — the same self-named pattern as the `monster-group` container (`monster.<category>/<category>`) …

to:

> Each skill group also has a **group-landing page `skill.group/<group>`** (e.g. `skill.group/crafting`) emitted by the **`skill-group`** parser (`internal/content/skill_group.go`). The unified `<type>.group/<member>` landing shape (also used by monster groups, `monster.group/<category>`) replaced the old self-named-leaf form (`skill.<g>/<g>`, `monster.<cat>/<cat>`) on 2026-06-09 — see `docs/superpowers/specs/2026-06-09-group-landing-scc-migration-design.md`. Site-side, the landing is **relocated to the group index** (`<root>/group/<member>.md` → `<root>/<member>/index.md` in `buildSection`) and its lore is folded above the card grid by `mergeGroupLanding`; there is no phantom `group/` folder card.

- [ ] **Step 2: Update `steel-etl/CLAUDE.md` — Monsters book**

In the "## Monsters book (statblocks)" section, change the SCC-hierarchy sentence:

> a group is `monster.<category>/<category>` (lore page, `monster/<category>/<category>.md`)

to:

> a group is `monster.group/<category>` (lore page; relocated to the Bestiary group index `monster/<category>/index.md` by the site builder)

- [ ] **Step 3: Update workspace `CLAUDE.md` — SCC registry paragraph**

In `/home/scott/code/steelCompendium/workspace/CLAUDE.md`, in the `## SCC (Steel Compendium Classification)` registry paragraph, append after the skills-nesting sentence:

> On **2026-06-09** the two group landings were unified to the `<type>.group/<member>` shape — `skill.group/<group>` and `monster.group/<category>` — retiring the self-named-leaf form (`skill.<g>/<g>`, `monster.<cat>/<cat>`); the site builder relocates each landing to its group index. 56 codes re-minted (5 skill + 51 monster); skill-leaf and statblock codes unchanged. See `steel-etl/docs/superpowers/plans/2026-06-09-group-landing-scc-migration.md`.

- [ ] **Step 4: Commit**

```bash
cd /home/scott/code/steelCompendium/workspace
git add steel-etl/CLAUDE.md CLAUDE.md
git commit -m "docs: record group-landing SCC unification (<type>.group/<member>)"
```

> Note: `steel-etl` is a sub-repo. Commit its `CLAUDE.md` change from inside `steel-etl/`, and the workspace `CLAUDE.md` (+ the steel-etl submodule pointer bump) from the workspace root, per the repo's usual split-commit flow.

---

## Done criteria

- `go test ./...` green.
- `classify --diff --all` shows exactly the 56 intended group-code changes, nothing else.
- Built site: `/Browse/skill/crafting/` is the lore+cards landing (scc `skill.group/crafting`), the `/crafting/crafting/` page is gone, no `skill/group/` or `monster/group/` phantom dirs, and `skill.group/*` + `monster.group/*` permalink stubs resolve to the group indexes.
- Heroes doc: 0 `skill.<g>/<g>` links, 164 `skill.group/<g>`, 177 leaf links intact.
- Docs updated in both CLAUDE.md files.
