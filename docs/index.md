# steel-etl docs

Deep references for steel-etl subsystems, one topic per file. The router/overview is
[`CLAUDE.md`](../CLAUDE.md); workspace-level docs (SCC change log, follow-up/roadmap
archives) live in the workspace `docs/`.

- [`site-builder.md`](site-builder.md) — per-file mechanics of `internal/site/`
  (cards, bestiary, ability/trait/statblock renderers, link rewriting, group-landing
  relocation)
- [`statblocks.md`](statblocks.md) — Monsters & Summoner statblock machinery: H7/H9
  deep headings, SCC hierarchy, code≠path hoist, power-roll forms, parser hardening
- [`linking-guide.md`](linking-guide.md) — rules for adding SCC cross-reference links
  to source docs (with dated convention notes per sweep)
- [`linking-reference.md`](linking-reference.md) — all linkable terms with display
  names, variants, and SCC codes
- [`rule-term-mapping.md`](rule-term-mapping.md) — every rules/glossary term's
  decision (new-rule / reuse-existing / skip) from the `rule.*` type introduction
- [`summoner-linking-reference.md`](summoner-linking-reference.md) — Summoner-book
  linking reference
- [`card-data-parity.md`](card-data-parity.md) — checklist for promoting
  card-surfaced fields into frontmatter + both schema copies
- [`superpowers/`](superpowers/) — per-effort plans/specs/prompts from skill-driven
  sessions (dated; each plan carries its own `## Status`)
