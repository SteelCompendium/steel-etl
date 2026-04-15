# Translation Guide

This guide explains how to contribute translations to the Steel Compendium.

## Overview

Translated content goes through the same pipeline as English content. You provide an annotated markdown file with the text translated, and `steel-etl` produces all output formats automatically.

## Getting Started

### 1. Generate a translation template

```bash
steel-etl strip --for-translation -o template.md input/heroes/Draw\ Steel\ Heroes.md
```

This creates a copy of the English source with a guide header explaining what to translate and what to preserve.

### 2. Translate the content

Open `template.md` and translate the human-readable text:

- Headings (`# Chapter Title`, `## Class Name`, etc.)
- Paragraphs and body text
- Table cells (the text, not the `|` formatting)
- Flavor text and effect descriptions
- Ability descriptions

### What NOT to translate

- **Annotations**: `<!-- @type: ability | @cost: 3 Ferocity -->` — leave these exactly as-is
- **YAML frontmatter**: The `---` block at the top of the file
- **Markdown formatting**: `**bold**`, `## heading`, `| table |`, `- list`
- **SCC codes**: `scc:mcdm.heroes.v1/feature.ability.fury.level-1/gouge`
- **Annotation keys**: `@type`, `@id`, `@cost`, `@level`, etc.

### Game terms

Work with your localization team to decide which game terms get translated:

- **Always translate**: Flavor text, descriptions, rules explanations
- **Team decision**: Class names, ability names, ancestry names — these may have official translations
- **Never translate**: Annotation values (`fury`, `gouge`, `ability`), SCC codes

### 3. Save the translated file

Place the completed translation at:

```
input/i18n/{locale}/Draw Steel Heroes.md
```

For example, Spanish goes to:

```
input/i18n/es/Draw Steel Heroes.md
```

### 4. Run the pipeline

```bash
steel-etl gen --locale es
```

This produces output at `{base_dir}/es/md/`, `{base_dir}/es/json/`, etc.

### 5. Verify output

Check a few output files to make sure:

- Content is translated
- Frontmatter is correct (SCC codes, type fields)
- File structure matches the English output
- No annotation fragments leaked into the output

## File Structure

```
input/
├── heroes/
│   └── Draw Steel Heroes.md          # English source (annotated)
└── i18n/
    ├── es/
    │   └── Draw Steel Heroes.md      # Spanish translation
    ├── pt-br/
    │   └── Draw Steel Heroes.md      # Brazilian Portuguese
    └── {locale}/
        └── Draw Steel Heroes.md      # Your translation
```

## Locale Codes

Use standard IETF language tags:

| Code | Language |
|------|----------|
| `es` | Spanish |
| `pt-br` | Brazilian Portuguese |
| `fr` | French |
| `de` | German |
| `ja` | Japanese |
| `ko` | Korean |
| `zh-hans` | Simplified Chinese |
| `zh-hant` | Traditional Chinese |

## How It Works

The pipeline uses the same annotations and content parsers for all locales. The SCC classification codes are **shared across locales** — the same ability gets the same SCC code regardless of language. This enables:

- Cross-locale linking (the website language switcher)
- Consistent API responses in any language
- Translation progress tracking (compare locale file sections against English)

## Tips

- **Start small**: Translate one class end-to-end (e.g., Fury) before doing the full book. Run the pipeline to verify.
- **Preserve structure**: The pipeline relies on heading levels and annotation positions. If you add or remove headings, parsing will break.
- **Power roll tables**: Keep the `| 11 or lower | 12-16 | 17+ |` header row structure. Translate the cell content only.
- **Ask questions**: If you're unsure whether something should be translated, leave it in English and flag it for review.
