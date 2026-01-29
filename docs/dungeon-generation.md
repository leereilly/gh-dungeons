# Dungeon Generation

A detailed explanation of how gh-dungeons creates procedural levels using Binary Space Partitioning.

---

## Overview

Each dungeon level is generated using a **Binary Space Partitioning (BSP) tree** algorithm. This creates organic-looking layouts with distinct rooms connected by L-shaped corridors.

The algorithm is deterministic: the same RNG seed always produces the same dungeon.

---

## The BSP Algorithm

### Step 1: Recursive Partitioning

The dungeon starts as a single rectangle (the full map). The BSP tree recursively splits this space into smaller regions.

**Split decision logic** (from `dungeon.go:BSPNode.Split()`):

```go
// Decide split direction based on shape
horizontal := rng.Float32() > 0.5

if float32(n.W)/float32(n.H) >= 1.25 {
    horizontal = false  // Wide region → vertical split
} else if float32(n.H)/float32(n.W) >= 1.25 {
    horizontal = true   // Tall region → horizontal split
}
```

**Split position:**
```go
maxSize := n.H - MinRoomSize  // or n.W for vertical
split := rng.Intn(maxSize-MinRoomSize) + MinRoomSize
```

**Recursion:** Each node splits up to **depth 4**, creating up to 16 leaf nodes (though most dungeons have 8-12 due to early termination when regions get too small).

**ASCII Diagram: BSP Tree Splitting**

```
Initial map (60x30):
┌──────────────────────────────────────────────────────────┐
│                                                          │
│                      (root node)                         │
│                                                          │
└──────────────────────────────────────────────────────────┘

After 1st split (vertical):
┌─────────────────────────┬────────────────────────────────┐
│                         │                                │
│      Left child         │       Right child              │
│                         │                                │
└─────────────────────────┴────────────────────────────────┘

After 2nd split (both children split horizontally):
┌─────────────────────────┬────────────────────────────────┐
│         A               │            C                   │
├─────────────────────────┼────────────────────────────────┤
│         B               │            D                   │
└─────────────────────────┴────────────────────────────────┘

After 4 splits (depth 4), you have 8-16 leaf regions.
```

---

### Step 2: Room Creation

Once the tree is fully split, each **leaf node** becomes a room.

**Room sizing logic** (from `dungeon.go:BSPNode.CreateRooms()`):

```go
// Room must fit within node with padding
maxW := min(MaxRoomSize, n.W-2)  // MaxRoomSize = 15
maxH := min(MaxRoomSize, n.H-2)

// Randomize dimensions (MinRoomSize = 6)
roomW := rng.Intn(maxW-MinRoomSize+1) + MinRoomSize
roomH := rng.Intn(maxH-MinRoomSize+1) + MinRoomSize

// Randomize position within node (with 1-tile padding)
roomX := n.X + rng.Intn(n.W-roomW-1) + 1
roomY := n.Y + rng.Intn(n.H-roomH-1) + 1
```

**Result:** Rooms are 6-15 tiles in each dimension, randomly positioned within their BSP region.

**ASCII Diagram: Rooms in Leaf Nodes**

```
Leaf nodes (A, B, C, D) with rooms placed inside:

┌─────────────────────────┬────────────────────────────────┐
│ A                       │ C                              │
│   ┌──────────┐          │    ┌────────┐                 │
│   │  Room 1  │          │    │ Room 3 │                 │
│   │          │          │    └────────┘                 │
│   └──────────┘          │                                │
├─────────────────────────┼────────────────────────────────┤
│ B                       │ D                              │
│      ┌─────────┐        │        ┌──────────┐           │
│      │ Room 2  │        │        │  Room 4  │           │
│      └─────────┘        │        └──────────┘           │
└─────────────────────────┴────────────────────────────────┘

Rooms are carved as TileFloor. Everything else is TileWall.
```

---

### Step 3: Corridor Carving

Rooms are connected by traversing the BSP tree and linking sibling nodes.

**Connection logic** (from `dungeon.go:connectRooms()`):

```go
func connectRooms(node *BSPNode, d *Dungeon, rng *rand.Rand) {
    if node.Left == nil || node.Right == nil {
        return  // Leaf node, no children to connect
    }

    // Recursively connect children first
    connectRooms(node.Left, d, rng)
    connectRooms(node.Right, d, rng)

    // Get representative rooms from left and right subtrees
    leftRoom := node.Left.GetRoom()
    rightRoom := node.Right.GetRoom()

    // Connect room centers with L-shaped corridor
    x1, y1 := leftRoom.Center()
    x2, y2 := rightRoom.Center()

    if rng.Float32() > 0.5 {
        d.carveHorizontalCorridor(x1, x2, y1)  // Move right/left
        d.carveVerticalCorridor(y1, y2, x2)    // Then up/down
    } else {
        d.carveVerticalCorridor(y1, y2, x1)    // Move up/down
        d.carveHorizontalCorridor(x1, x2, y2)  // Then right/left
    }
}
```

**ASCII Diagram: L-Shaped Corridors**

```
Connecting Room 1 and Room 3:

┌─────────────────────────┬────────────────────────────────┐
│                         │                                │
│   ┌──────────┐          │    ┌────────┐                 │
│   │  Room 1  │··········│····│ Room 3 │                 │
│   │     *    │          │    │   *    │                 │
│   └──────────┘          │    └────────┘                 │
├─────────────────────────┼────────────────────────────────┤
│                         │                                │
│      ┌─────────┐        │        ┌──────────┐           │
│      │ Room 2  │        │        │  Room 4  │           │
│      │    *    │········│········│     *    │           │
│      └─────────┘        │        └──────────┘           │
└─────────────────────────┴────────────────────────────────┘

* = room center
· = corridor (carved as TileFloor)

Room 1 → Room 3: Horizontal first, then vertical
Room 2 → Room 4: Vertical first, then horizontal (randomized)
```

**Why L-shaped?**
- Guarantees connectivity without complex pathfinding
- Creates interesting layouts (not just straight lines)
- Works well with BSP structure (connecting across splits)

---

## Key Constants

From `game/dungeon.go`:

```go
const (
    MinRoomSize = 6   // Minimum room width/height
    MaxRoomSize = 15  // Maximum room width/height
)
```

**BSP split depth:** 4 (hard-coded in `GenerateDungeon()`)

---

## Tile Types

```go
const (
    TileWall  Tile = iota  // '#' - Impassable
    TileFloor              // '.' or code char - Walkable
    TileDoor               // '>' - Stairs to next level
)
```

**Tile assignment:**
1. Initialize entire map as `TileWall`
2. Carve rooms as `TileFloor`
3. Carve corridors as `TileFloor`
4. Place one `TileDoor` in the last room

---

## Dungeon Function Reference

### `GenerateDungeon(width, height int, rng *rand.Rand, codeFile *CodeFile) *Dungeon`

**Purpose:** Main entry point for dungeon generation.

**Steps:**
1. Initialize tile map (all walls)
2. Create BSP root node
3. Split tree recursively (depth 4)
4. Create rooms in leaf nodes
5. Connect rooms with corridors
6. Return `Dungeon` struct

**Returns:** `*Dungeon` with tile map, room list, and code file reference.

### `BSPNode.Split(rng *rand.Rand, depth int)`

**Purpose:** Recursively partition the map into regions.

**Parameters:**
- `rng` — Random number generator (seeded)
- `depth` — Remaining recursion depth (stops at 0)

**Behavior:**
- Decides split direction (horizontal vs. vertical)
- Picks random split position
- Creates left and right children
- Recurses on both children

**Early termination:** If region is too small to split (< 2*MinRoomSize), returns without splitting.

### `BSPNode.CreateRooms(rng *rand.Rand)`

**Purpose:** Place rooms in leaf nodes.

**Behavior:**
- If node has children, recurse on them
- If leaf node, create a room with random size/position
- Skip if node is too small (< MinRoomSize+2 in either dimension)

### `connectRooms(node *BSPNode, d *Dungeon, rng *rand.Rand)`

**Purpose:** Link rooms with L-shaped corridors.

**Behavior:**
- Post-order traversal of BSP tree
- Connect left and right subtree representatives
- Randomly choose horizontal-first or vertical-first

### `Dungeon.carveHorizontalCorridor(x1, x2, y int)`

**Purpose:** Carve a horizontal line of floor tiles.

**Behavior:** Sets `Tiles[y][x] = TileFloor` for all x between x1 and x2.

### `Dungeon.carveVerticalCorridor(y1, y2, x int)`

**Purpose:** Carve a vertical line of floor tiles.

**Behavior:** Sets `Tiles[y][x] = TileFloor` for all y between y1 and y2.

### `Dungeon.PlaceDoor(rng *rand.Rand) (int, int)`

**Purpose:** Place the exit door in the last room.

**Behavior:**
- Picks the last room in the room list
- Randomizes position within the room (not on edges)
- Sets tile to `TileDoor`
- Returns door coordinates

---

## Code Text Backgrounds

Each dungeon level displays code from one of the scanned files as the floor background.

**Rendering logic** (from `game.go:render()`):

```go
// Use both y and x/40 to show 2x more code lines
lineIdx := (y*2 + x/40) % len(codeLines)
line := codeLines[lineIdx]

charIdx := x % 40
if x >= 40 {
    charIdx = x - 40
}

if charIdx < len(line) {
    ch = rune(line[charIdx])
} else {
    ch = '.'  // Fallback if line is shorter than x position
}
```

**Result:** Floor tiles show actual code characters, making each level visually unique.

---

## Modifying Dungeon Generation

### Change room sizes

Edit `game/dungeon.go`:

```go
const (
    MinRoomSize = 8   // Make rooms larger
    MaxRoomSize = 20
)
```

### Change BSP depth (more/fewer rooms)

Edit `game/dungeon.go:GenerateDungeon()`:

```go
root.Split(rng, 5)  // Increase depth = more rooms
```

**Warning:** Depth 5+ can create very small rooms or fail to generate.

### Change corridor style

Edit `game/dungeon.go:connectRooms()`:

Remove the random L-shape choice to always go horizontal-first:

```go
d.carveHorizontalCorridor(x1, x2, y1)
d.carveVerticalCorridor(y1, y2, x2)
```

### Add room decorations

Edit `game/state.go:generateLevel()` after room carving:

```go
// Place pillars in room centers
for _, room := range gs.Dungeon.Rooms {
    cx, cy := room.Center()
    gs.Dungeon.Tiles[cy][cx] = TileWall
}
```

---

## Testing Dungeon Generation

**Seed a specific dungeon:**

```go
rng := rand.New(rand.NewSource(12345))
dungeon := GenerateDungeon(80, 40, rng, nil)

// Verify room count
if len(dungeon.Rooms) < 5 {
    t.Errorf("Expected at least 5 rooms, got %d", len(dungeon.Rooms))
}
```

**Check connectivity:**

Use BFS to verify all rooms are reachable from the first room:

```go
visited := make(map[[2]int]bool)
queue := [][2]int{{startX, startY}}

for len(queue) > 0 {
    p := queue[0]
    queue = queue[1:]
    
    if dungeon.IsWalkable(p[0], p[1]) {
        visited[[2]int{p[0], p[1]}] = true
        // Add neighbors to queue...
    }
}

// Check if door is reachable
if !visited[[2]int{doorX, doorY}] {
    t.Error("Door is not reachable from start")
}
```

---

## Common Pitfalls

1. **Rooms clipping out of bounds:** Always check `x >= 0 && x < Width` before setting tiles.
2. **Disconnected rooms:** Rare, but can happen if `GetRoom()` returns nil. Ensure leaf nodes always create valid rooms.
3. **Overlapping corridors:** Not a bug—corridors can overlap, creating irregular shapes.
4. **Terminal too small:** Enforce minimum size (40x20) in `state.go:generateLevel()`.

---

## Further Reading

- [BSP Dungeon Generation Tutorial](http://www.roguebasin.com/index.php?title=Basic_BSP_Dungeon_generation) (RogueBasin)
- [architecture.md](./architecture.md) — System overview
- [modding.md](./modding.md) — How to customize the game
