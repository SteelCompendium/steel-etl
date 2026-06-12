# Printing Provenance and SCC Code Lifecycle (design)

**Date:** 2026-06-11
**Status:** Direction settled; implementation deferred (see "Decision status" at the end).
This document exists so the reasoning does not have to be rediscovered when MCDM
ships a printing that forces the deferred decisions.

## The incident that prompted this

The Heroes source PDF is errata printing **1.01b** (MCDM has shipped three errata
printings so far, with more expected). To record that, the `book:` frontmatter in
`input/heroes/Draw Steel Heroes.md` was changed from `mcdm.heroes.v1` to
`mcdm.heroes.v1_01b` — and the build "went crazy."

Why: the `book:` field becomes the **source segment** of every SCC code minted
from that document (`internal/pipeline/pipeline.go`). Changing it re-mints all
~1,915 heroes codes into a new namespace, which instantly dangles every
cross-reference link that still points at the old one:

| Where | Dangling links |
|---|---|
| Heroes doc internal links (`scc:mcdm.heroes.v1/...`) | ~17,526 |
| Summoner → Heroes cross-book links | 1,292 |
| Beastheart → Heroes cross-book links | 225 |

…plus every published `/scc/mcdm.heroes.v1/...` permalink and API key. The edit
was reverted; the printing is now recorded as a separate, currently-inert
`printing: "1.01b"` frontmatter field.

## The underlying need (the real requirement)

**Debugging provenance.** When someone reports "there's a typo in X," Scott wants
to take the SCC code / page they're looking at and determine *which source
printing* the data was generated from. That's the requirement — not that the
version literally appear inside the code string.

## Core principle: identity vs. provenance

These are two different needs that pull in opposite directions if conflated:

1. **Identity** — a permanent address for "the fury's Gouge ability." One page,
   one code, never moves, never breaks. This is what the SCC source segment is
   for. *Codes are forever.*
2. **Provenance** — given live data, knowing which printing it came from. This
   is **metadata about a build**, not part of an entity's identity.

The `.v1` in `mcdm.heroes.v1` is the **SCC namespace version**, not the PDF
printing. It only ever bumps for a genuinely breaking redefinition of the book's
content model (think a true Draw Steel 2nd Edition) — never for errata. This is
the same "X-is-not-identity" move as the format axis in SCC scheme v1.1
(`2026-06-09-scc-scheme-versioning-and-format-design.md`): printing, like
format, is not identity.

**Git already snapshots every source version, losslessly and for free.** Every
errata ingest is a commit to `input/heroes/Draw Steel Heroes.md`. Publishing
per-printing page snapshots would re-encode (at large cost) information the repo
history already stores.

## Rejected alternatives (and why)

- **Printing in the source segment** (`mcdm.heroes.v1_01b`) — the incident
  above. Every errata re-mints ~1,915 identities and breaks ~19k links +
  all permalinks/API keys. "Codes are forever, until the next errata" is not
  codes-are-forever.
- **Keep every version's pages live** (snapshot model: `v1_01a/...` and
  `v1_01b/...` pages both exist) — fails on scale (printings × ~1,915
  near-duplicate pages; the v2 site already had to move repos over GitHub
  Pages deploy-size limits) and fails on its own terms: a cross-book link can
  only point at *one* version, which quietly reinvents "one canonical identity"
  while paying for N copies.
- **One git repo per source-doc version** — same content explosion plus repo
  sprawl; duplicates what git history already provides.

## The model

### Provenance (handles errata printings — the common case)

- SCC source segment stays `mcdm.heroes.v1`. Errata = same entities, corrected
  text = same codes. Nothing re-mints, no links move.
- The source doc records its printing in non-identity frontmatter:
  `printing: "1.01b"` (already present, currently ignored by the pipeline).
- **When implemented**, the printing flows as a build stamp: registry →
  published SCC API JSON → rendered pages (e.g. a "Source: Draw Steel Heroes,
  printing 1.01b" line, like the existing footer SHA stamp). Debug workflow:
  code/URL → page or API says which printing + git SHA → `git show` the tagged
  source.
- Tag the input repo per ingested printing (e.g. `heroes-printing-1.01b`) so
  any historical source text is one `git show` away.

### Change taxonomy (what a new printing can do to an entity)

1. **Errata/typo** — same entity, corrected text. Same code; content updates in
   place; the printing stamp identifies the text generation. No design work
   needed.
2. **Balance change, same entity** — Gouge is nerfed but still Gouge. Same
   identity (the code names the entity, not its numbers). Same as (1).
3. **Removal/replacement** — Gouge is deleted; a new ability takes its slot.
   This needs the lifecycle model below.

### Code lifecycle (handles removals — not yet needed)

When an entity is removed or replaced:

- The replacement is a **new entity → new code**. Codes are never reused.
- The removed code **never 404s** — it becomes a **tombstone** carrying registry
  lifecycle metadata: `status: removed`, `removed_in: <printing>`, optional
  `superseded_by: <code>`. The page says "removed in printing 1.1, replaced by
  X" with a link. (Standard identifier-system practice: RFC obsoleted-by, DOIs
  never reassigned.)
- This scales with *removals* (rare, a handful per printing at most), not with
  printings × all codes. Cross-book links to a removed code land on a tombstone
  that points the reader onward — strictly better than a 404 or a re-minted
  namespace.

## Deferred decision: where tombstone content lives

Both options are viable; the choice depends on how MCDM actually handles
removals, which is unknown. **Documented here so the trade-offs don't have to be
rediscovered.**

**Option A — annotated retention in the source doc.** Removed entities stay in
`Draw Steel Heroes.md`, marked (e.g. `@removed-in: 1.1` annotation or a
"Removed Content" appendix). The pipeline renders a tombstone, optionally
preserving the last-known rules text ("as of printing 1.01b, Gouge read: …").
*Pro:* the source doc remains the single source of truth; tombstones are
self-contained and debuggable on-site. *Con:* the source doc slowly accretes a
graveyard.

**Option B — registry-only tombstones.** Removed entities are deleted from the
source; the registry persists the code with lifecycle metadata and the site
renders a minimal marker page from that. *Pro:* clean source doc. *Con:* old
text only recoverable via git tag; the registry becomes a second owner of
content-ish data. Also interacts with the registry rebuild behavior — `gen
--all` resets and re-derives the registry from current inputs
(`resetRegistryForRebuild`), so registry-persisted tombstones would need to
survive that reset (a freeze-like carve-out).

**Leaning (2026-06-11):** Option A, for the single-source-of-truth invariant.
Not decided.

**Decision triggers — revisit this doc when any of these happen:**
- MCDM ships a printing that *removes or replaces* an entity (forces the A/B
  choice).
- MCDM announces a true new edition ("v2") — that's the *breaking* branch:
  evaluate a new source segment (`mcdm.heroes.v2`) and/or SCC scheme bump
  (`scc.v2:`), which the v1.1 scheme-versioning machinery already anticipates
  (the resolver refuses to bind non-current scheme versions).

## Decision status

| Piece | Status |
|---|---|
| Identity/provenance separation; source segment never carries the printing | **Settled** (this doc) |
| `printing:` frontmatter field in the heroes source | **Done** (now consumed by the pipeline) |
| Printing stamp wired through registry → API → rendered pages; per-printing git tags | **Done** (2026-06-11) — plan: `docs/superpowers/plans/2026-06-11-printing-provenance-stamp.md` |
| Tombstone/lifecycle implementation, Option A vs B | **Deferred until a decision trigger fires** |
