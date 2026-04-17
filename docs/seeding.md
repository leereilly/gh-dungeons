# Deterministic Seeding

Every dungeon in `gh-dungeons` is generated from a deterministic 64-bit integer seed derived from your repository. Same repo + same commit + same code content ⇒ same dungeon. Different fork ⇒ different dungeon.

This document explains exactly how that seed is computed and what can and cannot change it.

Everything here is in `game/scanner.go`.

---

## The Recipe

```go
// game/scanner.go:computeSeed
func computeSeed(files []CodeFile) int64 {
    h := sha256.New()
    if repoID := getRepoIdentity();  repoID   != "" { h.Write([]byte(repoID))    }
    if commitSHA := getGitCommitSHA(); commitSHA != "" { h.Write([]byte(commitSHA)) }
    for _, f := range files {
        h.Write([]byte(f.SHA))     // f.SHA is the raw 32-byte SHA256 of the file content
    }
    sum := h.Sum(nil)
    return int64(binary.BigEndian.Uint64(sum[:8]))
}
```

If `findCodeFiles` returns nothing, `game.New` substitutes `seed = 42`.

The same `*rand.Rand` instance is used for:
- BSP splitting and room placement (per level)
- Door placement (per level)
- Merge-conflict trap placement (per level)
- Enemy spawns (count and YAML-registry draws)
- Potion spawns
- Corridor L-shape orientation
- Merge-conflict spread-tile shuffling
- Runtime animations that ask for random characters (System 1 fire after 2 turns)

Changing **any** consumer's RNG-call order or count shifts every subsequent result. See the "Modding Caveat" at the end.

---

## The Three Inputs

### 1. Repo Identity (`getRepoIdentity`)

```go
// preferred
cmd := exec.Command("git", "config", "--get", "remote.origin.url")

// fallback
cmd = exec.Command("git", "rev-parse", "--show-toplevel")  // then filepath.Base
```

- Prefers the remote origin URL (e.g., `https://github.com/leereilly/gh-dungeons.git`). Distinct across forks.
- Falls back to the basename of the repo root (e.g., `gh-dungeons`).
- If neither is available, the contribution is an empty string.

### 2. Commit SHA (`getGitCommitSHA`)

```go
exec.Command("git", "rev-parse", "HEAD")
```

Full 40-char SHA, or empty string if not in a repo. Uncommitted changes don't affect this.

### 3. Code File SHAs

From `findCodeFiles(cwd, minLines=60, maxFiles=5)`:

1. Recursively walk from `cwd`.
2. Skip any directory whose name starts with `.` (catches `.git`, `.github`, `.vscode`, ...) or matches `node_modules`, `vendor`, `dist`, `build`.
3. Filter files by extension against a 39-entry allow-list (`.go`, `.py`, `.ts`, `.rs`, `.ex`, `.clj`, ... — see `scanner.go:codeExtensions`).
4. Read each file via `bufio.Scanner`, collect all lines, require at least 60 lines.
5. Compute `SHA256(strings.Join(lines, "\n"))`. Note the `\n` join: this is not a byte-for-byte SHA of the file — trailing newlines and `\r\n` line endings are **normalized out**, so CRLF vs LF does not affect the seed.
6. Sort candidates by line count descending.
7. Keep the first 5.

Those top-5 file SHAs then feed the seed in the order they were sorted.

---

## What changes the seed

| Change                                          | Seed changes? |
| ----------------------------------------------- | ------------- |
| New commit                                      | ✅ (commit SHA) |
| Edit a scanned code file (uncommitted)          | ✅ (file SHA) |
| Add/remove a code file that enters/leaves the top-5 | ✅ |
| Edit a non-code file (README, image)            | ❌ |
| Edit a code file that is not in the top-5       | ❌ |
| Switch branches (if HEAD SHA changes)           | ✅ |
| Change `git remote origin.url`                  | ✅ |
| Rename the repo directory (if no origin URL set)| ✅ |
| Change terminal size                            | ❌ seed, ✅ map dimensions (w/h passed into `GenerateDungeon`) |
| Convert LF ↔ CRLF in a scanned file             | ❌ (join normalizes) |
| Change your OS, Go version, or username         | ❌ seed; see caveats below |

---

## Reproducibility Checklist

To get the exact same dungeon on two machines:

1. **Same repo identity.** Clone from the same URL, or ensure `git config --get remote.origin.url` returns the same string. If you rely on the directory-name fallback, rename the directory identically.
2. **Same commit.** `git checkout <sha>`; confirm with `git rev-parse HEAD`.
3. **Same working tree for scanned files.** No uncommitted edits to the top-5 longest code files. A safe belt-and-braces check: `git status --porcelain` comes back empty.
4. **Same terminal size** if you want identical maps. The seed determines the random sequence; the map *dimensions* are computed from the terminal. Two different terminal sizes + same seed ⇒ two different dungeons (both deterministic for their size).

**Not required** for reproducibility: OS, username, Go toolchain version (as long as `math/rand` semantics don't change — see below).

---

## Why Forks Differ

A fork has a different `remote.origin.url`:

```
origin: https://github.com/leereilly/gh-dungeons.git   # upstream
origin: https://github.com/alice/gh-dungeons.git       # alice's fork
```

Even at the identical commit SHA and identical code, the first input to the SHA256 differs, which avalanches through the whole hash. Alice gets a completely different dungeon.

This is deliberate: speedrun strategies cannot be "spoiled" just by cloning a famous fork.

---

## Seed Flow Diagram

```
┌──────────────────────┐
│ git remote origin URL │──┐
│  (or repo root name)  │  │
└──────────────────────┘  │
┌──────────────────────┐  │
│ git rev-parse HEAD    │──┼──► SHA256 ──► int64 (first 8 bytes, BE) ──► seed
└──────────────────────┘  │
┌──────────────────────┐  │
│ SHA256 of top-5 files │──┘
│ (≥60 lines, sorted    │
│  desc by line count)  │
└──────────────────────┘
         │
         ▼
  rand.New(rand.NewSource(seed))
         │
         ├──► Level 1: BSP, rooms, door, trap, spawns
         ├──► Level 2: same RNG, continues the stream
         ├──► ...
         └──► Level 5
```

The RNG is created **once** in `NewGameState`. Every level reuses it. There is no per-level reseed.

---

## Default Seed Fallback

If `findCodeFiles` returns an empty slice (empty directory, no matching files, or all files <60 lines), the seed is forced to `42`:

```go
// game/game.go:New
seed := computeSeed(codeFiles)
if len(codeFiles) == 0 {
    seed = 42
}
```

This guarantees the game always runs but makes "no-code-files" runs produce identical layouts.

Note: `computeSeed` can still produce a nonzero seed when `files` is empty (if repo identity or HEAD is set), because the fallback triggers on `len(codeFiles) == 0` regardless of what `computeSeed` returned. In practice this only matters in pathological "git repo with no qualifying code files" cases.

---

## RNG Guarantees

Go's `math/rand` is deterministic: same seed produces the same sequence across platforms and architectures. This is intentional and has been stable since Go 1.8.

Two caveats:

1. **If the Go standard library ever changes `math/rand`'s default algorithm** (unlikely; Go 1 compatibility covers this), seeds might produce different maps in a new Go version. At time of writing (`go 1.25.5`), this has not happened.
2. **We do not use `math/rand/v2`.** If a future refactor switches to `v2` (different algorithm), every existing seed will produce a different dungeon.

---

## Modding Caveat (important)

Changing what the RNG is consumed for changes the seed's effective meaning. Specifically:

- **Adding a new YAML monster** shifts `GetRandomMonster`'s index space — the same RNG draw now picks a different monster. Existing seeds produce different enemy layouts.
- **Adding a unique monster** adds extra RNG consumption per level (for its placement), shifting everything that follows.
- **Changing `numEnemies` or `numPotions` formulas** shifts consumption.
- **Inserting a new placement call into `generateLevel`** before another call shifts consumption.

Rule of thumb: if you care about reproducible runs, append new RNG consumers at the **end** of `generateLevel`, not in the middle.

---

## Debugging With Seeds

### Print the active seed

Temporarily add to `game/game.go:New`:

```go
fmt.Fprintf(os.Stderr, "seed=%d\n", seed)
```

(Don't print to `stdout` — tcell has the screen.)

### Pin a seed for testing

```go
// In a test
rng := rand.New(rand.NewSource(12345))
d   := GenerateDungeon(80, 27, rng, nil)
```

Or inject directly:

```go
gs := NewGameState(codeFiles, /*seed=*/12345, 80, 40)
```

### Diff two seeds

```bash
# Two different repos / commits
(cd repoA && go run .)  # observe map 1
(cd repoB && go run .)  # observe map 2
```

For deterministic A/B comparisons in tests, build a map fingerprint:

```go
func fingerprint(d *Dungeon) string {
    h := sha256.New()
    for _, row := range d.Tiles { h.Write([]byte(fmt.Sprintf("%v", row))) }
    return hex.EncodeToString(h.Sum(nil))
}
```

---

## Username Is Not Part of the Seed

`getUsername()` resolves from `gh api user --jq .login` (preferred) or `git config user.name` (fallback). It's used **only** for the welcome message ("Welcome adventurer, @you") shown for the first 10 moves.

It does not feed into `computeSeed`. Alice and Bob cloning the same repo at the same commit will get the same dungeon — only the welcome line differs.

---

## Seed Entropy Sketch

- Repo identity: 20–100 bytes.
- Commit SHA (as string): 40 bytes.
- Top-5 file SHA256s: 32 bytes × up to 5 = 160 bytes.

Total input: ~250 bytes. SHA256 output: 256 bits. We take the top 64 bits as the seed. That's 2⁶⁴ possible seeds — collision-wise, astronomically safe.

---

## See Also

- [architecture.md](./architecture.md) — where the seed plugs into `NewGameState`
- [dungeon-generation.md](./dungeon-generation.md) — what the RNG actually does
- [modding.md](./modding.md) — preserving reproducibility while adding content
