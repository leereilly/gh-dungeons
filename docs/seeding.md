# Deterministic Seeding

How gh-dungeons creates reproducible dungeons using deterministic RNG seeding.

---

## Overview

Every dungeon in gh-dungeons is generated from a **deterministic seed**. The same repository at the same commit with the same code files will always produce the same dungeon layouts, enemy placements, and item distributions.

This enables:
- **Reproducible gameplay** — Speedrunners can compete on the same seed
- **Debugging** — Replay the exact same dungeon to test fixes
- **Fork uniqueness** — Forks get different dungeons even at the same commit

---

## Seed Computation

### The Formula

From `game/scanner.go:computeSeed()`:

```go
func computeSeed(files []CodeFile) int64 {
    h := sha256.New()

    // 1. Include git repo identity (remote origin or repo name)
    if repoID := getRepoIdentity(); repoID != "" {
        h.Write([]byte(repoID))
    }

    // 2. Include current git commit SHA
    if commitSHA := getGitCommitSHA(); commitSHA != "" {
        h.Write([]byte(commitSHA))
    }

    // 3. Include code file content hashes
    for _, f := range files {
        h.Write([]byte(f.SHA))
    }

    sum := h.Sum(nil)
    return int64(binary.BigEndian.Uint64(sum[:8]))
}
```

**Result:** A 64-bit integer seed derived from three components.

---

## Seed Components

### 1. Repository Identity

**Purpose:** Ensure forks get different dungeons, even at the same commit.

**Resolution order:**

1. **Git remote origin URL** (preferred):
   ```bash
   git config --get remote.origin.url
   ```
   Example: `https://github.com/leereilly/gh-dungeons.git`

2. **Repository root directory name** (fallback):
   ```bash
   git rev-parse --show-toplevel | xargs basename
   ```
   Example: `gh-dungeons`

**Why remote origin URL?**
- Unique across forks (e.g., `yourname/gh-dungeons` vs `leereilly/gh-dungeons`)
- Persists across directory renames
- Distinguishes between different clones of the same repo

**Fallback behavior:**
- If no remote origin is configured, uses directory name
- If not in a git repo, repo identity is empty string

---

### 2. Commit SHA

**Purpose:** Different commits produce different dungeons.

**Command:**
```bash
git rev-parse HEAD
```

**Example:** `a3f2b1c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3`

**Behavior:**
- Full 40-character SHA is used
- Uncommitted changes don't affect the seed (only committed state matters)
- Stashes and working directory changes are ignored

**Fallback:** If not in a git repo or `HEAD` doesn't exist, commit SHA is empty string.

---

### 3. Code File Content Hashes

**Purpose:** Ensure code changes affect the seed, even within the same commit.

**File selection:**
- Scans repository for code files (see [Code File Scanning](#code-file-scanning))
- Keeps top 5 longest files (≥60 lines)
- Computes SHA256 hash of each file's content

**Per-file hash computation:**
```go
content := strings.Join(lines, "\n")
hash := sha256.Sum256([]byte(content))
sha := string(hash[:])  // Binary SHA, not hex
```

**Why file hashes?**
- Uncommitted changes (even single-line edits) change the seed
- Different code = different dungeon
- Ensures reproducibility across machines with same commit

**Fallback:** If no code files are found, uses default seed `42`.

---

## Code File Scanning

### File Discovery

From `game/scanner.go:findCodeFiles()`:

**Walk algorithm:**
1. Recursively traverse current directory
2. Skip hidden directories (`.git`, `.github`, etc.)
3. Skip common vendor directories (`node_modules`, `vendor`, `dist`, `build`)
4. Filter files by extension (see [Supported Extensions](#supported-extensions))
5. Read files and count lines
6. Keep files with ≥60 lines

**Why skip vendor directories?**
- Vendor code changes frequently but isn't "your" code
- Reduces scan time
- Focuses on actual project code

### Supported Extensions

From `game/scanner.go:codeExtensions`:

```
.go    .js    .ts    .tsx   .jsx   .py    .rb    .rs
.c     .cpp   .cc    .h     .hpp   .java  .cs    .swift
.kt    .scala .php   .pl    .sh    .bash  .zsh   .lua
.r     .m     .mm    .zig   .nim   .ex    .exs   .erl
.hs    .ml    .fs    .clj   .lisp  .el    .vim
```

**55 supported languages.** If your favorite language is missing, add it to the map!

### File Prioritization

Files are sorted by line count (longest first):

```go
sort.Slice(candidates, func(i, j int) bool {
    return len(candidates[i].Lines) > len(candidates[j].Lines)
})
```

**Why prioritize long files?**
- Makes floor backgrounds more interesting (more code to display)
- Core files tend to be longer
- Avoids tiny utility files

**Max files kept:** 5 (hard-coded in `game.go:New()`):

```go
codeFiles, err := findCodeFiles(cwd, 60, 5)
```

---

## Seed Flow Diagram

```
Repository State
├─> Git Remote Origin URL ─────┐
│   (or repo directory name)    │
│                                │
├─> Git Commit SHA ─────────────┤
│   (HEAD)                       │
│                                ├──> SHA256 Hash ──> int64 Seed
│                                │
└─> Code File Content Hashes ───┘
    (top 5 files, ≥60 lines)
         ↓
    Level 1 Dungeon RNG
         ↓
    Level 2 Dungeon RNG
         ↓
         ...
```

---

## Reproducibility Checklist

To get the **exact same dungeon** on two machines:

1. **Same repository**
   - Clone from the same remote origin URL
   - OR rename local directory to match

2. **Same commit**
   - `git checkout <commit-sha>`
   - Verify with `git rev-parse HEAD`

3. **Same code files**
   - No uncommitted changes
   - No untracked files in scanned directories

4. **Same terminal size** (for same map dimensions)
   - Level layout is deterministic, but map size affects room placement
   - Terminal size is stored in `GameState` and used in `generateLevel()`

**Not required:**
- Same operating system
- Same Go version (as long as `math/rand` is deterministic)
- Same username (doesn't affect seed)

---

## Why Forks Get Different Dungeons

**Scenario:** You fork `leereilly/gh-dungeons` to `yourname/gh-dungeons`.

**What changes:**
- Remote origin URL: `github.com/leereilly/...` → `github.com/yourname/...`
- Seed component 1 changes
- SHA256 hash produces a completely different seed
- You get a unique dungeon!

**Benefit:** Forkers can't "spoil" speedrun strategies by playing the same seed.

---

## Seed Stability

### What changes the seed

- **Committing code** — New commit SHA
- **Editing code** — Changes file content hashes
- **Adding/removing files** — Changes which files are scanned
- **Changing remote origin** — Changes repo identity
- **Renaming repo directory** (if no remote origin configured)

### What doesn't change the seed

- **Uncommitted changes to non-code files** (e.g., README, images)
- **Stashing changes**
- **Switching branches** (unless commit SHA changes)
- **Terminal size changes** (seed stays same, but map dimensions change)
- **Username changes** (username doesn't affect seed)

---

## Default Seed Fallback

If no code files are found (e.g., running in an empty directory):

```go
seed := computeSeed(codeFiles)
if len(codeFiles) == 0 {
    seed = 42  // Default seed
}
```

**Result:** You'll always get the same "empty repo" dungeon.

---

## RNG Usage

### Seed to RNG Instance

From `game/state.go:NewGameState()`:

```go
rng := rand.New(rand.NewSource(seed))
gs.RNG = rng
```

**RNG lifetime:**
- Created once at game start
- Persists across all levels
- Used for all randomization (dungeon gen, enemy placement, item placement, etc.)

### RNG Guarantees

Go's `math/rand` package is **deterministic**:
- Same seed = same sequence of random numbers
- Platform-independent (as of Go 1.8+)
- Not cryptographically secure (but we don't need that)

**Caveat:** If Go changes the `rand` implementation in a future version, seeds might produce different results. This is unlikely but possible.

---

## Debugging with Seeds

### Print the seed

Add to `game.go:New()`:

```go
fmt.Printf("Seed: %d\n", seed)
```

### Test a specific seed

Create a standalone test:

```go
func TestSpecificSeed(t *testing.T) {
    rng := rand.New(rand.NewSource(12345))
    dungeon := GenerateDungeon(80, 40, rng, nil)
    
    // Assert properties of this specific dungeon
    if len(dungeon.Rooms) != 8 {
        t.Errorf("Expected 8 rooms, got %d", len(dungeon.Rooms))
    }
}
```

### Compare two seeds

```go
seed1 := computeSeed(files1)
seed2 := computeSeed(files2)

if seed1 == seed2 {
    fmt.Println("Same dungeon")
} else {
    fmt.Println("Different dungeon")
}
```

---

## Seeding Best Practices

### For speedrunning

1. Agree on a specific commit SHA
2. Don't modify code files
3. Use the same terminal size if possible
4. Share the repo identity (remote origin URL)

### For testing

1. Use hard-coded seeds: `rand.New(rand.NewSource(12345))`
2. Test edge cases with known seeds
3. Don't rely on RNG for security-critical logic

### For modding

1. Be aware that changing constants (e.g., `MinRoomSize`) doesn't change the seed
2. Adding new enemies doesn't change dungeon layout (RNG is called in the same order)
3. Changing spawn formulas changes entity placement for the same seed

---

## Seed Entropy Analysis

**Inputs:**
- Repo identity: ~50-100 bytes (URL or directory name)
- Commit SHA: 40 bytes (hex string)
- File hashes: 32 bytes × 5 files = 160 bytes

**Total input:** ~250-300 bytes

**SHA256 output:** 256 bits of entropy

**Seed output:** 64 bits (first 8 bytes of SHA256)

**Collision probability:**
- 2^64 possible seeds (~18 quintillion)
- Birthday paradox: 50% collision chance after ~2^32 seeds (~4 billion repos)

**In practice:** Collisions are astronomically unlikely for real-world repos.

---

## Username Detection

**Not part of the seed**, but displayed in the welcome message.

From `game/scanner.go:getUsername()`:

**Resolution order:**

1. **GitHub username** (via `gh` CLI):
   ```bash
   gh api user --jq .login
   ```
   Result: `@username`

2. **Git user.name** (fallback):
   ```bash
   git config user.name
   ```
   Result: `Username` (no `@` prefix)

**Display logic** (from `game.go:render()`):

```go
if g.state.MoveCount < 10 && g.state.Username != "" {
    displayMsg = fmt.Sprintf("Welcome adventurer, %s", g.state.Username)
}
```

**Shown for:** First 10 moves, bottom-left message area.

---

## Further Reading

- [architecture.md](./architecture.md) — System overview
- [dungeon-generation.md](./dungeon-generation.md) — How RNG is used for BSP
- [modding.md](./modding.md) — How to control randomness in mods
