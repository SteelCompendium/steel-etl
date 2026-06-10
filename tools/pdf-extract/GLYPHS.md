# DrawSteelGlyphs / Wingdings glyph map

The Summoner PDF sets icons (action types, power-roll tiers, characteristics,
potency) in two custom fonts — `DrawSteelGlyphs-Regular` and `Wingdings-Regular`
— which encode those icons onto **ordinary ASCII codepoints**. A font-blind text
extractor would silently turn an icon into a letter and glue it into prose, so
`extract.py` tags every glyph-font character as `⟦g:0xNN⟧` (DrawSteelGlyphs) or
`⟦w:0xNN⟧` (Wingdings) instead.

`glyphs.json` maps each codepoint to the text it resolves to. That resolution is
used in **two** places, kept in lock-step:

1. **The fidelity wordbag** (`resolve.py` → `build_wordbag.py`): characteristic
   icons join their adjacent word, potency/tier markers become their text, and
   decorative glyphs are dropped.
2. **The markdown the converter writes**: same convention, so the fidelity gate
   stays exact. A mis-mapped glyph makes the slice/chunk fidelity check fail —
   the map is self-validating.

Verified visually against `The Summoner v1.0b` (pages 26–27 rendered) on
2026-06-09.

## The map

| Codepoint(s) | Resolves to | Meaning |
|---|---|---|
| `g:0x4d` / `g:0x6d` | `M` | **Might** characteristic icon (joins word → "Might"; alone → "M") |
| `g:0x41` / `g:0x61` | `A` | **Agility** |
| `g:0x52` / `g:0x72` | `R` | **Reason** |
| `g:0x49` / `g:0x69` | `I` | **Intuition** |
| `g:0x50` / `g:0x70` | `P` | **Presence** |
| `g:0x3c` | `<` | potency comparator |
| `g:0x77` | `weak` | potency strength (tier ≤11) |
| `g:0x76` | `average` | potency strength (tier 12-16) |
| `g:0x73` | `strong` | potency strength (tier 17+) |
| `g:0xe1` | `≤11` | power-roll tier 1 |
| `g:0xe9` | `12-16` | power-roll tier 2 |
| `g:0xed` | `17+` | power-roll tier 3 |
| `g:0x31`–`g:0x35` | `1`–`5` | numeric potency value (book prints numbers for some abilities) |
| `g:0xa2` | *(dropped)* | ability/keyword chevron decoration (×215) |
| `g:0xa3` | *(dropped)* | decoration |
| `g:0xa5` | *(dropped)* | charge mark (precedes "N charge:") |
| `g:0xae` | *(dropped)* | small-caps section ornament (e.g. before "Summoner Advancement") |
| `g:0x5d` | *(dropped)* | badge end-cap |
| `w:0x8a` | *(dropped)* | Wingdings ★ bullet |

## Conventions

- **Potency** renders as `<CHAR> < <STRENGTH|VALUE>` to match Draw Steel house
  style already used in heroes/beastheart (`M < AVERAGE`, `A < STRONG`). Use the
  **strength word** where the page shows WEAK/AVERAGE/STRONG and the **number**
  where the page shows a digit — the values are intentionally different per
  ability; render exactly what the page prints.
- **Power-roll tiers** render as the markdown list beastheart uses:
  `- **≤11:** …`, `- **12-16:** …`, `- **17+:** …`.
- **Distance / target icons** (📏 / 🎯) are NOT in these fonts — they are images
  or whitespace-width marks and do not appear in the extraction. The converter
  adds `📏`/`🎯` from the page image, matching the beastheart statblock table.
  They are not words, so they do not affect fidelity.

## Revisit notes

⚠️ **Decorative glyphs may carry meaning.** Per the project owner (2026-06-09),
some of the "dropped" glyphs (`0xa2`, `0xa3`, `0xa5`, `0xae`, `0x5d`, `w:0x8a`)
might actually be statblock ability-type icons that convey information rather than
pure ornamentation. They are dropped for now because they are not prose *words*
and dropping them does not affect word-for-word fidelity. If a statblock turns out
to need one of these to convey meaning (e.g. an action-type icon with no text
label), revisit the relevant entry here and give it a resolution.

If a future supplement uses a glyph codepoint not in this map, `resolve.py` drops
it by default (treats it as decorative). Re-run `extract.py` and inspect
`out/<book>/glyphs-found.json` for any new codepoints, then add them here.
