#!/usr/bin/env python3
"""One-shot, idempotent re-annotation of the heroes Treasures chapter.

State machine over the chapter region:

- Echelon/leveled category headers (#### N-Echelon Consumables/Trinkets,
  #### Leveled X Treasures, #### Other Leveled Treasures) -> @type: treasure-group
  with echelon + treasure-type. Mark "inside a group" so child items get annotated.
- `### Artifacts` -> @type: treasure-group with @tier: artifact (its items are H5
  directly underneath). Mark "inside a group".
- Other structural H3 (Found/Consumables/Trinkets/Leveled Treasures) -> leave;
  not a direct item parent (items live under their H4 groups).
- Prose H4 (What Does This Treasure Do?, Wearing/Wielding Treasures, Magic and
  Psionic Treasures, Stamina Bonuses and Damage Bonuses, Leveled Benefits, Carry
  Three Safely) -> annotation removed (folded into the chapter body); not a group.
- H5 items -> @type: treasure, but ONLY while inside a group (so the prose H5
  "Treasures and Kits" under Wielding Treasures is left untouched).
"""
import re
import pathlib

PATH = pathlib.Path("steel-etl/input/heroes/Draw Steel Heroes.md")

GROUP_RE = re.compile(
    r"^#### (?:(\d)(?:st|nd|rd|th)-Echelon (Consumables|Trinkets)"
    r"|Leveled (Armor|Implement|Weapon) Treasures"
    r"|(Other) Leveled Treasures)\s*$"
)
CAT = {"Consumables": "consumable", "Trinkets": "trinket",
       "Armor": "armor", "Implement": "implement", "Weapon": "weapon"}

lines = PATH.read_text().split("\n")
start = next(i for i, l in enumerate(lines)
             if l.strip() == "<!-- @type: chapter | @id: treasures -->")
end = next(i for i, l in enumerate(lines)
          if i > start and l.strip().startswith("<!-- @type: chapter"))


def drop_prev_treasure_ann(out):
    if out and out[-1].strip().startswith("<!-- @type: treasure"):
        out.pop()


out = []
in_group = False
groups = items = prose = 0
for i, line in enumerate(lines):
    if not (start < i < end):
        out.append(line)
        continue

    if line.startswith("### "):
        if line.strip() == "### Artifacts":
            drop_prev_treasure_ann(out)
            out.append("<!-- @type: treasure-group | @tier: artifact -->")
            out.append(line)
            in_group = True
            groups += 1
        else:
            in_group = False
            out.append(line)
        continue

    if line.startswith("#### "):
        m = GROUP_RE.match(line)
        if m:
            drop_prev_treasure_ann(out)
            ech, kind, lvl_kind, other = m.groups()
            if ech:
                ann = f"<!-- @type: treasure-group | @echelon: {ech} | @treasure-type: {CAT[kind]} -->"
            elif lvl_kind:
                ann = f"<!-- @type: treasure-group | @treasure-type: {CAT[lvl_kind]} -->"
            else:  # Other Leveled Treasures
                ann = "<!-- @type: treasure-group | @treasure-type: other -->"
            out.append(ann)
            out.append(line)
            in_group = True
            groups += 1
        else:  # prose H4 → fold
            drop_prev_treasure_ann(out)
            out.append(line)
            in_group = False
            prose += 1
        continue

    if line.startswith("##### "):
        if in_group:
            if not (out and out[-1].strip().startswith("<!-- @type: treasure")):
                out.append("<!-- @type: treasure -->")
                items += 1
            out.append(line)
        else:
            out.append(line)
        continue

    out.append(line)

PATH.write_text("\n".join(out))
print(f"groups={groups} items={items} prose={prose}")
