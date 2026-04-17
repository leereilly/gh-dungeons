# Dungeon Generation

How `gh-dungeons` builds each of its five floors using Binary Space Partitioning.

Everything in this file is implemented in `game/dungeon.go`, with driver calls from `game/state.go:generateLevel`.

---

## Overview

Each level goes through five phases:

```
1. Allocate tile grid (width × height), fill with TileWall.
2. Build a BSP tree (depth 4) over the full rectangle.
3. Place one room per leaf node of the BSP tree.
4. Carve rooms as TileFloor.
5. Walk the tree post-order; connect siblings with L-shaped corridors.
```

Then `state.go:generateLevel` decorates the finished dungeon:
- Places the player in the center of `Rooms[0]`.
- Calls `Dungeon.PlaceDoor` to stamp a `TileDoor` in the last room.
- Chooses `MergeConflictX/Y` (random floor tile — the fire trap).
- Calls `findCentralRoomCenter(dungeon)` to pick `MergeMarkerX/Y` (the `--merge` marker).
- Spawns enemies and potions on random floor tiles.

Dungeon generation is deterministic: same `rng` ⇒ same map. Decorators consume the same RNG afterward, so changing spawn counts shifts every subsequent random decision. See [seeding.md](./seeding.md).

---

## Constants and Types

```go
// game/dungeon.go
const (
    MinRoomSize = 6
    MaxRoomSize = 15
)

type Tile int
const (
    TileWall  Tile = iota  // '#'
    TileFloor              // '.' or a code character
    TileDoor               // '>'
)

type Room struct { X, Y, W, H int }
func (r Room) Center() (int, int)       // midpoint
func (r Room) Contains(x, y int) bool   // half-open rectangle

type BSPNode struct {
    X, Y, W, H  int
    Left, Right *BSPNode
    Room        *Room
}

type Dungeon struct {
    Width, Height int
    Tiles         [][]Tile // row-major: Tiles[y][x]
    Rooms         []*Room
    CodeFile      *CodeFile
}
```

Grid size comes from `state.go:generateLevel`:

```go
width := max(gs.TermWidth, 40)
height := max(gs.TermHeight - 3, 20)   // -3 reserves UI bar + message + buffer
```

So the map always has at least 40×20 and usually fills the terminal minus three lines.

---

## Step 1: BSP Split (`BSPNode.Split`)

```go
func (n *BSPNode) Split(rng *rand.Rand, depth int) {
    if depth <= 0 { return }

    horizontal := rng.Float32() > 0.5
    if float32(n.W)/float32(n.H) >= 1.25 {
        horizontal = false                 // wide region → vertical cut
    } else if float32(n.H)/float32(n.W) >= 1.25 {
        horizontal = true                  // tall region → horizontal cut
    }

    maxSize := n.H - MinRoomSize
    if !horizontal { maxSize = n.W - MinRoomSize }
    if maxSize <= MinRoomSize { return }   // too small → leaf
    split := rng.Intn(maxSize-MinRoomSize) + MinRoomSize

    if horizontal {
        n.Left  = NewBSPNode(n.X, n.Y,           n.W, split)
        n.Right = NewBSPNode(n.X, n.Y+split,     n.W, n.H-split)
    } else {
        n.Left  = NewBSPNode(n.X,        n.Y, split,      n.H)
        n.Right = NewBSPNode(n.X+split,  n.Y, n.W-split,  n.H)
    }
    n.Left.Split(rng, depth-1)
    n.Right.Split(rng, depth-1)
}
```

Two subtle but important rules:

1. **The 1.25 aspect-ratio heuristic** forces rectangles that are too long on one side to split on that axis. Without this, BSP can produce long slivers that fit at most a single 6-wide room.
2. **Early termination** when `maxSize <= MinRoomSize` turns the node into a leaf instead of splitting. This is why you sometimes see fewer than 16 rooms even though `depth = 4` could theoretically produce `2^4 = 16` leaves.

**Depth-4 tree at full expansion = up to 16 leaves.** In practice, most dungeons have 8–12 rooms depending on how the aspect checks fire.

```
Initial region (60×27):
┌────────────────────────────────────────────────────────────┐
│                          root                              │
└────────────────────────────────────────────────────────────┘

Cut 1 (vertical, aspect forced):
┌───────────────────┬────────────────────────────────────────┐
│       L           │                  R                     │
└───────────────────┴────────────────────────────────────────┘

Cut 2 (both children, direction randomized, then aspect-checked):
┌───────────────────┬────────────────────┬───────────────────┐
│        LL         │         RL         │                   │
├───────────────────┤                    │        RR         │
│        LR         ├────────────────────┤                   │
│                   │         RL2        │                   │
└───────────────────┴────────────────────┴───────────────────┘

...continues for up to depth 4.
```

---

## Step 2: Create Rooms (`BSPNode.CreateRooms`)

Called once after the full tree is built. Post-order traversal on internal nodes, room creation on leaves.

```go
if n.W < MinRoomSize+2 || n.H < MinRoomSize+2 { return }   // skip tiny regions

maxW := min(MaxRoomSize, n.W-2)   // at least 1 tile padding on each side
maxH := min(MaxRoomSize, n.H-2)

roomW := rng.Intn(maxW-MinRoomSize+1) + MinRoomSize   // [6, 15]
roomH := rng.Intn(maxH-MinRoomSize+1) + MinRoomSize

roomX := n.X + rng.Intn(n.W-roomW-1) + 1              // leave 1-tile border
roomY := n.Y + rng.Intn(n.H-roomH-1) + 1
```

Why the `-2` and `+1`? So rooms never touch the BSP partition boundary. Sibling corridors need a wall to cut through.

`n.Room` is set on each leaf. `GetRoom()` walks down from any node to find a representative room; `GetRooms()` returns the flat list.

```
Leaf region (20×10) with a 9×6 room placed inside:

┌────────────────────┐
│ leaf node          │
│   ┌───────────┐    │
│   │           │    │
│   │   Room    │    │   <- roomW × roomH chosen randomly within [6,15]
│   │           │    │
│   └───────────┘    │
│                    │
└────────────────────┘
```

---

## Step 3: Carve Rooms

After `CreateRooms`, `GenerateDungeon` flips every tile inside every room to `TileFloor`:

```go
for _, room := range d.Rooms {
    for y := room.Y; y < room.Y+room.H; y++ {
        for x := room.X; x < room.X+room.W; x++ {
            if y >= 0 && y < height && x >= 0 && x < width {
                d.Tiles[y][x] = TileFloor
            }
        }
    }
}
```

Rooms are always rectangular and axis-aligned.

---

## Step 4: Corridor Carving (`connectRooms`)

Post-order traversal. At every internal node we pick a room from the left subtree and one from the right subtree (`GetRoom` returns the first one found via DFS), then cut an L-shaped corridor between their centers. The randomness of which leg goes first makes the corridors feel less gridlike.

```go
func connectRooms(node *BSPNode, d *Dungeon, rng *rand.Rand) {
    if node.Left == nil || node.Right == nil { return }
    connectRooms(node.Left, d, rng)
    connectRooms(node.Right, d, rng)

    leftRoom  := node.Left.GetRoom()
    rightRoom := node.Right.GetRoom()
    if leftRoom == nil || rightRoom == nil { return }

    x1, y1 := leftRoom.Center()
    x2, y2 := rightRoom.Center()

    if rng.Float32() > 0.5 {
        d.carveHorizontalCorridor(x1, x2, y1)
        d.carveVerticalCorridor(y1, y2, x2)
    } else {
        d.carveVerticalCorridor(y1, y2, x1)
        d.carveHorizontalCorridor(x1, x2, y2)
    }
}
```

Corridors are 1 tile wide and may overlap with each other or with rooms. That's fine — re-setting a floor tile to floor is a no-op. `carveHorizontalCorridor` / `carveVerticalCorridor` sort the endpoints and bounds-check before writing.

**Connectivity proof sketch:** post-order recursion guarantees that before you connect siblings, each subtree is internally connected. So connecting the two representative rooms joins the subtrees. By induction the whole tree is connected.

```
Two leaves connected with an L-shape, horizontal-first:

┌────────────┐            ┌────────────┐
│  Room A    │            │  Room B    │
│     *──────────────┐    │     *      │
│            │       │    │     │      │
└────────────┘       │    └─────│──────┘
                     │          │
                     └──────────┘

* = room center      ── = carved corridor
```

---

## Step 5: Decorations (in `state.go:generateLevel`)

After `GenerateDungeon`, the decorator does the rest of the work:

```go
gs.Dungeon = GenerateDungeon(width, height, gs.RNG, codeFile)

// Visibility arrays
gs.Visible = make([][]bool, height)
gs.Explored = make([][]bool, height)
...

// Player in Rooms[0]
if len(gs.Dungeon.Rooms) > 0 {
    room := gs.Dungeon.Rooms[0]
    px, py := room.Center()
    // NewPlayer on first level, reposition on subsequent levels
}

gs.DoorX, gs.DoorY = gs.Dungeon.PlaceDoor(gs.RNG)        // last room

gs.MergeConflictX, gs.MergeConflictY = gs.randomFloorTile() // fire trap

// YAML-driven spawns
registry := GetMonsterRegistry()
for i := 0; i < 3+gs.Level*2; i++ {
    x, y := gs.randomFloorTile()
    def := registry.GetRandomMonster(gs.RNG)
    gs.Enemies = append(gs.Enemies, NewMonsterFromDef(def, x, y))
}
for _, def := range registry.GetUniqueMonsters() {
    x, y := gs.randomFloorTile()
    gs.Enemies = append(gs.Enemies, NewMonsterFromDef(def, x, y))
}

// Potions
for i := 0; i < 2+gs.Level+gs.RNG.Intn(2); i++ {
    x, y := gs.randomFloorTile()
    gs.Potions = append(gs.Potions, NewPotion(x, y))
}

gs.MergeMarkerX, gs.MergeMarkerY = findCentralRoomCenter(gs.Dungeon)
gs.updateVisibility()
```

`randomFloorTile` picks a random room, then a random tile inside it, retries up to 100 times to avoid the player, door, and merge-conflict trap positions. Falls back to map center on failure.

### `PlaceDoor`

```go
room := d.Rooms[len(d.Rooms)-1]                  // always the last room in the list
x := room.X + rng.Intn(room.W-2) + 1             // 1-tile margin from walls
y := room.Y + rng.Intn(room.H-2) + 1
d.Tiles[y][x] = TileDoor
```

### `findCentralRoomCenter`

Returns the center of the room whose center is closest (Euclidean-squared) to the geometric center of the whole map. This is what `--merge` mode uses to place its red `X` marker, so the warning and trigger logic operates on a visually prominent room rather than a random corner.

### `findNearestFloorTile`

BFS from a starting point, 8-directional, returning the first `TileFloor` it finds. Not used by the game loop today, but available for modders who want to snap an off-grid position to the nearest walkable tile.

---

## Code Text on Floors

The floor isn't rendered as `.` by default — it's rendered as characters from a real source file in your repository. From `game.go:render`:

```go
if len(codeLines) > 0 {
    // 2× line density: each tile row can show 2 different source lines
    lineIdx := (y*2 + x/40) % len(codeLines)
    line := codeLines[lineIdx]
    charIdx := x % 40
    if x >= 40 { charIdx = x - 40 }
    if charIdx < len(line) { ch = rune(line[charIdx]) } else { ch = '.' }
} else {
    ch = '.'
}
```

The code file for the floor is chosen by level:

```go
codeFile = &gs.CodeFiles[(gs.Level-1) % len(gs.CodeFiles)]
```

So each of the 5 levels uses a different top-5 file, cycling if there are fewer.

---

## Modifying Generation

### Bigger or smaller rooms
```go
// game/dungeon.go
const (
    MinRoomSize = 8   // was 6
    MaxRoomSize = 20  // was 15 — be careful, doesn't fit on small terminals
)
```

### More or fewer rooms
```go
// game/dungeon.go:GenerateDungeon
root.Split(rng, 5)   // depth 4 → 5
```
Going past depth 5 is almost always counterproductive: the `maxSize <= MinRoomSize` guard kicks in and many leaves refuse to split, but when combined with the 6-tile minimum room size you tend to get lots of hallway dungeons without the aesthetic you wanted.

### Straight-line corridors (no random leg order)
```go
// game/dungeon.go:connectRooms — replace the if/else
d.carveHorizontalCorridor(x1, x2, y1)
d.carveVerticalCorridor(y1, y2, x2)
```

### Non-rectangular rooms / pillars
After the room-carving loop in `GenerateDungeon`, stamp extra `TileWall`s inside rooms. Careful: you can block connectivity if you wall off a room center before corridor carving. Add decorations *after* `connectRooms`.

---

## Testing Dungeon Generation

Because the RNG is injected, unit-testing is straightforward:

```go
rng := rand.New(rand.NewSource(12345))
d := GenerateDungeon(80, 27, rng, nil)

// basic sanity
if len(d.Rooms) < 4 { t.Fatalf("expected at least 4 rooms, got %d", len(d.Rooms)) }

// connectivity via BFS from player start to door
start := d.Rooms[0].Center()
doorX, doorY := d.PlaceDoor(rng)
if !reachable(d, start, [2]int{doorX, doorY}) {
    t.Fatal("door unreachable from player start")
}
```

A handy constructor for tests: `newTestDungeon(w, h)` in `state_test.go` builds an all-floor dungeon with no rooms for isolated unit testing of movement/combat.

---

## Common Pitfalls

- **Out-of-bounds writes:** all carving helpers bounds-check; anything you add should too. `Tiles[y][x]` (not `[x][y]`).
- **Disconnected rooms:** rare, but if you change how `GetRoom` resolves representatives you can end up with sibling subtrees where the chosen nodes lie on opposite sides of a barely-carved corridor. Run the BFS connectivity check in tests.
- **Too-small terminals:** `generateLevel` enforces a 40×20 minimum. Don't lower this — the BSP at depth 4 with MinRoomSize 6 needs that much room to reliably produce any rooms at all.

---

## Further Reading

- [RogueBasin: Basic BSP Dungeon Generation](http://www.roguebasin.com/index.php?title=Basic_BSP_Dungeon_generation)
- [architecture.md](./architecture.md) — where BSP sits in the stack
- [seeding.md](./seeding.md) — why same seed ⇒ same map
- [modding.md](./modding.md) — adding content without breaking determinism
