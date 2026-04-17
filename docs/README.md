# Documentation Index

Welcome to the `gh-dungeons` technical docs. If the [top-level README](../README.md) is the tavern-side pitch, this directory is the wizard's workshop: annotated spellbooks, receipts for the reagents, and maps of the crypts beneath the code.

These docs describe what the code **actually does** today — not what an older draft thought it would do. Citations (`file.go:Function`) are sprinkled throughout so you can grep from here straight into the source.

---

## Documentation Files

### [architecture.md](./architecture.md)
A system-wide tour: entry points (`main.go` → `game.New` → `g.Run`), the render loop, turn pipeline, the `GameState` god-object, the embedded YAML monster registry, the two coexisting merge-conflict systems, the farewell `@` fade animation, and the actual dependency tree (`tcell/v2`, `yaml.v3`).

**Start here** if you want to know how a keystroke becomes a dead `z`.

### [dungeon-generation.md](./dungeon-generation.md)
Binary Space Partitioning in painful detail: the 1.25 aspect-ratio heuristic, room placement, post-order L-shaped corridor carving, `findCentralRoomCenter` (for merge-marker placement), `findNearestFloorTile`'s BFS, and the `y*2 + x/40` code-text background trick.

### [entities.md](./entities.md)
Everything that occupies a tile. The `Entity` struct (and all its YAML-fed fields), the player, every monster currently shipped (Bug, Scope Creep, Zombie, Hermit Crab), movement types, speed/turn-accumulator math, attack range, the `Invulnerable` / Konami path, and the subtle `KilledBy` / `getDeathMessage` case-matching bug.

### [monsters.md](./monsters.md)
Reference for `game/monsters.yaml`: every field, every legal value, how `unique: true` works, and how to add a new monster without writing a line of Go.

### [merge-conflict.md](./merge-conflict.md)
The single most confusing subsystem in the codebase, documented honestly: the random-floor-tile fire trap *and* the `--merge`-mode central-room marker are **two different systems** sharing prefixes. Also: the animated pattern state machine and the `findMergeConflict` repo scan.

### [seeding.md](./seeding.md)
How a 64-bit seed is brewed from the remote origin URL, HEAD SHA, and the SHA256s of the top 5 longest code files. Includes the reproducibility checklist and a note on Go's `math/rand` determinism.

### [modding.md](./modding.md)
From "add a monster in 30 seconds of YAML" all the way to "invent a new tile type and status effect." YAML-first, Go-second, with pitfalls about determinism and the legacy entity types.

---

## Quick Start by Goal

| I want to...                                   | Go to                                                              |
| ---------------------------------------------- | ------------------------------------------------------------------ |
| Understand the code at a glance                | [architecture.md](./architecture.md)                               |
| Add a new monster                              | [modding.md](./modding.md) → "Add a monster (YAML)"                |
| Understand how dungeons are shaped             | [dungeon-generation.md](./dungeon-generation.md)                   |
| Tune spawn rates or difficulty                 | [entities.md](./entities.md) → "Spawn formulas"                    |
| Understand why my dungeon is *this* dungeon    | [seeding.md](./seeding.md)                                         |
| Figure out what `--merge` actually does        | [merge-conflict.md](./merge-conflict.md)                           |
| Learn the Konami code                          | [entities.md](./entities.md) → "Konami code"                       |

---

## Conventions used in these docs

- **Citations:** `path/file.go:Thing` means "look at `Thing` (function/type/const) in that file."
- **ASCII diagrams** are preferred over images because this is a roguelike and we have standards.
- **Honest notes** — where the implementation has a quirk, dead code, or latent bug, we call it out rather than paper over it.

---

## Contributing Documentation

Found drift between these docs and the code? That's a bug. Open a PR. Guidelines:

- Cite code (file + function name) so the reader can verify.
- Don't invent features. If you want the docs to describe a behavior, implement the behavior first.
- Prefer ASCII diagrams for anything spatial.
- Keep the tone witty but precise.

---

*"Read the dungeon. Then write the scrolls."* — Dungeon Scribe
