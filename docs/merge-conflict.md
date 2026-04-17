# Merge Conflict Systems

`gh-dungeons` contains **two distinct merge-conflict subsystems** that share a naming prefix. They do different things and are triggered in different ways. This document untangles them.

---

## TL;DR

| System                                     | Active when   | Triggered by                              | Damage            | Placed at                         | Visible?           |
| ------------------------------------------ | ------------- | ----------------------------------------- | ----------------- | --------------------------------- | ------------------ |
| **Fire trap** (`MergeConflict{X,Y}`)       | always        | stepping on center tile; ambient damage   | 1/turn on center  | random floor tile (per level)     | invisible until triggered |
| **`--merge`-mode marker** (`MergeMarker{X,Y}`) | `--merge` flag | stepping on marker tile (`triggerMergeConflict`) | 2 on trigger      | center of most central room       | red `X` always visible |

Plus a third, nearly-unused element:
- **`findMergeConflict(cwd)`** scans the current repo for real `<<<<<<<`/`>>>>>>>` markers and stores the location on `state.MergeConflict`. Currently this result is captured but **not displayed or used** at runtime.

---

## System 1: The Fire Trap

### Fields

In `GameState`:

```go
MergeConflictX, MergeConflictY int        // where the trap lives this level
OnMergeConflict        bool               // player is currently on the center tile
MergeConflictTriggered bool               // trap has been hit at some point (persists for rest of level)
MergeConflictMovements int                // turns the player has stood on the center
MergeConflictSpread    [][2]int           // additional fire tiles spread around the core pattern
ColorRotation          int                // rotates the core pattern's red/orange/yellow palette
```

### Placement

From `state.go:generateLevel`:

```go
gs.MergeConflictX, gs.MergeConflictY = gs.randomFloorTile()
gs.OnMergeConflict = false
```

One trap per level, on a random floor tile, not colocated with the player or the stairs. **The tile looks identical to surrounding floor — you cannot see the trap before stepping on it.**

### Trigger and damage (`state.go:checkMergeConflict`)

Called every turn from `processTurn`.

```
if player is on (MergeConflictX, MergeConflictY):
    if this is the first step onto it:
        OnMergeConflict = true
        MergeConflictTriggered = true
        MergeConflictMovements = 0
        generateMergeConflictSpread()   # pick 7 adjacent walkable tiles randomly
    ColorRotation++
    if not Invulnerable:
        Player.TakeDamage(1)
        Message = "- 1 HP damage" (red, bold)
        if dead: KilledBy = "merge_conflict"
    else:
        Message = "The merge conflict burns around you, but your invulnerability protects you!"

else if MergeConflictTriggered:
    ColorRotation++                 # keep the fire animation rotating
    if OnMergeConflict and player not in conflict area: OnMergeConflict = false

else:
    OnMergeConflict = false
```

### Fire spread (`state.go:generateMergeConflictSpread`)

The core fire pattern occupies a 5-wide × 3-tall area centered on `(MergeConflictX, MergeConflictY)`. When the trap triggers, the engine finds all walkable tiles **adjacent** to that 5×3 area, shuffles them, and keeps up to 7.

These spread tiles:
- Are animated as conflict chars (`<`, `>`, `=`) in red/orange/yellow.
- Burn enemies that step into them (1 damage per action the enemy takes while inside; applied in `moveEnemies`).
- Persist for the rest of the level.

`isInMergeConflictArea(x, y)` returns true if `(x, y)` is in the 5×3 core **or** in the spread list.

### Animation

The rendered pattern cycles with `MergeConflictMovements`:

| `Movements` | Row 0    | Row 1    | Row 2    |
| ----------- | -------- | -------- | -------- |
| 0           | `<<<<<`  | `=====`  | `>>>>>`  |
| 1           | `>>>>>`  | `<<<<<`  | `=====`  |
| 2           | `=====`  | `>>>>>`  | `<<<<<`  |
| ≥3          | randomized from `<`, `>`, `=` per cell each frame |

The palette `[red, orange, yellow]` is rotated by `ColorRotation % 3`, and within the pattern each cell picks its color via `(x + y) % 3` — so the fire looks like it's wiggling, not strobing.

### Proximity warning

From `state.go:processTurn`:

```go
distance := gs.distanceToMergeConflict()   // Chebyshev
if distance <= 2 && distance > 0 && gs.Message == "" {
    gs.SetMessage(MergeConflictWarning) // "WARNING: MERGE CONFLICT DETECTED. TREAD CAREFULLY."
}
```

This is your only pre-trigger heads-up.

### Global side-effect: red walls

Once `MergeConflictTriggered` is true, `game.go:render` switches wall rendering from white/gray to red/orange for the rest of the level. Every wall on the map turns red — thematic and hard to miss.

---

## System 2: The `--merge`-mode Marker

Activated only when the user launches with `--merge`. This system is about explicitly *showing* a merge conflict, not about being surprised by one.

### Fields

In `GameState`:

```go
MergeConflict       *MergeConflictLocation  // repo-scanned conflict info (currently unused at runtime)
MergeMarkerX        int                     // -1 if none
MergeMarkerY        int
MergeAffectedTiles  map[int]bool            // key: y*width + x
MergeAnimationStep  int                     // increments with each player move while fire active
```

### Placement (`state.go:generateLevel`)

```go
gs.MergeMarkerX, gs.MergeMarkerY = findCentralRoomCenter(gs.Dungeon)
gs.MergeAffectedTiles = make(map[int]bool)
```

Unlike System 1, this marker is drawn every frame (under `--merge`) as a red bold `X` at `findCentralRoomCenter`.

### Trigger (`state.go:triggerMergeConflict`)

Called from `MovePlayer` when the player steps on `(MergeMarkerX, MergeMarkerY)`:

```go
if !Invulnerable: Player.TakeDamage(2)
SetMessage("MERGE CONFLICT! The code tears apart around you!")
for dy in -1..1, dx in -1..1:
    MergeAffectedTiles[key(ax, ay)] = true   # 3x3 around marker
if !Player.IsAlive(): GameOver = true; SetMessage("You died in a merge conflict!")
```

Differences from System 1:
- Deals **2 damage up front** instead of 1/turn.
- Doesn't set `KilledBy` — a death here falls to the default death message.
- Marks a **3×3** area as "affected" (stored as `int` keys in a map, not the `(int,int)` slice that System 1 uses).
- The affected tiles are animated by `render`: any such tile is drawn in red with cycling `<`, `>`, `=` using `MergeAnimationStep`.
- No fire spread around the area; no per-turn ambient damage; enemies walking through are not harmed.

### Pre-trigger warning

From `game.go:render` (inside `if g.mergeMode && ... len(MergeAffectedTiles) == 0`):

```go
if |Player.X - MergeMarkerX| <= 2 && |Player.Y - MergeMarkerY| <= 2 {
    show "WARNING: Merge conflict detected" in red
}
```

Note: this warning is separate from the `MergeConflictWarning` string System 1 uses. Two different warning messages for two different systems.

---

## Why two systems?

History. The fire trap (System 1) was the original mechanic — a hidden trap, ambient damage, level-wide red-walls dread. The `--merge` flag (System 2) was added later to visualize merge conflicts explicitly — presumably as a demo or teaching tool for the "your code files become the dungeon" theme. They were implemented side-by-side rather than unified, and the shared `Merge*` prefix hides how independent they are.

Consolidation is an obvious refactor opportunity, but:
- System 1 is always on; System 2 requires a flag.
- The two have different damage, different spread models, different animations.
- Removing either changes gameplay.

If you unify them, be prepared to update tests (`TestMergeConflictDamage`, `TestMergeConflictIntegration`, etc.) and rewrite this doc.

---

## System 3 (sort of): Real repo conflict scanning

When `--merge` is passed, `main.go` → `game.New` calls `findMergeConflict(cwd)` (`scanner.go`):

```go
walk cwd (skip .git, node_modules, vendor, dist, build)
for each file, scan lines:
    first <<<<<<< — record startLine
    first >>>>>>> after a start — record endLine, build MergeConflictLocation{File, StartLine, EndLine, CenterLine}
    return immediately on first conflict found
```

The result is stored on `GameState.MergeConflict` and **never read** by any render or gameplay code.

This is a latent feature. A future version could display the `File:CenterLine` under the `--merge` warning, teleport the marker to a tile representing that line, or drive the level's code-background file selection. None of that is wired today.

---

## Invulnerability interaction

- System 1 (fire trap): respects `Invulnerable`. Shows a flavor message instead of taking damage.
- System 2 (`--merge` marker): respects `Invulnerable`. Skips the `TakeDamage(2)` call entirely (the `SetMessage("MERGE CONFLICT!...")` still fires, as does the 3×3 affected-tile marking — so the map still lights up).
- Enemies standing in System 1's fire (`moveEnemies`): take 1 damage regardless of player state; there is no enemy-invulnerability concept.

---

## Testing

See `state_test.go`:
- `TestMergeConflictProximity` — Chebyshev distance check.
- `TestMergeConflictDamage` — 1-damage trigger.
- `TestMergeConflictNoDamageWhenNotOnTrap` — passive behavior.
- `TestMergeConflictInvulnerability` — invulnerable flag short-circuits damage.
- `TestMergeConflictIntegration` — end-to-end in a real `NewGameState`.
- `TestEnemyDamageByMergeConflictFire` / `TestEnemyNotDamagedWhenMergeConflictNotTriggered` — enemy damage inside fire area.

All of these target System 1. System 2 currently has no dedicated tests.

---

## See Also

- [entities.md](./entities.md) — Player, invulnerability, death attribution
- [architecture.md](./architecture.md) — Turn pipeline
- [modding.md](./modding.md) — Adding new traps
