# Architecture

A tour of the gh-dungeons codebase for modders, contributors, and curious dungeon-delvers.

---

## Overview

`gh-dungeons` is a terminal-based roguelike written in Go that generates procedural dungeons from your repository's code. Each run creates deterministic levels based on your repo identity, commit SHA, and code file hashes.

**Tech Stack:**
- **Language:** Go 1.21+
- **Terminal UI:** [tcell/v2](https://github.com/gdamore/tcell) for rendering and input handling
- **Build:** Standard `go build`

---

## Project Structure

```
gh-dungeons/
├── main.go           # Entry point, CLI argument parsing
├── game/             # Core game logic package
│   ├── game.go       # Game struct, main loop, rendering
│   ├── state.go      # GameState, turn processing, combat, visibility
│   ├── dungeon.go    # BSP generation, Room, Tile definitions
│   ├── entity.go     # Entity struct, player/enemy/item constructors
│   ├── scanner.go    # Code file scanning, seed computation
│   └── state_test.go # Unit tests
├── go.mod / go.sum   # Go module dependencies
└── README.md         # Player-facing documentation
```

---

## Entry Points

### 1. `main.go`

The command-line interface:

```go
func main() {
    mergeMode := false
    for _, arg := range os.Args[1:] {
        if arg == "--merge" {
            mergeMode = true
        }
    }

    g, err := game.New(game.WithMergeMode(mergeMode))
    defer g.Close()
    
    g.Run()
}
```

**What it does:**
- Parses `--merge` flag (enables merge conflict marker visualization)
- Calls `game.New()` to initialize the game
- Calls `g.Run()` to start the main loop
- Handles errors and cleanup

### 2. `game.New()` in `game/game.go`

The initialization sequence:

```
game.New()
  ├─> findCodeFiles()      # Scan repo for code files (60+ lines)
  ├─> computeSeed()        # Deterministic RNG seed from repo/commit/files
  ├─> tcell.NewScreen()    # Initialize terminal UI
  └─> NewGameState()       # Create initial game state
        └─> generateLevel() # Generate first dungeon level
```

**Returns:** A `Game` struct ready to run, or an error if initialization fails.

### 3. `g.Run()` in `game/game.go`

The main game loop:

```
for {
    g.render()           # Draw dungeon, entities, UI
    g.screen.Show()      # Flip buffer to screen
    
    ev := g.screen.PollEvent()
    switch ev := ev.(type) {
    case *tcell.EventResize:
        g.state.Resize(width, height)
    case *tcell.EventKey:
        if quit key: return
        if movement key:
            g.state.MovePlayer(dx, dy)
            g.state.CheckKonamiCode(key)
    }
}
```

**Loop structure:**
- **Render phase:** Draw everything to the tcell screen buffer
- **Input phase:** Block on `PollEvent()` until a key is pressed
- **Update phase:** Process player movement, enemy turns, combat, etc.

---

## Core Systems

### Game State (`game/state.go`)

**`GameState` struct** holds all mutable game data:
- Player entity and stats
- Enemies, potions, dungeon
- Visibility maps (fog of war)
- Level tracking, message log
- RNG instance (seeded)
- Konami code sequence tracking
- Merge conflict trap state

**Key methods:**
- `MovePlayer(dx, dy)` — Movement, collision, enemy attacks, turn processing
- `processTurn()` — Auto-attack, enemy movement, visibility updates
- `updateVisibility()` — Ray casting for fog of war (vision radius = 7)
- `CheckKonamiCode(key)` — Detect the Konami code sequence

### Dungeon Generation (`game/dungeon.go`)

**`GenerateDungeon()` function:**
1. Creates a BSP tree (Binary Space Partitioning)
2. Recursively splits the map into regions (depth = 4)
3. Places rooms in leaf nodes (6-15 tiles wide/tall)
4. Connects rooms with L-shaped corridors

**Key structs:**
- `Dungeon` — Width, height, tile map, list of rooms, code file reference
- `BSPNode` — Tree node with position, size, left/right children, optional room
- `Room` — Rectangle (X, Y, W, H) with center calculation

**Tile types:**
- `TileWall` — Impassable walls (`#`)
- `TileFloor` — Walkable floor (shows code text)
- `TileDoor` — Stairs to next level (`>`)

See [dungeon-generation.md](./dungeon-generation.md) for the full algorithm.

### Entities (`game/entity.go`)

**`Entity` struct:** Generic representation of player, enemies, items.

**Constructors:**
- `NewPlayer(x, y)` — 20 HP, 2 damage, `@` symbol
- `NewBug(x, y)` — 1 HP, 1 damage, `b` symbol
- `NewScopeCreep(x, y)` — 3 HP, 2 damage, `s` symbol
- `NewPotion(x, y)` — No HP/damage, `+` symbol, heals 3 HP

**Methods:**
- `IsAlive()` — Checks if HP > 0
- `TakeDamage(dmg)` — Reduces HP
- `Heal(amount)` — Increases HP (capped at MaxHP)
- `IsAdjacent(other)` — Chebyshev distance ≤ 1

### Code Scanning (`game/scanner.go`)

**`findCodeFiles()` function:**
- Walks the repository directory tree
- Skips `.git`, `node_modules`, `vendor`, `dist`, `build`
- Filters files by extension (`.go`, `.js`, `.py`, `.rs`, etc.)
- Keeps files with ≥60 lines
- Sorts by line count (prefers longer files)
- Returns top 5 candidates

**`computeSeed()` function:**

```
SHA256(repo identity + commit SHA + file hashes) → int64 seed
```

**Repo identity resolution:**
1. Try `git config --get remote.origin.url` (unique across forks)
2. Fall back to `git rev-parse --show-toplevel` basename

**Why deterministic?**
- Same repo + commit + code = same dungeon
- Forks get different dungeons (different origin URL)
- Speedrunners can compete on the same seed

---

## Rendering Pipeline (`game/game.go`)

**`render()` function steps:**

1. **Clear screen** — `g.screen.Clear()`
2. **Calculate offsets** — Center dungeon in terminal
3. **Render tiles** — Walls, floors (with code text), doors
4. **Render potions** — If visible
5. **Render merge conflict fire** — If triggered
6. **Render enemies** — If visible
7. **Render player** — Always visible
8. **Render UI bar** — HP, level, kills, invulnerability status
9. **Render message line** — Combat log, welcome message
10. **Render end screen** — Victory or game over

**Fog of war logic:**
- Visible tiles: Full brightness, normal colors
- Explored but not visible: Dimmed (Color240)
- Unexplored: Not rendered

**Code text backgrounds:**
- Each level uses one of the scanned code files
- Floor tiles display characters from code lines
- 2x density: Uses `y*2 + x/40` to show more lines

---

## Turn Processing

**Order of operations when player moves:**

1. **Collision check** — Can't move into walls
2. **Bump-to-attack** — If enemy at destination, attack it instead of moving
3. **Move player** — Update player X, Y
4. **Item pickup** — Potions heal immediately
5. **Merge conflict check** — Deal damage if on trap
6. **Door check** — Descend or win
7. **Process turn:**
   - Auto-attack adjacent enemies
   - Enemy movement (chase AI)
   - Enemy attacks
   - Update visibility (fog of war)
8. **Death check** — Set GameOver if HP ≤ 0

---

## Build and Test Commands

**Build the game:**
```bash
go build -o gh-dungeons
```

**Run locally:**
```bash
./gh-dungeons
```

**Run with merge mode (visualize conflicts):**
```bash
./gh-dungeons --merge
```

**Run tests:**
```bash
go test ./game
```

**Install as gh CLI extension:**
```bash
gh extension install leereilly/gh-dungeons
```

---

## Dependencies

From `go.mod`:

```go
require (
    github.com/gdamore/tcell/v2 v2.7.0
)
```

**Transitive dependencies:**
- `github.com/gdamore/encoding` — Terminal encoding
- `github.com/lucasb-eyer/go-colorful` — Color manipulation
- `github.com/mattn/go-runewidth` — Unicode width calculations
- `golang.org/x/sys` — System calls
- `golang.org/x/term` — Terminal utilities
- `golang.org/x/text` — Text encoding

---

## Key Constants

From `game/state.go`:
- `VisionRadius = 7` — Fog of war sight distance

From `game/dungeon.go`:
- `MinRoomSize = 6` — Smallest room dimension
- `MaxRoomSize = 15` — Largest room dimension

**Entity spawn formulas** (in `state.go:generateLevel()`):
```go
numEnemies := 3 + gs.Level*2       // Scales with level
numPotions := 2 + gs.Level + rand(2)  // Slightly randomized
```

**BSP split depth:** 4 (hard-coded in `dungeon.go:GenerateDungeon()`)

---

## Code Style Notes

- **No external roguelike libraries** — Everything is custom
- **Deterministic where it matters** — Seeding, dungeon gen
- **tcell for portability** — Works on Linux, macOS, Windows
- **Minimal dependencies** — Only tcell and Go stdlib
- **Error handling** — Graceful fallbacks (e.g., default seed if no code files)

---

## What's Next?

- [dungeon-generation.md](./dungeon-generation.md) — Deep dive into BSP algorithm
- [entities.md](./entities.md) — All entities and their behaviors
- [seeding.md](./seeding.md) — Reproducibility and determinism
- [modding.md](./modding.md) — How to add new content
