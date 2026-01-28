package game

import (
	"math/rand"
	"testing"

	"github.com/gdamore/tcell/v2"
)

func TestPlayerInitialHP(t *testing.T) {
	// Test that player starts with 20 HP
	player := NewPlayer(5, 5)
	
	if player.HP != 20 {
		t.Errorf("Player should start with 20 HP, got %d", player.HP)
	}
	
	if player.MaxHP != 20 {
		t.Errorf("Player MaxHP should be 20, got %d", player.MaxHP)
	}
}

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

	// Try to attack an enemy (shouldn't increment counter)
	enemy := NewBug(7, 6) // Place enemy at position we're trying to move to
	gs.Dungeon.Tiles[6][7] = TileFloor // Clear the wall
	gs.Enemies = []*Entity{enemy}
	initialMoveCount = gs.MoveCount
	gs.MovePlayer(1, 0) // Try to move right into enemy (attack)
	if gs.MoveCount != initialMoveCount {
		t.Errorf("MoveCount should not increment when attacking enemy, expected %d, got %d", initialMoveCount, gs.MoveCount)
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

func TestMergeConflictProximity(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:          1,
		MaxLevel:       5,
		RNG:            rand.New(rand.NewSource(42)),
		MergeConflictX: 10,
		MergeConflictY: 10,
	}

	// Create a player
	gs.Player = NewPlayer(8, 10)

	// Player is 2 spaces away, should detect proximity
	distance := gs.distanceToMergeConflict()
	if distance != 2 {
		t.Errorf("Expected distance 2, got %d", distance)
	}

	// Move player to 1 space away
	gs.Player.X = 9
	distance = gs.distanceToMergeConflict()
	if distance != 1 {
		t.Errorf("Expected distance 1, got %d", distance)
	}

	// Move player on top of merge conflict
	gs.Player.X = 10
	distance = gs.distanceToMergeConflict()
	if distance != 0 {
		t.Errorf("Expected distance 0, got %d", distance)
	}
}

func TestMergeConflictDamage(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:          1,
		MaxLevel:       5,
		RNG:            rand.New(rand.NewSource(42)),
		Invulnerable:   false,
		MergeConflictX: 10,
		MergeConflictY: 10,
	}

	// Create a player with 10 HP
	gs.Player = NewPlayer(10, 10)
	initialHP := gs.Player.HP

	// Check merge conflict when player is on it
	gs.checkMergeConflict()

	// Player should have taken 1 damage
	if gs.Player.HP != initialHP-1 {
		t.Errorf("Player should have taken 1 damage. HP: %d, expected: %d", gs.Player.HP, initialHP-1)
	}

	// OnMergeConflict flag should be set
	if !gs.OnMergeConflict {
		t.Error("OnMergeConflict flag should be true")
	}

	// Verify damage message format
	expectedMsg := "- 1 HP damage"
	if gs.Message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, gs.Message)
	}

	// Verify MessageStyle is set (not default/empty)
	if gs.MessageStyle == (tcell.Style{}) {
		t.Error("MessageStyle should be set after merge conflict damage")
	}
}

func TestMergeConflictNoDamageWhenNotOnTrap(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:          1,
		MaxLevel:       5,
		RNG:            rand.New(rand.NewSource(42)),
		Invulnerable:   false,
		MergeConflictX: 10,
		MergeConflictY: 10,
	}

	// Create a player away from the trap
	gs.Player = NewPlayer(5, 5)
	initialHP := gs.Player.HP

	// Check merge conflict when player is not on it
	gs.checkMergeConflict()

	// Player should not have taken damage
	if gs.Player.HP != initialHP {
		t.Errorf("Player should not have taken damage. HP: %d, expected: %d", gs.Player.HP, initialHP)
	}

	// OnMergeConflict flag should be false
	if gs.OnMergeConflict {
		t.Error("OnMergeConflict flag should be false")
	}
}

func TestMergeConflictInvulnerability(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:          1,
		MaxLevel:       5,
		RNG:            rand.New(rand.NewSource(42)),
		Invulnerable:   true,
		MergeConflictX: 10,
		MergeConflictY: 10,
	}

	// Create a player on the merge conflict
	gs.Player = NewPlayer(10, 10)
	initialHP := gs.Player.HP

	// Check merge conflict when player is invulnerable
	gs.checkMergeConflict()

	// Player should not have taken damage due to invulnerability
	if gs.Player.HP != initialHP {
		t.Errorf("Invulnerable player should not take damage. HP: %d, expected: %d", gs.Player.HP, initialHP)
	}

	// OnMergeConflict flag should still be set
	if !gs.OnMergeConflict {
		t.Error("OnMergeConflict flag should be true even when invulnerable")
	}
}

func TestMergeConflictIntegration(t *testing.T) {
	// Create a full dungeon with merge conflict
	codeFiles := []CodeFile{
		{
			Path:  "test.go",
			Lines: []string{"package main", "func main() {", "}"},
		},
	}
	
	gs := NewGameState(codeFiles, 12345, 80, 40)
	
	// Verify merge conflict was placed
	if gs.MergeConflictX == 0 && gs.MergeConflictY == 0 {
		// This is unlikely but possible, skip if at origin
		t.Skip("Merge conflict placed at origin")
	}
	
	// Verify it's on a walkable tile
	if !gs.Dungeon.IsWalkable(gs.MergeConflictX, gs.MergeConflictY) {
		t.Error("Merge conflict should be on a walkable tile")
	}
	
	// Verify it's not on the player
	if gs.Player.X == gs.MergeConflictX && gs.Player.Y == gs.MergeConflictY {
		t.Error("Merge conflict should not spawn on player")
	}
	
	// Verify it's not on the door
	if gs.DoorX == gs.MergeConflictX && gs.DoorY == gs.MergeConflictY {
		t.Error("Merge conflict should not spawn on door")
	}
	
	// Move player to merge conflict (if possible)
	initialHP := gs.Player.HP
	gs.Player.X = gs.MergeConflictX
	gs.Player.Y = gs.MergeConflictY
	
	// Trigger damage check
	gs.checkMergeConflict()
	
	// Verify damage was taken
	if gs.Player.HP != initialHP-1 {
		t.Errorf("Player should take 1 damage on merge conflict. HP: %d, expected: %d", gs.Player.HP, initialHP-1)
	}
	
	// Verify flag is set
	if !gs.OnMergeConflict {
		t.Error("OnMergeConflict flag should be true when on trap")
	}
	
	// Move player away
	gs.Player.X = gs.MergeConflictX + 5
	gs.Player.Y = gs.MergeConflictY + 5
	gs.checkMergeConflict()
	
	// Verify flag is cleared
	if gs.OnMergeConflict {
		t.Error("OnMergeConflict flag should be false when away from trap")
	}
}

func TestEnemyDamageMessage(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:        1,
		MaxLevel:     5,
		RNG:          rand.New(rand.NewSource(42)),
		Invulnerable: false,
	}

	// Create a player with 10 HP
	gs.Player = NewPlayer(5, 5)

	// Test bug attack (1 damage)
	enemy := NewBug(6, 5)
	gs.Enemies = []*Entity{enemy}
	gs.enemyAttacks()

	expectedMsg := "A bug attacked - 1 HP damage"
	if gs.Message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, gs.Message)
	}

	// Verify MessageStyle is set (not default/empty)
	if gs.MessageStyle == (tcell.Style{}) {
		t.Error("MessageStyle should be set after enemy attack")
	}

	// Test scope creep attack (2 damage)
	gs2 := &GameState{
		Level:        1,
		MaxLevel:     5,
		RNG:          rand.New(rand.NewSource(42)),
		Invulnerable: false,
	}
	gs2.Player = NewPlayer(5, 5)
	scopeCreep := NewScopeCreep(6, 5)
	gs2.Enemies = []*Entity{scopeCreep}
	gs2.enemyAttacks()

	expectedMsg2 := "A scope creep attacked - 2 HP damage"
	if gs2.Message != expectedMsg2 {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg2, gs2.Message)
	}

	// Verify MessageStyle is set for scope creep too
	if gs2.MessageStyle == (tcell.Style{}) {
		t.Error("MessageStyle should be set after scope creep attack")
	}
}

func TestMessageStyleClearing(t *testing.T) {
	// Create a game state
	gs := &GameState{
		Level:        1,
		MaxLevel:     5,
		RNG:          rand.New(rand.NewSource(42)),
		Invulnerable: false,
	}

	// Create a player and enemy
	gs.Player = NewPlayer(5, 5)
	enemy := NewBug(6, 5)
	gs.Enemies = []*Entity{enemy}

	// Enemy attacks, setting red damage message
	gs.enemyAttacks()

	expectedMsg := "A bug attacked - 1 HP damage"
	if gs.Message != expectedMsg {
		t.Errorf("Expected message '%s', got '%s'", expectedMsg, gs.Message)
	}

	// Verify MessageStyle is set (not default/empty)
	if gs.MessageStyle == (tcell.Style{}) {
		t.Error("MessageStyle should be set after enemy attack")
	}

	// Use SetMessage to set a normal message
	gs.SetMessage("You attack!")

	// Verify message is updated
	if gs.Message != "You attack!" {
		t.Errorf("Expected message 'You attack!', got '%s'", gs.Message)
	}

	// Verify MessageStyle is cleared (back to default/empty)
	if gs.MessageStyle != (tcell.Style{}) {
		t.Error("MessageStyle should be cleared after SetMessage")
	}
}

