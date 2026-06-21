# Summoner Retainer modeled like the Rival Summoner — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Show only the Devil Detective statblock + a shared Advancement Features featureblock on the Browse → Monsters → Retainer index, with the detective's three summoned minions and advancement card moved onto its own page — mirroring the Rival Summoner pattern.

**Architecture:** Three coordinated changes in steel-etl: (1) parser classification so the detective's minions nest as summons and the rival summons gain a parity `.statblock` segment; (2) source annotation that turns the detective's H8 advancement abilities into one `@type: featureblock` entity; (3) a site post-pass (`augmentSummonerRetainerPages`) that embeds the advancement card + a `## Summons` minion grid on the detective's page. The index pairing and URL hoisting are already handled by existing machinery.

**Tech Stack:** Go 1.26 (via devbox), steel-etl ETL + site builder, MkDocs Material output. Tests are Go `testing`.

## Global Constraints

- **Run all Go commands through devbox** from the `steel-etl/` directory: e.g. `devbox run -- go test ./internal/content/`. Go is not on the system PATH.
- **Minion SCC segment:** `monster.retainer.summoner.minion.statblock/<id>` (with `.statblock`). Rival summons: `monster.rival.<echelon>.summoner.minion.statblock/<id>` (add `.statblock`).
- **Browse URLs must not change** — the new `.statblock` segments are non-leaf and dropped by `hoistStatblockPath`. Verify, don't assume.
- **No schema/JSON data-contract changes.** Cards read existing frontmatter only.
- **Never hand-edit generated output** (`data/`, `v2/docs/Browse/…`). Regenerate via `gen`/`site`.
- **`advFeatSuffix`** constant (= `-advancement-features`) lives in `internal/site/advancement_pairs.go`; reuse it, don't re-spell the literal.
- Spec: `docs/superpowers/specs/2026-06-21-summoner-retainer-rival-pattern-design.md`.

---

## File Structure

- `internal/content/monster.go` — `StatblockParser.Classify` retainer + rival branches; `FeatureblockParser.Parse` retainer branch. (Task 1)
- `internal/content/monster_test.go` — parser classification tests. (Task 1)
- `input/summoner/Draw Steel Summoner.md` — source annotation for the advancement featureblock + section reorder. (Task 2)
- `internal/site/summoner_retainer.go` — **new** `augmentSummonerRetainerPages`. (Task 3)
- `internal/site/summoner_retainer_test.go` — **new** augment test. (Task 3)
- `internal/site/build.go` — wire the new augment beside `augmentRivalSummonerPages`. (Task 3)
- Docs: `CLAUDE.md`, `docs/statblocks.md`, `docs/site-builder.md`, `docs/summoner-linking-reference.md`, workspace `docs/scc-log.md`. (Task 4)

---

## Task 1: Parser classification — nest summoner minions, rival `.statblock` parity, allow summoner retainer advancement featureblock

**Files:**
- Modify: `internal/content/monster.go` (`StatblockParser.Classify` switch; `FeatureblockParser.Parse` retainer branch)
- Test: `internal/content/monster_test.go`

**Interfaces:**
- Consumes: `compactPath(...)`, `statblockDomain(ctx, level)`, `fm["organization"]`, existing `ParseRichFeatures`/`RichFeatureMaps`.
- Produces: TypePaths `monster/retainer/summoner/minion/statblock` (summoner minion), `monster/rival/<ech>/summoner/minion/statblock` (rival summon), `monster/retainer/advancement-features` (summoner retainer advancement featureblock). Detective stays `monster/retainer/statblock`.

- [ ] **Step 1: Write the failing tests**

Add to `internal/content/monster_test.go`:

```go
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
```

Then update the existing rival test `TestStatblockParser_SummonerRival` (≈line 351): change the summon `want` from `monster/rival/2nd-echelon/summoner/minion` to `monster/rival/2nd-echelon/summoner/minion/statblock`:

```go
		{"summon", "Skeleton", summonBody, "monster/rival/2nd-echelon/summoner/minion/statblock"},
```

- [ ] **Step 2: Run the tests to verify they fail**

Run: `devbox run -- go test ./internal/content/ -run 'SummonerRetainerMinionNests|SummonerRetainerAdvancement|SummonerRival' -v`
Expected: FAIL — minion gets `monster/retainer/statblock`, advancement falls through to `retainer/summoner`, rival summon lacks `/statblock`.

- [ ] **Step 3: Update `StatblockParser.Classify` retainer + rival branches**

In `internal/content/monster.go`, replace the `retainer` case:

```go
	case "retainer":
		// Both books' retainers live in the monster.* family. Monsters-book
		// retainers joined in Plan 6. Summoner-book retainers (@category: summoner):
		// the detective (organization Retainer) merges flat as monster.retainer.statblock,
		// but its summons (organization Minion) nest as
		// monster.retainer.summoner.minion.statblock — parallel to rival summons —
		// so they stay off the retainer index. The mcdm.summoner.v1 source segment +
		// the "Summoner ·" card eyebrow preserve provenance.
		if category == "summoner" {
			if org, _ := fm["organization"].(string); org == "Minion" {
				typePath = compactPath("monster", "retainer", "summoner", "minion", "statblock")
			} else {
				typePath = compactPath("monster", "retainer", "statblock")
			}
		} else {
			typePath = compactPath("monster", "retainer", category, subcategory, "statblock")
		}
```

And in the `rival` case, add the `.statblock` segment to the summon path:

```go
	case "rival":
		// The Rival Summoner NPC sits beside the Monsters-book rivals
		// (monster.rival.<echelon>.statblock); its minion summons nest under
		// monster.rival.<echelon>.summoner.minion.statblock. The source @category
		// ("summoner") is dropped; @subcategory is the echelon.
		if org, _ := fm["organization"].(string); org == "Minion" {
			typePath = compactPath("monster", "rival", subcategory, "summoner", "minion", "statblock")
		} else {
			typePath = compactPath("monster", "rival", subcategory, "statblock")
		}
```

- [ ] **Step 4: Lift the summoner guard in `FeatureblockParser.Parse` retainer branch**

In the same file, change the retainer-featureblock condition from:

```go
	if domain, category, _ := statblockDomain(ctx, section.HeadingLevel); domain == "retainer" && category != "summoner" {
```

to:

```go
	if domain, category, _ := statblockDomain(ctx, section.HeadingLevel); domain == "retainer" {
```

(The existing body already maps `category == "role-advancement"` → `kind = "role-advancement"` and everything else, including `"summoner"`, → `advancement-features`, and the typePath hardcodes `monster/retainer/<kind>`, so summoner mints `monster.retainer.advancement-features/<id>`. Update the branch comment to note it now also covers summoner-book retainers.)

- [ ] **Step 5: Run the tests to verify they pass**

Run: `devbox run -- go test ./internal/content/ -run 'SummonerRetainerMinionNests|SummonerRetainerAdvancement|SummonerRival|Retainer' -v`
Expected: PASS (incl. the unchanged `TestStatblockParser_Retainer`, `TestFeatureblockParser_RetainerAdvancement`, `TestFeatureblockParser_RoleAdvancement`).

- [ ] **Step 6: Run the full content package to catch regressions**

Run: `devbox run -- go test ./internal/content/`
Expected: PASS. (If `TestStatblockParser_SummonerRetainerMonsterFamily` still asserts the detective path, it stays valid — detective is unchanged.)

- [ ] **Step 7: Commit**

```bash
git add internal/content/monster.go internal/content/monster_test.go
git commit -m "fix(scc): nest summoner retainer minions + rival .statblock parity + summoner retainer advancement featureblock"
```

---

## Task 2: Source annotation — Devil Detective Advancement Features featureblock

**Files:**
- Modify: `input/summoner/Draw Steel Summoner.md` ("Retainer Summoner" section, ≈lines 3568–3733)

**Interfaces:**
- Consumes: the Task 1 `FeatureblockParser` retainer branch (mints `monster.retainer.advancement-features/devil-detective` from a `@type: featureblock | @id: devil-detective` section whose abilities sit under `> **Level N …**` blockquote labels).
- Produces: the generated `monster/retainer/devil-detective-advancement-features.md` page + the index pairing, and clean minion statblock pages (no leaked advancement abilities).

- [ ] **Step 1: Restructure the source section**

Edit the "Retainer Summoner" section so the order and annotation become:

1. `## Retainer Summoner` + the existing intro paragraph (unchanged).
2. `<!-- @type: monster-group | @domain: retainer | @category: summoner -->` / `##### —` (unchanged).
3. `<!-- @type: statblock -->` / `####### Devil Detective` — its stat grid + its three own blocks (`⭐️ Demon Summoner`, `🏹 Diabolic Probe`, `⭐️ True Name`). **Unchanged.**
4. **NEW featureblock section**, inserted immediately after Devil Detective's `⭐️ True Name` block:

   ```markdown
   <!-- @type: featureblock | @id: devil-detective -->
   ####### Devil Detective Advancement Features

   > **Level 4 Retainer Advancement Ability**

   <MOVE HERE: the existing `🏹 Soul Sleuth …` blockquote verbatim>

   <MOVE HERE: the existing `🏹 Summon Violents …` blockquote verbatim>

   > **Level 7 Retainer Advancement Ability**

   <MOVE HERE: the existing `🌀 Cleansing Flense …` blockquote verbatim>

   > **Level 10 Retainer Advancement Ability**

   <MOVE HERE: the existing `🏹 Blightwash …` blockquote verbatim>

   <MOVE HERE: the existing `🏹 Summon Gorrres …` blockquote verbatim>
   ```

   - **Consolidate** the two original `######## Level 4 …` H8 headings into one `> **Level 4 Retainer Advancement Ability**` blockquote label with both abilities beneath it; likewise the two `######## Level 10 …` into one label. (The detective picks one option per band.)
   - **Move the ability blockquotes verbatim** — do not retype the power-roll tables/effects; cut and paste so links and tier rows are byte-identical.
   - **Delete all five `######## Level N Retainer Advancement Ability` H8 headings** from their old inter-statblock positions.
5. `####### Razor` (stat grid + `⭐️ Teeth!`, `⭐️ Soulsight` only), then `####### Violent`, then `####### Gorrre` — now three consecutive `@type: statblock` minions with no advancement abilities between them.

- [ ] **Step 2: Sanity-check the edit textually**

Run: `devbox run -- bash -c "grep -n '########\\|@type: featureblock\\|Devil Detective Advancement\\|Level [0-9]* Retainer Advancement' 'input/summoner/Draw Steel Summoner.md' | sed -n '1,40p'"`
Expected: zero `########` lines remain in the retainer section; exactly one `@type: featureblock | @id: devil-detective`; three `> **Level N …**` labels (4, 7, 10).

- [ ] **Step 3: Regenerate the summoner book and verify classification**

Run: `devbox run -- go run ./cmd/steel-etl gen --config pipeline.yaml --book summoner`
Then:
`devbox run -- bash -c "find ../data/data-summoner -path '*retainer*' -name '*.md' | grep md-linked | sort"`
Expected paths (note hoisted URLs — no `statblock/` folder):
- `…/monster/retainer/statblock/devil-detective.md` *(raw data keeps statblock/ folder; the site hoist happens in Task 3/4)*
- `…/monster/retainer/advancement-features/devil-detective.md`
- `…/monster/retainer/summoner/minion/statblock/{razor,violent,gorrre}.md`

- [ ] **Step 4: Verify the advancement abilities left the minion pages**

Run: `devbox run -- bash -c "grep -l 'Soul Sleuth\\|Summon Violents\\|Cleansing Flense\\|Blightwash\\|Summon Gorrres' ../data/data-summoner/en/md-linked/monster/retainer/summoner/minion/*.md ../data/data-summoner/en/md-linked/monster/retainer/advancement-features/*.md 2>/dev/null"`
Expected: only `advancement-features/devil-detective.md` matches; no minion page does.

- [ ] **Step 5: Verify the featureblock captured leveled members**

Run: `devbox run -- bash -c "grep -n 'level:\\|features:\\|Soul Sleuth\\|Cleansing Flense\\|Blightwash' ../data/data-summoner/en/md-linked/monster/retainer/advancement-features/devil-detective.md | head -20"`
Expected: a `features:` list with `level: 4`, `level: 7`, `level: 10` members.

- [ ] **Step 6: Commit**

```bash
git add "input/summoner/Draw Steel Summoner.md"
git commit -m "content(summoner): make Devil Detective advancement abilities a featureblock"
```

---

## Task 3: Site augment — advancement card + summons on the detective's page

**Files:**
- Create: `internal/site/summoner_retainer.go`
- Create: `internal/site/summoner_retainer_test.go`
- Modify: `internal/site/build.go` (wire beside `augmentRivalSummonerPages`, ≈line 112)

**Interfaces:**
- Consumes: `splitFrontmatter`, `parseFrontmatterField`, `readFile`, `fileToTitle`, `card`, `advancementCardInner`, `rivalSummonsCards`, `listSummonFiles`, `advFeatSuffix`, `html.EscapeString`.
- Produces: `func augmentSummonerRetainerPages(sectionDir string) (int, []string)` — idempotent post-pass returning (pages modified, errors).

- [ ] **Step 1: Write the failing test**

Create `internal/site/summoner_retainer_test.go`:

```go
package site

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const detectiveFM = `name: Devil Detective
organization: Retainer
role: Controller
type: statblock
scc: mcdm.summoner.v1/monster.retainer.statblock/devil-detective
`

const retainerRazorFM = `name: Razor
organization: Minion
role: Harrier
type: statblock
scc: mcdm.summoner.v1/monster.retainer.summoner.minion.statblock/razor
`

// A Monsters-book retainer (not summoner) must NOT get a summons/advancement augment.
const monsterRetainerFM = `name: Angulotl Hopper
organization: Retainer
role: Harrier
type: statblock
scc: mcdm.monsters.v1/monster.retainer.statblock/angulotl-hopper
`

func TestAugmentSummonerRetainerPages(t *testing.T) {
	sec := t.TempDir()
	ret := filepath.Join(sec, "monster", "retainer")
	writeStatblockPage(t, filepath.Join(ret, "devil-detective.md"), detectiveFM, "Devil Detective")
	writeStatblockPage(t, filepath.Join(ret, "angulotl-hopper.md"), monsterRetainerFM, "Angulotl Hopper")
	// Advancement featureblock sibling (flattened name) with one leveled member.
	if err := os.WriteFile(filepath.Join(ret, "devil-detective-advancement-features.md"),
		[]byte("---\nname: Devil Detective\ntype: featureblock\nfeatures:\n  - name: Soul Sleuth\n    level: 4\n---\n\n# Devil Detective\n"), 0644); err != nil {
		t.Fatal(err)
	}
	minion := filepath.Join(ret, "summoner", "minion")
	writeStatblockPage(t, filepath.Join(minion, "razor.md"), retainerRazorFM, "Razor")

	n, errs := augmentSummonerRetainerPages(sec)
	if len(errs) != 0 {
		t.Fatalf("unexpected errors: %v", errs)
	}
	if n != 2 { // detective page (advancement + summons) + 1 minion back-link
		t.Errorf("augment count = %d, want 2", n)
	}

	dd := readFile(filepath.Join(ret, "devil-detective.md"))
	for _, want := range []string{
		"## Advancement Features",
		"Soul Sleuth", // advancementCardInner lists the member
		`href="../devil-detective-advancement-features/"`,
		"## Summons",
		`href="../summoner/minion/razor/"`,
	} {
		if !strings.Contains(dd, want) {
			t.Errorf("detective page missing %q:\n%s", want, dd)
		}
	}

	// Monsters-book retainer must be untouched.
	if ah := readFile(filepath.Join(ret, "angulotl-hopper.md")); strings.Contains(ah, "## Summons") {
		t.Errorf("angulotl-hopper must not get a Summons block")
	}

	// Minion back-link to the detective.
	rz := readFile(filepath.Join(minion, "razor.md"))
	if c := strings.Count(rz, "sb-backlink"); c != 1 {
		t.Errorf("razor sb-backlink count = %d, want 1", c)
	}
	if !strings.Contains(rz, `href="../../../devil-detective/"`) {
		t.Errorf("razor missing back-link href:\n%s", rz)
	}
	if strings.Index(rz, "sb-backlink") > strings.Index(rz, `<div class="sb-wrap"`) {
		t.Errorf("back-link should precede the sb-wrap card:\n%s", rz)
	}

	// Idempotent.
	n2, _ := augmentSummonerRetainerPages(sec)
	if n2 != 0 {
		t.Errorf("second run count = %d, want 0 (idempotent)", n2)
	}
}

func TestAugmentSummonerRetainerPages_NoTree(t *testing.T) {
	n, errs := augmentSummonerRetainerPages(t.TempDir())
	if n != 0 || len(errs) != 0 {
		t.Errorf("expected no-op, got n=%d errs=%v", n, errs)
	}
}
```

(`writeStatblockPage` already exists in `rival_summons_test.go`, same package.)

- [ ] **Step 2: Run the test to verify it fails**

Run: `devbox run -- go test ./internal/site/ -run TestAugmentSummonerRetainerPages -v`
Expected: FAIL — `augmentSummonerRetainerPages` undefined.

- [ ] **Step 3: Implement `augmentSummonerRetainerPages`**

Create `internal/site/summoner_retainer.go`:

```go
package site

import (
	"fmt"
	"html"
	"os"
	"path/filepath"
	"strings"
)

// augmentSummonerRetainerPages adds, to each summoner-book retainer page under
// sectionDir/monster/retainer, an "## Advancement Features" preview card (from the
// flattened <id>-advancement-features.md sibling) and a "## Summons" grid of the
// retainer's summoned minions (sectionDir/monster/retainer/summoner/minion), plus a
// "Summoned by" back-link on each minion page. Mirrors augmentRivalSummonerPages;
// runs after pages are written and is idempotent. Scoped to the summoner book
// (scc prefix mcdm.summoner.) and the conjurer (organization != "Minion"), so the
// Monsters-book retainers are untouched. There is exactly one summoner retainer
// today, so every minion under summoner/minion belongs to it. Returns the number of
// pages modified.
func augmentSummonerRetainerPages(sectionDir string) (int, []string) {
	retainerDir := filepath.Join(sectionDir, "monster", "retainer")
	if _, err := os.Stat(retainerDir); err != nil {
		return 0, nil
	}
	minionDir := filepath.Join(retainerDir, "summoner", "minion")
	summonFiles := listSummonFiles(minionDir) // nil when no summons subtree

	ents, err := os.ReadDir(retainerDir)
	if err != nil {
		return 0, []string{fmt.Sprintf("read %s: %v", retainerDir, err)}
	}

	count := 0
	var errs []string
	for _, e := range ents {
		if e.IsDir() || e.Name() == "index.md" || !strings.HasSuffix(e.Name(), ".md") {
			continue
		}
		if strings.HasSuffix(e.Name(), advFeatSuffix+".md") {
			continue // skip advancement-features pages themselves
		}
		path := filepath.Join(retainerDir, e.Name())
		fm, _ := splitFrontmatter(readFile(path))
		scc := strings.TrimSpace(parseFrontmatterField(fm, "scc"))
		org := strings.TrimSpace(parseFrontmatterField(fm, "organization"))
		if !strings.HasPrefix(scc, "mcdm.summoner.") || org == "Minion" {
			continue // only the summoner-book conjurer
		}
		base := strings.TrimSuffix(e.Name(), ".md")
		name := strings.TrimSpace(parseFrontmatterField(fm, "name"))
		if name == "" {
			name = fileToTitle(e.Name())
		}

		page := readFile(path)
		modified := false

		// Advancement Features preview card (if the sibling exists).
		advFile := base + advFeatSuffix + ".md"
		if _, err := os.Stat(filepath.Join(retainerDir, advFile)); err == nil &&
			!strings.Contains(page, "## Advancement Features") {
			inner := advancementCardInner(retainerDir, advFile)
			// href needs a "../" hop: the detective page is served as a dir.
			advCard := card("../"+advFile, "sword-cross", "Advancement Features", name, inner)
			page = strings.TrimRight(page, "\n") +
				"\n\n## Advancement Features\n\n<div class=\"sc-cards\">\n" + advCard + "</div>\n"
			modified = true
		}

		// Summons grid (hrefBase mirrors the rival case: one "../" hop up).
		if len(summonFiles) > 0 && !strings.Contains(page, "## Summons") {
			cards := rivalSummonsCards(minionDir, "../summoner/minion", summonFiles)
			page = strings.TrimRight(page, "\n") + "\n\n## Summons\n\n" + cards + "\n"
			modified = true
		}

		if modified {
			if err := os.WriteFile(path, []byte(page), 0644); err != nil {
				errs = append(errs, fmt.Sprintf("write %s: %v", path, err))
			} else {
				count++
			}
		}

		// Back-link on each minion page (3 hops: summoner/minion/<id>/ → retainer/).
		if len(summonFiles) > 0 {
			backlink := fmt.Sprintf(`<p class="sb-backlink">Summoned by <a href="../../../%s/">%s</a></p>`,
				base, html.EscapeString(name))
			for _, sf := range summonFiles {
				sp := filepath.Join(minionDir, sf)
				spage := readFile(sp)
				if strings.Contains(spage, "sb-backlink") {
					continue
				}
				i := strings.Index(spage, `<div class="sb-wrap"`)
				if i < 0 {
					continue
				}
				spage = spage[:i] + backlink + "\n\n" + spage[i:]
				if err := os.WriteFile(sp, []byte(spage), 0644); err != nil {
					errs = append(errs, fmt.Sprintf("write %s: %v", sp, err))
				} else {
					count++
				}
			}
		}
	}
	return count, errs
}
```

- [ ] **Step 4: Run the test to verify it passes**

Run: `devbox run -- go test ./internal/site/ -run TestAugmentSummonerRetainerPages -v`
Expected: PASS (both cases).

- [ ] **Step 5: Wire the augment into the build**

In `internal/site/build.go`, immediately after the `augmentRivalSummonerPages` loop (≈line 116), add a parallel loop:

```go
	// Summoner Retainer ⇄ summons + advancement-features cross-references, mirroring
	// the Rival Summoner pass: a "## Advancement Features" card + "## Summons" grid on
	// the detective page, and a back-link on each summon page. No-op without a summoner
	// monster/retainer tree.
	for _, s := range genericSections {
		if _, rErrs := augmentSummonerRetainerPages(filepath.Join(cfg.DocsDir, s.Name)); len(rErrs) > 0 {
			result.Errors = append(result.Errors, rErrs...)
		}
	}
```

- [ ] **Step 6: Run the full site package**

Run: `devbox run -- go test ./internal/site/`
Expected: PASS (existing rival/advancement-pair tests unaffected).

- [ ] **Step 7: Commit**

```bash
git add internal/site/summoner_retainer.go internal/site/summoner_retainer_test.go internal/site/build.go
git commit -m "feat(site): augment summoner retainer page with advancement card + summons grid"
```

---

## Task 4: Full regen, end-to-end verification, and docs

**Files:**
- Modify: `CLAUDE.md`, `docs/statblocks.md`, `docs/site-builder.md`, `docs/summoner-linking-reference.md`
- Modify: workspace `../docs/scc-log.md`

**Interfaces:** consumes the Task 1–3 behavior; produces verified generated output + synced docs.

- [ ] **Step 1: Full multi-book regen + SCC stability check**

Run: `devbox run -- go run ./cmd/steel-etl gen --all --config pipeline.yaml`
Then: `devbox run -- go run ./cmd/steel-etl validate --config pipeline.yaml --scc-stable`
Expected: validate passes (no unexpected code churn beyond the intended renames). If freeze blocks the renames, that is expected for *changed* codes — confirm the diff is exactly the rival/retainer-minion renames + the one new advancement code (next step).

- [ ] **Step 2: Confirm the exact registry delta**

Run: `devbox run -- go run ./cmd/steel-etl classify --config pipeline.yaml --diff | grep -i 'retainer\|rival.*minion\|advancement' | head -40`
Expected:
- `monster.retainer.summoner.minion.statblock/{razor,violent,gorrre}` present; old `monster.retainer.statblock/{razor,violent,gorrre}` gone.
- `monster.retainer.advancement-features/devil-detective` present (new, +1).
- rival summons now `…summoner.minion.statblock/<id>`.
- Devil Detective unchanged.

- [ ] **Step 3: Build the v2 site and verify the index + detective page**

Run: `devbox run -- go run ./cmd/steel-etl site --config ../v2/site.yaml`
Then verify the **index** shows only Devil Detective + its advancement card (no minion cards):
`devbox run -- bash -c "grep -o 'sc-card__name\">[^<]*' ../v2/docs/Browse/monster/retainer/index.md | grep -i 'devil\|razor\|violent\|gorrre\|advancement\|detective'"`
Expected: Devil Detective (+ its advancement card), and **no** Razor/Violent/Gorrre cards.

Verify the **detective page** has both augments and the **URLs are unchanged** (no `statblock` segment leaked into hrefs):
`devbox run -- bash -c "grep -n '## Advancement Features\\|## Summons\\|summoner/minion/razor/\\|devil-detective-advancement-features/' ../v2/docs/Browse/monster/retainer/devil-detective.md"`
Expected: all four present; no `summoner/minion/statblock/` in any href.

Verify a **minion page** has the back-link:
`devbox run -- bash -c "grep -n 'Summoned by\\|sb-backlink' ../v2/docs/Browse/monster/retainer/summoner/minion/razor.md"`
Expected: one back-link to `../../../devil-detective/`.

- [ ] **Step 4: Confirm no dangling links + clean build**

Run: `devbox run -- go run ./cmd/steel-etl site --config ../v2/site.yaml 2>&1 | grep -i 'warn\|404\|unresolch\|error' | head`
Expected: no new warnings about the retainer/rival trees (link resolver reported 0 inbound links to the re-minted codes during planning).

- [ ] **Step 5: Update steel-etl docs**

Apply these doc edits (router stays summary-only; detail in the topic docs):
- `docs/statblocks.md` — the summoner retainer now: detective `monster.retainer.statblock`, summons `monster.retainer.summoner.minion.statblock`, advancement `monster.retainer.advancement-features/devil-detective`; rival summons gain `.statblock`; mention `augmentSummonerRetainerPages`.
- `docs/site-builder.md` — document the new `augmentSummonerRetainerPages` pass beside the rival one.
- `docs/summoner-linking-reference.md` — update the retainer code table (minions → `…summoner.minion.statblock/<id>`, add `monster.retainer.advancement-features/devil-detective`); update the rival summon rows to `.statblock`.
- `CLAUDE.md` — the Statblocks bullet: summoner retainer now has nested summons + a real advancement featureblock; rival summons carry `.statblock`.

- [ ] **Step 6: Append a dated entry to the workspace SCC log**

Add to `../docs/scc-log.md` a `2026-06-21` entry: summoner retainer minions re-minted to `monster.retainer.summoner.minion.statblock/<id>`; rival summons gain `.statblock`; new `monster.retainer.advancement-features/devil-detective` (registry +1); 0 inbound links dangled.

- [ ] **Step 7: Commit docs**

```bash
git add CLAUDE.md docs/statblocks.md docs/site-builder.md docs/summoner-linking-reference.md ../docs/scc-log.md
git commit -m "docs: summoner retainer rival-pattern + rival summon .statblock parity"
```

> **Note:** generated `data/` and `v2/docs/` are rebuilt by the `just deploy*` recipes, which commit them themselves — do **not** hand-commit generated output here. The workspace submodule-pointer bump + deploy is a separate step per `docs/git-workflow.md`.

---

## Self-Review (completed)

- **Spec coverage:** A → Task 1 (statblock retainer branch) + Task 3 (index falls out free) ✓; B (rival `.statblock`) → Task 1 ✓; C (advancement featureblock) → Task 1 (FeatureblockParser guard) + Task 2 (source) ✓; D (index pairing) → verified in Task 4 Step 3 ✓; E (page augment) → Task 3 ✓. SCC/registry table → Task 4 Steps 1–2 ✓. Docs → Task 4 Steps 5–6 ✓.
- **Placeholder scan:** the only intentional `<MOVE HERE: …>` markers are in Task 2 Step 1, where retyping the ability blockquotes verbatim would risk corrupting power-roll tables/links — the instruction is to cut-paste existing bytes, which is more correct than a transcription.
- **Type consistency:** `augmentSummonerRetainerPages(string) (int, []string)`, `card(file, icon, typeLabel, name, inner string)`, `advancementCardInner(dir, advFile string) string`, `rivalSummonsCards(readDir, hrefBase string, files []string)`, `listSummonFiles(dir string) []string`, `advFeatSuffix` — all match their definitions in the codebase.
