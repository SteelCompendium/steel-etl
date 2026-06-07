# Link Audit + Fury "Stormwight Kits" Grouping — Record

> **Status:** COMPLETE (2026-06-06). Two linking rounds + one classifier/structural fix.
> Pipeline `gen --config pipeline.yaml` 0 WARN (1807 classified); `go test ./...` green;
> site rebuilds clean (2587 SCC stubs). SCC links in the heroes doc ~4,055 → ~4,595 (+540).
> Source committed & pushed (steel-etl `3bf359f`, workspace `5ab2176`). Remaining tail
> tracked in workspace `FOLLOWUPS.md` #6.

**Trigger:** User reported, on the v2 site, (a) wrongly/missing links — e.g. the "Rapid-Fire"
kit unlinked in the kit table, "I'm No Threat" unlinked, the "Disengage" move action
unlinked, "Domain Piety and Effects" appearing truncated to "Domain" in the censor Templar
trait — and (b) the **fury trait index not picking up the new card styling**.

## Part A — Fury "Stormwight Kits" structural fix (root cause: bad heading level + annotation)

**Symptom:** the fury trait index (`Browse/feature/trait/fury/`) had 8 ungrouped "dangling"
leaf entries; fury was the *only* class with this (every other class: 0 dangling top-level
trait files).

**Root cause:** `### Stormwight Kits` was an H3 tagged `<!-- @type: feature -->`. A plain
`feature` contributes **no path segment**, so its framework sub-features (Kit Features, Aspect
Benefits and Animal Form, Aspect of the Wild, Primordial Storm, Equipment, Kit Bonuses,
Growing Ferocity) got flat codes `feature.trait.fury/<id>` and landed directly in the class
root instead of a sub-folder — so the index had nothing to collapse them under.

**Fix:**
1. **Classifier** (`internal/content/feature.go`, `internal/content/ability.go`): a **named
   feature-group under a class/ancestry** now contributes its `@id` as a path segment, when
   there is no level context and no nearer kit. This mirrors the pre-existing behavior for
   the `common.*` branch (move-actions/maneuvers) and is gated so level feature-groups (which
   carry `@level`, not `@id`) and kit-scoped features (Boren/Corven/…) are unaffected.
2. **Input:** `### Stormwight Kits` → `<!-- @type: feature-group | @id: stormwight-kits -->`.
   The 7 framework features now classify as `feature.trait.fury.stormwight-kits/<id>` (and the
   Aspect of the Wild ability as `feature.ability.fury.stormwight-kits/aspect-of-the-wild`) →
   render in the index as one **"Stormwight Kits" folder/collapsible**, consistent with every
   other class. No more danglers.
3. **Inbound links redirected** (the old `feature.trait.fury/stormwight-kits` container has no
   page): `…/stormwight-kits` (×7) → `…stormwight-kits/kit-features`; `…/primordial-storm`
   (×5) → `…stormwight-kits/primordial-storm`; `feature.ability.fury/aspect-of-the-wild` (×3)
   → `…stormwight-kits/aspect-of-the-wild`. Also unlinked one false positive (`[kit features]`
   where "features" was a verb) and one self-link.

Note: the per-page `.sc-trait` card styling was never broken — only the *index grouping* was;
the user's "not picking up styling" was the dangling ungrouped list looking inconsistent.

## Part B — Link audit (two rounds)

Tooling added (reusable, `steel-etl/scripts/`):
- `link_audit.py` — whole-doc report of unlinked occurrences + truncation candidates, from
  `classification.json` + generated md frontmatter (authoritative display names), with
  hyphen↔space / plural / possessive variants.
- `link_audit_category.py <code-substr…>` — per-category report (e.g. `kit/`, `common.`).
- `link_apply.py '<regex w/ group 1>' '<code>' [start-end excl…] [--apply]` — safe single-rule
  linker: skips headings (incl. `>` blockquote headings), existing links, and comments;
  optional line-range exclusion for own-section; dry-run by default.

**Key finding:** entire **combat-mechanic** categories (`feature.{trait,ability}.common.*`
move actions / maneuvers / free strikes) were **absent from `linking-reference.md` and never
linked**, alongside the reported one-offs.

Linked (unambiguous, per-instance verified or high-precision-guarded):
- **Kits:** Rapid-Fire (hyphen variant — reference says "Rapid Fire", doc writes "Rapid-Fire",
  so it was skipped in the otherwise-complete kit table) + 2 Sniper cross-refs.
- **Move actions:** Advance / Disengage / Ride (guarded on a following "move action" + the
  Advance↔Disengage pairing; the kit "Disengage Bonus" stat label correctly excluded).
- **Maneuvers:** Catch Breath, Escape Grab, Aid Attack, Search for Hidden Creatures, Use
  Consumable, Stand Up, and the common-verb Hide/Grab/Knockback/Charge/Heal/Defend via
  high-precision guards (followed by "maneuver"/"main action", Grab↔Knockback pairings, and
  the glossary `**X Maneuver:**` / `**X Main Action:**` entries).
- **Free Strike:** ~138 generic refs → `common.main-actions/free-strike` (excludes the "Weapon
  Free Strike" ability names) + the Ranged Weapon Free Strike prose ref.
- **Abilities:** I'm No Threat (shadow), Strike Now (tactician) cross-refs.
- **Gods:** Cavall, Salorna, Adûn, Nebular, Thellasko (own-section excluded). Short names
  (Val/Ord/Kul/OV) left per the guide.
- **Truncation:** Templar `Domain Piety and Effects` → full conduit entity.

## Deliberately deferred (FOLLOWUPS #6)

The remaining tail is low-yield / ambiguity-heavy, *not* safely scriptable:
- Lowercase/mundane maneuver phrasings (~half the leftover Hide/Grab/Charge are genuinely
  mundane: "grab two dice", "in charge of").
- Full distinctive-ability cross-ref sweep — blocked by generic per-class terms
  (`Triggered Action` ×159, `Signature Ability` ×69 each map to one class but every class has
  its own), **identically-named cross-class domain features** (e.g. Invocation of the Heart for
  both censor and conduit), lowercase keyword/stat uses (`corruption immunity N`), and many
  1-occ self-section mentions. Needs **section-aware own-section exclusion + per-class
  disambiguation**, not the flat regex linker.
- `linking-reference.md` sync — the combat-mechanic section was added here (Combat Actions &
  Maneuvers); a fuller per-class ability index is still out of scope.

## Lessons reinforced

- **Always check hyphen↔space (and plural/possessive) variants** of a term before assuming it
  is linked — a single hyphen mismatch left "Rapid-Fire" the lone unlinked row in a complete
  table.
- A heading's **level + annotation type drives its SCC path segment**; an H3 `@type: feature`
  under a class silently flattens its children. Use `@type: feature-group | @id: …` to group.
