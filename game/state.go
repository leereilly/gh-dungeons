package game

import (
	"math/rand"
)

const VisionRadius = 7

type GameState struct {
	Player        *Entity
	Enemies       []*Entity
	Potions       []*Entity
	Dungeon       *Dungeon
	Level         int
	MaxLevel      int
	DoorX         int
	DoorY         int
	Visible       [][]bool
	Explored      [][]bool
	GameOver      bool
	Victory       bool
	EnemiesKilled int
	Message       string
	CodeFiles     []CodeFile
	RNG           *rand.Rand
	TermWidth     int
	TermHeight    int
}

func NewGameState(codeFiles []CodeFile, seed int64, termWidth, termHeight int) *GameState {
	rng := rand.New(rand.NewSource(seed))
	
	gs := &GameState{
		Level:      1,
		MaxLevel:   5,
		CodeFiles:  codeFiles,
		RNG:        rng,
		TermWidth:  termWidth,
		TermHeight: termHeight,
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
	
	// Spawn potions
	gs.Potions = nil
	numPotions := 2 + gs.RNG.Intn(3)
	for i := 0; i < numPotions; i++ {
		x, y := gs.randomFloorTile()
		gs.Potions = append(gs.Potions, NewPotion(x, y))
	}
	
	gs.updateVisibility()
	gs.Message = ""
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
			return x, y
		}
	}
	return gs.Dungeon.Width/2, gs.Dungeon.Height/2
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
					gs.Message = "You squashed a bug!"
				} else {
					gs.Message = "You eliminated a scope creep!"
				}
			} else {
				gs.Message = "You attack!"
			}
			// Enemy turn after player attacks
			gs.moveEnemies()
			gs.enemyAttacks()
			gs.updateVisibility()
			if !gs.Player.IsAlive() {
				gs.GameOver = true
				gs.Message = "You died!"
			}
			return
		}
	}
	
	gs.Player.X = newX
	gs.Player.Y = newY
	
	// Check for potion pickup
	for i, potion := range gs.Potions {
		if potion.X == newX && potion.Y == newY {
			gs.Player.Heal(3)
			gs.Potions = append(gs.Potions[:i], gs.Potions[i+1:]...)
			gs.Message = "You drink a health potion! (+3 HP)"
			break
		}
	}
	
	// Check for door
	if newX == gs.DoorX && newY == gs.DoorY {
		if gs.Level >= gs.MaxLevel {
			gs.Victory = true
			gs.Message = "You've escaped the dungeon! Victory!"
		} else {
			gs.Level++
			gs.generateLevel()
			gs.Message = "You descend deeper into the dungeon..."
		}
		return
	}
	
	gs.processTurn()
}

func (gs *GameState) processTurn() {
	// Auto-attack adjacent enemies
	gs.playerAutoAttack()
	
	// Enemy turn
	gs.moveEnemies()
	
	// Enemies attack player
	gs.enemyAttacks()
	
	// Update visibility
	gs.updateVisibility()
	
	// Check player death
	if !gs.Player.IsAlive() {
		gs.GameOver = true
		gs.Message = "You died!"
	}
}

func (gs *GameState) playerAutoAttack() {
	for _, enemy := range gs.Enemies {
		if enemy.IsAlive() && gs.Player.IsAdjacent(enemy) {
			enemy.TakeDamage(gs.Player.Damage)
			if !enemy.IsAlive() {
				gs.EnemiesKilled++
				if enemy.Type == EntityBug {
					gs.Message = "You squashed a bug!"
				} else {
					gs.Message = "You eliminated a scope creep!"
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
	for _, enemy := range gs.Enemies {
		if enemy.IsAlive() && gs.Player.IsAdjacent(enemy) {
			gs.Player.TakeDamage(enemy.Damage)
			if enemy.Type == EntityBug {
				gs.Message = "A bug bites you!"
			} else {
				gs.Message = "A scope creep attacks!"
			}
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
