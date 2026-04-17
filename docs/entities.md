# Entities

Every tile-occupying thing in the game: the player, the monsters (all YAML-driven), the potions, and the quasi-entity door. For the YAML schema specifically, see [monsters.md](./monsters.md).

---

## The `Entity` Struct

Every actor uses the same struct. From `game/entity.go`:

```go
type Entity struct {
    Type            EntityType
    X, Y            int
    HP, MaxHP       int
    Damage          int
    Symbol          rune
    Name            string       // display name and KilledBy value
    Color           tcell.Color  // currently unused at render time (see "quirks")
    Speed           float64      // <1.0 means "moves less often than once per turn"
    Movement        MovementType // straight|diagonal|any|stationary|horizontal
    AttackRange     int          // 1 = melee, 2+ = ranged (Chebyshev)
    ExperienceValue int          // currently inert — no XP/level-up system
    Abilities       []string     // currently inert — parsed but no consumer
    TurnAccumulator float64      // fractional-turn accumulator for slow (Speed<1) enemies
}

type EntityType int
const (
    EntityPlayer     EntityType = iota
    EntityMonster
    EntityPotion
    EntityBug         // legacy, used only by tests
    EntityScopeCreep  // legacy, used only by tests
)
```

**Methods:**

| Method                        | Definition                                               |
| ----------------------------- | -------------------------------------------------------- |
| `IsAlive() bool`              | `HP > 0`                                                 |
| `TakeDamage(dmg int)`         | `HP -= dmg`; clamped to `0`                              |
| `Heal(amount int)`            | `HP += amount`; clamped to `MaxHP`                       |
| `IsEnemy() bool`              | `Type ∈ {Monster, Bug, ScopeCreep}`                      |
| `DistanceTo(other) int`       | **Chebyshev** `max(|dx|, |dy|)`                          |
| `IsAdjacent(other) bool`      | Chebyshev ≤ 1, excluding self (must differ in ≥1 axis)   |

---

## The Player

- Symbol: `@` (bold white, always rendered on top).
- HP: 20 / MaxHP: 20.
- Damage: 2.
- Movement: `MovementAny` (8-directional, no speed penalty).
- Spawn: center of `Rooms[0]` on every new level.

**Controls** (`game.go:Run`):

| Direction      | Keys                                |
| -------------- | ----------------------------------- |
| Up             | `↑` / `k` / `w`                     |
| Down           | `↓` / `j` / `s`                     |
| Left           | `←` / `h` / `a`                     |
| Right          | `→` / `l` / `d`                     |
| Diagonal UL    | `y`                                 |
| Diagonal UR    | `u`                                 |
| Diagonal DL    | `b`                                 |
| Diagonal DR    | `n`                                 |
| Quit           | `q` / `Q` / `Esc` / `Ctrl-C`        |
| Dismiss end-screen | `Enter` / `Space`                |

**Special abilities:**

- **Bump-to-attack.** Moving into an adjacent enemy attacks instead of moves; does not increment `MoveCount`.
- **Auto-attack.** Every `processTurn` hits every adjacent enemy for `Player.Damage` — free, guaranteed, no miss chance.
- **Konami.** See below.

---

## Combat System

Combat has no misses, no crits, no armor. It's stat subtraction.

### Player → enemy

Two sources:

1. **Bump attack** (from `MovePlayer`): single hit on the specific enemy in the target tile.
2. **Auto-attack** (from `processTurn → playerAutoAttack`): hits **every** adjacent enemy, every turn.

```go
// state.go:playerAutoAttack
for _, enemy := range gs.Enemies {
    if enemy.IsAlive() && gs.Player.IsAdjacent(enemy) {
        enemy.TakeDamage(gs.Player.Damage)
        if !enemy.IsAlive() {
            gs.EnemiesKilled++
            gs.SetMessage(fmt.Sprintf("You defeated the %s!", enemy.Name))
        }
    }
}
```

> ⚠️ Bump-attack calls `moveEnemies` + `enemyAttacks` directly and returns before `processTurn`. So bumping doesn't also trigger your auto-attack pass that turn. The single bump *is* your attack.

### Enemy → player (`state.go:enemyAttacks`)

```go
if gs.Invulnerable { return }   // Konami short-circuits the whole pass
for _, enemy := range gs.Enemies {
    if !enemy.IsAlive() { continue }
    attackRange := enemy.AttackRange
    if attackRange <= 0 { attackRange = 1 }   // default melee
    if enemy.DistanceTo(gs.Player) <= attackRange {
        gs.Player.TakeDamage(enemy.Damage)
        gs.Message = fmt.Sprintf("A %s attacked - %d HP damage", enemy.Name, enemy.Damage)
        gs.MessageStyle = redBold
        if !gs.Player.IsAlive() { gs.KilledBy = enemy.Name }
    }
}
```

Note `AttackRange > 1` enables ranged attacks without projectile animation — a distant archer-type monster can hit through fog of war. Currently nothing in `monsters.yaml` uses range > 1, but the mechanic is live.

### Enemy AI (`state.go:moveEnemies`)

```
for each living enemy:
    if Speed < 1:
        TurnAccumulator += Speed
        if TurnAccumulator < 1: skip this turn
        TurnAccumulator -= 1
    if !hasLineOfSight(enemy, player): skip this turn  (enemies are dormant unless they see you)

    compute desired (dx, dy) stepping toward the player  (sign of Player.X - enemy.X, etc.)

    switch Movement:
      MovementStationary: continue (never moves, but can still attack at range)
      MovementStraight:   zero-out the shorter axis (Manhattan-dominant direction)
      MovementDiagonal:   if dx==0 or dy==0: skip turn (can only move diagonally)
      MovementHorizontal: dy = 0; if dx==0: skip
      MovementAny:        no restriction (prefer diagonal)

    Try (x+dx, y+dy). If blocked, fall back axis-by-axis (honoring the movement type).
    If the enemy is currently standing in merge-conflict fire, it takes 1 damage.
```

Key details:
- **Line-of-sight gate**: enemies only act when `hasLineOfSight(enemy, player)` returns true. This is a fast integer ray march through `Tiles`; it does not consider other entities as blockers.
- **Speed**: fractional. `Speed: 0.5` = moves every other turn. `Speed: 0.25` = every fourth. `Speed: 1.0` is the default; `>1.0` is legal but currently not used by any monster and would act every turn (the code only gates when `Speed < 1.0`).
- **Movement** restrictions are enforced both on the desired move and on the fallback single-axis moves.
- **Merge-conflict fire** damages enemies inside its area every time they act. This is intentional and is the only way to thin out a crowd that's camping the trap.

---

## Monsters (as of current `monsters.yaml`)

All spawns come from the YAML registry. The table below is the shipped set; see [monsters.md](./monsters.md) for the full schema.

| Name         | Token | Color  | HP | Str | Speed | Movement    | Range | XP | Unique | Description                                         |
| ------------ | ----- | ------ | -- | --- | ----- | ----------- | ----- | -- | ------ | --------------------------------------------------- |
| Bug          | `b`   | red    | 1  | 1   | 1.0   | any         | 1     | 5  | –      | Classic one-shot roguelike fodder                   |
| Scope Creep  | `s`   | yellow | 3  | 2   | 1.0   | any         | 1     | 10 | –      | Tankier, hits harder, named after the anti-pattern  |
| Zombie       | `z`   | green  | 3  | 2   | 0.5   | straight    | 1     | 15 | –      | Slow, cardinal-only, scary in corridors             |
| Hermit Crab  | `H`   | red    | 2  | 2   | 1.0   | horizontal  | 1     | 20 | ✓      | Sidles east/west only; never climbs                 |

All enemies currently render as red on screen regardless of the `Color` field — that's a render-layer wart, not a YAML issue.

### Spawn counts per level (`state.go:generateLevel`)

```go
numEnemies := 3 + gs.Level*2   // uniform random over non-unique YAML entries
// plus one of each unique monster
numPotions := 2 + gs.Level + rng.Intn(2)
```

| Level | Random enemies | + Uniques      | Potions |
| ----- | -------------- | -------------- | ------- |
| 1     | 5              | +1 Hermit Crab | 3–4     |
| 2     | 7              | +1 Hermit Crab | 4–5     |
| 3     | 9              | +1 Hermit Crab | 5–6     |
| 4     | 11             | +1 Hermit Crab | 6–7     |
| 5     | 13             | +1 Hermit Crab | 7–8     |

The random pool is uniform across all non-unique YAML entries (currently Bug, Scope Creep, Zombie), so each has ≈1/3 weight. There is no per-monster spawn weighting; to tune rarity, duplicate entries or add a weight field and plumb it through `monster.go`.

---

## Items

### Health Potion (`+`)

Not really an `Entity` in the combat sense — `NewPotion` only sets `Type`, `X/Y`, and `Symbol`. The heal amount is hard-coded in `MovePlayer`:

```go
gs.Player.Heal(3)
gs.Potions = append(gs.Potions[:i], gs.Potions[i+1:]...)
gs.SetMessage("You drink a health potion! (+3 HP)")
```

Pickup is automatic on stepping onto the tile. Max HP is still 20, so overheal is wasted.

---

## The Door / Stairs (`>`)

The door is a **tile**, not an entity — it's a `TileDoor` in `Dungeon.Tiles`. Placed by `Dungeon.PlaceDoor` in the last-generated room on every level.

Triggering (from `MovePlayer`):
- If `Level >= MaxLevel` (5): `Victory = true`, end screen.
- Otherwise: `Level++`, `generateLevel()`, `"You descend deeper into the dungeon..."`.

**Door transitions bypass `processTurn`.** Enemies don't get a free hit as you step on the stairs.

---

## The Merge Conflict Trap

There are actually **two** merge-conflict subsystems. This section covers the gameplay-facing one; see [merge-conflict.md](./merge-conflict.md) for the full story.

### The fire trap (`MergeConflictX/Y`)

- Placed by `generateLevel` on a random floor tile, every level.
- No marker on the map until you trigger it.
- Stepping on it:
  - Sets `OnMergeConflict = true`, `MergeConflictTriggered = true`.
  - Generates a 7-tile "fire spread" around the core 5×3 conflict pattern.
  - Deals 1 HP/turn while you stand on the center tile (unless invulnerable).
- Moving off the center keeps `MergeConflictTriggered` true forever — walls turn red, the fire keeps animating, and any enemies that step into the area take 1 damage.
- Proximity warning: when the player is within Chebyshev distance 2 of the trap center and no other message is set, the UI shows `"WARNING: MERGE CONFLICT DETECTED. TREAD CAREFULLY."`.

### The `--merge`-mode marker (`MergeMarkerX/Y`)

Only active when the game is launched with `--merge`. A red `X` is drawn at the central room's center and stepping on it calls `triggerMergeConflict`, which deals **2** damage immediately and marks a 3×3 area as "merge-affected" (tiles animate conflict characters for the rest of the level). Not to be confused with the random-tile fire trap above.

The two systems coexist in `GameState` but have independent state; see the dedicated doc for an untangling.

---

## Konami Code

Sequence: **↑ ↑ ↓ ↓ ← → ← → b a**

Implementation (`state.go:CheckKonamiCode`):

```go
konamiCode := []string{"up","up","down","down","left","right","left","right","b","a"}
gs.KonamiSequence = append(gs.KonamiSequence, key)
if len(gs.KonamiSequence) > 10 {
    gs.KonamiSequence = gs.KonamiSequence[len(gs.KonamiSequence)-10:]
}
if match && !gs.Invulnerable {
    gs.Invulnerable = true
    gs.SetMessage("KONAMI CODE ACTIVATED! You are now invulnerable!")
}
```

The driver in `game.go:Run` only assigns `konamiKey` for **arrow keys** and the literal runes `a` and `b`. So WASD-left does not count as "left" for Konami purposes, and vi-keys `h j k l` don't contribute either. You must use the arrow cluster plus the letter keys.

Effect: `enemyAttacks` early-returns, so no damage from monsters. The merge-conflict fire trap also checks `Invulnerable` and shows `"The merge conflict burns around you, but your invulnerability protects you!"` instead of dealing damage. The `--merge` marker's `triggerMergeConflict` also respects invulnerability. Konami is a full damage shield.

There is no toggle-off. Activation is a one-way trip.

---

## Death Attribution

`KilledBy` is set in two places:

- `state.go:enemyAttacks` — `gs.KilledBy = enemy.Name` (e.g., `"Bug"`, `"Zombie"`).
- `state.go:checkMergeConflict` — `gs.KilledBy = "merge_conflict"` (only from the fire trap; the `--merge` marker's `triggerMergeConflict` does **not** set `KilledBy`).

`game.go:getDeathMessage` then switches:

```go
switch g.state.KilledBy {
case "bug":            return "In GitHub Dungeons... bug squashes YOU"
case "merge_conflict": return fmt.Sprintf("Death by merge conflict. Just a typical %s.", time.Now().Weekday())
case "scope_creep":    return "Foiled by scope creep again!"
default:               return "The bugs and scope creeps won..."
}
```

> ⚠️ **Known mismatch.** The switch expects lowercase `"bug"` / `"scope_creep"`, but `enemyAttacks` writes the monster's proper-case `Name` (`"Bug"`, `"Scope Creep"`, `"Zombie"`, `"Hermit Crab"`). Every monster death therefore falls through to the default message. Only merge-conflict trap deaths render the themed message.
>
> If you fix this, the least-invasive approach is to normalize: `strings.ToLower(strings.ReplaceAll(gs.KilledBy, " ", "_"))`. Or switch directly on `gs.KilledBy == "Bug"` etc. Either works; the docs should be updated to match.

---

## Visibility (Fog of War)

From `state.go:updateVisibility` + `castRay`:

- `VisionRadius = 7` tiles.
- 180 rays cast at 2° intervals per turn.
- Each ray advances 1 unit per step, rounds to nearest cell, marks `Visible` and `Explored`, terminates on `TileWall`.
- Trigonometry is hand-rolled (`sin`, `cos`, `mod2pi` in `state.go`) using Taylor series. Platform-deterministic, no `math.Sin` dependency.

In the renderer:
- Visible tiles → normal colors, entities drawn.
- Explored but not currently visible → dimmed (`tcell.Color240`), entities hidden.
- Unexplored → blank.

---

## Quirks & Dead Code

- **`Entity.Color`** — parsed from YAML, stored on the entity, **never read at render time.** `game.go:render` uses a single `enemyStyle` (`ColorRed`). This is an obvious short PR opportunity.
- **`Entity.ExperienceValue`, `Entity.Abilities`** — plumbed through YAML → registry → entity, but no game logic consumes them. No XP, no status effects (yet).
- **Legacy `NewBug` / `NewScopeCreep` / `EntityBug` / `EntityScopeCreep`** — exist solely for existing unit tests. Production spawning goes exclusively through `NewMonsterFromDef`. Do not add new hardcoded constructors; add YAML entries instead.
- **`TurnAccumulator`** reset-on-level behavior — enemies are recreated every level, so accumulators reset implicitly. If you ever persist enemies across levels you'll need to reset explicitly.
- **`MergeConflict` field** (the `*MergeConflictLocation` loaded under `--merge`) is stored but currently unused — `findMergeConflict` runs, populates the state, and no game code queries it. Latent feature.

---

## Further Reading

- [monsters.md](./monsters.md) — YAML schema reference
- [merge-conflict.md](./merge-conflict.md) — Both merge-conflict subsystems in detail
- [modding.md](./modding.md) — Step-by-step mods, YAML-first
- [architecture.md](./architecture.md) — Where entities fit in the loop
