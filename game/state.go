package game

import (
	"fmt"
	"math/rand"

	"github.com/gdamore/tcell/v2"
)

const VisionRadius = 7
const MergeConflictWarning = "WARNING: MERGE CONFLICT DETECTED. TREAD CAREFULLY."

type GameState struct {
	Player                 *Entity
	Enemies                []*Entity
	Potions                []*Entity
	Dungeon                *Dungeon
	Level                  int
	MaxLevel               int
	DoorX                  int
	DoorY                  int
	Visible                [][]bool
	Explored               [][]bool
	GameOver               bool
	Victory                bool
	EnemiesKilled          int
	Message                string
	MessageStyle           tcell.Style           // Style for the message (e.g., red for damage)
	CodeFiles              []CodeFile
	RNG                    *rand.Rand
	TermWidth              int
	TermHeight             int
	KonamiSequence         []string
	Invulnerable           bool
	MoveCount              int
	Username               string
	MergeConflictX         int
	MergeConflictY         int
	OnMergeConflict        bool
	MergeConflictTriggered bool              // Track if merge conflict has ever been triggered (for persistent fire/wall effects)
	MergeConflictMovements int               // Track player movements on merge conflict
	KilledBy               string            // Track what killed the player for custom death messages
	MergeConflictSpread    [][2]int          // Additional fire spread tiles
	ColorRotation          int               // Track color rotation for merge conflict
	MergeConflict          *MergeConflictLocation
	MergeMarkerX           int
	MergeMarkerY           int
	MergeAffectedTiles     map[int]bool      // key: y*width + x
	MergeAnimationStep     int               // cycles merge conflict markers on each move
}

// SetMessage sets a message with default (green) style
func (gs *GameState) SetMessage(msg string) {
	gs.Message = msg
	gs.MessageStyle = tcell.Style{} // Clear custom style, use default
}

func NewGameState(codeFiles []CodeFile, seed int64, termWidth, termHeight int) *GameState {
	rng := rand.New(rand.NewSource(seed))

	gs := &GameState{
		Level:              1,
		MaxLevel:           5,
		CodeFiles:          codeFiles,
		RNG:                rng,
		TermWidth:          termWidth,
		TermHeight:         termHeight,
		KonamiSequence:     make([]string, 0),
		Invulnerable:       false,
		MoveCount:          0,
		Username:           getUsername(),
		MergeMarkerX:       -1,
		MergeMarkerY:       -1,
		MergeAffectedTiles: make(map[int]bool),
	}

	gs.generateLevel()
	return gs
}

func (gs *GameState) generateLevel() {
	// Reserve 3 lines for UI at bottom (status bar, message, buffer)
	width := gs.TermWidth
	height := gs.TermHeight - 3
	if width < 40 {
		width = 40
	}
	if height < 20 {
		height = 20
	}

	// Pick a code file for this level
	var codeFile *CodeFile
	if len(gs.CodeFiles) > 0 {
		codeFile = &gs.CodeFiles[(gs.Level-1)%len(gs.CodeFiles)]
	}

	gs.Dungeon = GenerateDungeon(width, height, gs.RNG, codeFile)

	// Initialize visibility arrays
	gs.Visible = make([][]bool, height)
	gs.Explored = make([][]bool, height)
	for y := 0; y < height; y++ {
		gs.Visible[y] = make([]bool, width)
		gs.Explored[y] = make([]bool, width)
	}

	// Place player in first room
	if len(gs.Dungeon.Rooms) > 0 {
		room := gs.Dungeon.Rooms[0]
		px, py := room.Center()
		if gs.Player == nil {
			gs.Player = NewPlayer(px, py)
		} else {
			gs.Player.X, gs.Player.Y = px, py
		}
	}

	// Place door
	gs.DoorX, gs.DoorY = gs.Dungeon.PlaceDoor(gs.RNG)

	
	// Place merge conflict trap (one per level) - place before enemies/potions
	gs.MergeConflictX, gs.MergeConflictY = gs.randomFloorTile()
	gs.OnMergeConflict = false
	
	// Spawn enemies
	gs.Enemies = nil
	numEnemies := 3 + gs.Level*2
	for i := 0; i < numEnemies; i++ {
		x, y := gs.randomFloorTile()
		if gs.RNG.Float32() > 0.4 {
			gs.Enemies = append(gs.Enemies, NewBug(x, y))
		} else {
			gs.Enemies = append(gs.Enemies, NewScopeCreep(x, y))
		}
	}

	// Spawn potions (scales with level)
	gs.Potions = nil
	numPotions := 2 + gs.Level + gs.RNG.Intn(2)
	for i := 0; i < numPotions; i++ {
		x, y := gs.randomFloorTile()
		gs.Potions = append(gs.Potions, NewPotion(x, y))
	}

	
	// Set merge conflict marker position (center of most central room)
	gs.MergeMarkerX, gs.MergeMarkerY = findCentralRoomCenter(gs.Dungeon)
	gs.MergeAffectedTiles = make(map[int]bool)
	
	gs.updateVisibility()
	gs.SetMessage("")
}

func (gs *GameState) randomFloorTile() (int, int) {
	for attempts := 0; attempts < 100; attempts++ {
		if len(gs.Dungeon.Rooms) == 0 {
			break
		}
		room := gs.Dungeon.Rooms[gs.RNG.Intn(len(gs.Dungeon.Rooms))]
		x := room.X + gs.RNG.Intn(room.W)
		y := room.Y + gs.RNG.Intn(room.H)

		if gs.Dungeon.IsWalkable(x, y) {
			// Check not on player or door
			if gs.Player != nil && x == gs.Player.X && y == gs.Player.Y {
				continue
			}
			if x == gs.DoorX && y == gs.DoorY {
				continue
			}
			// Check not on merge conflict trap (if already placed)
			if x == gs.MergeConflictX && y == gs.MergeConflictY {
				continue
			}
			return x, y
		}
	}
	return gs.Dungeon.Width / 2, gs.Dungeon.Height / 2
}

func (gs *GameState) MovePlayer(dx, dy int) {
	if gs.GameOver || gs.Victory {
		return
	}

	newX := gs.Player.X + dx
	newY := gs.Player.Y + dy

	// Check bounds and walkability
	if !gs.Dungeon.IsWalkable(newX, newY) {
		return
	}

	// Check for enemy at target position - bump to attack!
	for _, enemy := range gs.Enemies {
		if enemy.IsAlive() && enemy.X == newX && enemy.Y == newY {
			// Attack the enemy we bumped into
			enemy.TakeDamage(gs.Player.Damage)
			if !enemy.IsAlive() {
				gs.EnemiesKilled++
				if enemy.Type == EntityBug {
					gs.SetMessage("You squashed a bug!")
				} else {
					gs.SetMessage("You eliminated a scope creep!")
				}
			} else {
				gs.SetMessage("You attack!")
			}
			// Enemy turn after player attacks
			gs.moveEnemies()
			gs.enemyAttacks()
			gs.updateVisibility()
			if !gs.Player.IsAlive() {
				gs.GameOver = true
				gs.SetMessage("You died!")
			}
			return
		}
	}

	gs.Player.X = newX
	gs.Player.Y = newY
	gs.MoveCount++

	
	// Cycle merge conflict animation if active
	if len(gs.MergeAffectedTiles) > 0 {
		gs.MergeAnimationStep++
	}
	
	// Check for potion pickup
	for i, potion := range gs.Potions {
		if potion.X == newX && potion.Y == newY {
			gs.Player.Heal(3)
			gs.Potions = append(gs.Potions[:i], gs.Potions[i+1:]...)
			gs.SetMessage("You drink a health potion! (+3 HP)")
			break
		}
	}

	
	// Check for merge conflict marker
	if newX == gs.MergeMarkerX && newY == gs.MergeMarkerY {
		gs.triggerMergeConflict()
	}
	
	// Check for door
	if newX == gs.DoorX && newY == gs.DoorY {
		if gs.Level >= gs.MaxLevel {
			gs.Victory = true
			gs.SetMessage("You've escaped the dungeon! Victory!")
		} else {
			gs.Level++
			gs.generateLevel()
			gs.SetMessage("You descend deeper into the dungeon...")
		}
		return
	}

	gs.processTurn()
}

func (gs *GameState) distanceToMergeConflict() int {
	dx := gs.Player.X - gs.MergeConflictX
	dy := gs.Player.Y - gs.MergeConflictY
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	// Use Chebyshev distance (max of abs differences)
	if dx > dy {
		return dx
	}
	return dy
}

// isInMergeConflictArea checks if a position is within the merge conflict's fire area
func (gs *GameState) isInMergeConflictArea(x, y int) bool {
	// Check core 5x3 area
	dx := x - gs.MergeConflictX
	dy := y - gs.MergeConflictY
	if dx >= -2 && dx <= 2 && dy >= -1 && dy <= 1 {
		return true
	}
	// Check spread tiles
	for _, tile := range gs.MergeConflictSpread {
		if x == tile[0] && y == tile[1] {
			return true
		}
	}
	return false
}

// isPlayerInMergeConflictArea checks if the player is within the merge conflict's visual area
func (gs *GameState) isPlayerInMergeConflictArea() bool {
	return gs.isInMergeConflictArea(gs.Player.X, gs.Player.Y)
}

func (gs *GameState) checkMergeConflict() {
	// Check if player is on merge conflict trap center
	onTrapCenter := gs.Player.X == gs.MergeConflictX && gs.Player.Y == gs.MergeConflictY
	
	if onTrapCenter {
		if !gs.OnMergeConflict {
			// Player just stepped on the trap center
			gs.OnMergeConflict = true
			gs.MergeConflictTriggered = true
			gs.MergeConflictMovements = 0
			gs.generateMergeConflictSpread()
		}
		// Rotate colors on each movement
		gs.ColorRotation++
		// Deal 1 damage per turn while on the trap center
		if !gs.Invulnerable {
			gs.Player.TakeDamage(1)
			// Format merge conflict damage as "- X HP damage" in red
			gs.Message = "- 1 HP damage"
			gs.MessageStyle = tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)
			if !gs.Player.IsAlive() {
				gs.KilledBy = "merge_conflict"
			}
		} else {
			gs.SetMessage("The merge conflict burns around you, but your invulnerability protects you!")
		}
	} else if gs.MergeConflictTriggered {
		// Player moved off the center - keep animating fire even outside the area
		gs.ColorRotation++
		if gs.OnMergeConflict && !gs.isPlayerInMergeConflictArea() {
			// Player fully escaped the merge conflict area
			gs.OnMergeConflict = false
		}
	} else {
		gs.OnMergeConflict = false
	}
}

func (gs *GameState) processTurn() {
	// Auto-attack adjacent enemies
	gs.playerAutoAttack()

	
	// Check merge conflict proximity and damage
	gs.checkMergeConflict()
	
	// Enemy turn
	gs.moveEnemies()

	// Enemies attack player
	gs.enemyAttacks()

	// Update visibility
	gs.updateVisibility()

	
	// Increment merge conflict movement counter if on trap (at end of turn)
	if gs.OnMergeConflict {
		gs.MergeConflictMovements++
	}
	
	// Check player death
	if !gs.Player.IsAlive() {
		gs.GameOver = true
		gs.SetMessage("You died!")
		return
	}
	
	// Show warning message if player is near merge conflict and no other message
	distance := gs.distanceToMergeConflict()
	if distance <= 2 && distance > 0 && gs.Message == "" {
		gs.SetMessage(MergeConflictWarning)
	}
}

func (gs *GameState) playerAutoAttack() {
	for _, enemy := range gs.Enemies {
		if enemy.IsAlive() && gs.Player.IsAdjacent(enemy) {
			enemy.TakeDamage(gs.Player.Damage)
			if !enemy.IsAlive() {
				gs.EnemiesKilled++
				if enemy.Type == EntityBug {
					gs.SetMessage("You squashed a bug!")
				} else {
					gs.SetMessage("You eliminated a scope creep!")
				}
			}
		}
	}
}

func (gs *GameState) moveEnemies() {
	for _, enemy := range gs.Enemies {
		if !enemy.IsAlive() {
			continue
		}

		// Only move if player is visible (in line of sight)
		if !gs.hasLineOfSight(enemy.X, enemy.Y, gs.Player.X, gs.Player.Y) {
			continue
		}

		// Simple chase AI - move toward player
		dx, dy := 0, 0
		if enemy.X < gs.Player.X {
			dx = 1
		} else if enemy.X > gs.Player.X {
			dx = -1
		}
		if enemy.Y < gs.Player.Y {
			dy = 1
		} else if enemy.Y > gs.Player.Y {
			dy = -1
		}

		// Try to move (prefer diagonal, then cardinal)
		newX, newY := enemy.X+dx, enemy.Y+dy
		if gs.canEnemyMoveTo(newX, newY, enemy) {
			enemy.X, enemy.Y = newX, newY
		} else if dx != 0 && gs.canEnemyMoveTo(enemy.X+dx, enemy.Y, enemy) {
			enemy.X += dx
		} else if dy != 0 && gs.canEnemyMoveTo(enemy.X, enemy.Y+dy, enemy) {
			enemy.Y += dy
		}

		// Check if enemy is in merge conflict fire area and apply damage
		if gs.MergeConflictTriggered && gs.isInMergeConflictArea(enemy.X, enemy.Y) {
			enemy.TakeDamage(1)
		}
	}
}

func (gs *GameState) canEnemyMoveTo(x, y int, self *Entity) bool {
	if !gs.Dungeon.IsWalkable(x, y) {
		return false
	}
	if x == gs.Player.X && y == gs.Player.Y {
		return false
	}
	for _, e := range gs.Enemies {
		if e != self && e.IsAlive() && e.X == x && e.Y == y {
			return false
		}
	}
	return true
}

func (gs *GameState) enemyAttacks() {
	if gs.Invulnerable {
		// Player is invulnerable, enemies do no damage
		return
	}

	for _, enemy := range gs.Enemies {
		if enemy.IsAlive() && gs.Player.IsAdjacent(enemy) {
			gs.Player.TakeDamage(enemy.Damage)
			// Format damage message with monster type and damage in red
			if enemy.Type == EntityBug {
				gs.Message = fmt.Sprintf("A bug attacked - %d HP damage", enemy.Damage)
				if !gs.Player.IsAlive() {
					gs.KilledBy = "bug"
				}
			} else {
				gs.Message = fmt.Sprintf("A scope creep attacked - %d HP damage", enemy.Damage)
				if !gs.Player.IsAlive() {
					gs.KilledBy = "scope_creep"
				}
			}
			gs.MessageStyle = tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)
		}
	}
}

func (gs *GameState) hasLineOfSight(x1, y1, x2, y2 int) bool {
	dx := x2 - x1
	dy := y2 - y1

	steps := abs(dx)
	if abs(dy) > steps {
		steps = abs(dy)
	}

	if steps == 0 {
		return true
	}

	xInc := float64(dx) / float64(steps)
	yInc := float64(dy) / float64(steps)

	x := float64(x1)
	y := float64(y1)

	for i := 0; i < steps; i++ {
		x += xInc
		y += yInc
		ix, iy := int(x+0.5), int(y+0.5)
		if !gs.Dungeon.IsWalkable(ix, iy) {
			return false
		}
	}

	return true
}

func (gs *GameState) updateVisibility() {
	// Clear visible
	for y := range gs.Visible {
		for x := range gs.Visible[y] {
			gs.Visible[y][x] = false
		}
	}

	// Cast rays for fog of war
	px, py := gs.Player.X, gs.Player.Y
	for angle := 0; angle < 360; angle += 2 {
		gs.castRay(px, py, angle)
	}
}

func (gs *GameState) castRay(startX, startY, angle int) {
	// Convert angle to radians
	rad := float64(angle) * 3.14159265 / 180.0
	dx := cos(rad)
	dy := sin(rad)

	x := float64(startX)
	y := float64(startY)

	for dist := 0; dist <= VisionRadius; dist++ {
		ix, iy := int(x+0.5), int(y+0.5)

		if ix < 0 || ix >= gs.Dungeon.Width || iy < 0 || iy >= gs.Dungeon.Height {
			break
		}

		gs.Visible[iy][ix] = true
		gs.Explored[iy][ix] = true

		if gs.Dungeon.Tiles[iy][ix] == TileWall {
			break
		}

		x += dx
		y += dy
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func cos(rad float64) float64 {
	// Taylor series approximation
	rad = mod2pi(rad)
	x2 := rad * rad
	return 1 - x2/2 + x2*x2/24 - x2*x2*x2/720
}

func sin(rad float64) float64 {
	rad = mod2pi(rad)
	x2 := rad * rad
	return rad - rad*x2/6 + rad*x2*x2/120
}

func mod2pi(x float64) float64 {
	twoPi := 6.28318530718
	for x > 3.14159265 {
		x -= twoPi
	}
	for x < -3.14159265 {
		x += twoPi
	}
	return x
}

func (gs *GameState) Resize(termWidth, termHeight int) {
	gs.TermWidth = termWidth
	gs.TermHeight = termHeight
}

func (gs *GameState) generateMergeConflictSpread() {
	// Skip if no dungeon (for tests)
	if gs.Dungeon == nil {
		return
	}
	
	// Get all tiles in the core 5x3 pattern
	coreTiles := make(map[[2]int]bool)
	centerX := gs.MergeConflictX
	centerY := gs.MergeConflictY
	
	for row := -1; row <= 1; row++ {
		for col := -2; col <= 2; col++ {
			coreTiles[[2]int{centerX + col, centerY + row}] = true
		}
	}
	
	// Find all adjacent tiles to the core pattern
	var adjacentTiles [][2]int
	directions := [][2]int{{-1, -1}, {0, -1}, {1, -1}, {-1, 0}, {1, 0}, {-1, 1}, {0, 1}, {1, 1}}
	
	for tile := range coreTiles {
		for _, dir := range directions {
			newX := tile[0] + dir[0]
			newY := tile[1] + dir[1]
			newTile := [2]int{newX, newY}
			
			// Skip if already in core or out of bounds
			if coreTiles[newTile] {
				continue
			}
			if newX < 0 || newX >= gs.Dungeon.Width || newY < 0 || newY >= gs.Dungeon.Height {
				continue
			}
			if !gs.Dungeon.IsWalkable(newX, newY) {
				continue
			}
			
			// Check if already added
			alreadyAdded := false
			for _, t := range adjacentTiles {
				if t == newTile {
					alreadyAdded = true
					break
				}
			}
			if !alreadyAdded {
				adjacentTiles = append(adjacentTiles, newTile)
			}
		}
	}
	
	// Shuffle and pick 7 random tiles
	gs.RNG.Shuffle(len(adjacentTiles), func(i, j int) {
		adjacentTiles[i], adjacentTiles[j] = adjacentTiles[j], adjacentTiles[i]
	})
	
	numSpread := 7
	if len(adjacentTiles) < numSpread {
		numSpread = len(adjacentTiles)
	}
	gs.MergeConflictSpread = adjacentTiles[:numSpread]
}

// CheckKonamiCode checks if the given key press completes the Konami code
// Konami code: up, up, down, down, left, right, left, right, B, A
func (gs *GameState) CheckKonamiCode(key string) {
	konamiCode := []string{"up", "up", "down", "down", "left", "right", "left", "right", "b", "a"}

	gs.KonamiSequence = append(gs.KonamiSequence, key)

	// Keep only the last 10 keys
	if len(gs.KonamiSequence) > 10 {
		gs.KonamiSequence = gs.KonamiSequence[len(gs.KonamiSequence)-10:]
	}

	// Check if the sequence matches the Konami code
	if len(gs.KonamiSequence) == 10 {
		match := true
		for i := 0; i < 10; i++ {
			if gs.KonamiSequence[i] != konamiCode[i] {
				match = false
				break
			}
		}
		if match && !gs.Invulnerable {
			gs.Invulnerable = true
			gs.SetMessage("KONAMI CODE ACTIVATED! You are now invulnerable!")
		}
	}
}

// triggerMergeConflict handles the player stepping on a merge conflict marker
func (gs *GameState) triggerMergeConflict() {
	// Deal damage to player (unless invulnerable)
	if !gs.Invulnerable {
		gs.Player.TakeDamage(2)
	}
	gs.SetMessage("MERGE CONFLICT! The code tears apart around you!")
	
	// Mark surrounding tiles as affected (3x3 area around the marker)
	for dy := -1; dy <= 1; dy++ {
		for dx := -1; dx <= 1; dx++ {
			ax := gs.MergeMarkerX + dx
			ay := gs.MergeMarkerY + dy
			if ax >= 0 && ax < gs.Dungeon.Width && ay >= 0 && ay < gs.Dungeon.Height {
				key := ay*gs.Dungeon.Width + ax
				gs.MergeAffectedTiles[key] = true
			}
		}
	}
	
	// Check for player death
	if !gs.Player.IsAlive() {
		gs.GameOver = true
		gs.SetMessage("You died in a merge conflict!")
	}
}

// IsMergeAffected checks if a tile is affected by a merge conflict
func (gs *GameState) IsMergeAffected(x, y int) bool {
	key := y*gs.Dungeon.Width + x
	return gs.MergeAffectedTiles[key]
}
