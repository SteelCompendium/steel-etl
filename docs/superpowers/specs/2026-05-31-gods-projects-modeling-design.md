# Design: Deeper modeling of gods & downtime projects

**Date:** 2026-05-31
**Status:** Approved (design)
**Source:** FOLLOWUPS.md — "Deeper modeling of gods and downtime projects"

## Goal

Enrich the minimal `god` and `project` parsers with structured fields, annotate
the individual saints/heroes/evil-gods that are currently folded into their
parent god's body, and capture the ancestry purchased-trait point cost that is
currently discarded. Also correct documentation that overstates SCC codes as
"frozen/immutable."

All changes are **additive**: no existing SCC code changes or is removed, so the
registry's additive-only freeze guard (`ValidateAgainstFrozen`, which only checks
that frozen codes are not *missing*) continues to pass.

## Pieces

### 1. God `domains` field (parser-only)

`internal/content/god.go` — `GodParser` extracts the `**Domains:** X, Y, Z` line
(present on every god and saint) into a structured list:

```yaml
domains: [Creation, Knowledge, Life, Nature, Protection]
```

Use the existing comma-splitting behavior; reuse/extend `extractField` +
`strings.Split(", ")`. Omit the field when no Domains line is present.

### 2. Project structured fields (parser-only)

`internal/content/project.go` — `ProjectParser` extracts four regular bold-label
fields via the existing `extractField(body, name)` helper:

| Source label                      | Frontmatter key        | Notes                              |
|-----------------------------------|------------------------|------------------------------------|
| `**Item Prerequisite:**`          | `prerequisite`         | string                             |
| `**Project Source:**`             | `project_source`       | string                             |
| `**Project Roll Characteristic:**`| `roll_characteristic`  | string                             |
| `**Project Goal:**`               | `project_goal`         | string — value is a number ("3000") or "Varies" |

Each field is set only when present. `project_goal` stays a string because the
source value is sometimes non-numeric ("Varies").

### 3. Ancestry purchased-trait point cost (parser-only)

`CleanHeading` currently strips the `(N Point)` / `(N Piety)` suffix
(`costSuffixRe`) and discards it. Add a sibling helper in `helpers.go`:

```go
// extractCostSuffix returns the inner text of a trailing "(… cost …)" suffix,
// e.g. "Barbed Tail (1 Point)" -> "1 Point". Returns "" when absent.
func extractCostSuffix(s string) string
```

`FeatureParser` (`feature.go`, which handles `@type: feature` → `type: trait`,
including the ~69 ancestry purchased traits) sets `cost: "1 Point"` when the
heading carries a suffix. Faithful string; no semantic guessing. Other parsers
are unchanged.

### 4. Annotate the 27 unannotated figures (source `.md` + parser)

Add `<!-- @type: god | @id: <slug> -->` before each genuine deity/saint/hero in
`input/heroes/Draw Steel Heroes.md`. Adjudication from the gods-chapter hierarchy:

**Annotate (27):**

- Heroes of the Elves (patron Val): A Sea of Suns, The Taste of Morning, Ripples
  of Honey on a Shore of Gold, Yllin Dyrvis, Thyll Hylacae, Illwyv li Orchiax
- Heroes of the Dwarves (patron Ord): Zarok the Law-Giver, Valak-koth the Seeker,
  Stakros the Engineer
- Heroes of the Orcs (patron Kul): Khorvath Who Slew a Thousand, Grole the
  One-Handed, Khravila Who Ran Forty Leagues
- Heroes of the Hakaan (patron Kul): Mahsiti the Weaver, Prexaspes the Stargazer,
  Atossa the Shepherd
- Saints of Thellasko: Uryal the Subtle, Kuryalka the False Principle
- Saints of Adûn: Gaed the Confessor, Gryffyn the Stout
- Saints of Cavall: Llewellyn the Valiant, Gwenllian the Fell-Handed
- Saints of Salorna: Draighen the Warden, Eriarwen the Wroth
- Evil Gods (no patron): Nikros the Tyrant, Pentalion the Paladin, Cyrvis, Eseld
  of the Eye

**Leave as unannotated containers / narrative (skip):** the "Heroes of the X",
"Saints of Hell", "Evil Gods", "Devil Gods", "Human Gods of Vasloria", "Space
Gods of the Timescape" group headings; "Lords of Law and Chaos" and "Heralds of
the Space Gods" (narrative paragraphs, no sub-entries); "The Calling of Lady
Magnetar" and "The Calling of Cho'kassa the Time Rider" (story vignettes under
the space gods, not deities).

#### SCC modeling: nested path under patron

- Existing gods stay flat: `god/val`, `god/ord`, `god/kul`, `god/thellasko`,
  `god/adun`, `god/cavall`, `god/salorna`, `god/nebular`, `god/ov`.
- Saints/heroes nest under their patron: `god/val/a-sea-of-suns`,
  `god/thellasko/uryal`, `god/adun/gaed`, etc.
- Evil Gods have no patron god ancestor → flat `god/nikros`, `god/pentalion`,
  `god/cyrvis`, `god/eseld`.

`GodParser` builds `TypePath` by resolving the patron from the **section tree**
(`section.Parent`), not the context stack:

```go
typePath := []string{"god"}
if patron := findParentGodID(section); patron != "" {
    typePath = append(typePath, patron)
    fm["patron"] = patron   // structured field mirrors the path
}
```

where `findParentGodID` walks `section.Parent` upward and returns the `@id` of
the nearest ancestor whose `@type` is `god` (skipping unannotated containers):

```go
func findParentGodID(section *parser.Section) string {
    for p := section.Parent; p != nil; p = p.Parent {
        if p.Type() == "god" {
            return p.ID()
        }
    }
    return ""
}
```

Resulting code form: `mcdm.heroes.v1/god/<patron>/<id>` for saints,
`mcdm.heroes.v1/god/<id>` for top-level gods.

**Why tree-walk, not the context stack:** the context-stack helper
`findAncestorID(ctx, level, "god")` is unreliable here. `ContextStack.Push` only
clears levels at or *deeper* than the pushed level, and unannotated containers
("Evil Gods", "Human Gods of Vasloria") do not push at all. So a previous god's
entry can remain stale at an intermediate level — e.g. Nikros (an Evil God with
no patron) would wrongly inherit `salorna` from the preceding subtree. Walking
the real `section.Parent` chain avoids this: `Parent` is populated by the
document parser (`internal/parser/document.go`) and reflects true nesting.

Once a figure is annotated, `FullBodySource()` excludes it from the parent god's
body (it skips annotated children), so each saint renders as its own page and the
parent god body keeps only its own prose plus the unannotated "Heroes of the X"
intro paragraph.

### 5. Documentation corrections

SCC codes are **not** immutable/permanent — the registry is additive-only
(`ValidateAgainstFrozen` only forbids *removing* existing codes). Correct the
overstated language:

- `steel-etl/README.md` lines ~5, 31, 63, 79, 174 ("frozen", "permanent",
  "immutable once frozen", "cannot be removed").
- Sweep workspace `CLAUDE.md` and `ARCHITECTURE.md` for the same framing.
- Reframe as: codes are stable identifiers; the freeze guard prevents removal /
  renaming of existing codes, but new codes may be added.

## Out of scope

- Per-god domain links to a `domain` SCC type (domains remain plain strings).
- Project goal numeric typing / unit modeling.
- Restructuring the heading hierarchy of the gods chapter.
- Changing or removing the freeze enforcement mechanism itself (docs only).

## Testing

Table-driven unit tests (`go test -race ./...`):

- `god_test.go`: domains list parsed (multi-domain + single); patron set when an
  ancestor god is in context, absent for a top-level god.
- `project_test.go`: all four fields parsed; `project_goal` "Varies" preserved;
  fields omitted when absent.
- `feature_test.go` / `helpers_test.go`: `extractCostSuffix` present ("1 Point",
  "2 Points") and absent; `FeatureParser` sets `cost` only when present.
- Pipeline/classify check: the 27 new `god/...` codes are emitted with the
  expected nested/flat form, and `classify --diff` shows additions only (no
  existing code missing).

## Risks

- **Slug collisions** — long hero names ("Ripples of Honey on a Shore of Gold")
  slugify to long ids; verify uniqueness within each patron namespace.
- **`extractCostSuffix` scope** — the cost regex also matches "(N Piety)"; scope
  the new `cost` field to `FeatureParser` so domain abilities (parsed elsewhere)
  are unaffected.
