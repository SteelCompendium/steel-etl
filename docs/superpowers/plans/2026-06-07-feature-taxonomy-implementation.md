# Feature / Ability / Trait Taxonomy — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Narrow `trait` to its rulebook homes (ancestry + monster), make every other non-ability feature a plain `feature`, and reshape the SCC path to the hub-and-spoke form (`feature.<entity>` base; `feature.ability.*` / `feature.trait.*` marked) — a breaking SCC change.

**Architecture:** A single parser decision in `FeatureParser` (`internal/content/feature.go`) sets `fm["type"]` to `trait` (ancestry home) or `feature` (everything else) and inserts the `trait` path segment only for trait. Four downstream consumers that switch on `fm["type"]` learn the new `feature` value: `sdk_transform.go`, `dse.go`, `internal/site/ability_cards.go`, `internal/site/feature_index.go`. Then the registry is regenerated and source cross-reference links are rewritten from the authoritative old→new diff.

**Tech Stack:** Go 1.26 (steel-etl), goldmark parsing, `go test`; devbox toolchain; MkDocs Material (v2, downstream). Source spec: `steel-etl/docs/superpowers/specs/2026-06-07-feature-taxonomy-design.md`.

**Toolchain note:** Go is not on PATH. From the workspace root run Go via devbox:
`devbox run -- bash -c 'cd steel-etl && go test ./...'`. All `go`/`steel-etl` commands below assume this wrapper (shown explicitly in steps).

**Concurrency note:** Another agent is editing `steel-etl/input/heroes/Draw Steel Heroes.md` (rule-glossary linking). Phases 1–4 and 6–7 do **not** touch that file and are safe to run concurrently. **Phase 5 (regenerate + link rewrite) rewrites `Draw Steel Heroes.md` and `Draw Steel Beastheart.md` and MUST be the last code phase**, run after a fresh `git pull`, applying a scripted old→new mapping so it composes with the other agent's link additions rather than conflicting.

---

## File Structure

| File | Responsibility | Change |
|------|----------------|--------|
| `internal/content/feature.go` | `FeatureParser`: type + SCC path for non-ability features | **Core change** — trait vs feature decision; drop unconditional `trait` |
| `internal/content/ability.go` | `AbilityParser` | No change (abilities already `feature.ability.*`) |
| `internal/content/statblock_parse.go` | Monster statblock embedded features | No change (already emits `trait`/`ability`); add a guard test |
| `internal/output/sdk_transform.go` | ParsedContent → SDK feature.schema.json | Route `type:"feature"`; emit `feature_type` from `fm["type"]` |
| `internal/output/dse.go` | ParsedContent → DSE ds-feature codeblocks | Treat `type:"feature"` like trait |
| `internal/site/ability_cards.go` | Page-body card router | Route `type:"feature"` → `renderTraitCard` |
| `internal/site/feature_index.go` | Browse index preview cards | `featureKind`: non-ability feature dirs → recessed niche |
| `*_test.go` (content, output, site) | Assertions on the old shape | Flip class/college/domain/kit/companion expectations |
| `ANNOTATION-GUIDE.md`, `CLAUDE.md` (steel-etl + workspace), `docs/linking-*.md` | Docs | "meaning of trait narrowed" callout |
| `../data-sdk-npm/src/schema/feature.schema.json[.md]` | SDK schema docs | `feature_type` enum {ability, trait, feature} |

---

## Phase 1 — Core parser: trait vs feature decision

The whole semantic change lives here. `FeatureParser` is reached for non-ability features in class/ancestry/kit/companion/common context. The only **trait home** reachable here is an **ancestry** (monster traits are produced by `statblock_parse.go`; companions are explicitly *not* a trait home).

### Task 1.1: `FeatureParser` emits `feature` vs `trait` and the hub-and-spoke path

**Files:**
- Modify: `internal/content/feature.go:66-110` (the `fm` init and `typePath` construction)
- Test: `internal/content/content_test.go` (existing `feature.trait.*` expectations) + a new test

- [ ] **Step 1: Write failing tests for the new path shape**

Add to `internal/content/content_test.go` (new test function):

```go
func TestFeatureParser_TaxonomyPaths(t *testing.T) {
	cases := []struct {
		name      string
		homeType  string // "class" | "ancestry" | "companion"
		homeID    string
		wantType  string   // fm["type"]
		wantPath  []string // TypePath prefix (through entity)
	}{
		{"class feature is plain feature", "class", "shadow", "feature", []string{"feature", "shadow"}},
		{"ancestry feature is trait", "ancestry", "dwarf", "trait", []string{"feature", "trait", "dwarf"}},
		{"companion feature is plain feature", "companion", "wolf", "feature", []string{"feature", "companion", "wolf"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.NewContextStack(nil)
			if tc.homeType == "companion" {
				ctx.Push(2, context.Metadata{"type": "feature-group", "companion": tc.homeID})
			} else {
				ctx.Push(2, context.Metadata{"type": tc.homeType, "id": tc.homeID})
			}
			sec := &parser.Section{
				Heading:      "Sample Feature",
				HeadingLevel: 3,
				Annotation:   map[string]string{"type": "feature"},
			}
			p := &FeatureParser{}
			got, err := p.Parse(ctx, sec)
			if err != nil {
				t.Fatal(err)
			}
			if got.Frontmatter["type"] != tc.wantType {
				t.Errorf("type = %v, want %v", got.Frontmatter["type"], tc.wantType)
			}
			for i, seg := range tc.wantPath {
				if i >= len(got.TypePath) || got.TypePath[i] != seg {
					t.Errorf("TypePath = %v, want prefix %v", got.TypePath, tc.wantPath)
					break
				}
			}
		})
	}
}
```

(Check the exact `context.Metadata` constructor and `ctx.Push` signature against `internal/context/stack.go` and the existing `content_test.go` helpers before running; mirror whatever the existing tests use — e.g. `content_test.go` already builds context stacks for fury features.)

- [ ] **Step 2: Run the test to verify it fails**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestFeatureParser_TaxonomyPaths -v'
```
Expected: FAIL — class feature currently yields `type=trait` and `TypePath=[feature trait shadow]`.

- [ ] **Step 3: Implement the decision in `feature.go`**

Replace the `fm` initialization (currently `feature.go:66-69`):

```go
	// Trait is reserved for the rulebook's trait homes. The only trait home
	// reachable through FeatureParser is an ancestry (monster traits are emitted
	// by statblock_parse.go; companions are NOT a trait home — the Beastheart
	// book calls companion grants "features", never "traits"). See
	// docs/superpowers/specs/2026-06-07-feature-taxonomy-design.md.
	isTrait := ancestryID != ""
	featureKind := "feature"
	if isTrait {
		featureKind = "trait"
	}

	fm := map[string]any{
		"name": cleanName,
		"type": featureKind,
	}
```

Replace the `typePath` base (currently `feature.go:97`, `typePath := []string{"feature", "trait"}`) with:

```go
	// Hub-and-spoke: base case is unmarked; the `trait` marker is inserted only
	// for trait homes. Plain features take feature.<entity>...; abilities (in
	// ability.go) take feature.ability.<entity>...
	typePath := []string{"feature"}
	if isTrait {
		typePath = append(typePath, "trait")
	}
```

Leave the entity branches (`companion` / `classID` / `ancestryID` / `common`), the named-group block, the `level-N` append, and the `kit` append exactly as they are — they now run on top of the corrected base.

- [ ] **Step 4: Run the new test to verify it passes**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run TestFeatureParser_TaxonomyPaths -v'
```
Expected: PASS.

- [ ] **Step 5: Update the pre-existing content_test.go expectations**

Flip the class/companion/beastheart expectations to the new shape (ancestry expectations stay):
- `content_test.go:138` — `expected := []string{"feature", "fury", "level-1"}` (was `feature, trait, fury, level-1`).
- `content_test.go:485` — `want := "mcdm.beastheart.v1/feature.companion.wolf.level-3/my-what-big-teeth-you-have"` (drop `.trait`).
- `content_test.go:573` — `want := "mcdm.beastheart.v1/feature.beastheart.level-2/stormheart"` (drop `.trait`).

Run the whole content package:
```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/content/...'
```
Expected: PASS (fix any other `feature.trait.<class>` / `feature.trait.companion` literals the compiler/test surfaces; **keep** any `feature.trait.<ancestry>` literal).

- [ ] **Step 6: Commit**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git add internal/content/feature.go internal/content/content_test.go
git commit -m "feat: FeatureParser emits plain 'feature' except for ancestry trait homes"
```

### Task 1.2: Guard test — statblock passives stay `trait`, actions stay `ability`

**Files:**
- Test: `internal/content/statblock_parse_test.go` (already has `trait`/`ability` assertions at lines ~91, ~160-227)

- [ ] **Step 1: Confirm existing coverage, add an explicit guard if missing**

`statblock_parse.go` is unchanged by this refactor; this task only *locks in* that monster features keep the trait/ability split. Verify `statblock_parse_test.go` asserts both a passive (`feature_type: trait`) and an action (`feature_type: ability`). If only one is covered, add the missing assertion mirroring the existing test style.

- [ ] **Step 2: Run**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/content/ -run Statblock -v'
```
Expected: PASS.

- [ ] **Step 3: Commit (only if a test was added)**

```bash
git add internal/content/statblock_parse_test.go
git commit -m "test: lock statblock trait/ability split against taxonomy change"
```

---

## Phase 2 — Output transforms learn `type: "feature"`

### Task 2.1: SDK transform routes and labels `feature`

**Files:**
- Modify: `internal/output/sdk_transform.go:17-28` (switch) and `:72-79` (`transformTrait`)
- Test: `internal/output/sdk_transform_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/output/sdk_transform_test.go`:

```go
func TestTransformFeature_PlainFeature(t *testing.T) {
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{
			"name": "A Beyonding of Vision",
			"type": "feature",
		},
		Body:     "Your void sense reaches further.",
		TypePath: []string{"feature", "elementalist", "level-1"},
		ItemID:   "a-beyonding-of-vision",
	}
	out := TransformToSDKFormat("mcdm.heroes.v1/feature.elementalist.level-1/a-beyonding-of-vision", parsed)
	if out["type"] != "feature" {
		t.Errorf("type = %v, want feature", out["type"])
	}
	if out["feature_type"] != "feature" {
		t.Errorf("feature_type = %v, want feature", out["feature_type"])
	}
	if _, ok := out["effects"]; !ok {
		t.Error("expected effects[] on a plain feature")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/output/ -run TestTransformFeature_PlainFeature -v'
```
Expected: FAIL — `default` passthrough runs, no `feature_type`.

- [ ] **Step 3: Implement**

In `sdk_transform.go`, change the switch case (`:21`) from:
```go
	case "trait":
		return transformTrait(sccCode, parsed)
```
to:
```go
	case "trait", "feature":
		return transformTrait(sccCode, parsed)
```

In `transformTrait` (`:77-79`), replace the hardcoded `feature_type`:
```go
	// Required schema fields
	out["type"] = "feature"
	ftype, _ := fm["type"].(string) // "trait" or "feature"
	out["feature_type"] = ftype
```
Also update the function doc comment (`:72`) to say "traits and plain features". `buildTraitMetadata` already copies `feature_type` from `fm["type"]` and sets `action_type: "feature"`, so it is correct for both.

- [ ] **Step 4: Run to verify it passes**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/output/ -run TestTransformFeature_PlainFeature -v'
```
Expected: PASS.

- [ ] **Step 5: Fix pre-existing output tests, then run the package**

Flip class-feature expectations in `internal/output/`:
- `conformance_test.go:45-46` — if the asserted feature is a **class** feature, expect `feature` (and its SCC without `.trait`); if it's an **ancestry** feature, leave as `trait`. Inspect the fixture to decide.
- `coverage_test.go:215-216` — same rule by home.
- `sdk_transform_test.go:203-207` (`TestTransformTrait_Basic`) — keep as a `trait` case but ensure the fixture's `type` is `trait` (ancestry) so it still exercises the trait path.

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/output/...'
```
Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/output/sdk_transform.go internal/output/*_test.go
git commit -m "feat: SDK transform handles feature_type=feature (plain features)"
```

### Task 2.2: DSE generator treats `feature` like trait

**Files:**
- Modify: `internal/output/dse.go:60` and `:105`
- Test: `internal/output/dse_test.go`

- [ ] **Step 1: Write the failing test**

Add to `dse_test.go` (mirror `TestDSEGenerator_Trait` at `:122`, but with `type: feature`):

```go
func TestDSEGenerator_PlainFeature(t *testing.T) {
	dir := t.TempDir()
	gen := &DSEGenerator{BaseDir: dir}
	parsed := &content.ParsedContent{
		Frontmatter: map[string]any{"name": "Growing Ferocity", "type": "feature"},
		Body:        "You grow more ferocious.",
		TypePath:    []string{"feature", "fury", "level-1"},
		ItemID:      "growing-ferocity",
	}
	if err := gen.WriteSection("mcdm.heroes.v1/feature.fury.level-1/growing-ferocity", parsed); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(filepath.Join(dir, "feature", "fury", "level-1", "growing-ferocity.md"))
	if err != nil {
		t.Fatal(err)
	}
	out := string(data)
	if !strings.Contains(out, "```ds-feature") {
		t.Error("plain features should still get a ds-feature codeblock")
	}
	if !strings.Contains(out, "feature_type: feature") {
		t.Error("expected feature_type: feature")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/output/ -run TestDSEGenerator_PlainFeature -v'
```
Expected: FAIL — `type:"feature"` falls into the `default`/plain-markdown branch, no codeblock.

- [ ] **Step 3: Implement**

`dse.go:60`:
```go
	if featureType == "ability" || featureType == "trait" || featureType == "feature" {
```
`dse.go:105`:
```go
	if featureType == "ability" || featureType == "trait" || featureType == "feature" {
```
(The existing `else` branch already sets `action_type: "feature"`, correct for `feature`; `buildDSFeatureBlock`'s `getStringOr(..., "type", "ability")` already yields `feature`.)

- [ ] **Step 4: Run to verify it passes**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/output/ -run TestDSEGenerator_PlainFeature -v'
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/output/dse.go internal/output/dse_test.go
git commit -m "feat: DSE generator renders feature_type=feature as ds-feature block"
```

---

## Phase 3 — Site builder: render and index plain features

Plain features now have `type: feature` and live at `feature/<entity>/…` (no `trait`/`ability` segment). Two site code paths must learn this or plain features render as raw markdown and lose their browse preview cards.

### Task 3.1: Card router renders `feature` as the recessed niche

**Files:**
- Modify: `internal/site/ability_cards.go:40-47`
- Test: `internal/site/ability_cards_test.go` or `trait_cards_test.go`

- [ ] **Step 1: Write the failing test**

Add to `internal/site/trait_cards_test.go` (mirror the existing trait render test at `:15`, but feed `type: feature`):

```go
func TestBuildAbilityCardPage_PlainFeature(t *testing.T) {
	page := "---\ntype: feature\nname: A Beyonding of Vision\n---\n\nYour void sense reaches further.\n"
	out, ok := buildAbilityCardPage([]byte(page))
	if !ok {
		t.Fatal("expected plain feature to be rendered as a card")
	}
	if !strings.Contains(string(out), "sc-trait") {
		t.Error("plain feature should render the recessed .sc-trait niche")
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildAbilityCardPage_PlainFeature -v'
```
Expected: FAIL — `default:` returns `(data, false)`.

- [ ] **Step 3: Implement**

`ability_cards.go:43-44`, change:
```go
	case "trait":
		card = renderTraitCard(fm, body)
```
to:
```go
	case "trait", "feature":
		card = renderTraitCard(fm, body)
```
Update the function doc comment (`:30-33`) to mention plain features render as the niche too.

- [ ] **Step 4: Run to verify it passes**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestBuildAbilityCardPage_PlainFeature -v'
```
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/site/ability_cards.go internal/site/trait_cards_test.go
git commit -m "feat: site renders type=feature pages as the recessed trait niche"
```

### Task 3.2: Browse index gives plain-feature dirs preview cards

**Files:**
- Modify: `internal/site/feature_index.go:73-88` (`featureKind`)
- Test: `internal/site/feature_index_test.go`

- [ ] **Step 1: Write the failing test**

Add to `feature_index_test.go`:

```go
func TestFeatureKind_PlainFeatureDir(t *testing.T) {
	// Plain features now live at feature/<class>/... with no kind segment;
	// they should be treated as the recessed niche ("trait") for previews.
	if got := featureKind("Browse/feature/elementalist/level-1"); got != "trait" {
		t.Errorf("featureKind(plain feature dir) = %q, want trait", got)
	}
	if got := featureKind("Browse/feature/ability/Kits"); got != "ability" {
		t.Errorf("featureKind(ability dir) = %q, want ability", got)
	}
	if got := featureKind("Browse/feature/trait/dwarf"); got != "trait" {
		t.Errorf("featureKind(ancestry trait dir) = %q, want trait", got)
	}
	if got := featureKind("Browse/treasure/artifact"); got != "" {
		t.Errorf("featureKind(non-feature dir) = %q, want empty", got)
	}
}
```

- [ ] **Step 2: Run to verify it fails**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestFeatureKind_PlainFeatureDir -v'
```
Expected: FAIL — `featureKind("…/feature/elementalist/…")` returns `""`.

- [ ] **Step 3: Implement**

Replace `featureKind` (`feature_index.go:75-88`):

```go
// featureKind returns "ability" for feature/ability/** dirs and "trait" for any
// other dir under feature/** (ancestry/monster traits AND plain class/domain/
// college/kit/companion features — both render as the recessed niche). Returns
// "" for dirs outside the feature/ subtree. The first path segment after
// `feature` is either the reserved `ability` kind or an entity id; everything
// that is not `ability` is niche-styled.
func featureKind(dir string) string {
	parts := strings.Split(filepath.ToSlash(dir), "/")
	for i, p := range parts {
		if p == "feature" && i+1 < len(parts) {
			if parts[i+1] == "ability" {
				return "ability"
			}
			return "trait"
		}
	}
	return ""
}
```

- [ ] **Step 4: Run to verify it passes**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/site/ -run TestFeatureKind_PlainFeatureDir -v'
```
Expected: PASS.

- [ ] **Step 5: Run the whole site package**

```bash
devbox run -- bash -c 'cd steel-etl && go test ./internal/site/...'
```
Expected: PASS. Fix any `build_test.go` path literals that assumed `feature/trait/<class>/…` for a class feature (now `feature/<class>/…`); keep ancestry literals.

- [ ] **Step 6: Commit**

```bash
git add internal/site/feature_index.go internal/site/feature_index_test.go internal/site/build_test.go
git commit -m "feat: browse index previews plain-feature dirs as recessed niches"
```

### Task 3.3: Full suite green

- [ ] **Step 1: Run everything with the race detector**

```bash
devbox run -- bash -c 'cd steel-etl && go test -race ./...'
```
Expected: PASS. Resolve any remaining `feature.trait.<class|kit|companion>` literals (flip) vs `feature.trait.<ancestry|monster>` literals (keep) the suite surfaces.

- [ ] **Step 2: Commit any residual test fixes**

```bash
git add -A
git commit -m "test: align remaining fixtures with hub-and-spoke feature paths"
```

---

## Phase 4 — feature-group audit (book-named grants)

The spec keeps the `feature-group` mechanism but pins its meaning: level scaffolds stay no-code containers; book-named grants are real `feature`s. No parser change expected — this is a source-annotation audit.

### Task 4.1: Audit and reclassify

**Files:**
- Inspect: `internal/content/feature.go` (`FeatureGroupParser`), `input/heroes/Draw Steel Heroes.md`, `input/beastheart/Draw Steel Beastheart.md`

- [ ] **Step 1: List every feature-group heading**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
grep -n "@type: feature-group" input/heroes/*.md input/beastheart/*.md input/monsters/*.md
```

- [ ] **Step 2: Classify each**

For each hit, decide: is it a bare level/organization scaffold (e.g. "Nth-Level Features", "Level N <species> Advancement Feature") → **leave as `feature-group`**; or a heading the book names and *grants* as a feature → change its annotation to `@type: feature`. Known cases:
- "College Features" is already `@type: feature` — no change.
- The fury "Stormwight Kits" framework group — leave as `feature-group` (organizational; the kits themselves are the grants).
Document any change inline; most/all should need none.

- [ ] **Step 3: If any annotation changed, commit**

```bash
git add input/
git commit -m "chore: reclassify book-named feature-group grants as features"
```

---

## Phase 5 — Regenerate registry + rewrite source links  ⚠️ LAST CODE PHASE

This rewrites `Draw Steel Heroes.md` and `Draw Steel Beastheart.md`. Run only after Phases 1–4 are merged and after a fresh pull, to compose with the concurrent linking work.

### Task 5.1: Pull, regenerate, capture the diff

- [ ] **Step 1: Sync**

```bash
cd /home/vexa/code/steel_compendium/workspace/steel-etl
git pull --ff-only origin main
```

- [ ] **Step 2: Regenerate all books and capture the old→new SCC mapping**

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl classify --diff --all > /tmp/scc-diff.txt 2>&1'
cat /tmp/scc-diff.txt | head -60
```
Expected: only `feature.trait.<class|domain|college|kit|companion>…` → `feature.<…>…` changes; **no** changes to `feature.ability.*`, `feature.trait.<ancestry>`, or `feature.trait.<monster>`. (Exact `classify --diff` flags: confirm against `internal/cli/classify.go` — adjust the command to whatever emits a machine-readable old→new pair list. If no such flag exists, generate the registry twice — once on `git stash` of Phases 1–4 — and `diff` the two `classification.json` files.)

- [ ] **Step 3: Sanity-check the diff scope**

```bash
grep -c "feature.trait" /tmp/scc-diff.txt   # all should be class/kit/companion (changing)
grep "feature.trait.dwarf\|feature.trait.devil\|feature.trait.hakaan" /tmp/scc-diff.txt || echo "OK: no ancestry churn"
```

### Task 5.2: Rewrite `scc:` cross-reference links from the mapping

**Files:**
- Modify: `input/heroes/Draw Steel Heroes.md`, `input/beastheart/Draw Steel Beastheart.md`

- [ ] **Step 1: Build the rewrite from the authoritative mapping**

Drive the replacement from `/tmp/scc-diff.txt` (old→new pairs), not a blind regex — this guarantees ancestry/monster `feature.trait.*` links are untouched. Write a small one-off script (Go or `perl -pi`) that, for each `old → new` pair, replaces `scc:<old>` with `scc:<new>` across both source files. Example skeleton (adjust parsing to the actual diff format):

```bash
devbox run -- bash -c 'cd steel-etl && while IFS= read -r line; do
  old=$(printf "%s" "$line" | sed -n "s/^- \(.*\) -> .*/\1/p")
  new=$(printf "%s" "$line" | sed -n "s/^- .* -> \(.*\)/\1/p")
  [ -n "$old" ] && [ -n "$new" ] && perl -pi -e "s/\Qscc:$old\E/scc:$new/g" input/heroes/*.md input/beastheart/*.md
done < /tmp/scc-diff.txt'
```

- [ ] **Step 2: Verify no dangling old-shape class links remain**

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl validate --all'
```
Expected: PASS, no broken `scc:` references.

- [ ] **Step 3: Regenerate and confirm stability**

```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml --all && go run ./cmd/steel-etl validate --scc-stable'
```
Expected: clean.

- [ ] **Step 4: Commit**

```bash
git add input/ classification.json 2>/dev/null; git add input/
git commit -m "refactor: rewrite feature.trait class/companion links to hub-and-spoke codes"
```

---

## Phase 6 — Documentation: the "meaning of trait narrowed" callout

### Task 6.1: Propagate the glossary + callout

**Files (each gets a short, explicit "⚠️ `trait` narrowed 2026-06-07: ancestry + monster only" note + the three-value model):**
- `steel-etl/ANNOTATION-GUIDE.md` — `@type` table: `feature` (non-ability; inferred to `trait` only in ancestry/monster context, else `feature`); `ability`. Add a worked example for each of the three resulting codes.
- `steel-etl/CLAUDE.md` — Content embedding/taxonomy section.
- workspace `CLAUDE.md` — the SCC overview paragraph (note the path shape + trait narrowing).
- `steel-etl/docs/linking-guide.md` + `linking-reference.md` — fix any `feature.trait.<class>` example targets.
- Inline comments already added in `feature.go` (Task 1.1); add one-line pointers in `ability.go` and `statblock_parse.go` referencing the spec.

- [ ] **Step 1: Edit each file** with the callout + corrected examples (exact text per file; keep it short and unambiguous — define the narrowed meaning, don't just rename).

- [ ] **Step 2: Commit**

```bash
git add ANNOTATION-GUIDE.md CLAUDE.md docs/ internal/content/ability.go internal/content/statblock_parse.go
git commit -m "docs: document narrowed 'trait' meaning and hub-and-spoke feature paths"
cd /home/vexa/code/steel_compendium/workspace
git add CLAUDE.md
git commit -m "docs: note feature/ability/trait taxonomy in workspace SCC overview"
```

---

## Phase 7 — SDK schema docs (data-sdk-npm)

### Task 7.1: Enumerate the three `feature_type` values

**Files:**
- Modify: `../data-sdk-npm/src/schema/feature.schema.json` (`feature_type` description), `feature.schema.json.md`

- [ ] **Step 1: Update the `feature_type` property description** in `feature.schema.json` from "The type of the feature (ability or trait)" to enumerate three values and define each:
  - `ability` — a feature with the combat shape (keywords/usage/distance/target/power roll).
  - `trait` — a non-ability feature on an ancestry or monster (rulebook's "trait").
  - `feature` — any other non-ability feature (class/domain/college/kit/companion).

- [ ] **Step 2: Mirror the same in `feature.schema.json.md`** (the root-object table row for `feature_type`).

- [ ] **Step 3: Commit (in the data-sdk-npm repo)**

```bash
cd /home/vexa/code/steel_compendium/workspace/data-sdk-npm
git add src/schema/feature.schema.json src/schema/feature.schema.json.md
git commit -m "docs: feature_type now enumerates ability | trait | feature"
```

---

## Self-Review checklist (run before execution handoff)

- **Spec coverage:** vocabulary (Phase 1, 6, 7) · hub-and-spoke path (Phase 1) · `feature_type`=3 (Phase 2, 7) · inference rule incl. kit/companion not trait (Phase 1) · feature-group split (Phase 4) · builder-readiness (docs only, no task — correct, it is sketch-only) · docs propagation (Phase 6) · breaking migration + link rewrite (Phase 5). All seven acceptance criteria in the spec map to a task.
- **Reserved-word consistency:** `featureKind` (site) and the parser both treat segment-after-`feature` `== "ability"` as ability, everything else as niche/feature — consistent.
- **Ancestry/monster untouched:** Phase 1 only marks trait when `ancestryID != ""`; Phase 5 diff guard asserts no ancestry/monster churn.
