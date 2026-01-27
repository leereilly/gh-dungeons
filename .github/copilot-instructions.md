# GitHub Copilot Instructions for gh-dungeons

## Project Overview

This is a roguelike dungeon crawler GitHub CLI extension written in Go. It procedurally generates dungeons using a repository's code files, creating a playable game where the codebase becomes the dungeon floor.

## Technology Stack

- **Language**: Go 1.x
- **UI Library**: tcell (github.com/gdamore/tcell/v2) for terminal UI
- **CLI Framework**: GitHub CLI extension
- **Testing**: Standard Go testing (`go test`)
- **Build**: Standard Go build toolchain

## Code Style and Conventions

### Go Standards
- Follow standard Go formatting (`gofmt`, `goimports`)
- Use standard Go project layout with `main.go` at root and logic in `game/` package
- Prefer explicit error handling over panics
- Use descriptive variable names (no single-letter variables except for common idioms like `i` in loops)
- Keep functions focused and concise

### Game Architecture
- Game state management is centralized in `game/state.go`
- Dungeon generation uses Binary Space Partitioning (BSP) tree algorithm
- Entity management (player, enemies, items) is in `game/entity.go`
- File scanning logic is in `game/scanner.go`
- Deterministic random generation seeded by SHA hash of code files

### Naming Conventions
- Use PascalCase for exported functions and types
- Use camelCase for unexported functions and variables
- Struct types should be named clearly (e.g., `GameState`, `Entity`, `Dungeon`)
- Constants should use PascalCase (Go convention)

## Testing Guidelines

- Write unit tests for game logic in `game/` package
- Test files follow `*_test.go` naming convention
- Run tests with `go test ./...`
- Ensure deterministic behavior is testable (same seed = same dungeon)
- Mock file system operations when testing scanner functionality

## Building and Running

- Build: `go build -o gh-dungeons`
- Test: `go test ./...`
- Run: `gh dungeons` (requires GitHub CLI and extension installation)
- The game must be run from within a Git repository

## Game Design Principles

- Maintain roguelike feel with ASCII graphics and permadeath
- Keep gameplay simple but engaging (5 levels, auto-attack, fog of war)
- Code files should remain visible as background text in dungeons
- Dungeon generation must be deterministic based on repository contents
- Preserve classic roguelike controls (vi keys, arrow keys, WASD)

## Security and Best Practices

- Never commit secrets or sensitive data
- Validate file paths before reading to prevent directory traversal
- Handle file I/O errors gracefully
- Ensure game exits cleanly on all error conditions
- Respect user's repository - only read files, never modify them

## Dependencies

- Keep dependencies minimal and well-maintained
- Use Go modules (`go.mod`) for dependency management
- Document any new dependencies and their purpose
- Prefer standard library when possible

## Documentation

- Keep README.md up to date with features and usage
- Document any new gameplay mechanics or controls
- Include clear error messages for common issues
- Use clear commit messages following conventional commits style

## Performance Considerations

- Game should start quickly (< 1 second)
- File scanning should be efficient (only scan necessary files)
- Dungeon generation should be fast even for large repositories
- Terminal rendering should be smooth (no visible flickering)

## Compatibility

- Must work as a GitHub CLI extension (`gh extension`)
- Support common terminal emulators and sizes
- Handle both Unix-like systems and Windows
- Code should work with multiple Go versions (stay compatible with stable releases)
