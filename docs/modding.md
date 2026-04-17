# Modding Guide

How to extend `gh-dungeons`. The codebase is small (~2k lines), the monster list is YAML, and Go test coverage is friendly to iteration. If you can read Go, you can mod this game.

**Priority order for mods:**

1. **YAML first.** Adding, tuning, or removing monsters should always be a YAML edit.
2. **Stats and constants next.** Room sizes, spawn formulas, vision radius — one-line changes.
3. **Go code last.** New tile types, status effects, new item categories, UI overhauls.

---

## Quick Reference

| What to change                | Where                                            |
| ----------------------------- | ------------------------------------------------ |
| Add a monster                 | `game/monsters.yaml`                             |
| Adjust enemy counts per level | `game/state.go:generateLevel`                    |
| Adjust potion counts          | `game/state.go:generateLevel`                    |
| Change room sizes             | `game/dungeon.go` → `MinRoomSize`, `MaxRoomSize` |
| Change BSP depth              | `game/dungeon.go:GenerateDungeon` → `root.Split(rng, 4)` |
| Change vision radius          | `game/state.go` → `VisionRadius`                 |
| Change player stats           | `game/entity.go:NewPlayer`                       |
| Add a new item type           | `game/entity.go` + `game/state.go` (pickup) + `game/game.go` (render) |
| Change combat damage model    | `game/entity.go:TakeDamage`                      |
| Add a new tile type           | `game/dungeon.go` (constants) + `game/game.go:render` |
| Add a status effect           | `game/entity.go` (fields) + `game/state.go` (apply/tick) |
| Add a key binding             | `game/game.go:Run` (input switch)                |

---

## Philosophy: Keep It Deterministic

The game's charm is that your repo produces *this* dungeon, every time. Most mod breakages come from accidentally nondeterministic code.

**Do:**
- Always use `gs.RNG` for randomness. Never `math/rand.Float32()` directly.
- Append new RNG consumers at the **end** of `generateLevel`, not in the middle. Middle-insertions shift every subsequent random result for existing seeds.
- Keep YAML entry order stable unless you actually want the random pool to shuffle.

**Don't:**
- Don't read wall-clock time into gameplay (`time.Now().UnixNano()` into RNG is a classic determinism killer). The exit fade animation uses real time — that's fine because it runs after the game ends.
- Don't use goroutines for any game-state-affecting work. The event loop is single-threaded by design.

---

## Mod 1: Add a Monster (YAML)

This is the easiest kind of mod. Open `game/monsters.yaml` and append:

```yaml
  - name: "Tech Debt"
    description: "Interest compounds. Pay it down before it pays you a visit."
    token: "T"
    color: "purple"       # NOTE: color is parsed but currently unused at render time
    hp: 5
    strength: 1
    speed: 0.5            # moves every other turn
    movement: "any"
    attack_range: 1
    experience_value: 25
    abilities: []
```

Rebuild:

```bash
go build -o gh-dungeons
```

That's it. Tech Debt joins the random-spawn pool at uniform weight vs other non-unique monsters. See [monsters.md](./monsters.md) for the full schema.

**To make it a boss-type that always appears once per level**, add:

```yaml
    unique: true
```

to the entry. Unique monsters skip the random pool and spawn one per level on top of the random spawns.

### Common YAML gotchas

- **You must rebuild.** `monsters.yaml` is embedded at compile time (`//go:embed`).
- **Invalid YAML ⇒ empty registry.** The game falls back to a sentinel `Bug`. If spawns look uniform and broken, check YAML syntax.
- **`movement` must be one of**: `any`, `straight`, `diagonal`, `horizontal`, `stationary`. Unknown values behave like `any` but without explicit fallbacks — don't rely on this.
- **`color` is ignored at render time** today; every enemy draws red regardless. If you fix this, the right place is `game/game.go:render` where `enemyStyle` is defined.

### Test your monster

Add to `game/state_test.go`:

```go
func TestTechDebtIsPurpleAndSlow(t *testing.T) {
    registry := GetMonsterRegistry()
    def, ok := registry.GetMonster("Tech Debt")
    if !ok { t.Fatal("Tech Debt not registered") }
    if def.Speed != 0.5 { t.Errorf("expected speed 0.5, got %v", def.Speed) }
    if def.Color != "purple" { t.Errorf("expected purple, got %s", def.Color) }
}
```

Run:

```bash
go test ./game -count=1 -run TestTechDebt
```

---

## Mod 2: Tune Spawn Rates

All spawn logic lives in `game/state.go:generateLevel`. Relevant lines:

```go
numEnemies := 3 + gs.Level*2
...
numPotions := 2 + gs.Level + gs.RNG.Intn(2)
```

### Denser enemies, sparser potions

```go
numEnemies := 5 + gs.Level*3
numPotions := 1 + gs.Level/2
```

### Weighted random monsters

The stock code draws uniformly from `registry.names`. For weighted draws, extend `MonsterDef`:

```yaml
# game/monsters.yaml
  - name: "Bug"
    weight: 10
    ...
```

```go
// game/monster.go
type MonsterDef struct {
    ...
    Weight int `yaml:"weight"`
}

// in GetRandomMonster:
total := 0
for _, name := range r.names { total += r.monsters[name].Weight }
if total == 0 { /* fall back to uniform */ }
pick := rng.Intn(total)
for _, name := range r.names {
    pick -= r.monsters[name].Weight
    if pick < 0 { return r.monsters[name] }
}
```

Remember to default missing weights to 1 so existing YAML still works.

---

## Mod 3: Change Room Sizes / Dungeon Shape

```go
// game/dungeon.go
const (
    MinRoomSize = 8   // was 6
    MaxRoomSize = 20  // was 15
)
```

Bigger minimums mean fewer rooms per level (leaf regions refuse to split). On an 80×27 map, pushing `MinRoomSize` past ~9 often yields only 3–4 rooms.

Increase BSP depth for more (but smaller) rooms:

```go
// game/dungeon.go:GenerateDungeon
root.Split(rng, 5)   // was 4
```

### Always-horizontal-first corridors

```go
// game/dungeon.go:connectRooms — replace the if/else
d.carveHorizontalCorridor(x1, x2, y1)
d.carveVerticalCorridor(y1, y2, x2)
```

Loses a touch of character but produces a more predictable map — useful for testing.

---

## Mod 4: Add a New Item — Coffee (+5 HP, +1 Damage for the Rest of the Run)

### 4a. Constructor and type

```go
// game/entity.go
const (
    EntityPlayer     EntityType = iota
    EntityMonster
    EntityPotion
    EntityBug
    EntityScopeCreep
    EntityCoffee     // NEW
)

func NewCoffee(x, y int) *Entity {
    return &Entity{
        Type:   EntityCoffee,
        X:      x,
        Y:      y,
        Symbol: 'c',
    }
}
```

### 4b. Storage and spawning

```go
// game/state.go — on GameState
Coffee []*Entity
```

```go
// game/state.go:generateLevel — append at END to avoid shifting RNG for existing levels
gs.Coffee = nil
numCoffee := 1 + gs.RNG.Intn(2)
for i := 0; i < numCoffee; i++ {
    x, y := gs.randomFloorTile()
    gs.Coffee = append(gs.Coffee, NewCoffee(x, y))
}
```

> ⚠️ **Determinism note**: Adding RNG consumption here shifts every subsequent level's layout. If you want to preserve the stock seed's layout, you have to accept that existing seeds now produce subtly different dungeons.

### 4c. Pickup

```go
// game/state.go:MovePlayer — after the potion-pickup loop
for i, c := range gs.Coffee {
    if c.X == newX && c.Y == newY {
        gs.Player.Heal(5)
        gs.Player.Damage += 1
        gs.Coffee = append(gs.Coffee[:i], gs.Coffee[i+1:]...)
        gs.SetMessage("You drink coffee! (+5 HP, +1 damage)")
        break
    }
}
```

### 4d. Render

```go
// game/game.go:render — after the potion rendering
coffeeStyle := tcell.StyleDefault.Foreground(tcell.ColorOrange).Background(tcell.ColorBlack)
for _, c := range g.state.Coffee {
    if g.state.Visible[c.Y][c.X] {
        g.screen.SetContent(offsetX+c.X, offsetY+c.Y, c.Symbol, nil, coffeeStyle)
    }
}
```

### 4e. Tests

```go
func TestCoffeePickup(t *testing.T) {
    gs := &GameState{
        Dungeon:  newTestDungeon(10, 10),
        RNG:      rand.New(rand.NewSource(1)),
        Player:   NewPlayer(5, 5),
        Enemies:  []*Entity{},
        Potions:  []*Entity{},
        Visible:  make2D(10, 10),
        Explored: make2D(10, 10),
    }
    gs.Coffee = []*Entity{NewCoffee(6, 5)}
    gs.Player.HP = 10
    preDmg := gs.Player.Damage

    gs.MovePlayer(1, 0)

    if gs.Player.HP != 15 { t.Errorf("expected +5 HP, got HP=%d", gs.Player.HP) }
    if gs.Player.Damage != preDmg+1 { t.Errorf("expected +1 damage") }
    if len(gs.Coffee) != 0 { t.Errorf("coffee should be consumed") }
}
```

(You'll need a `make2D` helper; copy `TestMoveCounter`'s visibility-array setup if you want to avoid writing one.)

---

## Mod 5: Status Effects (Poison)

### 5a. Field on Entity

```go
// game/entity.go
type Entity struct {
    ...
    Poisoned int  // remaining turns of poison
}
```

### 5b. Apply on hit

```go
// game/state.go:enemyAttacks — within the dist <= attackRange branch
if enemy.Name == "Scope Creep" && gs.Player.Poisoned == 0 {
    gs.Player.Poisoned = 3
}
```

Or drive this off a new YAML field — add `inflicts: ["poison"]` to a monster and check `slices.Contains(enemy.Abilities, "poison")`. The `Abilities []string` field is already parsed from YAML and unused at runtime — this is a natural first consumer.

### 5c. Tick down and damage

```go
// game/state.go:processTurn — after updateVisibility
if gs.Player.Poisoned > 0 && !gs.Invulnerable {
    gs.Player.TakeDamage(1)
    gs.Player.Poisoned--
    if gs.Player.Poisoned == 0 {
        gs.SetMessage("The poison wears off.")
    }
    if !gs.Player.IsAlive() {
        gs.GameOver = true
        gs.KilledBy = "poison"
    }
}
```

### 5d. UI

```go
// game/game.go:render — near the UI bar construction
poisonStatus := ""
if g.state.Player.Poisoned > 0 {
    poisonStatus = fmt.Sprintf(" | POISONED(%d)", g.state.Player.Poisoned)
}
uiLine := fmt.Sprintf("HP: %d/%d | Level: %d/%d | Kills: %d%s%s | [q]uit",
    g.state.Player.HP, g.state.Player.MaxHP,
    g.state.Level, g.state.MaxLevel,
    g.state.EnemiesKilled,
    invulnStatus, poisonStatus)
```

### 5e. Custom death message

```go
// game/game.go:getDeathMessage
case "poison":
    return "You succumbed to the poison."
```

Unlike the `"bug"` / `"Bug"` mismatch described in [entities.md](./entities.md#death-attribution), `"poison"` is a value you control end-to-end, so there's no casing bug risk.

---

## Mod 6: Fix the `KilledBy` Case Mismatch

If you're going to touch `getDeathMessage`, fix this while you're there.

```go
// game/game.go:getDeathMessage
import "strings"

func (g *Game) getDeathMessage() string {
    normalized := strings.ToLower(strings.ReplaceAll(g.state.KilledBy, " ", "_"))
    switch normalized {
    case "bug":
        return "In GitHub Dungeons... bug squashes YOU"
    case "scope_creep":
        return "Foiled by scope creep again!"
    case "merge_conflict":
        return fmt.Sprintf("Death by merge conflict. Just a typical %s.", time.Now().Weekday())
    case "zombie":
        return "Brains... it had yours."
    case "hermit_crab":
        return "Pincered to death. Sideways."
    default:
        return "The bugs and scope creeps won..."
    }
}
```

This is a genuine bug fix, not stylistic. No test currently asserts on death messages, but you could add one.

---

## Mod 7: Add a New Trap (Revert Trap)

Teleports the player to a random room.

```go
// game/state.go — on GameState
RevertX, RevertY int
```

```go
// game/state.go:generateLevel — append at END
gs.RevertX, gs.RevertY = gs.randomFloorTile()
```

```go
// game/state.go:MovePlayer — after the merge-marker check
if newX == gs.RevertX && newY == gs.RevertY {
    room := gs.Dungeon.Rooms[gs.RNG.Intn(len(gs.Dungeon.Rooms))]
    cx, cy := room.Center()
    gs.Player.X, gs.Player.Y = cx, cy
    gs.SetMessage("git reset --hard! Back to an earlier state...")
    gs.RevertX, gs.RevertY = gs.randomFloorTile()   // relocate trap
    // Don't call processTurn for a revert — it's a free teleport, but that's your design choice
}
```

---

## Mod 8: Custom Game Modes (Boss Level)

```go
// game/state.go:generateLevel — after the normal enemy spawn
if gs.Level == gs.MaxLevel {
    // Clear random spawns; only the boss remains
    gs.Enemies = gs.Enemies[:0]
    center := gs.Dungeon.Rooms[len(gs.Dungeon.Rooms)/2]
    bx, by := center.Center()
    gs.Enemies = append(gs.Enemies, &Entity{
        Type: EntityMonster, X: bx, Y: by,
        HP: 50, MaxHP: 50, Damage: 5, Symbol: 'B',
        Name: "The Final Bug", Color: tcell.ColorRed,
        Speed: 1.0, Movement: MovementAny, AttackRange: 1,
    })
}
```

Boss definitions can also live in `monsters.yaml` with `unique: true` and a spawn gate you add:

```yaml
  - name: "The Final Bug"
    token: "B"
    color: "red"
    hp: 50
    strength: 5
    speed: 1.0
    movement: "any"
    attack_range: 1
    experience_value: 500
    abilities: []
    unique: true
    # Custom: min_level you'd need to wire into generateLevel
```

(`min_level` would be a new field; plumb it through `MonsterDef` and filter in `GetUniqueMonsters`.)

---

## Testing Your Mods

### Run everything

```bash
go test ./game -count=1
```

### Use a fixed seed in tests

```go
rng := rand.New(rand.NewSource(12345))
d := GenerateDungeon(80, 27, rng, nil)
```

### Hardcode a seed for manual play

```go
// game/game.go:New — for debugging only
seed := int64(12345)  // <<< temporary override
// seed := computeSeed(codeFiles)
```

Revert before committing. Seriously.

### Print to stderr, not stdout

tcell owns stdout while the game is running. Debug prints must go to stderr:

```go
fmt.Fprintf(os.Stderr, "spawn count = %d\n", numEnemies)
```

---

## Common Pitfalls

1. **Reading `Tiles[x][y]` instead of `Tiles[y][x]`.** Row-major. Always.
2. **Forgetting the render layer.** New entity types *must* be added to `game.go:render` or they won't draw. Visibility gates should usually apply.
3. **Shifting RNG consumption mid-function.** Inserting a `gs.RNG.Intn` call in the middle of `generateLevel` shifts every subsequent random decision for existing seeds. Append at the end, or embrace the new randomness.
4. **Relying on YAML-only changes taking effect without a rebuild.** `monsters.yaml` is embedded via `//go:embed`. `go build` is required.
5. **Goroutines.** Don't. The loop is single-threaded; touching `GameState` from another goroutine will race.
6. **Panics in hot paths.** A panic while tcell holds the terminal leaves the user's shell in raw mode. Prefer `if` checks over uncontrolled indexing.
7. **Windows line endings in YAML.** `yaml.v3` handles them, but if you're editing via a weird tool check for stray `\r` characters.

---

## Contributing

- Fork the repo.
- Run `go test ./game -count=1` before and after your change.
- If you added a new field to `MonsterDef`, add at least one test that exercises it (see `TestHermitCrabIsRedH` for a template).
- If you change `generateLevel`'s RNG consumption, say so in the PR description — other users' seeds will shift.
- Update the relevant doc in `docs/`. Cite code locations (`file.go:Function`).

---

## See Also

- [monsters.md](./monsters.md) — YAML schema in detail
- [entities.md](./entities.md) — Runtime behavior reference (incl. known bugs to fix)
- [architecture.md](./architecture.md) — Where everything lives
- [seeding.md](./seeding.md) — Determinism constraints
- [merge-conflict.md](./merge-conflict.md) — Before extending merge mechanics, read this
