# Card ⇄ Data Field Parity

**Rule:** When a parser is upgraded to surface a field on an index card (or any
other card / page), that field MUST also be promoted into the structured data
formats — unless it is purely presentational (truncation, icon choice, layout).

The index/preview cards (`internal/site/cards.go`, `feature_index.go`,
`ability_cards.go`, `trait_cards.go`) are *site-only* and may read either
frontmatter or the page body. The **page body is not a data contract** — only
frontmatter flows into the JSON/YAML outputs (via `transformPassthrough` /
`copyFrontmatter`) and is governed by the schemas. So a card that scrapes a
value out of the body is a parity gap: the website shows data the data repos
lack.

## Checklist for adding a card-surfaced field

1. **Extract once, in the parser.** Add `fm["<field>"] = …` in the relevant
   `internal/content/<type>.go` parser so the value lands in frontmatter and
   flows automatically into JSON/YAML. Share extraction helpers (e.g.
   `firstFlavorParagraph` in `internal/content/flavor.go`) so the card and the
   data never diverge.
2. **Declare it in BOTH schema copies.** `steel-etl/schemas/<type>.schema.json`
   AND `data-sdk-npm/src/schema/<type>.schema.json`. They are hand-synced and
   use `unevaluatedProperties: false`, so an undeclared field is invalid.
   Verify with `diff -q` between the two copies, and make sure the declared
   `type` matches what the parser actually emits (e.g. `project_goal` is emitted
   as a string, so the schema accepts `["integer", "string"]`).
3. **Update the allowlist test.** Add the key to `schemaAllowedFields` in
   `internal/output/schema_validation_test.go` and exercise it in
   `TestSchema_NoUnevaluatedProperties` (these tests validate field *names*
   against the allowlist, not values against the JSON schema files).
4. **Make the card read the frontmatter field** (with a body fallback only as a
   safety net — e.g. `cardFlavor(fm, body)`), so the parser is the single source
   of truth.
5. **The data-sdk-npm consumer side is its own effort.** Adding the field to the
   TS SDK (DTOs, model classes, markdown/json/yaml readers+writers, tests) is
   tracked separately — see
   `docs/superpowers/plans/2026-06-08-card-data-field-parity-sdk.md`.

## Precedent

The first sweep (2026-06-08) promoted `flavor` for every card type (class,
ancestry, career, culture, title, complication, perk, kit) plus treasure
`project_goal` / `project_roll_characteristic` / `echelon`. See
`docs/superpowers/plans/2026-06-08-card-data-field-parity.md`.
