# Architecture

A guided tour of `gh-dungeons` for contributors, modders, and curious dungeon-delvers. This document covers the code as it exists today.

---

## Overview

`gh-dungeons` is a terminal-based roguelike written in Go. It reads the code files in the current Git repository, hashes them together with the repo's remote origin URL and HEAD SHA, and uses that as the seed for a deterministic, procedurally generated 5-level dungeon. The walls of each floor are drawn from characters of a real source file in the repo.

**Tech stack (verified against `go.mod`):**

| Dependency                                      | Version   | Role                                      |
| ----------------------------------------------- | --------- | ----------------------------------------- |
| Go                                              | `1.25.5`  | Language/runtime                          |
| `github.com/gdamore/tcell/v2`                   | `v2.13.7` | Terminal rendering, input, colors         |
| `gopkg.in/yaml.v3`                              | `v3.0.1`  | Parses the embedded `monsters.yaml`       |

Indirect deps (rendering infrastructure): `gdamore/encoding`, `lucasb-eyer/go-colorful`, `rivo/uniseg`, `golang.org/x/{sys,term,text}`.

The game is distributed as a [GitHub CLI extension](https://cli.github.com/manual/gh_extension), but the binary also runs standalone as `./gh-dungeons`.

---

## Project Layout

```
gh-dungeons/
├── main.go                  # CLI entry point, --merge flag parsing
├── game/
│   ├── game.go              # Game struct, main loop, render(), exit fade animation
│   ├── state.go             # GameState: turn pipeline, combat, visibility, Konami
│   ├── dungeon.go           # BSP, Room, Tile, corridor carving, PlaceDoor
│   ├── entity.go            # Entity struct and constructors (Player, monsters, Potion)
│   ├── monster.go           # YAML monster registry, MovementType, ColorFromString
│   ├── monsters.yaml        # Embedded monster definitions (edit to add enemies)
│   ├── scanner.go           # Code file walk, seed computation, getUsername, findMergeConflict
│   └── state_test.go        # Unit tests (Konami, merge conflict, hermit crab, ...)
├── go.mod / go.sum
├── README.md                # Player-facing README
├── images/, assets/         # Screenshots, static assets
└── docs/                    # You are here
```

A single binary, no config files, no persistent state on disk.

---

## Entry Points

### 1. `main.go`

```go
func main() {
    mergeMode := false
    for _, arg := range os.Args[1:] {
        if arg == "--merge" { mergeMode = true; break }
    }
    g, err := game.New(game.WithMergeMode(mergeMode))
    ...
    defer g.Close()
    g.Run()
}
```

The CLI does exactly two things: look for `--merge` and hand off to `game.New`.

### 2. `game.New()` — `game/game.go`

Initialization order matters; this is the order on disk:

```
game.New()
  ├─ os.Getwd()
  ├─ findCodeFiles(cwd, minLines=60, maxFiles=5)   # scanner.go
  ├─ if --merge: findMergeConflict(cwd)            # scans repo for <<<<<<< markers
  ├─ computeSeed(codeFiles)                        # SHA256(repo identity ∥ HEAD ∥ file SHAs)
  │   └─ if no code files: seed = 42  (fallback)
  ├─ tcell.NewScreen() + screen.Init()
  └─ NewGameState(codeFiles, seed, w, h)
        └─ generateLevel()   # called immediately, generates level 1
```

`game.New` returns a fully initialized `*Game` with a live terminal. You **must** `defer g.Close()` or the terminal will be left in raw mode.

### 3. `g.Run()` — the main loop

```go
for {
    g.render()
    g.screen.Show()
    ev := g.screen.PollEvent()
    switch ev := ev.(type) {
    case *tcell.EventResize:
        g.screen.Sync()
        g.state.Resize(w, h)
    case *tcell.EventKey:
        // quit: Esc / Ctrl-C / q / Q
        // if game over or victory: Enter/Space exits
        // otherwise: map to (dx, dy) + konamiKey, call CheckKonamiCode + MovePlayer
    }
}
```

The loop is pull-based: `PollEvent` blocks until input arrives, then exactly one turn is processed. There is no tick thread. Nothing moves while the player is idle.

### 4. `g.Close()` — the exit fade

When the game closes, `playExitAnimation` prints a lone `@` that walks right and fades from bright grayscale (ANSI 256-color 231/255) to black (232) over 20 frames × 80ms. It writes directly to `stdout` with ANSI escape codes — it runs **after** `tcell.Screen.Fini()`, so it's deliberately outside the tcell buffer.

---

## The Render Pipeline (`game.go:render`)

Called once per loop iteration. Order matters because later writes overwrite earlier cells:

| # | Layer                          | Visibility gate                   | Notes                                                              |
| - | ------------------------------ | --------------------------------- | ------------------------------------------------------------------ |
| 1 | Tiles (wall, floor, door)      | `Explored[y][x]`                  | Unexplored tiles draw as blank spaces                              |
| 2 | Floor characters from code     | `Explored` (dim) / `Visible` (lit)| 2× line density: `lineIdx = (y*2 + x/40) % len(codeLines)`         |
| 3 | Merge-affected tiles override  | `Visible`                         | Red, cycles chars `<`, `>`, `=` via `(x + y + MergeAnimationStep)` |
| 4 | Potions                        | `Visible[potion.Y][potion.X]`     | Always drawn in `'+''`                                             |
| 5 | Merge-conflict fire spread     | `MergeConflictTriggered`          | See `merge-conflict.md`                                            |
| 6 | Enemies                        | `Visible[enemy.Y][enemy.X]`       | Drawn in `ColorRed` (not per-monster `Color` — a known wart)       |
| 7 | Player                         | always                            | Bold white `@`, even if unseen (camera always locked to player)    |
| 8 | `--merge` mode `X` marker      | always when `--merge`             | Red `X` at `findCentralRoomCenter(dungeon)`                        |
| 9 | UI bar (`height-2`)            | always                            | `HP: H/M | Level: L/M | Kills: K[ | INVULNERABLE] | [q]uit`         |
|10 | Message line (`height-1`)      | always                            | Welcome, combat, merge warning, etc.                               |
|11 | End screen overlay             | GameOver/Victory                  | Centered ASCII box, uses `getDeathMessage()`                       |

**Fog of war** has three states per tile:
- **Visible:** full color, entities drawn.
- **Explored but not currently visible:** dimmed to `tcell.Color240`, no entities.
- **Unexplored:** blank space (rendering short-circuits).

When a merge conflict has been triggered, wall colors become red (visible) / orange (fogged) instead of white/gray — a global visual cue.

---

## Turn Pipeline

A "turn" happens when the player presses a movement key and either:
1. Moves into a floor, **or**
2. Bumps an enemy (triggering an attack).

Pressing into a wall does *nothing* and does **not** consume a turn (verified in `TestMoveCounter`).

### Bump-to-attack (early return)

From `state.go:MovePlayer`:

```
check bounds & walkability → if blocked: return (no turn consumed)
for each enemy at (newX,newY):
    enemy.TakeDamage(Player.Damage)
    if dead: EnemiesKilled++; message = "You defeated the X!"
    else:    message = "You attack!"
    moveEnemies(); enemyAttacks(); updateVisibility()
    if Player dead: GameOver = true
    return   # note: MoveCount is NOT incremented on bump-attack
```

### Regular move

```
Player.X, Player.Y = newX, newY
MoveCount++
if any merge-conflict fire is active: MergeAnimationStep++
potion pickup?       → Heal(3), remove potion
MergeMarker hit?     → triggerMergeConflict() (the --merge-mode marker)
Door hit?            → Level++ or Victory; early return (no processTurn call)
processTurn()
```

### `processTurn` (`state.go`)

```
1. playerAutoAttack()         # free hit on every adjacent enemy every turn
2. checkMergeConflict()       # damage if on MergeConflictX/Y trap center
3. moveEnemies()              # respects Speed and MovementType
4. enemyAttacks()              # respects AttackRange; invulnerability short-circuits
5. updateVisibility()         # 180 rays, 2° apart
6. if OnMergeConflict: MergeConflictMovements++
7. if !Player.IsAlive(): GameOver = true
8. proximity warning: if 0 < distanceToMergeConflict() ≤ 2 and Message == "", show warning
```

Note the split: **door transitions bypass `processTurn`** entirely — descending or winning does not let enemies attack on the way out.

---

## Core Systems

### `GameState` (`state.go`)

Central god-object. Key fields, grouped:

- **World:** `Dungeon`, `Level`, `MaxLevel = 5`, `DoorX/Y`, `CodeFiles`.
- **Actors:** `Player *Entity`, `Enemies []*Entity`, `Potions []*Entity`.
- **Vision:** `Visible [][]bool`, `Explored [][]bool` (both sized `height × width`).
- **RNG:** `RNG *rand.Rand` — **single instance** seeded once; used for every randomized decision for the whole game. Changing the order of RNG consumption is how you shift every future random result.
- **Meta:** `GameOver`, `Victory`, `EnemiesKilled`, `MoveCount`, `Message`, `MessageStyle`, `Username`.
- **Konami:** `KonamiSequence []string`, `Invulnerable bool`.
- **Merge conflict (trap system):** `MergeConflictX/Y`, `OnMergeConflict`, `MergeConflictTriggered`, `MergeConflictMovements`, `MergeConflictSpread`, `ColorRotation`.
- **Merge conflict (`--merge` marker system):** `MergeConflict *MergeConflictLocation`, `MergeMarkerX/Y`, `MergeAffectedTiles map[int]bool`, `MergeAnimationStep`.
- **Death attribution:** `KilledBy string` (set to either `"merge_conflict"` or the monster's `Name`).

> ⚠️ `KilledBy` is set to the **monster Name** (e.g. `"Bug"`, `"Scope Creep"`) but `game.go:getDeathMessage` only matches lowercase underscore tokens (`"bug"`, `"scope_creep"`). In practice every monster death currently falls through to the default `"The bugs and scope creeps won..."`. See [entities.md](./entities.md) for details.

### `Dungeon` (`dungeon.go`)

`Tiles [][]Tile` (row-major, `[y][x]`), `Rooms []*Room`, optional `CodeFile`. Generated by BSP with depth 4, rooms 6×6 to 15×15, L-shaped corridors connecting siblings up the BSP tree. See [dungeon-generation.md](./dungeon-generation.md).

### `Entity` (`entity.go`)

Unified struct for player, every monster, and potions. Reading the fields is the best one-minute summary of the game's depth:

```go
type Entity struct {
    Type            EntityType
    X, Y            int
    HP, MaxHP       int
    Damage          int
    Symbol          rune
    Name            string       // from YAML; also used in messages and KilledBy
    Color           tcell.Color  // currently not used for enemy rendering — they're all red
    Speed           float64      // <1 means "moves less often than every turn"
    Movement        MovementType // straight | diagonal | any | stationary | horizontal
    AttackRange     int          // 1 = melee; 2+ = ranged (uses Chebyshev DistanceTo)
    ExperienceValue int          // XP on kill (not yet consumed by any UI/level-up system)
    Abilities       []string     // reserved — currently inert
    TurnAccumulator float64      // fractional-turn bookkeeping for slow monsters
}
```

Constructors:
- `NewPlayer(x,y)` — 20 HP, 2 damage, `'@'`, `MovementAny`.
- `NewMonsterFromDef(def, x, y)` — builds a monster from a `MonsterDef` loaded from YAML.
- `NewBug(x,y)`, `NewScopeCreep(x,y)` — **legacy constructors**, used only by tests. Production spawning goes through the YAML registry. The legacy `EntityBug` / `EntityScopeCreep` enum values exist for backwards compatibility with these tests.
- `NewPotion(x,y)` — `'+'` (no HP/Damage fields set; heal amount is hard-coded in `MovePlayer`).

Methods: `IsAlive`, `TakeDamage` (clamps at 0), `Heal` (clamps at `MaxHP`), `IsEnemy` (Monster|Bug|ScopeCreep), `DistanceTo` (Chebyshev), `IsAdjacent` (Chebyshev ≤ 1, strictly not self).

### YAML monster registry (`monster.go` + `monsters.yaml`)

`monsters.yaml` is embedded at build time (`//go:embed monsters.yaml`). On first access, `GetMonsterRegistry()` parses it into:

- `monsters map[string]MonsterDef` — lookup by name.
- `names []string` — non-unique monsters, drawn uniformly at random by `GetRandomMonster(rng)`.
- `uniqueNames []string` — monsters with `unique: true`. Exactly one of each is spawned per level.

**Spawn algorithm** (`state.go:generateLevel`):

```go
numEnemies := 3 + gs.Level*2           // Lv1=5, Lv5=13
for i := 0; i < numEnemies; i++ {
    x, y := gs.randomFloorTile()
    def := registry.GetRandomMonster(gs.RNG)   // uniform over non-unique
    gs.Enemies = append(gs.Enemies, NewMonsterFromDef(def, x, y))
}
for _, def := range registry.GetUniqueMonsters() {
    x, y := gs.randomFloorTile()
    gs.Enemies = append(gs.Enemies, NewMonsterFromDef(def, x, y))
}
```

See [monsters.md](./monsters.md) for the YAML schema and [entities.md](./entities.md) for per-monster stats.

### Code scanning & seed (`scanner.go`)

- `findCodeFiles(root, minLines=60, maxFiles=5)` — walks the repo, skipping hidden dirs and `node_modules`, `vendor`, `dist`, `build`. Filters by a 39-extension allow-list. Keeps files with ≥60 lines. Sorts by line count desc, truncates to 5.
- `computeSeed(files)` — `int64(bigEndian(SHA256(repoID || commitSHA || sha256(file1) || sha256(file2) || ...)[:8]))`. Repo identity is `git config --get remote.origin.url`, falling back to the basename of `git rev-parse --show-toplevel`.
- `getUsername()` — tries `gh api user --jq .login` (prefixed with `@`), falls back to `git config user.name`.
- `findMergeConflict(root)` — walks the repo for any file containing `<<<<<<<` … `>>>>>>>` marker pairs and returns the first one's location. Used only when `--merge` is passed (but note: this result is stored in `state.MergeConflict` and is currently **not rendered or referenced anywhere else** in the running game; it exists as data only). The visible `--merge` marker is the red `X` placed at the central room's center.

See [seeding.md](./seeding.md) for the full reproducibility story.

### Visibility (`state.go:updateVisibility`, `castRay`)

180 rays at 2° intervals (angles 0, 2, 4, …, 358) are cast from the player's position up to `VisionRadius = 7` tiles. Each ray marches one unit per step, rounds to the nearest cell, marks `Visible` and `Explored`, and terminates on hitting a wall.

**Unusual detail:** trigonometry is implemented from scratch via Taylor series in `state.go:sin`, `cos`, and `mod2pi`. No `math.Sin` import. This is deterministic and platform-independent (not that Go's `math` package isn't, but it makes the intent explicit).

### Konami code (`state.go:CheckKonamiCode`)

Sequence: `up up down down left right left right b a`. `KonamiSequence` is appended per movement keypress, truncated to last 10 entries, and compared as a whole. **Only arrow keys count for directions** — WASD and vi keys are not Konami-eligible (see `game.go:Run`: `konamiKey` is only set for arrow keys and the `a`/`b` runes). Activating sets `Invulnerable = true`; there is no toggle-off.

---

## Build, Run, Test

```bash
go build -o gh-dungeons          # produces the binary in repo root
./gh-dungeons                    # run standalone
./gh-dungeons --merge            # run with merge-conflict marker visible
go test ./game -count=1          # run all unit tests (no network, no git required)
gh extension install leereilly/gh-dungeons   # install as a gh CLI extension
gh dungeons                      # run as a gh subcommand
```

The tests live in `game/state_test.go` and do not require a terminal (they construct `GameState` directly, bypassing `tcell`).

---

## Known Quirks (not bugs, just personality)

- **`KilledBy` case mismatch** — enemy deaths always yield the default "The bugs and scope creeps won..." message. See [entities.md](./entities.md#death-attribution).
- **Two merge-conflict systems** share the `Merge*` prefix but do different things. See [merge-conflict.md](./merge-conflict.md).
- **`Entity.Color` is ignored at render time** — all enemies draw as `tcell.ColorRed` regardless of YAML color.
- **`ExperienceValue` and `Abilities`** are parsed from YAML but no code currently reads them.
- **Door transitions skip `processTurn`** — enemies get no free swing as you descend.
- **`MergeConflict *MergeConflictLocation`** (the repo-scanned conflict metadata under `--merge`) is stored but not displayed.

Each of these is an opportunity for a PR.

---

## Further Reading

- [dungeon-generation.md](./dungeon-generation.md) — BSP in detail
- [entities.md](./entities.md) — Player, monsters, combat, Konami, death attribution
- [monsters.md](./monsters.md) — YAML schema reference
- [merge-conflict.md](./merge-conflict.md) — Both merge-conflict subsystems
- [seeding.md](./seeding.md) — Determinism and reproducibility
- [modding.md](./modding.md) — How to add content
