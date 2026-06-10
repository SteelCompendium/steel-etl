# SCC Scheme Versioning & Format Negotiation — Design

**Date:** 2026-06-09
**Status:** Design (approved, pending implementation plan)
**Touches:** `reference/scc-specification.md`, `classification.json` registry, `steel-etl` SCC emitter/resolver, v2 website SCC routing

## 1. Problem

SCC codes are a permanent, "semi-readable GUID" the community can use to reference and fetch
Draw Steel content. Two gaps threaten that contract long-term:

1. **No scheme version.** The classification *grammar* (delimiters, type nesting, source format)
   is intended to freeze forever, but if a breaking change is ever truly needed — a misjudged
   structure, or a wholesale re-mint — old and new codes would be **indistinguishable strings**.
   A consumer who cached `mcdm.heroes.v1/class/fury` under v1 rules has no way to know a later
   string was minted under different rules. We want insurance against grammar changes, frozen-code
   re-pointing, and a full SCC v2 re-mint — without those scenarios silently corrupting caches.

2. **No format axis.** The same entity exists in multiple representations (markdown, json, yaml,
   dse-markdown = markdown with embedded yaml). The current format has no way to say "the json view
   of this entity," and naively bolting format onto the code would pollute the cache key — third-party
   tools using SCC as a primary key would treat `…/fury` and `…/fury.json` as different entities when
   they are the same thing.

### Use cases that shape the design

- **Human dialogue:** *"I tried downloading `scc:foo/bar/baz` and it has a typo."* — names the entity.
- **Builder autocomplete:** *"at level 5 I'll add `scc:heroes/feature.fury.level-5/gouge`."* — names the entity.
- **Rule-logic DSL:** `choice(1, {scc:foo/bar/baz, scc:foo/bar/nargles})` — entity as a stable key.
- **Representation comparison:** *"in the `:markdown` view the 2nd effect is nested, but in `:json` it's flattened."* — names a *representation* of one entity.
- **SDK inline-expander (lowest priority):** a tool scans text for SCC codes and fetches+inlines the
  content, possibly mixing formats per occurrence. Needs the *reference* to optionally carry its format.

In every case the **identity is format-free**; format, when it appears, rides on the *reference* and
must normalize back to that identity.

## 2. The three axes

The design separates three concerns that the original spec conflated or omitted. **Naming them
distinctly is itself a goal** — there are three different "versions" in play (see §2.1).

| Axis | Answers | Where it lives | In the cache key / identity? |
|---|---|---|---|
| **Identity** | *which entity* | `source/type/item` (frozen, immutable) | **Yes — it IS the key** |
| **Scheme version** | *under which grammar was this minted* | `scc.vN` prefix | **Yes — part of the name** |
| **Format** | *which representation* | fetch-time qualifier, normalizes out | **No — selects a `(key, format)` variant** |

Scheme version is part of identity because a hypothetical `scc.v2:.../fury` may be minted under
different rules and need not denote the same thing as `scc.v1:.../fury`. Format is *not* part of
identity because every representation denotes the same entity.

### 2.1 Three "versions", disambiguated

To prevent the exact confusion this design guards against, the spec must name these separately:

| Name | Token | Meaning | Bumps when |
|---|---|---|---|
| **Book edition** | `v1` inside `mcdm.heroes.v1` | the publisher's edition of the *content* | MCDM reprints with rules changes |
| **Registry-file schema** | `version: 1` in `classification.json` | the JSON shape of the registry file | the registry file format changes |
| **SCC scheme version** | `scc.v1` prefix | the *grammar* for constructing codes | a breaking grammar change / re-mint |

These are orthogonal. The scheme prefix (`scc.v1`) and the book edition (`…heroes.v1`) never collide
because one is the leading prefix and the other lives inside the `source` component.

## 3. Scheme version

### 3.1 Notation — `scc.vN` swaps in wherever the `scc` namespace token already appears

```
Link / URN / DSL:     scc.v1:mcdm.heroes.v1/class/fury
Website URL:          steelcompendium.io/scc.v1/mcdm.heroes.v1/class/fury
Bare (implicit v1):   scc:mcdm.heroes.v1/class/fury        ≡  scc.v1:…
Future breaking ver:  scc.v2:mcdm.heroes.v1/class/fury     (coexists with v1)
```

- **Canonical form is explicit `scc.v1`.** Tooling SHOULD emit explicit.
- **Bare `scc:` (and the bare `/scc/` route) is a permanent implicit-v1 alias.** `scc:X` ≡ `scc.v1:X`.
  This is why no existing code, link, URL, or bookmark breaks — they are all valid implicit v1.
  (Mirrors `Accept-Language`: omittable, but explicit removes ambiguity.)

### 3.2 What bumps the scheme version

A `scc.vN` bump is reserved for **breaking grammar changes** only:

- a delimiter's meaning changes,
- the type-nesting rules change incompatibly,
- the `source` component format changes,
- a wholesale re-mint of the code-set.

**Additive** taxonomy work (new types, new nesting, new groups) stays within the current scheme
version under the existing freeze + alias rules — it does **not** bump `scc.vN`. The many taxonomy
migrations already done (skills nesting, treasure hierarchy, group-landing migration, etc.) would all
have been v1-internal.

### 3.3 Coexistence & registry

- When `scc.v2` ever appears, **v1 is untouched**: every v1 code keeps resolving forever. The two
  code-sets coexist exactly as `heroes.v1` and a future `heroes.v2` would.
- The registry gains a `scheme_version` field (initially `1`), distinct from the existing registry-file
  `version` field. A future v2 re-mint is a **separate registry partition**, not edits to the v1 registry.
- The `scc:`-protocol resolver treats a missing prefix as `scc.v1`.

## 4. Format negotiation

### 4.1 Identity is format-free; format is selected at fetch time

- **Over HTTP** (the website / API): the canonical mechanism is the **`Accept` header**
  (`Accept: application/json`). A **`?format=json` query param** is a convenience for curl/browser
  users (lives in the URL query, never in the path/identity). **No file extension** — `…/fury.json`
  is explicitly rejected; it reads as a filename and pollutes identity.
- **In a builder/DSL:** the tool holds the bare `scc:…` and asks its data source for whatever
  representation it needs. Format is the tool's concern, not the reference's.

Cache semantics: the **primary key is the full name including scheme version**
(`scc.v1:source/type/item`). Representations are stored as `(key, format)` variants under that one
key — exactly how an HTTP cache keeps variants via `Vary: Accept`. Selecting a format never produces
a new key.

### 4.2 Reserved (not built): the `#format` reference qualifier

For the non-HTTP cases (dialogue, DSL, the SDK inline-expander) a reference MAY *optionally carry* a
format qualifier. **We reserve the syntax now but do not build it** — YAGNI. Reserving costs nothing
and keeps us un-cornered.

- **Delimiter: `#`** (URI-fragment style). The web already treats a fragment as "a view of the same
  resource" and strips it from the identity sent to the server, so tooling normalizes it out for free.
- **Shape:** `scc.v1:mcdm.heroes.v1/class/fury#json`.
- **Normalization rule:** strip from `#` onward → canonical identity. `…/fury#json` and `…/fury#yaml`
  and `…/fury` are the **same cache key**; the qualifier only selects which stored variant.
- The `/`, `.`, and `:` delimiters are claimed by path / scheme-prefix / URN separators respectively;
  `#` is the one punctuation the grammar promises never to claim for identity.

The format token vocabulary (initially `markdown`, `json`, `yaml`, `dse-markdown`) is an open,
extensible list defined when the qualifier is actually built.

## 5. Backwards compatibility & migration

- **Nothing breaks today.** Bare `scc:` / `/scc/` = implicit v1, so all ~2,734 registry codes,
  ~17,527 in-prose links, URLs, and bookmarks remain valid unchanged.
- **Restamp is deferred.** Rewriting existing bare `scc:` links to explicit `scc.v1:` across the
  source inputs is a mechanical but high-churn sweep. It is intentionally **not** part of this work;
  it will be done in a later pass over all inputs (a new sourcebook is in flight on another agent).
  Tracked as **FOLLOWUPS #8**.
- Going forward, newly emitted links and the registry SHOULD use explicit `scc.v1`, but bare remains
  forever valid.

## 6. Scope of implementation (for the plan)

In scope:

1. Update `reference/scc-specification.md`: bump to spec **v1.1** (additive — scheme version + format
   negotiation), add the three-axis model, the three-versions disambiguation, `scc.vN` notation, the
   `#format` reservation, and HTTP format negotiation. The frozen *code-set* is unchanged.
2. Add `scheme_version: 1` to the registry (`classification.json`) and its schema.
3. `steel-etl` resolver/parser: accept and normalize the `scc.vN` prefix (bare ⇒ v1); strip a `#…`
   qualifier to canonical when present (recognize-and-normalize only; no per-format fetch yet).
4. v2 website SCC routing: serve `/scc.v1/…` as an explicit alias of the existing bare `/scc/…` route.
5. HTTP format negotiation entry point (`Accept` / `?format=`) — design the contract; build scope TBD
   with the plan (may be deferred if the API surface isn't ready).

Out of scope (deferred / future):

- The `#format` qualifier implementation and per-format content emission for representations not
  already produced.
- The mass restamp of bare → explicit `scc.v1` (FOLLOWUPS #8).
- Any actual `scc.v2` re-mint.

## 7. Open questions for the plan

- Does the live website route on bare `/scc/…` or bare `/{code}` today? (Spec §3 vs. CLAUDE.md's
  `/scc/{code}/` disagree.) The plan must reconcile this before adding the `/scc.v1/` alias.
- Which representations does the API actually emit today, and is the `Accept`/`?format=` entry point
  worth building now or deferring with the qualifier?
