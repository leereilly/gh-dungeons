# Modding Guide

How to extend gh-dungeons with new enemies, items, mechanics, and more.

---

## Philosophy

gh-dungeons is designed to be moddable. The codebase is small (~2000 lines), well-structured, and uses simple patterns. If you can read Go, you can mod this game.

**Key principles:**
- All game logic is in the `game/` package
- Entities are data (stats + position), not classes
- Dungeon generation is deterministic (controlled by RNG seed)
- Rendering is separate from game state

---

## Quick Reference

| What to Mod | File to Edit | Function/Struct |
|-------------|--------------|-----------------|
| Add enemy type | `entity.go` | Create `NewFoo()` constructor |
| Change spawn rates | `state.go` | `generateLevel()` |
| Add item type | `entity.go` + `state.go` | New `EntityType` + pickup logic |
| Change room sizes | `dungeon.go` | `MinRoomSize`, `MaxRoomSize` |
| Add custom AI | `state.go` | `moveEnemies()` |
| Change combat | `state.go` | `enemyAttacks()`, `playerAutoAttack()` |
| Add status effects | `entity.go` + `state.go` | New fields on `Entity` |

---

## Adding a New Enemy

Let's add **Tech Debt** — a slow, tanky enemy that doesn't chase the player.

### Step 1: Add the Entity Type

Edit `game/entity.go`:

```go
const (
    EntityPlayer EntityType = iota
    EntityBug
    EntityScopeCreep
    EntityTechDebt  // Add this
    EntityPotion
)
```

### Step 2: Create a Constructor

Add to `game/entity.go`:

```go
func NewTechDebt(x, y int) *Entity {
    return &Entity{
        Type:   EntityTechDebt,
        X:      x,
        Y:      y,
        HP:     5,      // Tankier than scope creep
        MaxHP:  5,
        Damage: 1,      // Low damage
        Symbol: 't',    // 't' for tech debt
    }
}
```

### Step 3: Add Spawn Logic

Edit `game/state.go:generateLevel()`:

Find the enemy spawning code and modify it:

```go
numEnemies := 3 + gs.Level*2
for i := 0; i < numEnemies; i++ {
    x, y := gs.randomFloorTile()
    
    roll := gs.RNG.Float32()
    if roll > 0.7 {
        gs.Enemies = append(gs.Enemies, NewBug(x, y))
    } else if roll > 0.4 {
        gs.Enemies = append(gs.Enemies, NewScopeCreep(x, y))
    } else {
        gs.Enemies = append(gs.Enemies, NewTechDebt(x, y))  // New!
    }
}
```

**New spawn rates:**
- 30% Bugs
- 30% Scope Creeps
- 40% Tech Debt

### Step 4: Add Death Message

Edit `game/state.go:playerAutoAttack()` and `state.go:enemyAttacks()`:

```go
if !enemy.IsAlive() {
    gs.EnemiesKilled++
    if enemy.Type == EntityBug {
        gs.SetMessage("You squashed a bug!")
    } else if enemy.Type == EntityScopeCreep {
        gs.SetMessage("You eliminated a scope creep!")
    } else if enemy.Type == EntityTechDebt {
        gs.SetMessage("You refactored the tech debt!")  // New!
    }
}
```

### Step 5: Add Custom AI (Optional)

Tech Debt doesn't chase the player—it just wanders randomly.

Edit `game/state.go:moveEnemies()`:

```go
for _, enemy := range gs.Enemies {
    if !enemy.IsAlive() {
        continue
    }

    // Tech Debt doesn't chase the player
    if enemy.Type == EntityTechDebt {
        // Random walk
        dx := gs.RNG.Intn(3) - 1  // -1, 0, or 1
        dy := gs.RNG.Intn(3) - 1
        newX, newY := enemy.X+dx, enemy.Y+dy
        if gs.canEnemyMoveTo(newX, newY, enemy) {
            enemy.X, enemy.Y = newX, newY
        }
        continue
    }

    // Only move if player is visible (existing chase AI)
    if !gs.hasLineOfSight(enemy.X, enemy.Y, gs.Player.X, gs.Player.Y) {
        continue
    }

    // ... rest of chase AI
}
```

### Step 6: Add Custom Death Message

Edit `game/game.go:getDeathMessage()`:

```go
switch g.state.KilledBy {
case "bug":
    return "In GitHub Dungeons... bug squashes YOU"
case "merge_conflict":
    dayName := time.Now().Weekday().String()
    return fmt.Sprintf("Death by merge conflict. Just a typical %s.", dayName)
case "scope_creep":
    return "Foiled by scope creep again!"
case "tech_debt":  // New!
    return "Crushed under the weight of tech debt."
default:
    return "The bugs and scope creeps won..."
}
```

And set the killer in `state.go:enemyAttacks()`:

```go
if !gs.Player.IsAlive() {
    if enemy.Type == EntityBug {
        gs.KilledBy = "bug"
    } else if enemy.Type == EntityScopeCreep {
        gs.KilledBy = "scope_creep"
    } else if enemy.Type == EntityTechDebt {
        gs.KilledBy = "tech_debt"  // New!
    }
}
```

### Done!

Rebuild and test:

```bash
go build -o gh-dungeons
./gh-dungeons
```

You should now see `t` enemies wandering around that don't chase you.

---

## Adding a New Item

Let's add **Coffee** — restores 5 HP and grants +1 damage for the rest of the level.

### Step 1: Add the Entity Type

Edit `game/entity.go`:

```go
const (
    EntityPlayer EntityType = iota
    EntityBug
    EntityScopeCreep
    EntityPotion
    EntityCoffee  // Add this
)
```

### Step 2: Create a Constructor

Add to `game/entity.go`:

```go
func NewCoffee(x, y int) *Entity {
    return &Entity{
        Type:   EntityCoffee,
        X:      x,
        Y:      y,
        Symbol: 'c',  // 'c' for coffee
    }
}
```

### Step 3: Add Coffee List to GameState

Edit `game/state.go:GameState` struct:

```go
type GameState struct {
    Player    *Entity
    Enemies   []*Entity
    Potions   []*Entity
    Coffee    []*Entity  // Add this
    // ... rest of fields
}
```

### Step 4: Add Spawn Logic

Edit `game/state.go:generateLevel()`:

```go
// Spawn coffee (1-2 per level)
gs.Coffee = nil
numCoffee := 1 + gs.RNG.Intn(2)
for i := 0; i < numCoffee; i++ {
    x, y := gs.randomFloorTile()
    gs.Coffee = append(gs.Coffee, NewCoffee(x, y))
}
```

### Step 5: Add Pickup Logic

Edit `game/state.go:MovePlayer()`, after the potion pickup code:

```go
// Check for coffee pickup
for i, coffee := range gs.Coffee {
    if coffee.X == newX && coffee.Y == newY {
        gs.Player.Heal(5)
        gs.Player.Damage += 1
        gs.Coffee = append(gs.Coffee[:i], gs.Coffee[i+1:]...)
        gs.SetMessage("You drink coffee! (+5 HP, +1 damage)")
        break
    }
}
```

### Step 6: Add Rendering

Edit `game/game.go:render()`, after rendering potions:

```go
// Render coffee
coffeeStyle := tcell.StyleDefault.Foreground(tcell.ColorBrown).Background(tcell.ColorBlack)
for _, coffee := range g.state.Coffee {
    if g.state.Visible[coffee.Y][coffee.X] {
        g.screen.SetContent(offsetX+coffee.X, offsetY+coffee.Y, coffee.Symbol, nil, coffeeStyle)
    }
}
```

### Done!

Coffee now spawns, heals 5 HP, and grants permanent +1 damage.

---

## Modifying Spawn Rates

### Change Enemy Density

Edit `game/state.go:generateLevel()`:

```go
numEnemies := 5 + gs.Level*3  // More enemies (was 3 + gs.Level*2)
```

### Change Potion Frequency

```go
numPotions := 1 + gs.Level/2  // Fewer potions (was 2 + gs.Level + rand(2))
```

### Make Bugs Rarer

```go
if gs.RNG.Float32() > 0.8 {  // 20% chance (was 60%)
    gs.Enemies = append(gs.Enemies, NewBug(x, y))
} else {
    gs.Enemies = append(gs.Enemies, NewScopeCreep(x, y))
}
```

---

## Changing Dungeon Generation

### Make Rooms Bigger

Edit `game/dungeon.go`:

```go
const (
    MinRoomSize = 8   // Was 6
    MaxRoomSize = 20  // Was 15
)
```

### Generate More Rooms

Edit `game/dungeon.go:GenerateDungeon()`:

```go
root.Split(rng, 5)  // Depth 5 instead of 4 = more rooms
```

**Warning:** Depth 5+ can fail to generate if terminal is too small.

### Change Corridor Style

Edit `game/dungeon.go:connectRooms()`:

Replace random L-shape with always horizontal-first:

```go
d.carveHorizontalCorridor(x1, x2, y1)
d.carveVerticalCorridor(y1, y2, x2)
// Remove the `if rng.Float32() > 0.5` check
```

---

## Adding Status Effects

Let's add a **poison** status effect that deals 1 damage per turn for 3 turns.

### Step 1: Add Fields to Entity

Edit `game/entity.go:Entity` struct:

```go
type Entity struct {
    Type     EntityType
    X, Y     int
    HP       int
    MaxHP    int
    Damage   int
    Symbol   rune
    Poisoned int  // Turns of poison remaining
}
```

### Step 2: Create a Poison Attack

Modify Scope Creep to inflict poison:

Edit `game/state.go:enemyAttacks()`:

```go
for _, enemy := range gs.Enemies {
    if enemy.IsAlive() && gs.Player.IsAdjacent(enemy) {
        gs.Player.TakeDamage(enemy.Damage)
        
        // Scope Creeps inflict poison
        if enemy.Type == EntityScopeCreep && gs.Player.Poisoned == 0 {
            gs.Player.Poisoned = 3
            gs.Message = fmt.Sprintf("A scope creep attacked - %d HP damage (POISONED!)", enemy.Damage)
        } else {
            gs.Message = fmt.Sprintf("...")  // Normal messages
        }
        
        // ... rest of attack logic
    }
}
```

### Step 3: Apply Poison Damage

Edit `game/state.go:processTurn()`, at the end:

```go
// Apply poison damage
if gs.Player.Poisoned > 0 && !gs.Invulnerable {
    gs.Player.TakeDamage(1)
    gs.Player.Poisoned--
    if gs.Player.Poisoned > 0 {
        gs.SetMessage(fmt.Sprintf("Poison damage! (%d turns remaining)", gs.Player.Poisoned))
    } else {
        gs.SetMessage("The poison wears off.")
    }
    
    if !gs.Player.IsAlive() {
        gs.GameOver = true
        gs.KilledBy = "poison"
    }
}
```

### Step 4: Display Poison Status

Edit `game/game.go:render()`, in the UI bar:

```go
poisonStatus := ""
if g.state.Player.Poisoned > 0 {
    poisonStatus = fmt.Sprintf(" | POISONED(%d)", g.state.Player.Poisoned)
}

uiLine := fmt.Sprintf("HP: %d/%d | Level: %d/%d | Kills: %d%s%s | [q]uit",
    g.state.Player.HP, g.state.Player.MaxHP,
    g.state.Level, g.state.MaxLevel,
    g.state.EnemiesKilled,
    invulnStatus,
    poisonStatus)  // Add poison status
```

### Done!

Scope Creeps now inflict a 3-turn poison effect.

---

## Changing Combat Mechanics

### Add Critical Hits (20% chance for 2x damage)

Edit `game/state.go:playerAutoAttack()`:

```go
for _, enemy := range gs.Enemies {
    if enemy.IsAlive() && gs.Player.IsAdjacent(enemy) {
        damage := gs.Player.Damage
        if gs.RNG.Float32() < 0.2 {  // 20% crit chance
            damage *= 2
            gs.SetMessage("CRITICAL HIT!")
        }
        enemy.TakeDamage(damage)
        // ... rest of logic
    }
}
```

### Add Armor (reduce damage by 1, minimum 1)

Edit `game/entity.go:Entity` struct:

```go
type Entity struct {
    // ... existing fields
    Armor int  // Damage reduction
}
```

Edit `game/entity.go:TakeDamage()`:

```go
func (e *Entity) TakeDamage(dmg int) {
    dmg -= e.Armor
    if dmg < 1 {
        dmg = 1  // Minimum 1 damage
    }
    e.HP -= dmg
    if e.HP < 0 {
        e.HP = 0
    }
}
```

Give the player some armor:

Edit `game/entity.go:NewPlayer()`:

```go
func NewPlayer(x, y int) *Entity {
    return &Entity{
        Type:   EntityPlayer,
        HP:     20,
        MaxHP:  20,
        Damage: 2,
        Armor:  1,  // Player has 1 armor
        Symbol: '@',
    }
}
```

---

## Adding New Traps

Let's add a **Revert Trap** that teleports the player to a random room.

### Step 1: Track Trap Location

Edit `game/state.go:GameState` struct:

```go
type GameState struct {
    // ... existing fields
    RevertTrapX int
    RevertTrapY int
}
```

### Step 2: Place Trap

Edit `game/state.go:generateLevel()`:

```go
// Place revert trap (one per level)
gs.RevertTrapX, gs.RevertTrapY = gs.randomFloorTile()
```

### Step 3: Trigger Trap

Edit `game/state.go:MovePlayer()`, after merge conflict check:

```go
// Check for revert trap
if newX == gs.RevertTrapX && newY == gs.RevertTrapY {
    // Teleport player to random room
    room := gs.Dungeon.Rooms[gs.RNG.Intn(len(gs.Dungeon.Rooms))]
    gs.Player.X = room.X + room.W/2
    gs.Player.Y = room.Y + room.H/2
    gs.SetMessage("REVERT! You've been teleported!")
    
    // Move trap to new location
    gs.RevertTrapX, gs.RevertTrapY = gs.randomFloorTile()
}
```

### Done!

The revert trap now teleports the player and relocates itself.

---

## Testing Your Mods

### Use a Fixed Seed

Edit `game/game.go:New()`:

```go
seed := int64(12345)  // Hard-code for testing
rng := rand.New(rand.NewSource(seed))
```

### Add Debug Logging

Edit `game/state.go:generateLevel()`:

```go
fmt.Printf("Level %d: %d enemies, %d potions\n", gs.Level, len(gs.Enemies), len(gs.Potions))
```

### Write Unit Tests

Create `game/mymod_test.go`:

```go
package game

import (
    "math/rand"
    "testing"
)

func TestTechDebtSpawns(t *testing.T) {
    rng := rand.New(rand.NewSource(42))
    
    // Count tech debt spawns over 100 iterations
    techDebtCount := 0
    for i := 0; i < 100; i++ {
        x, y := 0, 0
        roll := rng.Float32()
        if roll < 0.4 {
            techDebtCount++
        }
    }
    
    // Expect ~40 tech debt out of 100
    if techDebtCount < 30 || techDebtCount > 50 {
        t.Errorf("Expected ~40 tech debt, got %d", techDebtCount)
    }
}
```

Run tests:

```bash
go test ./game
```

---

## Common Pitfalls

### Forgetting to Initialize New Fields

If you add a field to `GameState`, initialize it in `NewGameState()` and reset it in `generateLevel()`.

### Breaking Determinism

If you use non-seeded randomness (e.g., `math/rand.Float32()` directly), different runs will produce different results. Always use `gs.RNG`.

### Off-by-One Errors

Remember that tiles are `[y][x]`, not `[x][y]`. Check bounds before accessing:

```go
if y >= 0 && y < height && x >= 0 && x < width {
    tile := dungeon.Tiles[y][x]
}
```

### Not Rendering New Entities

If you add a new entity type, remember to render it in `game.go:render()`, otherwise it won't appear.

---

## Advanced: Custom Game Modes

### Boss Fight Mode

Add a `BossLevel` field to `GameState`:

```go
type GameState struct {
    // ...
    BossLevel bool
}
```

Edit `game/state.go:generateLevel()`:

```go
if gs.Level == gs.MaxLevel {
    gs.BossLevel = true
    
    // Spawn boss in center room
    centerRoom := gs.Dungeon.Rooms[len(gs.Dungeon.Rooms)/2]
    bx, by := centerRoom.Center()
    boss := NewBoss(bx, by)
    gs.Enemies = []*Entity{boss}  // Only the boss
}
```

Create the boss:

```go
func NewBoss(x, y int) *Entity {
    return &Entity{
        Type:   EntityBug,  // Reuse type or create EntityBoss
        HP:     50,
        MaxHP:  50,
        Damage: 5,
        Symbol: 'B',
    }
}
```

### Speedrun Timer

Add a timer field:

```go
type GameState struct {
    // ...
    StartTime time.Time
}
```

Initialize in `NewGameState()`:

```go
gs.StartTime = time.Now()
```

Display in UI:

```go
elapsed := time.Since(g.state.StartTime).Seconds()
uiLine := fmt.Sprintf("HP: %d/%d | Time: %.1fs | ...", gs.Player.HP, gs.Player.MaxHP, elapsed)
```

---

## Contributing Your Mods

If you create a cool mod:

1. Fork the repo
2. Add your changes
3. Update README with mod description
4. Submit a pull request

Or share your fork in GitHub Discussions!

---

## Further Reading

- [architecture.md](./architecture.md) — Understand the codebase structure
- [entities.md](./entities.md) — Deep dive into entity system
- [dungeon-generation.md](./dungeon-generation.md) — Modify map generation
- [seeding.md](./seeding.md) — Maintain determinism in mods
