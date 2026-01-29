# Entities

A complete reference to all entities in gh-dungeons: player, enemies, items, and interactive objects.

---

## Entity System Overview

All game objects that occupy a tile are represented by the `Entity` struct in `game/entity.go`.

**`Entity` struct definition:**

```go
type Entity struct {
    Type   EntityType  // Player, Bug, ScopeCreep, Potion
    X, Y   int         // Position on the map
    HP     int         // Current hit points
    MaxHP  int         // Maximum hit points
    Damage int         // Damage dealt per attack
    Symbol rune        // Character displayed on screen
}
```

---

## Player

**Symbol:** `@`  
**Starting HP:** 20  
**Max HP:** 20  
**Damage:** 2  
**Movement:** WASD, arrow keys, vim keys (hjklyubn)

**Constructor:**
```go
func NewPlayer(x, y int) *Entity {
    return &Entity{
        Type:   EntityPlayer,
        X:      x,
        Y:      y,
        HP:     20,
        MaxHP:  20,
        Damage: 2,
        Symbol: '@',
    }
}
```

**Spawn location:** Center of the first room (from `state.go:generateLevel()`).

**Special abilities:**
- **Auto-attack:** Automatically attacks all adjacent enemies each turn
- **Bump-to-attack:** Moving into an enemy triggers an attack instead of movement
- **Konami code:** `↑ ↑ ↓ ↓ ← → ← → B A` grants invulnerability

**Movement details:**
- Cardinal directions: Up, Down, Left, Right
- Diagonal directions: Y (up-left), U (up-right), B (down-left), N (down-right)
- Cannot move into walls
- Cannot move into enemies (attacks them instead)

---

## Enemies

### Bug

**Symbol:** `b`  
**HP:** 1  
**Max HP:** 1  
**Damage:** 1  

**Constructor:**
```go
func NewBug(x, y int) *Entity {
    return &Entity{
        Type:   EntityBug,
        X:      x,
        Y:      y,
        HP:     1,
        MaxHP:  1,
        Damage: 1,
        Symbol: 'b',
    }
}
```

**Flavor:** Weak, one-shot enemies. The traditional roguelike fodder.

**Spawn rate:** 60% chance per enemy slot (from `state.go:generateLevel()`):

```go
if gs.RNG.Float32() > 0.4 {
    gs.Enemies = append(gs.Enemies, NewBug(x, y))
}
```

**Death message:** `"You squashed a bug!"`

---

### Scope Creep

**Symbol:** `s`  
**HP:** 3  
**Max HP:** 3  
**Damage:** 2  

**Constructor:**
```go
func NewScopeCreep(x, y int) *Entity {
    return &Entity{
        Type:   EntityScopeCreep,
        X:      x,
        Y:      y,
        HP:     3,
        MaxHP:  3,
        Damage: 2,
        Symbol: 's',
    }
}
```

**Flavor:** Tougher enemies that hit harder. Named after the project management anti-pattern.

**Spawn rate:** 40% chance per enemy slot.

**Death message:** `"You eliminated a scope creep!"`

**Note:** The README incorrectly lists the symbol as `c`, but the code uses `s`.

---

### Enemy AI

**Chase behavior** (from `state.go:moveEnemies()`):

```go
// Only move if player is visible (line of sight check)
if !gs.hasLineOfSight(enemy.X, enemy.Y, gs.Player.X, gs.Player.Y) {
    continue
}

// Simple chase AI - move toward player
dx, dy := 0, 0
if enemy.X < gs.Player.X {
    dx = 1
} else if enemy.X > gs.Player.X {
    dx = -1
}
if enemy.Y < gs.Player.Y {
    dy = 1
} else if enemy.Y > gs.Player.Y {
    dy = -1
}

// Try diagonal move first, then cardinal directions
```

**Key behavior:**
- **Dormant when not visible:** Enemies don't move unless they have line of sight to the player
- **Diagonal preferred:** Tries to move both dx and dy simultaneously
- **Collision avoidance:** Won't move into walls, player, or other enemies
- **Attacks when adjacent:** Automatically attacks player if next to them

**Line of sight:** Uses Bresenham-like ray casting (from `state.go:hasLineOfSight()`). Blocked by walls only, not by other entities.

---

### Enemy Spawn Formula

From `state.go:generateLevel()`:

```go
numEnemies := 3 + gs.Level*2
```

**Spawn count by level:**
- Level 1: 5 enemies
- Level 2: 7 enemies
- Level 3: 9 enemies
- Level 4: 11 enemies
- Level 5: 13 enemies

**Composition:** 60% Bugs, 40% Scope Creeps (on average).

---

## Items

### Health Potion

**Symbol:** `+`  
**HP:** N/A  
**Damage:** N/A  
**Heal amount:** 3 HP

**Constructor:**
```go
func NewPotion(x, y int) *Entity {
    return &Entity{
        Type:   EntityPotion,
        X:      x,
        Y:      y,
        Symbol: '+',
    }
}
```

**Pickup behavior:**
- Automatically consumed when player moves onto the tile
- Restores 3 HP (capped at MaxHP)
- Removed from the map after use

**Pickup message:** `"You drink a health potion! (+3 HP)"`

**Spawn formula** (from `state.go:generateLevel()`):

```go
numPotions := 2 + gs.Level + gs.RNG.Intn(2)
```

**Spawn count by level:**
- Level 1: 3-4 potions
- Level 2: 4-5 potions
- Level 3: 5-6 potions
- Level 4: 6-7 potions
- Level 5: 7-8 potions

---

## Interactive Objects

### Door (Stairs)

**Symbol:** `>`  
**Tile type:** `TileDoor`  
**Not an Entity:** Stored in `Dungeon.Tiles`, not as a separate entity.

**Placement:** One door per level, placed in the last room (from `dungeon.go:PlaceDoor()`):

```go
room := d.Rooms[len(d.Rooms)-1]  // Last room
x := room.X + rng.Intn(room.W-2) + 1
y := room.Y + rng.Intn(room.H-2) + 1
d.Tiles[y][x] = TileDoor
```

**Behavior when player enters:**
- If `Level < MaxLevel` (5): Increment level, generate new dungeon
- If `Level >= MaxLevel`: Set `Victory = true`, show victory screen

**Message:** `"You descend deeper into the dungeon..."` or `"You've escaped the dungeon! Victory!"`

---

## Hidden Content: Merge Conflict Trap

**Symbol:** `<` `>` `=` (animated)  
**Damage:** 1 HP per turn while standing on it  
**Trigger:** One hidden trap per level, placed on a random floor tile

**Visual effect:**
- When triggered, displays a 5x3 pattern of conflict markers
- Fire spreads to 7 adjacent tiles
- Walls turn red
- Pattern animates with color rotation (red → orange → yellow)

**Placement logic** (from `state.go:generateLevel()`):

```go
gs.MergeConflictX, gs.MergeConflictY = gs.randomFloorTile()
```

**Damage logic** (from `state.go:checkMergeConflict()`):

```go
if !gs.Invulnerable {
    gs.Player.TakeDamage(1)
    gs.Message = "- 1 HP damage"
}
```

**Death message:** `"Death by merge conflict. Just a typical [DayOfWeek]."`

**Merge mode:** Run with `gh dungeons --merge` to see an `X` marker at the trap location.

---

## Combat System

### Attack Resolution

**Player attacks enemy:**
```go
enemy.TakeDamage(gs.Player.Damage)  // Always hits, always deals 2 damage
if !enemy.IsAlive() {
    gs.EnemiesKilled++
}
```

**Enemy attacks player:**
```go
gs.Player.TakeDamage(enemy.Damage)  // Always hits
if !gs.Player.IsAlive() {
    gs.GameOver = true
}
```

**No miss chance, no critical hits, no armor.** Combat is deterministic based on stats.

### Auto-Attack

From `state.go:playerAutoAttack()`:

```go
for _, enemy := range gs.Enemies {
    if enemy.IsAlive() && gs.Player.IsAdjacent(enemy) {
        enemy.TakeDamage(gs.Player.Damage)
    }
}
```

Triggers **every turn** after player movement. This means:
- You don't need to bump into enemies to attack them
- Standing next to an enemy and moving in place attacks them
- Multiple adjacent enemies are all attacked each turn

### Adjacency Check

From `entity.go:IsAdjacent()`:

```go
func (e *Entity) IsAdjacent(other *Entity) bool {
    dx := abs(e.X - other.X)
    dy := abs(e.Y - other.Y)
    return dx <= 1 && dy <= 1 && (dx+dy > 0)
}
```

Uses **Chebyshev distance** (8-way adjacency). Diagonal counts as adjacent.

---

## Entity Methods

### `IsAlive() bool`

Returns `HP > 0`. Used for:
- Skipping dead enemies in movement/attack logic
- Determining when to show death messages

### `TakeDamage(dmg int)`

```go
func (e *Entity) TakeDamage(dmg int) {
    e.HP -= dmg
    if e.HP < 0 {
        e.HP = 0
    }
}
```

Simple subtraction, clamped at 0.

### `Heal(amount int)`

```go
func (e *Entity) Heal(amount int) {
    e.HP += amount
    if e.HP > e.MaxHP {
        e.HP = e.MaxHP
    }
}
```

Used by potions. Cannot exceed `MaxHP`.

### `IsEnemy() bool`

```go
return e.Type == EntityBug || e.Type == EntityScopeCreep
```

Used for filtering entities (not currently used in code, but available for modding).

### `DistanceTo(other *Entity) int`

```go
dx := abs(e.X - other.X)
dy := abs(e.Y - other.Y)
return max(dx, dy)  // Chebyshev distance
```

Not currently used in game logic, but available for distance checks.

---

## Konami Code

**Sequence:** `↑ ↑ ↓ ↓ ← → ← → B A`

**Effect:** Sets `Invulnerable = true`, player takes no damage from any source.

**Implementation** (from `state.go:CheckKonamiCode()`):

```go
konamiCode := []string{"up", "up", "down", "down", "left", "right", "left", "right", "b", "a"}

gs.KonamiSequence = append(gs.KonamiSequence, key)

// Keep only last 10 keys
if len(gs.KonamiSequence) > 10 {
    gs.KonamiSequence = gs.KonamiSequence[len(gs.KonamiSequence)-10:]
}

// Check for match
if len(gs.KonamiSequence) == 10 {
    match := true
    for i := 0; i < 10; i++ {
        if gs.KonamiSequence[i] != konamiCode[i] {
            match = false
            break
        }
    }
    if match && !gs.Invulnerable {
        gs.Invulnerable = true
        gs.SetMessage("KONAMI CODE ACTIVATED! You are now invulnerable!")
    }
}
```

**Notes:**
- Must use arrow keys for directional inputs (WASD doesn't count)
- Must press `b` and `a` keys specifically (not `B` or diagonal movement)
- One-time activation per game (doesn't toggle off)

---

## Visibility and Rendering

### Fog of War

**Vision radius:** 7 tiles (from `state.go:VisionRadius`)

**Ray casting:** 180 rays cast every 2 degrees (from `state.go:updateVisibility()`):

```go
for angle := 0; angle < 360; angle += 2 {
    gs.castRay(px, py, angle)
}
```

**Visibility states:**
- **Visible:** Full color, entities rendered
- **Explored but not visible:** Dimmed (Color240), no entities
- **Unexplored:** Not rendered

### Rendering Order

From `game.go:render()`:

1. Tiles (walls, floors, doors)
2. Potions
3. Merge conflict fire (if triggered)
4. Enemies
5. Player

**Result:** Player is always rendered on top, even if multiple entities occupy the same tile (shouldn't happen, but graceful if it does).

---

## Custom Death Messages

From `game.go:getDeathMessage()`:

```go
switch g.state.KilledBy {
case "bug":
    return "In GitHub Dungeons... bug squashes YOU"
case "merge_conflict":
    dayName := time.Now().Weekday().String()
    return fmt.Sprintf("Death by merge conflict. Just a typical %s.", dayName)
case "scope_creep":
    return "Foiled by scope creep again!"
default:
    return "The bugs and scope creeps won..."
}
```

**KilledBy tracking:** Set in `state.go` when player HP reaches 0:
- `"bug"` — Killed by a Bug's attack
- `"scope_creep"` — Killed by a Scope Creep's attack
- `"merge_conflict"` — Killed by standing on the merge conflict trap

---

## Modding Entities

See [modding.md](./modding.md) for full examples, but here's a quick reference:

### Add a new enemy type

1. Add to `EntityType` enum in `entity.go`
2. Create constructor (e.g., `NewTechDebt(x, y int)`)
3. Add spawn logic in `state.go:generateLevel()`
4. Add death message in `state.go:enemyAttacks()`
5. (Optional) Add custom AI in `state.go:moveEnemies()`

### Change enemy stats

Edit the constructors in `entity.go`:

```go
func NewBug(x, y int) *Entity {
    return &Entity{
        HP:     2,   // Was 1, now tankier
        Damage: 2,   // Was 1, now hits harder
        // ...
    }
}
```

### Change spawn rates

Edit `state.go:generateLevel()`:

```go
numEnemies := 5 + gs.Level*3  // More enemies per level
numPotions := 1 + gs.Level    // Fewer potions
```

Or change the Bug/Scope Creep ratio:

```go
if gs.RNG.Float32() > 0.7 {  // 30% Bugs, 70% Scope Creeps
    gs.Enemies = append(gs.Enemies, NewBug(x, y))
}
```

---

## Further Reading

- [architecture.md](./architecture.md) — System overview
- [modding.md](./modding.md) — How to add new entities
- [seeding.md](./seeding.md) — Deterministic entity placement
