# Monsters YAML Reference

All enemies in `gh-dungeons` are defined in `game/monsters.yaml`. The file is **embedded into the binary at build time** via `//go:embed`, so adding a monster requires a rebuild but no separate asset deployment.

---

## File Location and Loading

```go
// game/monster.go
//go:embed monsters.yaml
var monstersYAML []byte
```

Parsed lazily on first call to `GetMonsterRegistry()`. A bad YAML file logs nothing and returns an empty registry, at which point `GetRandomMonster` falls back to a hard-coded `Bug` sentinel — so a broken YAML file is silently survivable (annoying, but not a crash). If you're debugging a mod and "nothing is spawning right," check for YAML syntax errors first.

---

## Schema

Each monster is an entry under the top-level `monsters:` list. Example:

```yaml
monsters:
  - name: "Zombie"
    description: "A shambling corpse that moves slowly in straight lines"
    token: "z"
    color: "green"
    hp: 3
    strength: 2
    speed: 0.5
    movement: "straight"
    attack_range: 1
    experience_value: 15
    abilities: []
    # unique: false   (omit or false for normal monsters)
```

### Field reference

| Field              | Type      | Required | Used by code?               | Notes |
| ------------------ | --------- | -------- | --------------------------- | ----- |
| `name`             | string    | yes      | `Entity.Name`, messages, `KilledBy` | Also the registry key. |
| `description`      | string    | no       | **Unused** at runtime       | Documentation only. |
| `token`            | string    | yes      | `Entity.Symbol = rune(token[0])` | Only the first rune is used. Empty string → `'m'` fallback. |
| `color`            | string    | no       | `Entity.Color` (via `ColorFromString`) | ⚠️ Currently unused at render time; all enemies draw red. |
| `hp`               | int       | yes      | `Entity.HP`, `Entity.MaxHP` | |
| `strength`         | int       | yes      | `Entity.Damage`             | |
| `speed`            | float64   | yes      | `Entity.Speed`              | `1.0` = normal. `0.5` = every other turn. `< 1.0` triggers `TurnAccumulator`. |
| `movement`         | string    | yes      | `Entity.Movement`           | See below. |
| `attack_range`     | int       | yes      | `Entity.AttackRange`        | `1` = melee, `≥2` = ranged (Chebyshev). `≤0` silently treated as `1` in `enemyAttacks`. |
| `experience_value` | int       | no       | `Entity.ExperienceValue`    | ⚠️ Currently unused (no XP system). |
| `abilities`        | string[]  | no       | `Entity.Abilities`          | ⚠️ Currently unused (no status-effect system). |
| `unique`           | bool      | no       | spawn grouping              | See below. |

### `movement` values (`MovementType` enum in `monster.go`)

| Value         | Behavior                                                                 |
| ------------- | ------------------------------------------------------------------------ |
| `any`         | 8-directional; prefers diagonal, falls back to cardinal if blocked.       |
| `straight`    | Cardinal only; moves along the axis with greater distance to the player. |
| `diagonal`    | Diagonal only; skips turn if on same row or column as the player.        |
| `horizontal`  | Left/right only; skips turn if in the same column as the player.         |
| `stationary`  | Never moves. Can still attack if the player is within `attack_range`.    |

Unknown values are not validated; the YAML just won't match any branch in `moveEnemies`, and the enemy will behave as if `any`-default (it will still apply general fallback logic). **Always use one of the five strings above.**

### `unique: true`

- Non-unique (`unique: false` or omitted) monsters go into the random-spawn pool. Each random-spawn slot picks one uniformly at random from this pool.
- Unique monsters are spawned **one per level, every level**, on top of the random spawns. Use this for mini-bosses, gimmick enemies, or flavor encounters.

Currently `Hermit Crab` is the only unique monster in the shipped YAML.

---

## Full Shipped `monsters.yaml`

```yaml
monsters:
  - name: "Zombie"
    description: "A shambling corpse that moves slowly in straight lines"
    token: "z"
    color: "green"
    hp: 3
    strength: 2
    speed: 0.5
    movement: "straight"
    attack_range: 1
    experience_value: 15
    abilities: []

  - name: "Bug"
    description: "A nasty little software bug"
    token: "b"
    color: "red"
    hp: 1
    strength: 1
    speed: 1.0
    movement: "any"
    attack_range: 1
    experience_value: 5
    abilities: []

  - name: "Scope Creep"
    description: "An insidious feature that keeps growing"
    token: "s"
    color: "yellow"
    hp: 3
    strength: 2
    speed: 1.0
    movement: "any"
    attack_range: 1
    experience_value: 10
    abilities: []

  - name: "Hermit Crab"
    description: "A stubborn crustacean that only moves sideways"
    token: "H"
    color: "red"
    hp: 2
    strength: 2
    speed: 1.0
    movement: "horizontal"
    attack_range: 1
    experience_value: 20
    abilities: []
    unique: true
```

---

## Color values recognized by `ColorFromString`

Even though `Color` isn't currently rendered, the following strings are legal (anything else silently becomes white):

`red`, `green`, `blue`, `yellow`, `cyan`, `magenta`, `white`, `orange`, `purple`, `gray` / `grey`.

If/when enemy rendering starts respecting per-monster color, these are the values that will "just work."

---

## Adding a Monster — minimal example

```yaml
  - name: "Tech Debt"
    description: "Accumulates interest. Ignore at your peril."
    token: "T"
    color: "purple"
    hp: 5
    strength: 1
    speed: 0.5
    movement: "any"
    attack_range: 1
    experience_value: 25
    abilities: []
```

Rebuild:

```bash
go build -o gh-dungeons
```

That's it. The registry picks it up on next launch, and it joins the random-spawn pool with uniform weight relative to other non-unique monsters. See [modding.md](./modding.md) for more involved examples and test patterns.

---

## Caveats

- **Embedding means rebuild.** Editing `monsters.yaml` has no effect until `go build`.
- **Silent failures.** A malformed YAML file produces an empty registry. Test your changes by running `go test ./game -count=1`; at minimum `TestHermitCrabIsRedH` should pass and non-empty registry behavior should hold.
- **No validation.** Unknown movement types, negative HP, empty tokens — all accepted. Invalid data manifests as weird in-game behavior.
- **Case sensitivity.** `movement` values are lowercase strings. `color` strings are lowercase. `name` is free-form; it's used verbatim in messages and as `KilledBy`.
- **Order matters for determinism.** The YAML load order determines `names` / `uniqueNames` slice order, which is what `GetRandomMonster(rng)` indexes. Reordering entries in `monsters.yaml` will shift which monsters spawn in which slots for a given seed. If speedrun-identical runs matter to you, don't rearrange without reason.

---

## See Also

- [entities.md](./entities.md) — runtime behavior, AI, combat
- [modding.md](./modding.md) — end-to-end modding walkthroughs
- [seeding.md](./seeding.md) — why YAML order affects deterministic runs
