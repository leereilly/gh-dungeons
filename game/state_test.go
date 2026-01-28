package game

import (
	"math/rand"
	"testing"
)

func TestKonamiCode(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:          1,
		MaxLevel:       5,
		RNG:            rand.New(rand.NewSource(42)),
		KonamiSequence: make([]string, 0),
		Invulnerable:   false,
	}

	// Test correct Konami code sequence
	konamiCode := []string{"up", "up", "down", "down", "left", "right", "left", "right", "b", "a"}

	for i, key := range konamiCode {
		gs.CheckKonamiCode(key)
		if i < 9 {
			// Should not be invulnerable yet
			if gs.Invulnerable {
				t.Errorf("Player became invulnerable too early at step %d", i)
			}
		}
	}

	// After all 10 keys, should be invulnerable
	if !gs.Invulnerable {
		t.Error("Player should be invulnerable after entering Konami code")
	}
}

func TestKonamiCodeIncorrectSequence(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:          1,
		MaxLevel:       5,
		RNG:            rand.New(rand.NewSource(42)),
		KonamiSequence: make([]string, 0),
		Invulnerable:   false,
	}

	// Test incorrect sequence
	incorrectSequence := []string{"up", "down", "left", "right", "up", "down", "left", "right", "b", "a"}

	for _, key := range incorrectSequence {
		gs.CheckKonamiCode(key)
	}

	// Should not be invulnerable with incorrect sequence
	if gs.Invulnerable {
		t.Error("Player should not be invulnerable with incorrect sequence")
	}
}

func TestInvulnerabilityPreventsAttacks(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:        1,
		MaxLevel:     5,
		RNG:          rand.New(rand.NewSource(42)),
		Invulnerable: true,
	}

	// Create a player with 10 HP
	gs.Player = NewPlayer(5, 5)
	initialHP := gs.Player.HP

	// Create an enemy adjacent to the player
	enemy := NewBug(6, 5)
	gs.Enemies = []*Entity{enemy}

	// Enemy attacks
	gs.enemyAttacks()

	// Player HP should not have changed
	if gs.Player.HP != initialHP {
		t.Errorf("Player took damage while invulnerable. HP: %d, expected: %d", gs.Player.HP, initialHP)
	}
}

func TestVulnerablePlayerTakesDamage(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:        1,
		MaxLevel:     5,
		RNG:          rand.New(rand.NewSource(42)),
		Invulnerable: false,
	}

	// Create a player with 10 HP
	gs.Player = NewPlayer(5, 5)
	initialHP := gs.Player.HP

	// Create an enemy adjacent to the player
	enemy := NewBug(6, 5)
	gs.Enemies = []*Entity{enemy}

	// Enemy attacks
	gs.enemyAttacks()

	// Player HP should have decreased
	if gs.Player.HP >= initialHP {
		t.Errorf("Player should have taken damage. HP: %d, initial: %d", gs.Player.HP, initialHP)
	}
}

func TestMoveCounter(t *testing.T) {
	// Create a minimal dungeon for testing
	dungeon := &Dungeon{
		Width:  10,
		Height: 10,
		Tiles:  make([][]Tile, 10),
	}
	for i := range dungeon.Tiles {
		dungeon.Tiles[i] = make([]Tile, 10)
		for j := range dungeon.Tiles[i] {
			dungeon.Tiles[i][j] = TileFloor
		}
	}

	gs := &GameState{
		Level:     1,
		MaxLevel:  5,
		RNG:       rand.New(rand.NewSource(42)),
		Dungeon:   dungeon,
		MoveCount: 0,
		Player:    NewPlayer(5, 5),
		Enemies:   []*Entity{},
		Potions:   []*Entity{},
		Visible:   make([][]bool, 10),
		Explored:  make([][]bool, 10),
	}

	// Initialize visibility arrays
	for i := range gs.Visible {
		gs.Visible[i] = make([]bool, 10)
		gs.Explored[i] = make([]bool, 10)
	}

	// Test that move counter starts at 0
	if gs.MoveCount != 0 {
		t.Errorf("MoveCount should start at 0, got %d", gs.MoveCount)
	}

	// Move player right
	gs.MovePlayer(1, 0)
	if gs.MoveCount != 1 {
		t.Errorf("MoveCount should be 1 after one move, got %d", gs.MoveCount)
	}

	// Move player down
	gs.MovePlayer(0, 1)
	if gs.MoveCount != 2 {
		t.Errorf("MoveCount should be 2 after two moves, got %d", gs.MoveCount)
	}

	// Try to move into a wall (shouldn't increment counter)
	// Player is now at (6, 6), set wall at (7, 6)
	gs.Dungeon.Tiles[6][7] = TileWall
	initialMoveCount := gs.MoveCount
	gs.MovePlayer(1, 0) // Try to move right into wall
	if gs.MoveCount != initialMoveCount {
		t.Errorf("MoveCount should not increment when blocked by wall, expected %d, got %d", initialMoveCount, gs.MoveCount)
	}
}

func TestUsernameInitialization(t *testing.T) {
	// Create a game state with username
	gs := &GameState{
		Username: "@testuser",
	}

	if gs.Username != "@testuser" {
		t.Errorf("Username should be set correctly, got %s", gs.Username)
	}

	// Test with empty username
	gs2 := &GameState{
		Username: "",
	}

	if gs2.Username != "" {
		t.Errorf("Username should be empty, got %s", gs2.Username)
	}
}
