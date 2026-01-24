# gh-dungeons 🎮

A roguelike dungeon crawler that turns your codebase into a playable game! This GitHub CLI extension procedurally generates dungeons using your repository's code files.

![Demo GIF](assets/demo.gif)

## Installation

```bash
gh extension install leereilly/gh-dungeons
```

Or build from source:
```bash
git clone https://github.com/leereilly/gh-dungeons
cd gh-dungeons
go build -o gh-dungeons
```

## Usage

Navigate to any Git repository and run:
```bash
gh dungeons
```

The game scans your repository for code files (60+ lines) and uses them to:
- Display as dark gray background text in the dungeon
- Seed the random dungeon generation (same repo = same dungeons!)

## Controls

| Key | Action |
|-----|--------|
| `↑` `w` `k` | Move up |
| `↓` `s` `j` | Move down |
| `←` `a` `h` | Move left |
| `→` `d` `l` | Move right |
| `y` `u` `b` `n` | Diagonal movement |
| `q` `Esc` | Quit |

## Gameplay

- **You** are `@` with 10 HP
- **Bugs** `b` - Weak enemies (1 HP, 1 damage)
- **Scope Creeps** `c` - Tougher enemies (3 HP, 2 damage)
- **Health Potions** `+` - Restore 3 HP
- **Door** `>` - Descend to the next level

### Features

- 🗺️ **BSP-tree dungeon generation** - procedurally created rooms and corridors
- 👁️ **Fog of war** - limited vision radius, explored areas stay visible
- 🤖 **Enemy AI** - enemies chase you when in line of sight
- ⚔️ **Auto-attack** - automatically attack adjacent enemies
- 📊 **Stats tracking** - kills and levels cleared

### Objective

Survive 5 dungeon levels by finding the hidden door `>` on each floor. Kill bugs and scope creeps, collect potions, and make it to the end!

## How It Works

1. Scans the current directory for code files (.go, .js, .py, .rs, etc.)
2. Selects 3-5 files with 60+ lines of code
3. Computes a SHA hash of the files to seed the RNG
4. Generates deterministic dungeons using Binary Space Partitioning
5. Your code appears as the dungeon floor background!

### Dungeon Generation

Dungeons are procedurally generated using **Binary Space Partitioning (BSP) trees**:

1. **Partitioning** - The map starts as a single rectangle, then recursively splits into smaller sections (either horizontally or vertically) based on the area's aspect ratio
2. **Room creation** - Each leaf node of the BSP tree becomes a room with randomized dimensions (6-15 tiles)
3. **Corridor carving** - Rooms are connected via L-shaped corridors by traversing the BSP tree and linking sibling nodes
4. **Deterministic seeding** - A SHA hash of your code files seeds the RNG, so the same repository always generates the same dungeon layouts

## License

MIT
