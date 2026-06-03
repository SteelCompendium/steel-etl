# Beastheart Advancement Table — Link Generic Entries Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make the nine currently-unlinked entries in the Beastheart Advancement Table (the "Wild Nature Feature", "Wild Nature Ability", and "N-Ferocity Ability" cells) into working cross-reference links, matching how the Fury class already handles the identical entries.

**Architecture:** The unlinked cells reference container sections ("2nd-Level Wild Nature Feature", "7-Ferocity Ability", etc.) that currently carry **no annotation**, so they generate no SCC code and cannot be link targets. We annotate those nine headers as `@type: feature` with an explicit `@id` and `@level` (Beastheart's per-feature convention — its level wrappers are not `feature-group`s, unlike Fury), which produces `feature.trait.beastheart.level-N/<slug>` codes. Then we wrap the nine table cells in `[text](scc:...)` links pointing at those new codes. The single source edit propagates to all three generated renderings of the table (book-faithful subtree rendering emits it under `class/`, `chapter/`, and `feature/.../wild-nature.md`).

**Tech Stack:** Annotated Markdown source (`input/beastheart/Draw Steel Beastheart.md`), the Go `steel-etl` pipeline (`gen` command), MkDocs site build (`just deploy-v2`).

---

## Background & Verification (read before starting)

**The unlinked cells** (source file `input/beastheart/Draw Steel Beastheart.md`, table at lines 324–337):

| Table row | Bare cell text | Target header (source line) | Header markup |
|-----------|----------------|------------------------------|---------------|
| 2nd | Wild Nature Feature | `2nd-Level Wild Nature Feature` (1584) | `#####` |
| 2nd | Wild Nature Ability | `2nd-Level Wild Nature Ability` (1619) | `###` |
| 3rd | 7-Ferocity Ability | `7-Ferocity Ability` (1786) | `####` |
| 5th | Wild Nature Feature | `5th-Level Wild Nature Feature` (1893) | `#####` |
| 5th | 9-Ferocity Ability | `9-Ferocity Ability` (1925) | `####` |
| 6th | Wild Nature Ability | `6th-Level Wild Nature Ability` (2011) | `###` |
| 8th | Wild Nature Feature | `8th-Level Wild Nature Feature` (2187) | `#####` |
| 8th | 11-Ferocity Ability | `11-Ferocity Ability` (2224) | `####` |
| 9th | Wild Nature Ability | `9th-Level Wild Nature Ability` (2299) | `###` |

**Why this is safe (verified against the parser):**
- `internal/content/feature.go` builds the code path as `feature.trait.{classID}.level-{N}/{id}`. `classID` resolves to `beastheart` for headers at any depth (proven: existing `feature.trait.beastheart.level-1/beastheart-abilities` comes from an `###` header). `level-{N}` comes from the header's own `@level` annotation (pushed to the context stack at `pipeline.go:114` before parsing, then read by `ctx.Lookup` at `feature.go:73`).
- `collectAbilityChildren` (`feature.go:147`) only embeds a child ability when **exactly one** exists and only recurses through **unannotated** intermediaries. The "Wild Nature Feature" headers have `@type: feature` children (not abilities → not embedded). The "Wild Nature Ability" / "N-Ferocity" headers have **many** ability descendants → `len != 1` → no embed. So annotating these containers does **not** absorb children and does **not** change any existing child SCC code. Children keep their own standalone pages.
- `freeze: false` in `pipeline.yaml` → adding nine new codes is allowed.

**The canonical pattern we mirror** — Fury (`input/heroes/Draw Steel Heroes.md`, table line 9508+) already does exactly this:
```
[Aspect Feature](scc:mcdm.heroes.v1/feature.trait.fury.level-2/2nd-level-aspect-feature)
[7-Ferocity Ability](scc:mcdm.heroes.v1/feature.trait.fury.level-3/7-ferocity-ability)
```
with the target headers annotated `<!-- @type: feature -->`. Display text is the short cell label; the code uses the full header slug.

**The nine new codes this plan creates:**
```
feature.trait.beastheart.level-2/2nd-level-wild-nature-feature
feature.trait.beastheart.level-2/2nd-level-wild-nature-ability
feature.trait.beastheart.level-3/7-ferocity-ability
feature.trait.beastheart.level-5/5th-level-wild-nature-feature
feature.trait.beastheart.level-5/9-ferocity-ability
feature.trait.beastheart.level-6/6th-level-wild-nature-ability
feature.trait.beastheart.level-8/8th-level-wild-nature-feature
feature.trait.beastheart.level-8/11-ferocity-ability
feature.trait.beastheart.level-9/9th-level-wild-nature-ability
```

**Commands** (devbox-gated — Go is not on PATH):
- Gen: `devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml'`
- Site build/deploy: `just deploy-v2` (run from workspace root)

---

## Task 1: Capture the RED baseline

**Files:** none (read-only verification)

- [ ] **Step 1: Regenerate from current source so output matches HEAD**

Run from the workspace root:
```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml'
```
Expected: completes without error.

- [ ] **Step 2: Confirm the nine target pages do NOT yet exist (RED)**

Run from the workspace root:
```bash
for c in \
  feature/trait/beastheart/level-2/2nd-level-wild-nature-feature \
  feature/trait/beastheart/level-2/2nd-level-wild-nature-ability \
  feature/trait/beastheart/level-3/7-ferocity-ability \
  feature/trait/beastheart/level-5/5th-level-wild-nature-feature \
  feature/trait/beastheart/level-5/9-ferocity-ability \
  feature/trait/beastheart/level-6/6th-level-wild-nature-ability \
  feature/trait/beastheart/level-8/8th-level-wild-nature-feature \
  feature/trait/beastheart/level-8/11-ferocity-ability \
  feature/trait/beastheart/level-9/9th-level-wild-nature-ability ; do
  test -f "data/data-beastheart/en/md-linked/$c.md" && echo "EXISTS: $c" || echo "missing: $c"
done
```
Expected: all nine print `missing:` — confirming there is nothing to link to yet.

- [ ] **Step 3: Confirm the table cells are currently bare text (RED)**

Run from the workspace root:
```bash
grep -nE "Wild Nature Feature, Wild Nature Ability|, 7-Ferocity Ability| 9-Ferocity Ability|, 11-Ferocity Ability" \
  data/data-beastheart/en/md-linked/class/beastheart.md
```
Expected: prints rows showing `Wild Nature Feature`, `7-Ferocity Ability`, etc. as plain text (no `](` link markup around them).

---

## Task 2: Annotate the nine container headers

**Files:**
- Modify: `input/beastheart/Draw Steel Beastheart.md` (nine header lines, in the Beastheart class section)

Each step inserts one annotation comment immediately **above** an existing header line. The header text is unique in the file, so each `old_string` is unambiguous. Do **not** change the header text itself.

- [ ] **Step 1: Annotate "2nd-Level Wild Nature Feature" (line ~1584)**

Replace:
```
##### 2nd-Level Wild Nature Feature
```
with:
```
<!-- @type: feature | @id: 2nd-level-wild-nature-feature | @level: 2 -->
##### 2nd-Level Wild Nature Feature
```

- [ ] **Step 2: Annotate "2nd-Level Wild Nature Ability" (line ~1619)**

Replace:
```
### 2nd-Level Wild Nature Ability
```
with:
```
<!-- @type: feature | @id: 2nd-level-wild-nature-ability | @level: 2 -->
### 2nd-Level Wild Nature Ability
```

- [ ] **Step 3: Annotate "7-Ferocity Ability" (line ~1786)**

Replace:
```
#### 7-Ferocity Ability
```
with:
```
<!-- @type: feature | @id: 7-ferocity-ability | @level: 3 -->
#### 7-Ferocity Ability
```

- [ ] **Step 4: Annotate "5th-Level Wild Nature Feature" (line ~1893)**

Replace:
```
##### 5th-Level Wild Nature Feature
```
with:
```
<!-- @type: feature | @id: 5th-level-wild-nature-feature | @level: 5 -->
##### 5th-Level Wild Nature Feature
```

- [ ] **Step 5: Annotate "9-Ferocity Ability" (line ~1925)**

Replace:
```
#### 9-Ferocity Ability
```
with:
```
<!-- @type: feature | @id: 9-ferocity-ability | @level: 5 -->
#### 9-Ferocity Ability
```

- [ ] **Step 6: Annotate "6th-Level Wild Nature Ability" (line ~2011)**

Replace:
```
### 6th-Level Wild Nature Ability
```
with:
```
<!-- @type: feature | @id: 6th-level-wild-nature-ability | @level: 6 -->
### 6th-Level Wild Nature Ability
```

- [ ] **Step 7: Annotate "8th-Level Wild Nature Feature" (line ~2187)**

Replace:
```
##### 8th-Level Wild Nature Feature
```
with:
```
<!-- @type: feature | @id: 8th-level-wild-nature-feature | @level: 8 -->
##### 8th-Level Wild Nature Feature
```

- [ ] **Step 8: Annotate "11-Ferocity Ability" (line ~2224)**

Replace:
```
#### 11-Ferocity Ability
```
with:
```
<!-- @type: feature | @id: 11-ferocity-ability | @level: 8 -->
#### 11-Ferocity Ability
```

- [ ] **Step 9: Annotate "9th-Level Wild Nature Ability" (line ~2299)**

Replace:
```
### 9th-Level Wild Nature Ability
```
with:
```
<!-- @type: feature | @id: 9th-level-wild-nature-ability | @level: 9 -->
### 9th-Level Wild Nature Ability
```

- [ ] **Step 10: Regenerate and confirm the nine pages now exist (GREEN for Task 2)**

Run from the workspace root:
```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml'
for c in \
  feature/trait/beastheart/level-2/2nd-level-wild-nature-feature \
  feature/trait/beastheart/level-2/2nd-level-wild-nature-ability \
  feature/trait/beastheart/level-3/7-ferocity-ability \
  feature/trait/beastheart/level-5/5th-level-wild-nature-feature \
  feature/trait/beastheart/level-5/9-ferocity-ability \
  feature/trait/beastheart/level-6/6th-level-wild-nature-ability \
  feature/trait/beastheart/level-8/8th-level-wild-nature-feature \
  feature/trait/beastheart/level-8/11-ferocity-ability \
  feature/trait/beastheart/level-9/9th-level-wild-nature-ability ; do
  test -f "data/data-beastheart/en/md-linked/$c.md" && echo "OK: $c" || echo "MISSING: $c"
done
```
Expected: all nine print `OK:`.

- [ ] **Step 11: Confirm no existing beastheart feature codes changed**

Run from the workspace root:
```bash
git -C steel-etl/../data/data-beastheart status --short 2>/dev/null | grep -E "^ ?D|deleted" || echo "no deletions"
```
Expected: `no deletions` — annotating containers added pages but removed/renamed none. (If `data-beastheart` is not its own git repo, skip this check; the Step 10 result already proves children were untouched.)

- [ ] **Step 12: Commit the source annotations**

```bash
git add steel-etl/input/beastheart/Draw\ Steel\ Beastheart.md
git commit -m "feat: annotate Beastheart wild-nature/ferocity container headers as link targets"
```

---

## Task 3: Link the nine table cells

**Files:**
- Modify: `input/beastheart/Draw Steel Beastheart.md` (six table rows, lines 329–336)

Each step replaces one full table row. Use the **entire row line** as `old_string` — the short cell labels ("Wild Nature Feature", "Wild Nature Ability") repeat across rows and link to different per-level codes, so partial matches are ambiguous. Display text stays the short label; only the link wrapper is added. (Note: "Everyone's Best Friend" uses a curly apostrophe `’` — copy the row exactly.)

- [ ] **Step 1: Link the 2nd-level row (Wild Nature Feature + Wild Nature Ability)**

Replace:
```
| 2nd | [Perk](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-2/perk), [Everyone’s Best Friend](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-2/everyones-best-friend), Wild Nature Feature, Wild Nature Ability | Signature, 3, 5 | 5 |
```
with:
```
| 2nd | [Perk](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-2/perk), [Everyone’s Best Friend](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-2/everyones-best-friend), [Wild Nature Feature](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-2/2nd-level-wild-nature-feature), [Wild Nature Ability](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-2/2nd-level-wild-nature-ability) | Signature, 3, 5 | 5 |
```

- [ ] **Step 2: Link the 3rd-level row (7-Ferocity Ability)**

Replace:
```
| 3rd | [Companion Advancement Feature](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-3/companion-advancement-feature), 7-Ferocity Ability | Signature, 3, 5, 7 | 5 |
```
with:
```
| 3rd | [Companion Advancement Feature](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-3/companion-advancement-feature), [7-Ferocity Ability](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-3/7-ferocity-ability) | Signature, 3, 5, 7 | 5 |
```

- [ ] **Step 3: Link the 5th-level row (Wild Nature Feature + 9-Ferocity Ability)**

Replace:
```
| 5th | Wild Nature Feature, 9-Ferocity Ability | Signature, 3, 5, 7, 9 | 5 |
```
with:
```
| 5th | [Wild Nature Feature](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-5/5th-level-wild-nature-feature), [9-Ferocity Ability](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-5/9-ferocity-ability) | Signature, 3, 5, 7, 9 | 5 |
```

- [ ] **Step 4: Link the 6th-level row (Wild Nature Ability)**

Replace:
```
| 6th | [Perk](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-6/perk), [Become the Beast](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-6/become-the-beast), Wild Nature Ability | Signature, 3, 5, 7, 9 | 5, 9 |
```
with:
```
| 6th | [Perk](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-6/perk), [Become the Beast](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-6/become-the-beast), [Wild Nature Ability](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-6/6th-level-wild-nature-ability) | Signature, 3, 5, 7, 9 | 5, 9 |
```

- [ ] **Step 5: Link the 8th-level row (Wild Nature Feature + 11-Ferocity Ability)**

Replace:
```
| 8th | Wild Nature Feature, [Perk](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-8/perk), 11-Ferocity Ability | Signature, 3, 5, 7, 9, 11 | 5, 9 |
```
with:
```
| 8th | [Wild Nature Feature](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-8/8th-level-wild-nature-feature), [Perk](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-8/perk), [11-Ferocity Ability](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-8/11-ferocity-ability) | Signature, 3, 5, 7, 9, 11 | 5, 9 |
```

- [ ] **Step 6: Link the 9th-level row (Wild Nature Ability)**

Replace:
```
| 9th | [Avatar of the Green](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-9/avatar-of-the-green), Wild Nature Ability | Signature, 3, 5, 7, 9, 11 | 5, 9, 11 |
```
with:
```
| 9th | [Avatar of the Green](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-9/avatar-of-the-green), [Wild Nature Ability](scc:mcdm.beastheart.v1/feature.trait.beastheart.level-9/9th-level-wild-nature-ability) | Signature, 3, 5, 7, 9, 11 | 5, 9, 11 |
```

---

## Task 4: Regenerate, verify links resolve, deploy

**Files:** none new (regenerates `data/data-beastheart/`, builds `v2/`)

- [ ] **Step 1: Regenerate the pipeline**

Run from the workspace root:
```bash
devbox run -- bash -c 'cd steel-etl && go run ./cmd/steel-etl gen --config pipeline.yaml'
```
Expected: completes without error.

- [ ] **Step 2: Confirm every formerly-bare cell now renders as a resolved `.md` link (GREEN)**

Run from the workspace root:
```bash
grep -nE "\[Wild Nature Feature\]\(|\[Wild Nature Ability\]\(|\[7-Ferocity Ability\]\(|\[9-Ferocity Ability\]\(|\[11-Ferocity Ability\]\(" \
  data/data-beastheart/en/md-linked/class/beastheart.md
```
Expected: nine matches total across the six rows, each wrapped in `[...](../feature/trait/beastheart/level-N/...md)`. No occurrence should still be bare.

- [ ] **Step 3: Assert no bare entries remain in the generated table**

Run from the workspace root:
```bash
grep -nE ", Wild Nature Feature,| Wild Nature Ability \|| Wild Nature Feature, 9-Ferocity| 7-Ferocity Ability \|| 11-Ferocity Ability \|" \
  data/data-beastheart/en/md-linked/class/beastheart.md \
  && echo "FAIL: bare entries remain" || echo "PASS: no bare entries"
```
Expected: `PASS: no bare entries`.

- [ ] **Step 4: Confirm no broken/unresolved SCC references were introduced**

Run from the workspace root:
```bash
grep -rn "scc:mcdm.beastheart.v1" data/data-beastheart/en/md-linked/ && echo "FAIL: raw scc: links remain" || echo "PASS: all scc links resolved"
```
Expected: `PASS: all scc links resolved` (the generator rewrites every valid `scc:` URL to a relative `.md` path; a remaining raw `scc:` indicates an unresolved/typo'd code).

- [ ] **Step 5: Build the v2 site**

Run from the workspace root:
```bash
just deploy-v2
```
Expected: MkDocs build completes without error.

- [ ] **Step 6: Spot-check one rendered target page exists in the site**

Run from the workspace root:
```bash
ls v2/site/scc/mcdm.beastheart.v1/feature.trait.beastheart.level-3/7-ferocity-ability/index.html 2>/dev/null \
  && echo "OK: permalink stub generated" \
  || find v2/site -path "*7-ferocity-ability*" -name "index.html" | head
```
Expected: prints `OK:` or lists the generated page path, confirming the new target is reachable on the site.

- [ ] **Step 7: Commit**

```bash
git add steel-etl/input/beastheart/Draw\ Steel\ Beastheart.md
git commit -m "feat: link Wild Nature / Ferocity entries in Beastheart Advancement Table"
```

- [ ] **Step 8: Commit the regenerated data + site (per the workspace's normal deploy commit flow)**

Follow the repo's existing convention for committing generated output (the recent history uses `chore: bump steel-etl ...` style commits and the `steel-etl` submodule pointer). If `data/` and `v2/` are tracked submodules, commit within each and bump the pointers from the workspace; if they are plain directories, include them in the commit above. Verify with:
```bash
git status
```
Expected: working tree clean after the appropriate commits.

---

## Self-Review Notes

- **Spec coverage:** All nine unlinked cells from the table (2nd×2, 3rd×1, 5th×2, 6th×1, 8th×2, 9th×1) are addressed — Task 2 creates the nine targets, Task 3 links the nine cells across six rows.
- **Type consistency:** Every `@id` in Task 2 is the exact slug used in the matching link in Task 3 and in the verification greps (e.g. `2nd-level-wild-nature-feature`, `7-ferocity-ability`). Levels match the table row (Wild Nature Ability at row 6 → `level-6`, etc.).
- **No placeholders:** Every edit shows exact before/after text; every command is runnable as written.
- **Pattern parity:** Mirrors the Fury class's existing treatment of the identical "Aspect Feature / N-Ferocity Ability" entries.
