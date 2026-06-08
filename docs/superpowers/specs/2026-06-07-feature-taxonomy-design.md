# Design: Feature / Ability / Trait taxonomy

**Date:** 2026-06-07
**Status:** Approved (design)
**Source:** Brainstorm — canonical naming of features/abilities/traits before the SCC freeze
**Scope:** steel-etl (primary), data-sdk-npm (schema docs), workspace + v2 docs

## Goal

Settle the canonical vocabulary and SCC shape for the three related concepts —
**feature**, **ability**, **trait** — while SCC codes are still unfrozen, even at
the cost of a breaking change. Two prime directives, in order:

1. **Be faithful to the rulebook's language.**
2. **Be clear about data types** (the output is an industry-standard format
   consumed by third-party Draw Steel tools, not just the v2 site).

A secondary motivation — not implemented here, but the taxonomy must not preclude
it — is a future character builder that can read the JSON/YAML and know which
features are granted automatically vs. chosen.

## The problem we are fixing

"Feature" is overloaded across three layers, and "trait" is applied where the
book never uses it:

| Layer | "feature" today means | "trait" today means |
|-------|-----------------------|---------------------|
| **Rulebook** | the umbrella ("1st-Level College **Features**", domain "grants **features**") | **only** ancestry traits (signature/purchased) and **monster** statblock passives ("Crafty") |
| **Author annotation** (`@type:`) | the *non-ability* subtype (`@type: feature`) | — (authors never type "trait") |
| **Output** (SDK + SCC) | the umbrella (`type: feature`, `feature.*`) | the non-ability subtype (`feature_type: trait`, `feature.trait.*`) |

So one concept — "a non-ability feature" — is called `feature` on input and
`trait` on output; and "feature" simultaneously names the umbrella and (on input)
one of its children. The example **A Beyonding of Vision** is listed in the book
as an elementalist *feature* (Void column), yet is coded
`feature.trait.elementalist.level-1/a-beyonding-of-vision` — a word the book
never applies to it.

## Core model

An **ability is a feature plus rigor** (keywords + usage + distance + target +
power roll/effect). It is a *specialization* of feature, never a sibling. A
**trait is a feature in a specific home** (ancestry / monster / companion), which
is the only place the rulebook uses the word. Therefore there is one base type
(`feature`) with two recognized specializations (`ability`, `trait`).

### 1. Vocabulary

- **`feature`** — the umbrella. Every grantable thing is a feature. A feature may
  recursively contain features, effects, and abilities. Unchanged; matches book +
  SDK.
- **`ability`** — a feature carrying the rigorous combat shape. A specialization.
- **`trait`** — ⚠️ **narrowed.**
  - **OLD:** *any* non-ability feature (class, domain, college, kit, ancestry,
    monster, companion — everything).
  - **NEW:** a non-ability feature whose home is an **ancestry** or a
    **monster/statblock** — the only two books that use the word "trait." Class /
    domain / college / kit / **companion** non-ability features are **no longer
    traits**; they are plain `feature`s.

#### Source verification (2026-06-07)

The two trait homes were confirmed against the source markdown, and one assumed
home was rejected:

- **Heroes / ancestries → trait ✅** — "Ancestry Traits," "signature traits,"
  "purchased traits," "Signature Trait: Silver Tongue," "Purchased Dwarf Traits."
- **Monsters → trait ✅** — the book *defines* it: "Many creatures have **traits**,
  which are features that don't require a main action, a maneuver, or a triggered
  action to activate, such as the Crafty trait" (and the ⭐️ icon legend: "A trait
  of the creature, often a feature that is always in effect").
- **Beastheart / companions → feature, NOT trait ❌** — the Beastheart book uses
  the word "trait" **zero** times. Companion grants are "Features" (table column
  "Features"; headings like "Level 3 Basilisk Advancement Feature"; authored
  `@type: feature`). A companion is mechanically a creature, but its book calls
  its features "features," so fidelity requires `feature_type: feature`. Companion
  signature abilities remain `ability`.

### 2. SCC path — hub-and-spoke (base unmarked, specializations marked)

The base case is unmarked; each specialization inserts a reserved segment:

```
feature.<entity>...          base:   plain class/domain/college/kit/companion feature   (A Beyonding of Vision)
feature.ability.<entity>...  marked: rigorous ability (any source)                    (Shared Void Sense)
feature.trait.<entity>...    marked: ancestry/monster feature                          (Mighty, Crafty)
```

Companion examples: `feature.companion.basilisk.level-3/foes-forever-frozen`
(plain feature) and `feature.ability.companion.basilisk.level-1/petrify`
(signature ability).

`ability` and `trait` are **reserved words** in segment 2; anything else there is
the entity id (`elementalist`, `shadow`, `dwarf`, `common`, `companion`, …). A
consumer reads segment 2: if it is `ability` or `trait`, the code is a marked
kind; otherwise it is a plain feature and segment 2 begins the entity path.

**Why asymmetric is correct, not odd:** marking the base case (`feature.feature.*`)
would imply the default is special. Marking only the specializations is the honest
encoding of "ability ⊂ feature" and "trait = feature-in-a-home."

**Known cost (accepted):** siblings sit at different depths
(`feature.ability.shadow.level-1/x` vs `feature.shadow.level-1/y`), so any
"group the browse tree by segment 2" logic must special-case the two reserved
words. Cheap, and tools that need the distinction can read `feature_type` from
frontmatter instead of parsing the path.

### 3. `feature_type` field — three values, aligned 1:1 with the path

`feature_type ∈ { ability, trait, feature }`, matching the path's marked/unmarked
kind exactly (`feature.trait.dwarf/mighty` ⇒ `feature_type: trait`;
`feature.shadow.level-1/college-features` ⇒ `feature_type: feature`).

**Parser inference rule (minimizes annotation churn):**

- `@type: ability` ⇒ `feature_type: ability`, path inserts `ability`. (Any home.)
- `@type: feature` ⇒
  - `feature_type: trait`, path inserts `trait`, **iff** the nearest typed home ∈
    { `ancestry`, `monster`/`statblock` };
  - else `feature_type: feature`, path inserts no marker.

Authors keep writing `@type: feature` for non-ability features everywhere; the
parser decides trait-vs-feature from context. **`kit` and `companion` are NOT
trait homes** — the book calls kit grants "benefits" and companion grants
"features," not "traits," so their non-ability features are plain `feature`s.
Statblock parsing already labels passives `trait` and actions `ability`, so it
already conforms; **the existing companion branch in `FeatureParser` that prepends
`trait` must be removed** so companion non-ability features take the base shape.

### 4. `feature-group` clarification

`feature-group` keeps its mechanism (level/scope context, optional path segment,
no own code), but its semantic role is pinned down:

- **Level scaffolds** ("Nth-Level Features" wrappers) → remain **no-code
  structural containers**. The book does not grant "1st-Level Features" as a
  thing; it is just the level bucket.
- **Named grouping features the book grants** (e.g. **College Features**, which
  "grants you one or two features"; the fury's **Stormwight Kits** framework) →
  are real `feature`s (`feature_type: feature`) that nest children and keep a
  code. A group is simply a `feature` whose children are its sub-features.

**Implementation task:** audit existing `@type: feature-group` usages; any that
correspond to a book-named grant become `@type: feature`. Pure level/organization
scaffolds stay `feature-group`. (Note: "College Features" is already authored as
`@type: feature` and already gets a code — it only needs the trait→feature
reclassification, not a structural change.)

### 5. Builder-readiness appendix (sketch only — NOT implemented here)

Captured so the taxonomy leaves the right hooks for a future character builder.
No fields below are emitted by this spec's work.

A chooser is just a `feature` whose children are the options. A future grant/choice
layer would add, per feature:

```yaml
grant: automatic | choice      # do you just get it, or pick?
choose: 1                        # how many, when grant == choice
options: [ <scc>, <scc>, ... ]   # or rely on nested children as the option set
prerequisites: [ ... ]           # optional gating
```

Confirmed non-blocking: because grouping is structural recursion (feature →
features), "choose one or two College Features" maps to a `feature` with
`grant: choice, choose: 1..2` over its child features — no schema dead-end. This
section exists to prevent the taxonomy from painting the builder into a corner; it
does not commit to the field names above.

## Documentation & migration

### Docs to update (the "meaning of trait changed" propagation)

A shared glossary blurb + an explicit "⚠️ meaning of `trait` narrowed on
2026-06-07" callout must land in:

- `steel-etl/ANNOTATION-GUIDE.md` — the `@type` table (feature vs ability;
  trait is inferred, never authored) + worked examples.
- `steel-etl/CLAUDE.md` — content-embedding / taxonomy notes.
- workspace `CLAUDE.md` — the SCC overview paragraph.
- `steel-etl/docs/linking-guide.md` + `linking-reference.md` — any `feature.trait.*`
  class targets that change shape.
- `data-sdk-npm/src/schema/feature.schema.json` (`feature_type` description) and
  `feature.schema.json.md` — enumerate the three values and define each.
- Inline comments in `internal/content/feature.go`, `internal/content/ability.go`,
  `internal/content/statblock_parse.go` describing the inference rule.

Because a large amount of existing code, comments, and docs say "trait" with the
OLD broad meaning, every touched location must be made unambiguous: state the
narrowed definition, not just rename.

### Breaking SCC change

Every `feature.trait.<class|domain|college|kit|companion>…` code → `feature.<…>…`
(the `trait` segment is dropped for every home except ancestry and
monster/statblock). This is a large fraction of the ~1,807 heroes codes plus the
beastheart companion features. SCC is unfrozen, so regenerate:

- `steel-etl classify --diff` quantifies the churn before committing.
- Source cross-reference links (`scc:` links in `Draw Steel Heroes.md` **and
  `Draw Steel Beastheart.md`**) that point at reclassified class/companion
  features must be rewritten to the new codes.
- `feature.trait.*` codes for **ancestries and monsters only** are **unchanged**.
  Companion `feature.trait.companion.*` codes **do** change to
  `feature.companion.*`.

## Out of scope

- Implementing grant/choice metadata across the corpus (appendix is a sketch only).
- Any change to the umbrella word `feature` or the SDK `type: "feature"` value.
- The sibling-namespace restructure (`ability.*` / `feature.*` / `trait.*` as
  three top-level namespaces) — rejected in favor of keeping `feature.*` as the
  registry root that mirrors the SDK umbrella.
- Reordering the path (e.g. entity-before-kind) — separate concern, not addressed.

## Acceptance criteria

1. A plain class/domain/college non-ability feature emits `feature_type: feature`
   and an SCC code with **no** `trait`/`ability` segment
   (`feature.elementalist.level-1/a-beyonding-of-vision`).
2. An ability emits `feature_type: ability` and `feature.ability.<entity>…`
   regardless of home.
3. An **ancestry or monster** non-ability feature emits `feature_type: trait`
   and `feature.trait.<entity>…`.
4. `kit` **and companion** non-ability features emit `feature_type: feature`
   (not `trait`): e.g. `feature.companion.basilisk.level-3/foes-forever-frozen`.
   Companion signature abilities stay `feature.ability.companion.*`.
5. `steel-etl validate` passes; `classify --diff` shows only the intended
   trait→feature reshaping (class/domain/college/kit/companion) and **no** churn
   to ability codes or to ancestry/monster `feature.trait.*` codes.
6. The three example headings resolve correctly: **9th-Level College Ability** →
   `feature.ability.shadow…`; **A Beyonding of Vision** →
   `feature.elementalist…`; **1st-Level College Features** →
   `feature.shadow…` as a `feature` that nests its granted sub-features.
7. The "meaning of `trait` narrowed" callout is present in every doc listed above.
