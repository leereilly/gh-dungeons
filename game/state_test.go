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
