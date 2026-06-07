# Rule-term → SCC mapping

Canonical mapping from every rules/glossary **term** in the *Draw Steel Heroes* book to the
SCC code it should link to. This drives the later annotation + linking effort (publish
`rule.*` codes, annotate the anchor sections, then sweep the document linking headwords).

## How to read this table

- **Term** — the glossary headword.
- **Variants** — case/plural/inflected surface forms to match when linking.
- **Decision** — `new-rule` (mint a new `rule.<group>/<id>` code), `reuse` (link an existing
  typed code), or `skip` (no rules meaning / no home).
- **Code** — full SCC code. For `new-rule`: `mcdm.heroes.v1/rule.<group>/<id>`. For `reuse`:
  the existing code. Blank for `skip`.
- **Anchor heading (line)** — for `new-rule`, the existing heading whose section defines the
  rule (the section we will annotate to mint the code), with its source line number. For
  `reuse`: `(existing)`.
- **Notes** — disambiguation, missing-heading flags, collision resolutions.

Source doc: `steel-etl/input/heroes/Draw Steel Heroes.md` (glossary entries are the bold
`**Term:** definition` run on lines 112–584). Existing codes verified against
`classification.json`. There are currently **no** `rule.*` codes, so every `new-rule` code
below is collision-free against the registry; uniqueness among the new codes was checked
separately (see "Collisions resolved" at the bottom).

## Mapping

| Term | Variants | Decision | Code | Anchor heading (line) | Notes |
|------|----------|----------|------|------------------------|-------|
| Ability | ability, abilities | new-rule | mcdm.heroes.v1/rule.general/ability | `#### Ability Keywords` (4231) | Generic game term; keyword section is the closest definitional home. Very common word — link conservatively. |
| Ability Roll | ability roll, ability rolls | new-rule | mcdm.heroes.v1/rule.dice/ability-roll | `#### Ability Roll` (4379) | |
| Adjacent | adjacent | new-rule | mcdm.heroes.v1/rule.combat/adjacent | `##### Adjacent` (4558) | "Within 1 square." Distinct heading exists. |
| Agility | Agility | new-rule | mcdm.heroes.v1/rule.character/agility | `#### Agility` (645) | Characteristic. Match capitalized form only to avoid the common noun. |
| Ally | ally, allies, allied | new-rule | mcdm.heroes.v1/rule.combat/ally | `##### Ally` (4361) | Its own dedicated heading. |
| Ancestry | ancestry, ancestries | reuse | mcdm.heroes.v1/chapter/ancestries | (existing) | Links to the Ancestries chapter; individual ancestries have `ancestry/<id>` codes. |
| Area of Effect | area of effect, areas of effect, AoE | new-rule | mcdm.heroes.v1/rule.combat/area-of-effect | `#### Area Abilities` (4305) | Covers aura/burst/cube/line/wall collectively. |
| Argument | argument, arguments | reuse | mcdm.heroes.v1/chapter/negotiation | (existing) | Negotiation plea; defined in the Negotiation chapter. No standalone heading; chapter is nearest typed home. |
| Artifact | artifact, artifacts | new-rule | mcdm.heroes.v1/rule.treasure/artifact | NO HEADING — needs new heading OR map to nearest parent: Treasures chapter. Best parent `mcdm.heroes.v1/chapter/treasures` | Glossary-only; no artifact section in heroes book (artifacts are a teaser). |
| Artisan | artisan, artisans | reuse | mcdm.heroes.v1/career/artisan | (existing) | Glossary defines artisan as a crafting follower; the Artisan career is the typed entity. Judgment call: could instead be a follower-type rule — see Notes for Follower. |
| Aura | aura, auras | new-rule | mcdm.heroes.v1/rule.combat/aura | `##### Aura` (4315) | Area-ability subtype. |
| Background | background, backgrounds | reuse | mcdm.heroes.v1/chapter/background | (existing) | "Culture and career"; Background chapter. |
| Bane | bane, banes | new-rule | mcdm.heroes.v1/rule.dice/bane | `##### Bane` (733) | −2 penalty. |
| Bonus | bonus, bonuses | new-rule | mcdm.heroes.v1/rule.dice/bonuses-and-penalties | `#### Bonuses and Penalties` (757) | Primary for the Bonuses and Penalties section; Penalty reuses this code. |
| Breakthrough | breakthrough, breakthroughs | new-rule | mcdm.heroes.v1/rule.downtime/breakthrough | NO HEADING — needs new heading OR map to nearest parent: Project Roll. Best parent `mcdm.heroes.v1/rule.downtime/project-roll` (`### Project Roll` 22720) | Natural 19–20 on a project roll; defined inline under project rolls. |
| Burst | burst, bursts | new-rule | mcdm.heroes.v1/rule.combat/burst | `##### Burst` (4319) | Area-ability subtype. |
| Capital | Capital, capitals | new-rule | mcdm.heroes.v1/rule.world/capital | `#### Capital` (964) | Setting term ("largest city in Orden"). |
| Career | career, careers | reuse | mcdm.heroes.v1/chapter/careers | (existing) | Careers chapter; individual careers have `career/<id>`. |
| Characteristics | characteristic, characteristics | new-rule | mcdm.heroes.v1/rule.character/characteristic | `### Characteristics` (637) | Umbrella for Might/Agility/Reason/Intuition/Presence. |
| Ceiling | ceiling, ceilings | reuse | mcdm.heroes.v1/rule.general/ground | (shares Ground section) | The Ground and Ceiling section defines both; collapsed to the Ground primary. |
| Class | class, classes | reuse | mcdm.heroes.v1/chapter/classes | (existing) | Classes chapter; individual classes `class/<id>`. |
| Climb | climb, climbs, climbing | reuse | mcdm.heroes.v1/movement/climb-or-swim | (existing) | Movement mode. Climb and Swim share `movement/climb-or-swim`. |
| Combat Round | combat round, combat rounds, round, rounds | new-rule | mcdm.heroes.v1/rule.combat/combat-round | `#### Combat Round` (21354) | "round" is common — prefer "combat round" surface form when linking bare. |
| Complication | complication, complications | reuse | mcdm.heroes.v1/chapter/complications | (existing) | Complications chapter; individual `complication/<id>`. |
| Concealment | concealment | new-rule | mcdm.heroes.v1/rule.combat/concealment | `### Concealment` (21927) | |
| Condition | condition, conditions | new-rule | mcdm.heroes.v1/rule.combat/condition | `#### Conditions` (4582) | Umbrella term; individual conditions are `condition/<id>` (reuse those for named conditions). |
| Consequence | consequence, consequences | new-rule | mcdm.heroes.v1/rule.general/consequence | NO HEADING — needs new heading OR map to nearest parent: Test outcomes. Best parent `mcdm.heroes.v1/rule.test/test` (`### How to Make a Test` 20353) | Test-related setback; defined in glossary, discussed under tests/rewards. |
| Consumable | consumable, consumables | new-rule | mcdm.heroes.v1/rule.treasure/consumable | `### Consumables` (23803) | Treasure category. Individual consumables are `treasure.<echelon>.consumable/<id>`. |
| Cover | cover | new-rule | mcdm.heroes.v1/rule.combat/cover | `### Cover` (21923) | Common word — link only in ranged-combat/concealment rules context. |
| Crafting Project | crafting project, crafting projects | new-rule | mcdm.heroes.v1/rule.downtime/crafting-project | `### Crafting Projects` (22789) | |
| Crawl | crawl, crawls, crawling | reuse | mcdm.heroes.v1/movement/crawl | (existing) | Movement mode. |
| Creature | creature, creatures | new-rule | mcdm.heroes.v1/rule.general/creature | `### Creatures and Objects` (808) | Primary for the Creatures and Objects section; Object reuses this code. Extremely common word — link very sparingly. |
| Critical Hit | critical hit, critical hits, crit | new-rule | mcdm.heroes.v1/rule.combat/critical-hit | `#### Critical Hit` (4493) | |
| Cube | cube, cubes | new-rule | mcdm.heroes.v1/rule.combat/cube | `##### Cube` (4323) | Area-ability subtype. |
| Culture | culture, cultures | reuse | mcdm.heroes.v1/chapter/cultures | (existing) | Cultures chapter; individual `culture/<id>`. |
| Damage | damage, damages | new-rule | mcdm.heroes.v1/rule.damage/damage | `### Damage` (21935) | Very common word; link conservatively. |
| Damage Immunity | damage immunity, immunity, immune | new-rule | mcdm.heroes.v1/rule.damage/damage-immunity | `##### Damage Immunity` (21945) | |
| Damage Type | damage type, damage types | new-rule | mcdm.heroes.v1/rule.damage/damage-type | `#### Damage Types` (21939) | |
| Damage Weakness | damage weakness, weakness | new-rule | mcdm.heroes.v1/rule.damage/damage-weakness | `##### Damage Weakness` (21955) | |
| Dig Maneuver | dig maneuver, dig | reuse | mcdm.heroes.v1/movement/burrow | `###### Dig Maneuver` (21477) | Judgment call: maps to the burrow movement type it depends on. Alternative: new-rule combat maneuver. Chose reuse since it has no common-trait code and is a burrow application. |
| Director | Director, directors, GM | new-rule | mcdm.heroes.v1/rule.general/director | NO HEADING — needs new heading OR map to nearest parent: For the Director chapter. Best parent `mcdm.heroes.v1/chapter/for-the-director` | Role definition; the For the Director chapter is the closest typed home. |
| Distance | distance, distances | new-rule | mcdm.heroes.v1/rule.combat/distance | `#### Distance` (4281) | Ability "Distance" entry. Note: there is also a kit `#### Distance Bonus`; this term is the ability/measurement concept. |
| Double Bane | double bane, double banes | reuse | mcdm.heroes.v1/rule.dice/bane | (shares Bane section) | Glossary-only; no own heading. The Bane section (733) is where it is defined; collapsed to the Bane primary (one heading mints one code). |
| Double Edge | double edge, double edges | reuse | mcdm.heroes.v1/rule.dice/edge | (shares Edge section) | Glossary-only; no own heading. The Edge section (727) is where it is defined; collapsed to the Edge primary. |
| Downtime Project | downtime project, downtime projects, downtime | reuse | mcdm.heroes.v1/chapter/downtime-projects | (existing) | Downtime Projects chapter is the typed home. (A `rule.downtime/downtime-project` could be minted at `# Downtime Projects` 22688 if a finer anchor is wanted; chose chapter reuse.) |
| Dying | dying, dies, died, death | new-rule | mcdm.heroes.v1/rule.health/dying | `#### Dying and Death` (21981) | id=dying. |
| Echelon | echelon, echelons | new-rule | mcdm.heroes.v1/rule.general/echelon | `### Echelons of Play` (892) | |
| Edge | edge, edges | new-rule | mcdm.heroes.v1/rule.dice/edge | `##### Edge` (727) | +2 bonus. |
| EoT | EoT | new-rule | mcdm.heroes.v1/rule.combat/end-of-turn | `##### End of Next Turn (EoT)` (4540) | Abbreviation for end-of-turn effect duration. |
| Enemy | enemy, enemies | new-rule | mcdm.heroes.v1/rule.combat/enemy | `##### Enemy` (4357) | Its own dedicated heading. |
| Enhancement | enhancement, enhancements | new-rule | mcdm.heroes.v1/rule.treasure/enhancement | `#### Imbue Treasure` (22892) | Crafting-applied treasure property; defined under Imbue Treasure (armor/implement/weapon enhancement subsections follow). |
| Experience (XP) | Experience, XP, experience points | new-rule | mcdm.heroes.v1/rule.resource/experience | `#### Experience` (854) | |
| Falling | falling, fall, falls, fell | new-rule | mcdm.heroes.v1/rule.health/falling | NO HEADING — needs new heading OR map to nearest parent: Forced Into a Fall. Best parent `mcdm.heroes.v1/rule.health/dying` sibling under combat; nearest section `##### Forced Into a Fall` (21680) | Falling damage rule; defined in glossary, mechanics under Forced Into a Fall. Flag for a dedicated heading. |
| Flanking | flanking, flank, flanked | new-rule | mcdm.heroes.v1/rule.combat/flanking | `### Flanking` (21915) | |
| Follower | follower, followers | new-rule | mcdm.heroes.v1/rule.general/follower | `#### Follower Types` (26894) | Defined in the Follower Types section. Artisan/Sage/Retainer are follower kinds. |
| Free Maneuver | free maneuver, free maneuvers | new-rule | mcdm.heroes.v1/rule.combat/free-maneuver | `#### Free Maneuvers` (21420) | |
| Free Triggered Action | free triggered action, free triggered actions | reuse | mcdm.heroes.v1/rule.combat/triggered-action | (shares Triggered Action section) | The Triggered Actions and Free Triggered Actions section defines both; collapsed to the Triggered Action primary. |
| God | god, gods, deity, deities | reuse | mcdm.heroes.v1/chapter/gods-and-religion | (existing) | Gods and Religion chapter; individual gods are `god/<id>`. Glossary "God" → chapter; named gods → `god/<id>`. |
| Ground | ground, grounds | new-rule | mcdm.heroes.v1/rule.general/ground | `#### Ground and Ceiling` (4576) | Primary for the Ground and Ceiling section; Ceiling reuses this code. |
| Group Test | group test, group tests | new-rule | mcdm.heroes.v1/rule.test/group-test | `### Group Tests` (21157) | |
| Guide | guide, guides | new-rule | mcdm.heroes.v1/rule.downtime/guide | `#### Guides` (22742) | Downtime manual granting project points. |
| Hero | hero, heroes | new-rule | mcdm.heroes.v1/rule.general/hero | NO HEADING — needs new heading OR map to nearest parent: Making a Hero chapter. Best parent `mcdm.heroes.v1/chapter/making-a-hero` | Core but glossary-only definition; extremely common — link sparingly. |
| Heroic Ability | heroic ability, heroic abilities | new-rule | mcdm.heroes.v1/rule.general/heroic-ability | `##### Heroic Abilities` (4221) | Distinct heading of its own. Group is `rule.general` — defines a class of ability, not a resource. (Multiple per-class `##### Heroic Abilities` headings exist; anchor the definitional one at 4221.) |
| Hero Tokens | hero token, hero tokens | new-rule | mcdm.heroes.v1/rule.resource/hero-token | `### Hero Tokens` (769) | |
| Heroic Resource | heroic resource, heroic resources | new-rule | mcdm.heroes.v1/rule.resource/heroic-resource | `#### Heroic Resources` (860) | Sole new-rule at this anchor; Heroic Ability now anchors its own `##### Heroic Abilities` (4221). |
| Humanoid | humanoid, humanoids | new-rule | mcdm.heroes.v1/rule.general/humanoid | NO HEADING — needs new heading OR map to nearest parent: Ancestries chapter. Best parent `mcdm.heroes.v1/chapter/ancestries` | Creature category; glossary-only. |
| Implement | implement, implements | new-rule | mcdm.heroes.v1/rule.treasure/implement | `##### Imbue Implement` (23075) | Treasure/equipment category. Distinct from the named treasure "Implement of Wrath" (an ability). |
| Interest | interest | new-rule | mcdm.heroes.v1/rule.negotiation/interest | `#### Interest` (22163) | Negotiation stat. Common word — match in negotiation context. |
| Intuition | Intuition | new-rule | mcdm.heroes.v1/rule.character/intuition | `#### Intuition` (653) | Characteristic. Match capitalized form. |
| Item Prerequisite | item prerequisite, item prerequisites | new-rule | mcdm.heroes.v1/rule.downtime/item-prerequisite | `#### Item Prerequisite` (22702) | |
| Jump | jump, jumps, jumping, jumped | reuse | mcdm.heroes.v1/movement/jump | (existing) | Movement. (Also a `skill/jump`; the glossary term is the movement action.) |
| Kit | kit, kits | reuse | mcdm.heroes.v1/chapter/kits | (existing) | Kits chapter; individual `kit/<id>`. |
| Level | level, levels | new-rule | mcdm.heroes.v1/rule.general/level | NO HEADING — needs new heading OR map to nearest parent: Echelons of Play. Best parent `mcdm.heroes.v1/rule.general/echelon` (`### Echelons of Play` 892) | Very common word; link sparingly. |
| Leveled Treasure | leveled treasure, leveled treasures | new-rule | mcdm.heroes.v1/rule.treasure/leveled-treasure | `### Leveled Treasures` (25021) | |
| Line | line, lines | new-rule | mcdm.heroes.v1/rule.combat/line | `##### Line` (4327) | Area-ability subtype. Common word — match area-of-effect context. |
| Line of Effect | line of effect | new-rule | mcdm.heroes.v1/rule.combat/line-of-effect | `#### Line of Effect` (4562) | |
| Main Action | main action, main actions | reuse | mcdm.heroes.v1/rule.combat/turn | (shares Taking a Turn section) | The `### Main Actions` heading is a non-code `feature-group` container, so it can't mint a code; the action economy is defined in `### Taking a Turn`. Individual common main actions are `feature.trait.common.main-actions/<id>`. |
| Malice | malice | skip | | | Monsters-book resource ("See Draw Steel: Monsters"); no rules content or home in the heroes book. |
| Maneuver | maneuver, maneuvers | reuse | mcdm.heroes.v1/rule.combat/turn | (shares Taking a Turn section) | The `### Maneuvers` heading is a non-code `feature-group` container; action economy defined in `### Taking a Turn`. Individual common maneuvers are `feature.trait.common.maneuvers/<id>`. |
| Manifold | manifold, manifolds | reuse | mcdm.heroes.v1/rule.world/orden | (shares Orden section) | The Orden and the Timescape section defines Orden/Timescape/Manifold together; collapsed to the Orden primary. |
| Melee | melee | new-rule | mcdm.heroes.v1/rule.combat/melee | `##### Melee` (4247) | Ability keyword. (4247 is the Ability Keywords > Melee subheading.) |
| Melee Free Strike | melee free strike, melee free strikes | reuse | mcdm.heroes.v1/feature.trait.common.main-actions/free-strike | (existing) | A free strike made with a melee ability; maps to the Free Strike common action. |
| Might | Might | new-rule | mcdm.heroes.v1/rule.character/might | `#### Might` (641) | Characteristic. Match capitalized form. |
| Montage Test | montage test, montage tests | new-rule | mcdm.heroes.v1/rule.test/montage-test | `### Montage Tests` (21179) | |
| Motivation | motivation, motivations | new-rule | mcdm.heroes.v1/rule.negotiation/motivation | `#### Motivations` (22179) | No `/motivation/` typed code exists in the registry, so this is new-rule (not reuse). |
| Mounted Combat | mounted combat | new-rule | mcdm.heroes.v1/rule.combat/mounted-combat | `### Mounted Combat` (22038) | |
| Move Action | move action, move actions | reuse | mcdm.heroes.v1/rule.combat/turn | (shares Taking a Turn section) | The `### Move Actions` heading is a non-code `feature-group` container; action economy defined in `### Taking a Turn`. Individual common move actions are `feature.trait.common.move-actions/<id>`. |
| Movement | movement, move, moves, moving, moved | reuse | mcdm.heroes.v1/movement/walk | `### Movement` (21436) | Judgment call: the generic "Movement" concept. Mapped to the Movement section conceptually; "walk" is the default move. Alternative: new-rule combat/movement at `### Movement` (21436). Chose reuse to keep movement terms unified; reviewer should confirm. |
| Mundane | mundane | reuse | mcdm.heroes.v1/rule.general/supernatural | (shares Supernatural section) | The Supernatural or Mundane section (818) defines both; collapsed to the Supernatural primary. |
| Natural 19 or 20 | natural 19, natural 20, natural 19 or 20, nat 19, nat 20 | new-rule | mcdm.heroes.v1/rule.dice/natural-19-20 | `##### Natural 19 or 20: Success With a Reward` (20442) | Its own distinct heading (re-anchored off Critical Hit 4493 to avoid sharing that anchor with the Critical Hit row). |
| Natural Roll | natural roll, natural rolls | new-rule | mcdm.heroes.v1/rule.dice/natural-roll | `##### Natural Roll` (717) | |
| Negotiation | negotiation, negotiations, negotiate | reuse | mcdm.heroes.v1/chapter/negotiation | (existing) | Negotiation chapter. |
| No Action | no action, no actions | new-rule | mcdm.heroes.v1/rule.combat/no-action | NO HEADING — needs new heading OR map to nearest parent: action categories. Best parent `mcdm.heroes.v1/rule.combat/main-action` (`### Main Actions` 21836) | Glossary-defined "no action" activity type; no dedicated heading. Flag. |
| NPC | NPC, NPCs, nonplayer character | new-rule | mcdm.heroes.v1/rule.general/npc | `### PCs and NPCs` (822) | Defined in the PCs and NPCs section. |
| Object | object, objects | reuse | mcdm.heroes.v1/rule.general/creature | (shares Creature section) | The Creatures and Objects section (808) defines both; Object Stamina is durability, not the term "object". Common word — link sparingly. |
| Objective | objective, objectives | new-rule | mcdm.heroes.v1/rule.combat/objective | `#### Objective Endings` (22058) | Combat encounter goal. |
| Opportunity Attack | opportunity attack, opportunity attacks | new-rule | mcdm.heroes.v1/rule.combat/opportunity-attack | `#### Opportunity Attacks` (21877) | Its own dedicated heading. |
| Opposed Power Roll | opposed power roll, opposed power rolls | new-rule | mcdm.heroes.v1/rule.dice/opposed-power-roll | `#### Opposed Power Rolls` (20550) | |
| Orden | Orden | new-rule | mcdm.heroes.v1/rule.world/orden | `### Orden and the Timescape` (916) | Primary for the Orden and the Timescape section; Timescape and Manifold reuse this code. |
| Patience | patience | new-rule | mcdm.heroes.v1/rule.negotiation/patience | `#### Patience` (22169) | Negotiation stat. |
| Penalty | penalty, penalties | reuse | mcdm.heroes.v1/rule.dice/bonuses-and-penalties | (shares Bonus section) | The Bonuses and Penalties section defines both; collapsed to the Bonus primary. |
| Perk | perk, perks | reuse | mcdm.heroes.v1/chapter/perks | (existing) | Perks chapter; individual `perk/<id>`. |
| Pitfall | pitfall, pitfalls | new-rule | mcdm.heroes.v1/rule.negotiation/pitfall | `#### Pitfalls` (22185) | Negotiation trait. |
| Potency | potency, potencies | new-rule | mcdm.heroes.v1/rule.character/potency | `#### Potencies` (4433) | |
| Power Roll | power roll, power rolls | new-rule | mcdm.heroes.v1/rule.dice/power-roll | `### Power Rolls` (683) | |
| Presence | Presence | new-rule | mcdm.heroes.v1/rule.character/presence | `#### Presence` (657) | Characteristic. Match capitalized form. |
| Project Event | project event, project events | new-rule | mcdm.heroes.v1/rule.downtime/project-event | `#### For the Director: Project Events` (22750) | |
| Project Goal | project goal, project goals | new-rule | mcdm.heroes.v1/rule.downtime/project-goal | NO HEADING — needs new heading OR map to nearest parent: Project Roll / Discover Lore Project Goals. Best parent `mcdm.heroes.v1/rule.downtime/project-points` (`#### Project Points` 3619) | Defined in glossary; "project points needed". Flag. |
| Project Points | project points | new-rule | mcdm.heroes.v1/rule.downtime/project-points | `#### Project Points` (3619) | |
| Project Roll | project roll, project rolls | new-rule | mcdm.heroes.v1/rule.downtime/project-roll | `### Project Roll` (22720) | |
| Project Source | project source, project sources | new-rule | mcdm.heroes.v1/rule.downtime/project-source | `#### Project Source` (22706) | |
| Pull | pull, pulls, pulled, pulling | reuse | mcdm.heroes.v1/movement/forced-movement | (existing) | Forced movement form. |
| Push | push, pushes, pushed, pushing | reuse | mcdm.heroes.v1/movement/forced-movement | (existing) | Forced movement form. |
| Ranged | ranged | new-rule | mcdm.heroes.v1/rule.combat/ranged | `##### Ranged` (4255) | Ability keyword (Ability Keywords > Ranged). |
| Ranged Free Strike | ranged free strike, ranged free strikes | reuse | mcdm.heroes.v1/feature.trait.common.main-actions/free-strike | (existing) | Free strike made with a ranged ability. |
| Reactive Test | reactive test, reactive tests | new-rule | mcdm.heroes.v1/rule.test/reactive-test | `### Reactive Tests` (20560) | |
| Reason | Reason | new-rule | mcdm.heroes.v1/rule.character/reason | `#### Reason` (649) | Characteristic. Match capitalized form. |
| Recoveries | recovery, recoveries | new-rule | mcdm.heroes.v1/rule.health/recoveries | `#### Recoveries and Recovery Value` (21971) | Primary for the Recoveries and Recovery Value section; Recovery Value reuses this code. |
| Recovery Value | recovery value | reuse | mcdm.heroes.v1/rule.health/recoveries | (shares Recoveries section) | The Recoveries and Recovery Value section defines both; collapsed to the Recoveries primary. |
| Renown | renown | new-rule | mcdm.heroes.v1/rule.resource/renown | `## Renown` (26857) | Two headings exist (`#### Renown` 3611 brief mention; `## Renown` 26857 the rules section). Anchor the rules section. |
| Research Project | research project, research projects | new-rule | mcdm.heroes.v1/rule.downtime/research-project | `### Research Projects` (23371) | |
| Respite | respite, respites | new-rule | mcdm.heroes.v1/rule.resource/respite | `#### Respite` (882) | Defined in The Basics (882). (Downtime detail uses respites but the definition sits here.) |
| Respite Activity | respite activity, respite activities | new-rule | mcdm.heroes.v1/rule.downtime/respite-activity | NO HEADING — needs new heading OR map to nearest parent: Respite. Best parent `mcdm.heroes.v1/rule.resource/respite` (`#### Respite` 882) | Glossary-only; one activity per respite. Flag. |
| Retainer | retainer, retainers | new-rule | mcdm.heroes.v1/rule.general/retainer | `##### Retainer` (26904) | Its own dedicated heading (a follower who adventures along). Note: the Monsters book also has a `retainer` type. |
| Reward | reward, rewards | reuse | mcdm.heroes.v1/chapter/rewards | (existing) | Rewards chapter is the typed home. (Could mint `rule.general/reward` at `### How to Make a Test` reward subsection; chose chapter reuse.) |
| Rolled Damage | rolled damage | new-rule | mcdm.heroes.v1/rule.damage/rolled-damage | `#### Rolled Damage` (4429) | |
| Sage | sage, sages | reuse | mcdm.heroes.v1/career/sage | (existing) | Glossary defines sage as a research follower; the Sage career is the typed entity. (Parallel to Artisan.) |
| Saint | saint, saints | new-rule | mcdm.heroes.v1/rule.world/saint | `### Gods and Religion` (26952) | No `/saint/` type exists; saints are discussed in Gods and Religion (Saints and Domains table 27069). Setting/world term. |
| Save Ends | save ends, (save ends) | reuse | mcdm.heroes.v1/rule.general/saving-throw | (shares Saving Throw section) | The Saving Throw (Save Ends) section defines both; collapsed to the Saving Throw primary. |
| Saving Throw | saving throw, saving throws, save | new-rule | mcdm.heroes.v1/rule.general/saving-throw | `##### Saving Throw (Save Ends)` (4544) | Primary for the section; Save Ends reuses this code. |
| Side | side, sides | new-rule | mcdm.heroes.v1/rule.combat/side | `#### Sides` (21346) | Combat side. Common word — link in combat context. |
| Signature Ability | signature ability, signature abilities | new-rule | mcdm.heroes.v1/rule.combat/signature-ability | `##### Signature Abilities` (4227) | Ability usable without spending a resource. Many class `##### Signature Ability` headings exist; anchor the definitional one under Ability Keywords (4227). |
| Size | size, sizes | new-rule | mcdm.heroes.v1/rule.character/size | `#### Size and Space` (21323) | Primary for the Size and Space section; Space reuses this code. |
| Skill | skill, skills | reuse | mcdm.heroes.v1/chapter/skills | (existing) | Skills chapter; individual skills `skill/<id>`. Glossary "Skill" → chapter; named skills → `skill/<id>`. |
| Slide | slide, slides, slid, sliding | reuse | mcdm.heroes.v1/movement/forced-movement | (existing) | Forced movement form. |
| Space | space, spaces | reuse | mcdm.heroes.v1/rule.character/size | (shares Size section) | The Size and Space section defines both; collapsed to the Size primary (one heading mints one code). |
| Speed | speed, speeds | new-rule | mcdm.heroes.v1/rule.character/speed | `### Starting Size and Speed` (1511) | Shares this Basics anchor with Stability; distinct id. (Also a kit `#### Speed Bonus` — different concept.) |
| Square | square, squares | new-rule | mcdm.heroes.v1/rule.combat/square | NO HEADING — needs new heading OR map to nearest parent: measurement/Distance. Best parent `mcdm.heroes.v1/rule.combat/distance` (`#### Distance` 4281) | Unit of measurement; glossary-only. Common word — link sparingly. |
| Stability | stability | new-rule | mcdm.heroes.v1/rule.character/stability | `##### Stability` (21689) | Detailed section under Forced Movement. (Also defined in Starting Size and Speed 1511; chose the dedicated Stability section as anchor.) |
| Stamina | stamina | new-rule | mcdm.heroes.v1/rule.health/stamina | `### Stamina` (21965) | |
| Strained | strained | reuse | mcdm.heroes.v1/feature.trait.talent.level-1/clarity-and-strain | (existing) | Talent-only clarity state, defined by the talent's Clarity and Strain feature (verified in classification.json). |
| Strike | strike, strikes | new-rule | mcdm.heroes.v1/rule.combat/strike | `##### Strike` (4259) | Ability keyword. Distinct from Free Strike (common action). Common verb — link sparingly; only the Ability/keyword context, never narrative "strikes". |
| Subclass | subclass, subclasses | new-rule | mcdm.heroes.v1/rule.general/subclass | `### Subclasses` (4185) | Its own dedicated heading. |
| Suffocating | suffocating, suffocate, suffocation | new-rule | mcdm.heroes.v1/rule.health/suffocating | `### Suffocating` (22032) | |
| Supernatural | supernatural | new-rule | mcdm.heroes.v1/rule.general/supernatural | `### Supernatural or Mundane` (818) | Primary for the Supernatural or Mundane section; Mundane reuses this code. (Re-anchored from Magic and Psionic Treasures, which does not define the term.) |
| Surge | surge, surges | new-rule | mcdm.heroes.v1/rule.resource/surge | `#### Surges` (4505) | id=surge. |
| Surprised | surprised, surprise | new-rule | mcdm.heroes.v1/rule.combat/surprised | `#### Determine Surprise` (21362) | |
| Swim | swim, swims, swimming, swam | reuse | mcdm.heroes.v1/movement/climb-or-swim | (existing) | Movement mode; shares `movement/climb-or-swim` with Climb. |
| Target | target, targets, targeted | new-rule | mcdm.heroes.v1/rule.combat/target | `#### Target` (4343) | Ability target. Common word — link in ability context. |
| Temporary Stamina | temporary stamina, temp stamina | new-rule | mcdm.heroes.v1/rule.health/temporary-stamina | `#### Temporary Stamina` (22007) | |
| Test | test, tests | new-rule | mcdm.heroes.v1/rule.test/test | `### How to Make a Test` (20353) | Core test rule. Anchor is the How-to section under the Tests chapter (`# Tests` 20337). Common word — link in rules context. |
| Tier Outcome | tier outcome, tier outcomes, tier | new-rule | mcdm.heroes.v1/rule.dice/tier-outcome | `##### Power Roll Outcomes` (701) | Primary for the Power Roll Outcomes section; Tier 1/2/3 reuse this code. |
| Tier 1 | tier 1, tier one | reuse | mcdm.heroes.v1/rule.dice/tier-outcome | (shares Tier Outcome section) | The Power Roll Outcomes section defines all tiers; collapsed to the Tier Outcome primary. |
| Tier 2 | tier 2, tier two | reuse | mcdm.heroes.v1/rule.dice/tier-outcome | (shares Tier Outcome section) | The Power Roll Outcomes section defines all tiers; collapsed to the Tier Outcome primary. |
| Tier 3 | tier 3, tier three | reuse | mcdm.heroes.v1/rule.dice/tier-outcome | (shares Tier Outcome section) | The Power Roll Outcomes section defines all tiers; collapsed to the Tier Outcome primary. |
| Title | title, titles | reuse | mcdm.heroes.v1/chapter/titles | (existing) | Titles chapter; individual `title/<id>`. |
| Timescape | Timescape | reuse | mcdm.heroes.v1/rule.world/orden | (shares Orden section) | The Orden and the Timescape section (916) defines both; collapsed to the Orden primary. (Detail at `#### The Myriad Worlds of the Timescape` 998 — not used as anchor.) |
| Treasure | treasure, treasures | reuse | mcdm.heroes.v1/chapter/treasures | (existing) | Treasures chapter; categories Consumable/Trinket/Leveled/etc. are new-rule rows above. |
| Triggered Action | triggered action, triggered actions | new-rule | mcdm.heroes.v1/rule.combat/triggered-action | `#### Triggered Actions and Free Triggered Actions` (21410) | Primary for the section; Free Triggered Action reuses this code. |
| Trinket | trinket, trinkets | new-rule | mcdm.heroes.v1/rule.treasure/trinket | `### Trinkets` (24484) | Treasure category. Individual trinkets are `treasure.<echelon>.trinket/<id>`. |
| Turn | turn, turns | new-rule | mcdm.heroes.v1/rule.combat/turn | `### Taking a Turn` (21469) | Defined in glossary; "main action, maneuver, move action." Common word — link in combat context. Anchored on Taking a Turn; Main/Maneuver/Move Action reuse this code. |
| Unattended Object | unattended object, unattended objects | new-rule | mcdm.heroes.v1/rule.general/unattended-object | `#### Object Stamina` (22017) | Object not worn/held/controlled; Object Stamina is its nearest object-rules heading. Sole new-rule at this anchor (Object now reuses Creature). |
| Underwater combat | underwater combat | new-rule | mcdm.heroes.v1/rule.combat/underwater-combat | `### Underwater Combat` (22028) | |
| Untyped Damage | untyped damage | new-rule | mcdm.heroes.v1/rule.damage/untyped-damage | NO HEADING — needs new heading OR map to nearest parent: Damage Types. Best parent `mcdm.heroes.v1/rule.damage/damage-type` (`#### Damage Types` 21939) | Glossary-only; damage with no type. Flag. |
| Vasloria | Vasloria | new-rule | mcdm.heroes.v1/rule.world/vasloria | `#### Vasloria` (924) | Setting continent. |
| Vertical | vertical | reuse | mcdm.heroes.v1/movement/forced-movement | (existing) | Vertical forced movement → forced-movement code. |
| Victories | victory, victories | new-rule | mcdm.heroes.v1/rule.resource/victories | `#### Victories` (838) | |
| Wall | wall, walls | new-rule | mcdm.heroes.v1/rule.combat/wall | `##### Wall` (4331) | Area-ability subtype. Common word — match area-of-effect context. |
| Wealth | wealth | new-rule | mcdm.heroes.v1/rule.resource/wealth | `#### Wealth` (3615) | |
| Winded | winded | new-rule | mcdm.heroes.v1/rule.health/winded | `#### Winded` (21975) | |

## Summary

- **Total terms:** 170 (every glossary headword on lines 112–584).
- **new-rule:** 123 · **reuse:** 46 · **skip:** 1 (`Malice`).
- **new-rule terms with NO suitable heading (flagged in Notes):** 14. Each carries a
  "NO HEADING — …" note with a best nearest-parent code, so links resolve even before a
  dedicated heading is authored. These want either a new headed section or acceptance of the
  nearest-parent mapping during the annotation phase (Phase 3).
- **Shared-heading invariant:** one heading mints exactly one `new-rule` code. Where 2+ terms
  are defined under a single heading, one term stays `new-rule` (the primary) and the rest are
  `reuse` pointing at the primary's code. No two `new-rule` rows with a real heading anchor
  share the same heading line (verified).

## Collisions resolved (shared headings → one primary, rest reuse)

**One heading mints exactly one `new-rule` code.** Where 2+ terms are defined under a single
heading, the primary term stays `new-rule` and the others become `reuse` pointing at the
primary's code (the shared section defines them all, so the link is still correct). Verified:
no two `new-rule` rows with a real heading anchor share a heading line; no `new-rule` code
duplicates another; none collides with an existing `classification.json` code (no `rule.*`
codes exist yet).

- **Size / Space** share `#### Size and Space` (21323) → primary **Size** (`size`); Space reuses.
- **Bonus / Penalty** share `#### Bonuses and Penalties` (757) → primary **Bonus**
  (`bonuses-and-penalties`); Penalty reuses.
- **Edge / Double Edge** — Edge has its own `##### Edge` (727); **Double Edge** is glossary-only
  and defined in the Edge section → reuses `edge`.
- **Bane / Double Bane** — Bane has its own `##### Bane` (733); **Double Bane** is glossary-only
  → reuses `bane`.
- **Recoveries / Recovery Value** share `#### Recoveries and Recovery Value` (21971) → primary
  **Recoveries** (`recoveries`); Recovery Value reuses.
- **Ground / Ceiling** share `#### Ground and Ceiling` (4576) → primary **Ground** (`ground`);
  Ceiling reuses.
- **Tier Outcome / Tier 1 / Tier 2 / Tier 3** share `##### Power Roll Outcomes` (701) → primary
  **Tier Outcome** (`tier-outcome`); Tier 1/2/3 reuse.
- **Triggered Action / Free Triggered Action** share `#### Triggered Actions and Free Triggered
  Actions` (21410) → primary **Triggered Action** (`triggered-action`); Free Triggered Action
  reuses.
- **Saving Throw / Save Ends** share `##### Saving Throw (Save Ends)` (4544) → primary
  **Saving Throw** (`saving-throw`); Save Ends reuses.
- **Creature / Object** — Creature anchors `### Creatures and Objects` (808); **Object** reuses
  `creature` (Object Stamina at 22017 defines durability, not the term "object", and is left to
  Unattended Object).
- **Supernatural / Mundane** share `### Supernatural or Mundane` (818) → primary
  **Supernatural** (`supernatural`); Mundane reuses. (Re-anchored from Magic and Psionic
  Treasures, which does not define these terms.)
- **Orden / Timescape / Manifold** share `### Orden and the Timescape` (916) → primary
  **Orden** (`orden`); Timescape and Manifold reuse.

Distinct headings confirmed (kept separate `new-rule`, no collapse needed):

- **Heroic Resource** anchors `#### Heroic Resources` (860); **Heroic Ability** has its own
  `##### Heroic Abilities` (4221), regrouped to `rule.general/heroic-ability`.
- **Critical Hit** anchors `#### Critical Hit` (4493); **Natural 19 or 20** re-anchored to its
  own `##### Natural 19 or 20: Success With a Reward` (20442).
- **Speed** anchors `### Starting Size and Speed` (1511); **Stability** anchors its dedicated
  `##### Stability` (21689).
- **Unattended Object** keeps `#### Object Stamina` (22017) as its sole `new-rule` anchor.

## Judgment calls a reviewer should double-check

- **Movement** → mapped (`reuse`) to `movement/walk` to keep the movement family unified; could
  instead be a `rule.combat/movement` at `### Movement` (21436).
- **Artisan / Sage** → mapped to the `career/<id>` entities; the glossary defines them as
  follower roles. If a generic "follower-kind" rule is preferred, revisit alongside
  **Follower** (`#### Follower Types` 26894) / **Retainer** (`##### Retainer` 26904).
- **Dig Maneuver** → `reuse` `movement/burrow` (it is a burrow application with no common-trait
  code); alternative is a `rule.combat` maneuver.
- **Opportunity Attack** → `new-rule` at its own `#### Opportunity Attacks` (21877), even though
  it is mechanically a melee free strike.
- **Strained** → `reuse` `feature.trait.talent.level-1/clarity-and-strain` (defined by the
  talent's Clarity and Strain feature; not a general rule).
- **Reward / Downtime Project / Treasure / God / Skill / Ancestry / etc.** → mapped to their
  **chapter** codes rather than minting `rule.*`; named members already have typed codes
  (`skill/<id>`, `god/<id>`, …). If finer rule anchors are wanted, they can be minted at the
  chapter/section headings noted.
- **Common-word terms** (`Damage`, `Creature`, `Hero`, `Level`, `Object`, `Square`, `Target`,
  `Line`, `Wall`, `Side`, `Test`, `Interest`, `Round`, characteristic names): link
  conservatively / in-context during the sweep to avoid false positives.
