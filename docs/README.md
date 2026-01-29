# Documentation Index

Welcome to the gh-dungeons documentation! This directory contains technical documentation for modders, contributors, and anyone who wants to understand how the game works.

---

## Documentation Files

### [architecture.md](./architecture.md)
**For:** Contributors, developers, curious players  
**Topics:**
- Repository structure and entry points
- Core systems overview (GameState, rendering, input)
- Build and test commands
- Code style and conventions

**Start here** if you want a high-level tour of the codebase.

---

### [dungeon-generation.md](./dungeon-generation.md)
**For:** Modders, algorithm enthusiasts, map designers  
**Topics:**
- Binary Space Partitioning (BSP) algorithm explained
- Room creation and sizing
- L-shaped corridor carving
- ASCII diagrams showing the generation process
- How to modify room sizes and BSP depth

**Start here** if you want to understand or modify how dungeons are built.

---

### [entities.md](./entities.md)
**For:** Modders, game designers, balance tweakers  
**Topics:**
- Complete entity reference (player, enemies, items)
- Combat system and auto-attack mechanics
- Enemy AI and line-of-sight logic
- Spawn formulas and rates
- Konami code implementation
- Merge conflict trap details

**Start here** if you want to add new enemies, change stats, or understand combat.

---

### [seeding.md](./seeding.md)
**For:** Speedrunners, testers, reproducibility nerds  
**Topics:**
- Deterministic RNG seed computation
- What goes into the seed (repo URL, commit SHA, file hashes)
- Why forks get different dungeons
- Reproducibility checklist
- Code file scanning logic

**Start here** if you want to understand why your dungeon is unique or how to get the same dungeon twice.

---

### [modding.md](./modding.md)
**For:** Modders, hackers, tinkerers  
**Topics:**
- Step-by-step guide to adding new enemies
- Adding new items with custom effects
- Changing spawn rates and dungeon parameters
- Adding status effects (poison example)
- Adding new traps
- Testing your mods

**Start here** if you want to create content for the game.

---

## Quick Start by Goal

**"I want to understand the code"**  
→ Start with [architecture.md](./architecture.md)

**"I want to add a new enemy type"**  
→ Jump to [modding.md](./modding.md) → "Adding a New Enemy"

**"I want to change how dungeons look"**  
→ Read [dungeon-generation.md](./dungeon-generation.md) → "Modifying Dungeon Generation"

**"I want the same dungeon on two machines"**  
→ Check [seeding.md](./seeding.md) → "Reproducibility Checklist"

**"I want to know what the Konami code does"**  
→ See [entities.md](./entities.md) → "Konami Code"

---

## Contributing Documentation

Found an error? Want to expand a section? PRs welcome!

**Guidelines:**
- Keep the tone witty but precise (see existing docs for examples)
- Always cite code locations (file names and function names)
- Use ASCII diagrams for spatial/procedural concepts
- Don't invent features—document what's actually in the code
- Test your examples before submitting

---

## Player-Facing Documentation

For gameplay instructions, controls, and general info, see the main [README.md](../README.md) in the root directory.

This `docs/` directory is specifically for **technical documentation** aimed at people who want to read or modify the code.

---

*"Read the dungeon. Then write the scrolls."* — Dungeon Scribe
