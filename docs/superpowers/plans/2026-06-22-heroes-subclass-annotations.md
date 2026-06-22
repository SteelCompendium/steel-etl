# Heroes subclass → `@subclass` annotation migration — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Inject `@subclass: <slug>` into the heroes doc's existing annotation comments for all 410 non-null subclass facts in `data-gen/input/heroes/metadata.json`, reaching 410/410.

**Architecture:** A throwaway Go program under `steel-etl/cmd/subclass-migrate/` imports `internal/content` and reuses `content.Slugify` / `content.CleanHeading` for zero id-derivation drift. It walks the heroes doc, indexes annotated `@type: ability|feature|trait` headings by `(class, bucket, slug)`, joins the metadata (two-tier match: exact then type-relaxed), injects the annotation idempotently, and prints a residue report. The residue (~16) is resolved via an override map, then the tool is removed.

**Tech Stack:** Go (devbox toolchain), steel-etl `internal/content` package, the deprecated `data-gen/input/heroes/metadata.json` (read-only reference).

## Global Constraints

- All Go/just commands run under devbox: `devbox run -- go …` (Go is not on PATH).
- Value format: lowercase-hyphen slug via `content.Slugify(displayName)`.
- `data-gen/input/heroes/metadata.json` is read-only — never edit it.
- Only annotation comments in `input/heroes/Draw Steel Heroes.md` may change; never touch bare unannotated headings (reproductions).
- `subclass` is frontmatter-only and never in the SCC path — `validate --scc-stable` must stay green.
- Injection is idempotent: skip a comment that already has `@subclass`.
- Branch: `feat/heroes-subclass-annotations` (already created in `steel-etl`).

---

### Task 1: Build the migration tool (match + report, dry-run)

**Files:**
- Create: `steel-etl/cmd/subclass-migrate/main.go`

**Interfaces:**
- Consumes: `content.Slugify(string) string`, `content.CleanHeading(string) string` (from `github.com/SteelCompendium/steel-etl/internal/content`).
- Produces: a CLI with flags `-doc` (heroes md path), `-metadata` (metadata.json path), `-apply` (default false = dry-run report only). In dry-run it prints match tiers and the residue list; with `-apply` it rewrites the doc.

- [ ] **Step 1: Scaffold the tool with parsing + indexing + dry-run report**

Create `steel-etl/cmd/subclass-migrate/main.go`:

```go
// Command subclass-migrate is a one-off migration: it injects @subclass
// annotations into the heroes source doc from the legacy data-gen
// metadata.json. Throwaway — removed after the migration run.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"

	"github.com/SteelCompendium/steel-etl/internal/content"
)

var (
	annRe   = regexp.MustCompile(`^\s*<!--(.*?)-->\s*$`)
	headRe  = regexp.MustCompile(`^\s*>?\s*#{1,6}\s+(.*)$`)
	typeRe  = regexp.MustCompile(`@type:\s*(\w+)`)
	idRe    = regexp.MustCompile(`@id:\s*([a-z0-9-]+)`)
	subRe   = regexp.MustCompile(`@subclass:`)
)

type entry struct {
	Subclass *string `json:"subclass"`
}

// key identifies a heading by class context, bucket (ability|feature) and slug.
type key struct{ class, bucket, slug string }

// headingRef points at the annotation comment line for a matched heading.
type headingRef struct {
	commentLine int // 0-based index into lines
}

func bucketOf(t string) string {
	if t == "ability" {
		return "ability"
	}
	return "feature"
}

// indexDoc returns, per (class,bucket,slug), the comment lines of every
// ANNOTATED ability/feature/trait heading. Bare headings are ignored.
func indexDoc(lines []string) map[key][]headingRef {
	idx := map[key][]headingRef{}
	curClass := ""
	for i := 0; i < len(lines); i++ {
		m := annRe.FindStringSubmatch(lines[i])
		if m == nil {
			continue
		}
		body := m[1]
		tm := typeRe.FindStringSubmatch(body)
		if tm == nil {
			continue
		}
		t := tm[1]
		if t == "class" {
			if id := idRe.FindStringSubmatch(body); id != nil {
				curClass = id[1]
			}
			continue
		}
		if t != "ability" && t != "feature" && t != "trait" {
			continue
		}
		// find the next non-blank line; it must be a heading
		j := i + 1
		for j < len(lines) && strings.TrimSpace(lines[j]) == "" {
			j++
		}
		if j >= len(lines) {
			continue
		}
		hm := headRe.FindStringSubmatch(lines[j])
		if hm == nil {
			continue
		}
		slug := ""
		if id := idRe.FindStringSubmatch(body); id != nil {
			slug = id[1]
		} else {
			slug = content.Slugify(content.CleanHeading(hm[1]))
		}
		k := key{curClass, bucketOf(t), slug}
		idx[k] = append(idx[k], headingRef{commentLine: i})
	}
	return idx
}

// parseKey splits a metadata key into (class, bucket, slug).
// e.g. mcdm.heroes.v1:feature.ability.elementalist.1st-level-feature:explosive-assistance
func parseKey(k string) (class, bucket, slug string, ok bool) {
	colon := strings.Split(k, ":")
	if len(colon) < 3 {
		return "", "", "", false
	}
	slug = colon[len(colon)-1]
	path := strings.Split(colon[1], ".")
	if path[0] == "feature" && len(path) >= 3 {
		return path[2], bucketOf(path[1]), slug, true
	}
	return "", "", slug, false // kit-ability.* etc. — all null, skipped anyway
}

// overrides maps a metadata key whose (class,bucket,slug) does not auto-match to
// the resolved (class,bucket,slug) of the correct canonical annotated heading.
// Populated in Task 3.
var overrides = map[string]key{}

func main() {
	docPath := flag.String("doc", "input/heroes/Draw Steel Heroes.md", "heroes markdown")
	mdPath := flag.String("metadata", "../data-gen/input/heroes/metadata.json", "legacy metadata.json")
	apply := flag.Bool("apply", false, "rewrite the doc (default: dry-run report)")
	flag.Parse()

	raw, err := os.ReadFile(*docPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	lines := strings.Split(string(raw), "\n")
	idx := indexDoc(lines)

	mdRaw, err := os.ReadFile(*mdPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	var md map[string]entry
	if err := json.Unmarshal(mdRaw, &md); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	// Deterministic order for stable reporting.
	keys := make([]string, 0, len(md))
	for k := range md {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	type hit struct {
		mdKey   string
		ref     headingRef
		slugVal string
	}
	var exact, relaxed, viaOverride []hit
	var residue, already []string
	nonNull := 0

	for _, mk := range keys {
		e := md[mk]
		if e.Subclass == nil || *e.Subclass == "" {
			continue
		}
		nonNull++
		class, bucket, slug, ok := parseKey(mk)
		if !ok {
			residue = append(residue, mk+"  (unparseable / kit)")
			continue
		}
		val := content.Slugify(*e.Subclass)

		if k, has := overrides[mk]; has {
			if refs := idx[k]; len(refs) == 1 {
				viaOverride = append(viaOverride, hit{mk, refs[0], val})
				continue
			}
			residue = append(residue, mk+"  (override did not resolve to a unique heading)")
			continue
		}
		// Tier 1: exact (class, bucket, slug)
		if refs := idx[key{class, bucket, slug}]; len(refs) == 1 {
			exact = append(exact, hit{mk, refs[0], val})
			continue
		}
		// Tier 2: type-relaxed (class, slug) — try the other bucket
		other := "feature"
		if bucket == "feature" {
			other = "ability"
		}
		if refs := idx[key{class, other, slug}]; len(refs) == 1 {
			relaxed = append(relaxed, hit{mk, refs[0], val})
			continue
		}
		residue = append(residue, fmt.Sprintf("%s  (class=%s bucket=%s slug=%s)", mk, class, bucket, slug))
	}

	// Mark hits whose comment already carries @subclass (idempotency report).
	allHits := append(append(append([]hit{}, exact...), relaxed...), viaOverride...)
	for _, h := range allHits {
		if subRe.MatchString(lines[h.ref.commentLine]) {
			already = append(already, h.mdKey)
		}
	}

	fmt.Printf("non-null entries: %d\n", nonNull)
	fmt.Printf("exact:      %d\n", len(exact))
	fmt.Printf("relaxed:    %d\n", len(relaxed))
	fmt.Printf("override:   %d\n", len(viaOverride))
	fmt.Printf("already set: %d\n", len(already))
	fmt.Printf("RESIDUE:    %d\n", len(residue))
	for _, r := range residue {
		fmt.Println("  -", r)
	}

	if !*apply {
		fmt.Println("\n(dry-run; pass -apply to rewrite the doc)")
		return
	}

	// Apply: inject ` | @subclass: <val>` before the closing --> of each comment.
	injected := 0
	for _, h := range allHits {
		ln := lines[h.ref.commentLine]
		if subRe.MatchString(ln) {
			continue // idempotent
		}
		// single-line comment: insert before the final -->
		idxClose := strings.LastIndex(ln, "-->")
		if idxClose < 0 {
			fmt.Fprintf(os.Stderr, "WARN multi-line comment at %d not handled: %q\n", h.ref.commentLine+1, ln)
			continue
		}
		head := strings.TrimRight(ln[:idxClose], " ")
		lines[h.ref.commentLine] = head + " | @subclass: " + h.slugVal + " -->"
		injected++
	}
	if err := os.WriteFile(*docPath, []byte(strings.Join(lines, "\n")), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
	fmt.Printf("\ninjected: %d\n", injected)
}
```

- [ ] **Step 2: Build it**

Run: `cd steel-etl && devbox run -- go build ./cmd/subclass-migrate/`
Expected: builds with no errors.

- [ ] **Step 3: Dry-run and capture the residue**

Run (from `steel-etl/`): `devbox run -- go run ./cmd/subclass-migrate -doc "input/heroes/Draw Steel Heroes.md" -metadata "../data-gen/input/heroes/metadata.json"`
Expected: `non-null entries: 410`, `exact: 391`, `relaxed: 3`, `RESIDUE: 16`, with the 16 residue keys listed.

- [ ] **Step 4: Commit the tool**

```bash
cd steel-etl && git add cmd/subclass-migrate/main.go && git commit -m "feat: add throwaway subclass-migrate tool (dry-run)"
```

---

### Task 2: Resolve the residue into the override map

**Files:**
- Modify: `steel-etl/cmd/subclass-migrate/main.go` (`overrides` map only)

**Interfaces:**
- Consumes: the dry-run residue list from Task 1.
- Produces: a populated `overrides` map such that a re-run reports `RESIDUE: 0`.

- [ ] **Step 1: Investigate each residue key**

For each residue key, find the correct canonical annotated heading in `input/heroes/Draw Steel Heroes.md`:

```bash
cd steel-etl
# Example pattern per slug — confirm the heading that has a preceding @type comment:
grep -nE "@type|#{2,6}.*Oracular Warning" "input/heroes/Draw Steel Heroes.md" | grep -iE "oracular|@type"
```

For each, determine the `(class, bucket, slug)` of its annotated heading as `indexDoc` would compute it (mind the class context it actually sits under, and the actual heading text → `Slugify(CleanHeading(...))`). For statblock/renamed cases (e.g. `source-of-earth-statblock`), find the real annotated entity the subclass fact belongs to (`summon-source-of-earth`) — if the target is a `@type: statblock` (not ability/feature), note it: statblocks aren't indexed, so either point the override at the ability that grants it or record it as intentionally-skipped in the plan's completion notes.

- [ ] **Step 2: Populate the `overrides` map**

Replace `var overrides = map[string]key{}` with the resolved entries, e.g.:

```go
var overrides = map[string]key{
	"mcdm.heroes.v1:feature.trait.conduit.<level>:oracular-warning": {"<resolved-class>", "feature", "oracular-warning"},
	// ... one line per residue key, with the verified (class, bucket, slug)
}
```

(Exact keys/values come from Step 1 — copy the metadata key verbatim and the resolved heading's computed key.)

- [ ] **Step 3: Re-run dry-run to confirm zero residue**

Run: `cd steel-etl && devbox run -- go run ./cmd/subclass-migrate -doc "input/heroes/Draw Steel Heroes.md" -metadata "../data-gen/input/heroes/metadata.json"`
Expected: `RESIDUE: 0` and `exact + relaxed + override == 410` (minus any explicitly-skipped statblock case, which must be called out in stdout/notes, not silently dropped).

- [ ] **Step 4: Commit the override map**

```bash
cd steel-etl && git add cmd/subclass-migrate/main.go && git commit -m "feat: resolve subclass-migrate residue via override map"
```

---

### Task 3: Apply the injection to the heroes doc

**Files:**
- Modify: `steel-etl/input/heroes/Draw Steel Heroes.md` (annotation comments only)

**Interfaces:**
- Consumes: the zero-residue tool from Task 2.
- Produces: the heroes doc with ~410 `@subclass` annotations injected.

- [ ] **Step 1: Apply**

Run: `cd steel-etl && devbox run -- go run ./cmd/subclass-migrate -doc "input/heroes/Draw Steel Heroes.md" -metadata "../data-gen/input/heroes/metadata.json" -apply`
Expected: `injected: 410` (or `410 - <explicitly-skipped>`), no WARN lines.

- [ ] **Step 2: Sanity-check the diff shape**

Run: `cd steel-etl && git diff --stat "input/heroes/Draw Steel Heroes.md" && git diff "input/heroes/Draw Steel Heroes.md" | grep -E '^[+-]' | grep -v '@subclass' | grep -vE '^(\+\+\+|---)'`
Expected: the second grep prints nothing — every changed line differs only by the added `@subclass` field.

- [ ] **Step 3: Spot-check known values**

Run: `cd steel-etl && grep -nE "Explosive Assistance|Caustic|Coat the Blade" "input/heroes/Draw Steel Heroes.md" | head; grep -n "@subclass: fire" "input/heroes/Draw Steel Heroes.md" | head -3; grep -c "@subclass:" "input/heroes/Draw Steel Heroes.md"`
Expected: `@subclass:` count ≈ 410; `explosive-assistance` comment now carries `@subclass: fire`; multi-word slugs like `caustic-alchemy` present.

- [ ] **Step 4: Commit the doc edit**

```bash
cd steel-etl && git add "input/heroes/Draw Steel Heroes.md" && git commit -m "feat: add @subclass annotations to heroes abilities/features"
```

---

### Task 4: Verify against the pipeline

**Files:** none (verification only)

**Interfaces:**
- Consumes: the annotated doc.
- Produces: confidence that subclass flows to frontmatter and no SCC codes drifted.

- [ ] **Step 1: SCC stability**

Run: `cd steel-etl && devbox run -- go run ./cmd/steel-etl validate --scc-stable --config pipeline.yaml`
Expected: no code-change errors (subclass is path-invisible). If `validate` needs different flags, use `devbox run -- go run ./cmd/steel-etl validate --help` to confirm.

- [ ] **Step 2: Generate heroes output and confirm frontmatter**

Run: `cd steel-etl && devbox run -- go run ./cmd/steel-etl gen --config pipeline.yaml`
Then: `grep -rl "subclass: fire" ../data/ | head; grep -rh "^subclass:" ../data/ | sort | uniq -c | sort -rn | head -20`
Expected: leaf pages for fire-element elementalist abilities carry `subclass: fire`; the histogram shows the expected subclass slugs (`fire`, `war`, `black-ash`, `caustic-alchemy`, …) with sane counts.

- [ ] **Step 3: Confirm full subclass coverage count**

Run: `cd steel-etl && grep -rh "^subclass:" ../data/en/ 2>/dev/null | wc -l`
Expected: a count consistent with 410 source facts (a single fact can surface on multiple Browse/Read renders, so this is ≥410; the source `@subclass:` count of ~410 is the authoritative figure).

- [ ] **Step 4: Run the package tests** (guards against accidental parser regression)

Run: `cd steel-etl && devbox run -- go test ./internal/content/... ./internal/output/...`
Expected: PASS.

---

### Task 5: Remove the throwaway tool and finalize

**Files:**
- Delete: `steel-etl/cmd/subclass-migrate/`

**Interfaces:**
- Consumes: completed migration.
- Produces: clean tree; the doc edits are the durable artifact.

- [ ] **Step 1: Remove the tool**

Run: `cd steel-etl && git rm -r cmd/subclass-migrate`

- [ ] **Step 2: Confirm build still green without it**

Run: `cd steel-etl && devbox run -- go build ./...`
Expected: builds.

- [ ] **Step 3: Commit removal**

```bash
cd steel-etl && git commit -m "chore: remove throwaway subclass-migrate tool"
```

- [ ] **Step 4: Note follow-up**

The superproject pointer bump + deploy (regenerating `data/` and the v2 site) is a separate step per `docs/git-workflow.md` — do it when ready to ship (not part of this migration's source change).

---

## Self-Review notes

- **Spec coverage:** value format (Task 1 `content.Slugify`), one-time/no-metadata-edit (read-only, never written), two-tier match (Task 1), residue→410/410 (Task 2), idempotency (Task 1 `subRe` guard), reproductions untouched (only annotated headings indexed), verification incl. `--scc-stable` + `gen` (Task 4), tool cleanup (Task 5) — all covered.
- **Decision:** residue resolution uses an in-tool `overrides` map rather than hand-editing the doc, so the full mapping is reviewable in one place and the run stays reproducible.
- **Statblock edge:** `source-of-earth-statblock` may resolve to a `@type: statblock` entity that `indexDoc` does not index. If so it is explicitly reported as skipped (not silently dropped) and noted at completion — the only allowable shortfall from 410.
