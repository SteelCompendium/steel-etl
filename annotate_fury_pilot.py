#!/usr/bin/env python3
"""
Annotate the Fury section of Draw Steel Heroes.md as a pilot for the
steel-etl annotation scheme.

This script:
1. Copies the source markdown
2. Adds YAML frontmatter at the top
3. Inserts annotation comments before headings in the Fury section
4. Writes the annotated file to steel-etl/input/heroes/

Annotations are defined as (original_line_content, annotation_text) tuples.
The script matches lines exactly and inserts the annotation on the line before.
"""

import re
import sys
from pathlib import Path

SRC = Path(__file__).parent.parent / "data-gen" / "input" / "heroes" / "Draw Steel Heroes.md"
DST = Path(__file__).parent / "input" / "heroes" / "Draw Steel Heroes.md"

FRONTMATTER = """\
---
book: mcdm.heroes.v1
source: MCDM
title: Draw Steel Heroes
---

"""

# Each entry: (line_number_in_original, annotation_lines_to_insert_before_it)
# Line numbers are 1-based from the original file.
# We'll insert the annotation on a blank line before the heading.
ANNOTATIONS = [
    # === Fury class ===
    (8842, "<!-- @type: class | @id: fury -->"),

    # === 1st-Level Features ===
    (8889, "<!-- @type: feature-group | @level: 1 -->"),
    (8893, "<!-- @type: feature -->"),  # Primordial Aspect
    (8903, "<!-- @type: feature -->"),  # Ferocity
    (8925, "<!-- @type: feature -->"),  # Growing Ferocity
    (8953, "<!-- @type: feature -->"),  # 1st-Level Aspect Features
    (8965, "<!-- @type: feature -->"),  # Beast Shape
    (8969, "<!-- @type: feature | @id: kit-feature -->"),  # Kit (generic name)
    (8973, "<!-- @type: feature -->"),  # Primordial Cunning
    (8979, "<!-- @type: feature -->"),  # Primordial Strength
    (8985, "<!-- @type: feature -->"),  # Relentless Hunter
    (8989, "<!-- @type: feature -->"),  # Aspect Triggered Action

    # Triggered action abilities (blockquoted)
    (9001, "<!-- @type: ability | @subtype: triggered -->"),  # Furious Change
    (9015, "<!-- @type: ability | @subtype: triggered -->"),  # Lines of Force
    (9029, "<!-- @type: ability | @subtype: triggered -->"),  # Unearthly Reflexes

    (9043, "<!-- @type: feature -->"),  # Mighty Leaps

    # Signature abilities
    (9055, "<!-- @type: ability | @subtype: signature -->"),  # Brutal Slam
    (9069, "<!-- @type: ability | @subtype: signature -->"),  # Hit and Run
    (9085, "<!-- @type: ability | @subtype: signature | @id: impaled -->"),  # Impaled!
    (9099, "<!-- @type: ability | @subtype: signature | @id: to-the-death -->"),  # To the Death!

    # 3-Ferocity abilities
    (9123, "<!-- @type: ability | @cost: 3 Ferocity | @id: back -->"),  # Back!
    (9137, "<!-- @type: ability | @cost: 3 Ferocity | @id: out-of-the-way -->"),  # Out of the Way!
    (9153, "<!-- @type: ability | @cost: 3 Ferocity -->"),  # Tide of Death
    (9171, "<!-- @type: ability | @cost: 3 Ferocity | @id: your-entrails-are-your-extrails -->"),  # Your Entrails...

    # 5-Ferocity abilities
    (9191, "<!-- @type: ability | @cost: 5 Ferocity | @id: blood-for-blood -->"),  # Blood for Blood!
    (9207, "<!-- @type: ability | @cost: 5 Ferocity | @id: make-peace-with-your-god -->"),  # Make Peace With Your God!
    (9217, "<!-- @type: ability | @cost: 5 Ferocity -->"),  # Thunder Roar
    (9233, "<!-- @type: ability | @cost: 5 Ferocity -->"),  # To the Uttermost End

    # === 2nd-Level Features ===
    (9249, "<!-- @type: feature-group | @level: 2 -->"),
    (9253, "<!-- @type: feature -->"),  # Perk
    (9257, "<!-- @type: feature -->"),  # 2nd-Level Aspect Feature
    (9269, "<!-- @type: feature -->"),  # Inescapable Wrath
    (9273, "<!-- @type: feature -->"),  # Tooth and Claw
    (9277, "<!-- @type: feature -->"),  # Unstoppable Force

    # 2nd-Level Aspect Abilities (5 Ferocity each)
    (9289, "<!-- @type: ability | @cost: 5 Ferocity -->"),  # Special Delivery
    (9299, "<!-- @type: ability | @cost: 5 Ferocity -->"),  # Wrecking Ball
    (9321, "<!-- @type: ability | @cost: 5 Ferocity | @id: death-death -->"),  # Death... Death!
    (9335, "<!-- @type: ability | @cost: 5 Ferocity -->"),  # Phalanx-Breaker
    (9355, "<!-- @type: ability | @cost: 5 Ferocity -->"),  # Apex Predator
    (9371, "<!-- @type: ability | @cost: 5 Ferocity -->"),  # Visceral Roar

    # === 3rd-Level Features ===
    (9387, "<!-- @type: feature-group | @level: 3 -->"),
    (9391, "<!-- @type: feature -->"),  # 3rd-Level Aspect Feature
    (9403, "<!-- @type: feature -->"),  # Immovable Object
    (9409, "<!-- @type: feature -->"),  # Nature's Knight
    (9415, "<!-- @type: feature -->"),  # See Through Their Tricks

    # 7-Ferocity abilities
    (9423, "<!-- @type: ability | @cost: 7 Ferocity -->"),  # Demon Unleashed
    (9433, "<!-- @type: ability | @cost: 7 Ferocity | @id: face-the-storm -->"),  # Face the Storm!
    (9443, "<!-- @type: ability | @cost: 7 Ferocity -->"),  # Steelbreaker
    (9453, "<!-- @type: ability | @cost: 7 Ferocity -->"),  # You Are Already Dead

    # === 4th-Level Features ===
    (9463, "<!-- @type: feature-group | @level: 4 -->"),
    (9467, "<!-- @type: feature -->"),  # Characteristic Increase
    (9471, "<!-- @type: feature -->"),  # Damaging Ferocity
    (9475, "<!-- @type: feature -->"),  # Growing Ferocity Improvement
    (9479, "<!-- @type: feature -->"),  # Perk
    (9483, "<!-- @type: feature -->"),  # Primordial Attunement
    (9487, "<!-- @type: feature -->"),  # Primordial Strike
    (9491, "<!-- @type: feature -->"),  # Skill

    # === 5th-Level Features ===
    (9495, "<!-- @type: feature-group | @level: 5 -->"),
    (9499, "<!-- @type: feature -->"),  # 5th-Level Aspect Feature
    (9511, "<!-- @type: feature -->"),  # Bounder
    (9515, "<!-- @type: feature -->"),  # Stormborn
    (9519, "<!-- @type: feature -->"),  # Unfettered

    # 9-Ferocity abilities
    (9527, "<!-- @type: ability | @cost: 9 Ferocity -->"),  # Debilitating Strike
    (9543, "<!-- @type: ability | @cost: 9 Ferocity | @subtype: triggered -->"),  # My Turn!
    (9561, "<!-- @type: ability | @cost: 9 Ferocity -->"),  # Rebounding Storm
    (9577, "<!-- @type: ability | @cost: 9 Ferocity | @id: to-stone -->"),  # To Stone!

    # === 6th-Level Features ===
    (9593, "<!-- @type: feature-group | @level: 6 -->"),
    (9597, "<!-- @type: feature -->"),  # Marauder of the Primordial Chaos
    (9603, "<!-- @type: feature -->"),  # Primordial Portal
    (9609, "<!-- @type: feature -->"),  # Perk

    # 6th-Level Aspect Abilities (9 Ferocity each)
    (9621, "<!-- @type: ability | @cost: 9 Ferocity -->"),  # Avalanche Impact
    (9637, "<!-- @type: ability | @cost: 9 Ferocity -->"),  # Force of Storms
    (9657, "<!-- @type: ability | @cost: 9 Ferocity | @subtype: triggered -->"),  # Death Strike
    (9669, "<!-- @type: ability | @cost: 9 Ferocity -->"),  # Seek and Destroy
    (9691, "<!-- @type: ability | @cost: 9 Ferocity -->"),  # Pounce
    (9707, "<!-- @type: ability | @cost: 9 Ferocity -->"),  # Riders on the Storm

    # === 7th-Level Features ===
    (9719, "<!-- @type: feature-group | @level: 7 -->"),
    (9723, "<!-- @type: feature -->"),  # Characteristic Increase
    (9727, "<!-- @type: feature -->"),  # Elemental Form
    (9733, "<!-- @type: feature -->"),  # Greater Ferocity
    (9737, "<!-- @type: feature -->"),  # Growing Ferocity Improvement
    (9741, "<!-- @type: feature -->"),  # Skill

    # === 8th-Level Features ===
    (9745, "<!-- @type: feature-group | @level: 8 -->"),
    (9749, "<!-- @type: feature -->"),  # Perk
    (9753, "<!-- @type: feature -->"),  # 8th-Level Aspect Feature
    (9765, "<!-- @type: feature -->"),  # Menagerie
    (9769, "<!-- @type: feature -->"),  # A Step Ahead
    (9773, "<!-- @type: feature -->"),  # Strongest There Is

    # 11-Ferocity abilities
    (9781, "<!-- @type: ability | @cost: 11 Ferocity -->"),  # Elemental Ferocity
    (9791, "<!-- @type: ability | @cost: 11 Ferocity -->"),  # Overkill
    (9807, "<!-- @type: ability | @cost: 11 Ferocity -->"),  # Primordial Rage
    (9817, "<!-- @type: ability | @cost: 11 Ferocity -->"),  # Relentless Death

    # === 9th-Level Features ===
    (9833, "<!-- @type: feature-group | @level: 9 -->"),
    (9837, "<!-- @type: feature -->"),  # Harbinger of the Primordial Chaos

    # 9th-Level Aspect Abilities (11 Ferocity each)
    (9849, "<!-- @type: ability | @cost: 11 Ferocity | @id: death-comes-for-you-all -->"),  # Death Comes for You All!
    (9865, "<!-- @type: ability | @cost: 11 Ferocity -->"),  # Primordial Vortex
    (9885, "<!-- @type: ability | @cost: 11 Ferocity -->"),  # Primordial Bane
    (9901, "<!-- @type: ability | @cost: 11 Ferocity -->"),  # Shower of Blood
    (9921, "<!-- @type: ability | @cost: 11 Ferocity -->"),  # Death Rattle
    (9935, "<!-- @type: ability | @cost: 11 Ferocity -->"),  # Deluge

    # === 10th-Level Features ===
    (9951, "<!-- @type: feature-group | @level: 10 -->"),
    (9955, "<!-- @type: feature -->"),  # Chaos Incarnate
    (9963, "<!-- @type: feature -->"),  # Characteristic Increase
    (9967, "<!-- @type: feature -->"),  # Growing Ferocity Improvement
    (9971, "<!-- @type: feature -->"),  # Perk
    (9975, "<!-- @type: feature -->"),  # Primordial Ferocity
    (9979, "<!-- @type: feature -->"),  # Primordial Power
    (9989, "<!-- @type: feature -->"),  # Skill

    # === Stormwight Kits ===
    (9993, "<!-- @type: feature -->"),  # Stormwight Kits (intro section)
    (10001, "<!-- @type: feature -->"),  # Aspect Benefits and Animal Form
    (10005, "<!-- @type: feature -->"),  # Aspect of the Wild

    # Aspect of the Wild ability
    (10009, "<!-- @type: ability | @subtype: signature -->"),  # Aspect of the Wild (ability)

    (10021, "<!-- @type: feature -->"),  # Primordial Storm
    (10025, "<!-- @type: feature -->"),  # Equipment
    (10029, "<!-- @type: feature -->"),  # Kit Bonuses
    (10033, "<!-- @type: feature -->"),  # Signature Ability (intro)
    (10037, "<!-- @type: feature -->"),  # Growing Ferocity (intro)

    # Boren kit
    (10041, "<!-- @type: kit | @id: boren -->"),  # Boren
    (10045, "<!-- @type: feature -->"),  # Aspect Benefits
    (10049, "<!-- @type: feature -->"),  # Animal Form: Bear
    (10053, "<!-- @type: feature -->"),  # Hybrid Form: Bear
    (10057, "<!-- @type: feature -->"),  # Primordial Storm: Blizzard
    (10069, "<!-- @type: ability | @subtype: signature -->"),  # Bear Claws

    # Corven kit
    (10098, "<!-- @type: kit | @id: corven -->"),  # Corven
    (10102, "<!-- @type: feature -->"),  # Aspect Benefits
    (10106, "<!-- @type: feature -->"),  # Animal Form: Crow
    (10110, "<!-- @type: feature -->"),  # Hybrid Form: Crow
    (10114, "<!-- @type: feature -->"),  # Primordial Storm: Anabatic Wind
    (10127, "<!-- @type: ability | @subtype: signature -->"),  # Wing Buffet

    # Raden kit
    (10158, "<!-- @type: kit | @id: raden -->"),  # Raden
    (10162, "<!-- @type: feature -->"),  # Aspect Benefits
    (10166, "<!-- @type: feature -->"),  # Animal Form: Rat
    (10170, "<!-- @type: feature -->"),  # Hybrid Form: Rat
    (10174, "<!-- @type: feature -->"),  # Primordial Storm: Rat Flood
    (10187, "<!-- @type: ability | @subtype: signature -->"),  # Driving Pounce

    # Vuken kit
    (10218, "<!-- @type: kit | @id: vuken -->"),  # Vuken
    (10222, "<!-- @type: feature -->"),  # Aspect Benefits
    (10226, "<!-- @type: feature -->"),  # Animal Form: Wolf
    (10230, "<!-- @type: feature -->"),  # Hybrid Form: Wolf
    (10234, "<!-- @type: feature -->"),  # Primordial Storm: Lightning Storm
    (10247, "<!-- @type: ability | @subtype: signature -->"),  # Unbalancing Attack
]


def main():
    if not SRC.exists():
        print(f"Source file not found: {SRC}", file=sys.stderr)
        sys.exit(1)

    lines = SRC.read_text(encoding="utf-8").splitlines(keepends=True)

    # Build a dict of line_number -> annotation text
    # Line numbers are 1-based
    annotation_map = {}
    for lineno, annotation in ANNOTATIONS:
        annotation_map[lineno] = annotation

    # Build output
    output_lines = []

    # Add frontmatter at the very top
    output_lines.append(FRONTMATTER)

    for i, line in enumerate(lines):
        lineno = i + 1  # 1-based

        if lineno in annotation_map:
            # Insert annotation before this line
            ann = annotation_map[lineno]
            output_lines.append(ann + "\n")

        output_lines.append(line)

    # Write output
    DST.parent.mkdir(parents=True, exist_ok=True)
    DST.write_text("".join(output_lines), encoding="utf-8")

    total = len(ANNOTATIONS)
    print(f"Annotated {total} headings in Fury section")
    print(f"Output: {DST}")
    print(f"Original lines: {len(lines)}")
    print(f"Output lines: {sum(l.count(chr(10)) for l in output_lines) + len(output_lines)}")


if __name__ == "__main__":
    main()
