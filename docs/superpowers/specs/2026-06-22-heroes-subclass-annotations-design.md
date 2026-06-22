# Heroes subclass → `@subclass` annotation migration

**Date:** 2026-06-22
**Status:** Design approved
**Scope:** `steel-etl` only (edits `input/heroes/Draw Steel Heroes.md`; throwaway migration tool)

## Problem

The legacy `data-gen` pipeline associated subclass info to heroes abilities/features
via a sidecar `data-gen/input/heroes/metadata.json` (481 entries keyed by **old** SCC
codes, e.g. `mcdm.heroes.v1:feature.ability.elementalist.1st-level-feature:explosive-assistance`
→ `{"subclass": "Fire"}`). The new `steel-etl` pipeline carries subclass as a
**`@subclass` annotation** on the source heading instead, but that data was never
migrated — the heroes doc currently has **zero** `@subclass` annotations.

During the migration many SCC codes changed (notably the level segment:
`1st-level-feature` → `level-1`), so the old keys can't be mapped by code. The stable
join is `(class, slug)`: the class segment and the final slug are unchanged.

## Goal

Inject `@subclass: <slug>` into the heroes doc's existing annotation comments for all
**410 non-null** subclass facts in `metadata.json`. Target **410/410** — no fact
silently dropped.

## Decisions (locked)

- **`metadata.json` is read-only reference.** It lives in the deprecated `data-gen`
  repo; leave it untouched. The heroes doc becomes the sole source of truth for
  subclass.
- **One-time migration**, not a build-time sidecar. `steel-etl` does not read
  `metadata.json`.
- **Value format = lowercase-hyphen slug**, produced by `content.Slugify` on the
  display name. Matches the beastheart precedent (`@subclass: punisher`).
  `"Black Ash"` → `black-ash`, `"Caustic Alchemy"` → `caustic-alchemy`, `"Fire"` → `fire`.
- **Tooling = throwaway Go program** that imports `internal/content` and reuses the
  real `content.Slugify` / `content.CleanHeading` — zero id-derivation drift.
- **Residue resolved by hand**, targeting 410/410.

## Key facts established during exploration

- Target headings **already carry `@type` annotations** (e.g.
  `<!-- @type: ability | @subtype: triggered -->`) with **no `@id`** — the slug is
  derived from the heading text. The work is *injecting `@subclass`* into those
  existing comments, not adding new ones.
- Heading id derivation = `Slugify(CleanHeading(heading))`:
  `CleanHeading` strips a trailing `(N Cost)` parenthetical
  (`> ###### Lightning Lord (9 Piety)` → `Lightning Lord`); `Slugify` lowercases,
  drops apostrophes, and hyphenates non-alphanumerics. (`internal/content/helpers.go`.)
- **`subclass` is frontmatter-only and never enters the SCC path** (guaranteed by
  `TestAbilitySubclassFrontmatter`). Injecting changes no SCC code; `validate
  --scc-stable` stays green.
- `parseSubclass` (`internal/content/ability.go`) stores a single value as a string
  and a comma-separated list as `[]string`. Metadata is one subclass per entry, so all
  values are scalars.
- Some features appear **multiple times** in the doc: a canonical annotated instance
  (`<!-- @type: feature -->`) plus bare unannotated reproductions in domain/overview
  sections. Only annotated headings are indexed, so reproductions are never touched.

## Distribution (481 entries)

| | count | handling |
|---|---|---|
| Non-null subclass | 410 | inject |
| Null subclass (general elementalist + all kit abilities) | 71 | skip |

Non-null spread across 8 classes: censor, conduit, elementalist, fury, shadow,
tactician, talent, troubadour.

## Matching algorithm

1. Walk `Draw Steel Heroes.md` line by line, tracking the current class context from
   `@type: class | @id: <class>` annotations.
2. For every **annotated** `@type: ability|feature|trait` heading, compute
   `id = Slugify(CleanHeading(headingText))` (honoring an `@id` override when present)
   and record `(class, bucket, id) → comment-line`. `bucket` = `ability` for
   `@type: ability`, else `feature`.
3. Parse each metadata key into `(class, bucket, slug, subclass)`. Key shape:
   `…:feature.<ability|trait|feature>.<class>.<level>:<slug>` (and `kit-ability.*`,
   which are all null → skipped). `bucket` = `ability` for `feature.ability.*`, else
   `feature`.
4. Match in two tiers:
   - **Exact** `(class, bucket, slug)`.
   - **Type-relaxed** `(class, slug)` within the same class — recovers items the doc
     annotates as `@type: ability` while metadata bucketed them as trait/feature
     (e.g. `blessing-of-secrets`, `take-two`, `we-cant-be-upstaged`).
5. Inject `@subclass: Slugify(displayName)`:
   - single-line comment → insert ` | @subclass: <slug>` before the closing `-->`.
   - multi-line comment → add a `@subclass: <slug>` line before `-->`.
   - **Idempotent** — skip if `@subclass` already present.
6. Emit a **residue report**: unmatched entries, already-present, and any in-class
   slug collision.

### Expected automatic coverage

Validated by a dry-run probe against the current doc:

| tier | count |
|---|---|
| Exact `(class, bucket, slug)` | 391 |
| Type-relaxed `(class, slug)` | 3 |
| Residue (needs hand resolution) | 16 |
| **Total non-null** | **410** |

Zero ambiguous (multi-heading) matches under `(class, bucket, slug)` scoping.

## Residue resolution (the ~16)

These have a canonical annotated instance in the doc but don't auto-match because of:

- **Class-context divergence** — e.g. conduit domain features (`oracular-warning`,
  `light-of-revelation`, `seance`, `invocation-of-the-heart`, …) whose annotated home
  resolves to a different class context than metadata's `conduit`.
- **Slug change** — e.g. `source-of-earth-statblock` (metadata) vs the doc's
  `summon-source-of-earth` statblock.
- **Renamed/restructured** items.

For each: locate the correct canonical annotated heading and inject `@subclass` there,
either by hand or via a small override map fed to the tool
(`{old-key → resolved (class, slug)}`). No entry is silently dropped; the run ends with
410/410 accounted for (injected or explicitly mapped).

## Verification

- `steel-etl validate --scc-stable` → no code drift (subclass is path-invisible).
- `steel-etl gen` for heroes → `subclass:` frontmatter present on ~410 items;
  spot-check `explosive-assistance: fire`, a conduit domain feature, and a multi-word
  slug like `caustic-alchemy`.
- Doc diff review: only annotation comments changed; changed-comment count ≈ 410.

## Out of scope

- No parser or schema changes (`@subclass` is already parsed and emitted).
- No `metadata.json` edits.
- No SCC scheme/registry changes.

## Cleanup

Remove the throwaway migration tool (`cmd/subclass-migrate/` or equivalent) after the
run; the doc edits are the durable artifact.
