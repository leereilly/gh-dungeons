package game

import (
	"fmt"
	"os"
	"time"

	"github.com/gdamore/tcell/v2"
)

type Game struct {
	screen    tcell.Screen
	state     *GameState
	mergeMode bool
}

// GameOption configures Game creation
type GameOption func(*gameOptions)

type gameOptions struct {
	mergeMode bool
}

// WithMergeMode enables merge conflict display mode
func WithMergeMode(enabled bool) GameOption {
	return func(o *gameOptions) {
		o.mergeMode = enabled
	}
}

func New(opts ...GameOption) (*Game, error) {
	// Apply options
	options := &gameOptions{}
	for _, opt := range opts {
		opt(options)
	}

	// Find code files in current directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	codeFiles, err := findCodeFiles(cwd, 60, 5)
	if err != nil {
		return nil, fmt.Errorf("scanning code files: %w", err)
	}

	// Find merge conflict location if in merge mode
	var mergeConflict *MergeConflictLocation
	if options.mergeMode {
		mergeConflict = findMergeConflict(cwd)
	}

	// Compute seed from code files
	seed := computeSeed(codeFiles)
	if len(codeFiles) == 0 {
		seed = 42 // Default seed if no code files found
	}

	screen, err := tcell.NewScreen()
	if err != nil {
		return nil, fmt.Errorf("creating screen: %w", err)
	}

	if err := screen.Init(); err != nil {
		return nil, fmt.Errorf("initializing screen: %w", err)
	}

	screen.SetStyle(tcell.StyleDefault.Foreground(tcell.ColorWhite))
	screen.Clear()

	width, height := screen.Size()
	state := NewGameState(codeFiles, seed, width, height)
	state.MergeConflict = mergeConflict

	return &Game{
		screen:    screen,
		state:     state,
		mergeMode: options.mergeMode,
	}, nil
}

func (g *Game) Close() {
	if g.screen != nil {
		g.screen.Fini()
	}
}

func (g *Game) Run() error {
	for {
		g.render()
		g.screen.Show()

		ev := g.screen.PollEvent()
		switch ev := ev.(type) {
		case *tcell.EventResize:
			g.screen.Sync()
			width, height := g.screen.Size()
			g.state.Resize(width, height)
		case *tcell.EventKey:
			if ev.Key() == tcell.KeyEscape || ev.Key() == tcell.KeyCtrlC {
				return nil
			}
			if ev.Rune() == 'q' || ev.Rune() == 'Q' {
				return nil
			}

			if g.state.GameOver || g.state.Victory {
				// Any key to exit on game over/victory
				if ev.Key() == tcell.KeyEnter || ev.Rune() == ' ' {
					return nil
				}
				continue
			}

			// Movement
			dx, dy := 0, 0
			konamiKey := ""
			switch ev.Key() {
			case tcell.KeyUp:
				dy = -1
				konamiKey = "up"
			case tcell.KeyDown:
				dy = 1
				konamiKey = "down"
			case tcell.KeyLeft:
				dx = -1
				konamiKey = "left"
			case tcell.KeyRight:
				dx = 1
				konamiKey = "right"
			default:
				switch ev.Rune() {
				case 'h', 'a':
					dx = -1
					if ev.Rune() == 'a' {
						konamiKey = "a"
					}
				case 'l', 'd':
					dx = 1
				case 'k', 'w':
					dy = -1
				case 'j', 's':
					dy = 1
				case 'y': // diagonal up-left
					dx, dy = -1, -1
				case 'u': // diagonal up-right
					dx, dy = 1, -1
				case 'b': // diagonal down-left
					dx, dy = -1, 1
					konamiKey = "b"
				case 'n': // diagonal down-right
					dx, dy = 1, 1
				}
			}

			// Check for Konami code
			if konamiKey != "" {
				g.state.CheckKonamiCode(konamiKey)
			}

			if dx != 0 || dy != 0 {
				g.state.MovePlayer(dx, dy)
			}
		}
	}
}

func (g *Game) render() {
	g.screen.Clear()

	width, height := g.screen.Size()
	dungeon := g.state.Dungeon

	// Calculate offsets to center the dungeon
	offsetX := (width - dungeon.Width) / 2
	offsetY := (height - dungeon.Height - 3) / 2 // -3 for UI bar and message
	if offsetX < 0 {
		offsetX = 0
	}
	if offsetY < 0 {
		offsetY = 0
	}

	// Styles - walls turn red (visible) or orange (fog) when merge conflict triggered
	var wallStyle, fogWallStyle tcell.Style
	if g.state.MergeConflictTriggered {
		wallStyle = tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)
		fogWallStyle = tcell.StyleDefault.Foreground(tcell.ColorOrange).Background(tcell.ColorBlack)
	} else {
		wallStyle = tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
		fogWallStyle = tcell.StyleDefault.Foreground(tcell.Color240).Background(tcell.ColorBlack)
	}
	uiStyle := tcell.StyleDefault.Foreground(tcell.ColorLightGreen).Background(tcell.ColorBlack)
	codeStyle := tcell.StyleDefault.Foreground(tcell.Color238).Background(tcell.ColorBlack)
	playerStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack).Bold(true)
	enemyStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack)
	potionStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	doorStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack).Bold(true)
	fogStyle := tcell.StyleDefault.Foreground(tcell.Color240).Background(tcell.ColorBlack)
	mergeAffectedStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)

	// Get code lines for background
	var codeLines []string
	if dungeon.CodeFile != nil && len(dungeon.CodeFile.Lines) > 0 {
		codeLines = dungeon.CodeFile.Lines
	}

	// Render dungeon
	for y := 0; y < min(dungeon.Height, height-2); y++ {
		for x := 0; x < min(dungeon.Width, width); x++ {
			tile := dungeon.Tiles[y][x]
			visible := g.state.Visible[y][x]
			explored := g.state.Explored[y][x]

			if !explored {
				g.screen.SetContent(offsetX+x, offsetY+y, ' ', nil, tcell.StyleDefault)
				continue
			}

			var ch rune
			var style tcell.Style

			switch tile {
			case TileWall:
				ch = '#'
				if visible {
					style = wallStyle
				} else {
					style = fogWallStyle
				}
			case TileFloor:
				// Show code character if available (2x density)
				if len(codeLines) > 0 {
					// Use both y and x/40 to show 2x more code lines
					lineIdx := (y*2 + x/40) % len(codeLines)
					line := codeLines[lineIdx]
					charIdx := x % 40
					if x >= 40 {
						charIdx = x - 40
					}
					if charIdx < len(line) {
						ch = rune(line[charIdx])
					} else {
						ch = '.'
					}
				} else {
					ch = '.'
				}
				if visible {
					style = codeStyle
				} else {
					style = fogStyle
				}
			case TileDoor:
				ch = '>'
				if visible {
					style = doorStyle
				} else {
					style = fogStyle
				}
			}

			// Override style for merge-affected tiles (show in red with conflict chars)
			if g.state.IsMergeAffected(x, y) && visible {
				style = mergeAffectedStyle
				// Change character to conflict markers, cycling with player movement
				conflictChars := []rune{'<', '>', '='}
				ch = conflictChars[(x+y+g.state.MergeAnimationStep)%len(conflictChars)]
			}

			g.screen.SetContent(offsetX+x, offsetY+y, ch, nil, style)
		}
	}

	// Render potions
	for _, potion := range g.state.Potions {
		if g.state.Visible[potion.Y][potion.X] {
			g.screen.SetContent(offsetX+potion.X, offsetY+potion.Y, potion.Symbol, nil, potionStyle)
		}
	}
	
	// Render merge conflict if it has been triggered (fire persists after leaving)
	if g.state.MergeConflictTriggered {
		g.renderMergeConflict(offsetX, offsetY)
	}

	// Render enemies
	for _, enemy := range g.state.Enemies {
		if enemy.IsAlive() && g.state.Visible[enemy.Y][enemy.X] {
			g.screen.SetContent(offsetX+enemy.X, offsetY+enemy.Y, enemy.Symbol, nil, enemyStyle)
		}
	}

	// Render player
	g.screen.SetContent(offsetX+g.state.Player.X, offsetY+g.state.Player.Y, g.state.Player.Symbol, nil, playerStyle)

	// Render merge conflict marker (red X at center of the most central room)
	if g.mergeMode {
		mergeStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)
		markerX, markerY := findCentralRoomCenter(dungeon)
		if markerX >= 0 && markerY >= 0 {
			g.screen.SetContent(offsetX+markerX, offsetY+markerY, 'X', nil, mergeStyle)
		}
	}

	// Render UI bar at bottom left of screen
	uiY := height - 2
	invulnStatus := ""
	if g.state.Invulnerable {
		invulnStatus = " | INVULNERABLE"
	}
	uiLine := fmt.Sprintf("HP: %d/%d | Level: %d/%d | Kills: %d%s | [q]uit",
		g.state.Player.HP, g.state.Player.MaxHP,
		g.state.Level, g.state.MaxLevel,
		g.state.EnemiesKilled,
		invulnStatus)

	for i, ch := range uiLine {
		if i < width {
			g.screen.SetContent(i, uiY, ch, nil, uiStyle)
		}
	}

	// Render message at bottom left of screen
	msgY := height - 1
	// Clear the message line first to avoid leftover characters
	for i := 0; i < width; i++ {
		g.screen.SetContent(i, msgY, ' ', nil, tcell.StyleDefault)
	}
	if g.state.Message != "" {
		msgStyle := uiStyle
		// Show warning message in red
		if g.state.Message == MergeConflictWarning {
			msgStyle = tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)
		}
		for i, ch := range g.state.Message {
			if i < width {
				g.screen.SetContent(i, msgY, ch, nil, msgStyle)
			}
		}
	}

	// Render merge conflict warning if player is within 2 chars of merge marker center
	// Only show warning if conflict hasn't been triggered yet (no affected tiles)
	if g.mergeMode && g.state.MergeMarkerX >= 0 && g.state.MergeMarkerY >= 0 && len(g.state.MergeAffectedTiles) == 0 {
		dx := g.state.Player.X - g.state.MergeMarkerX
		dy := g.state.Player.Y - g.state.MergeMarkerY
		if dx < 0 {
			dx = -dx
		}
		if dy < 0 {
			dy = -dy
		}
		if dx <= 2 && dy <= 2 {
			warningStyle := tcell.StyleDefault.Foreground(tcell.ColorRed).Background(tcell.ColorBlack).Bold(true)
			warningMsg := "WARNING: Merge conflict detected"
			msgY := height - 1
			for i, ch := range warningMsg {
				if i < width {
					g.screen.SetContent(i, msgY, ch, nil, warningStyle)
				}
			}
		}
	}

	// Game over / Victory screen
	if g.state.GameOver || g.state.Victory {
		g.renderEndScreen(width, height)
	}
}

func (g *Game) renderMergeConflict(offsetX, offsetY int) {
	// Colors for merge conflict: red, orange, yellow - rotate based on movement
	baseColors := []tcell.Color{
		tcell.ColorRed,
		tcell.ColorOrange,
		tcell.ColorYellow,
	}
	// Rotate colors based on ColorRotation
	rotation := g.state.ColorRotation % 3
	colors := make([]tcell.Color, 3)
	for i := 0; i < 3; i++ {
		colors[i] = baseColors[(i+rotation)%3]
	}
	
	centerX := g.state.MergeConflictX
	centerY := g.state.MergeConflictY
	
	// Define the patterns based on movement count (3 rows x 5 cols)
	var pattern []string
	movements := g.state.MergeConflictMovements
	
	if movements == 0 {
		// Initial pattern (when player first steps on trap)
		pattern = []string{
			"<<<<<",
			"=====",
			">>>>>",
		}
	} else if movements == 1 {
		// After 1st turn on trap
		pattern = []string{
			">>>>>",
			"<<<<<",
			"=====",
		}
	} else if movements == 2 {
		// After 2nd turn on trap
		pattern = []string{
			"=====",
			">>>>>",
			"<<<<<",
		}
	} else {
		// After 2+ turns, randomize between <, >, and =
		pattern = make([]string, 3)
		chars := []rune{'<', '>', '='}
		for row := 0; row < 3; row++ {
			rowStr := ""
			for col := 0; col < 5; col++ {
				charIdx := g.state.RNG.Intn(len(chars))
				rowStr += string(chars[charIdx])
			}
			pattern[row] = rowStr
		}
	}
	
	// Calculate the size of the pattern
	patternHeight := len(pattern)
	patternWidth := 5 // All patterns are 5 characters wide
	
	// Render centered on the merge conflict position
	startY := -(patternHeight / 2)
	startX := -(patternWidth / 2)
	
	for row := 0; row < patternHeight; row++ {
		for col := 0; col < patternWidth && col < len(pattern[row]); col++ {
			mcX := centerX + startX + col
			mcY := centerY + startY + row
			
			// Skip if out of bounds
			if mcX < 0 || mcX >= g.state.Dungeon.Width || mcY < 0 || mcY >= g.state.Dungeon.Height {
				continue
			}
			
			// Only show on walkable tiles (always show when player is on merge conflict)
			if !g.state.Dungeon.IsWalkable(mcX, mcY) {
				continue
			}
			
			ch := rune(pattern[row][col])
			if ch != ' ' {
				// Deterministic color based on position and rotation
				colorIdx := (mcX + mcY) % 3
				mcStyle := tcell.StyleDefault.Foreground(colors[colorIdx]).Background(tcell.ColorBlack)
				g.screen.SetContent(offsetX+mcX, offsetY+mcY, ch, nil, mcStyle)
			}
		}
	}
	
	// Render fire spread tiles
	spreadChars := []rune{'<', '>', '='}
	for i, tile := range g.state.MergeConflictSpread {
		mcX := tile[0]
		mcY := tile[1]
		
		// Skip if out of bounds
		if mcX < 0 || mcX >= g.state.Dungeon.Width || mcY < 0 || mcY >= g.state.Dungeon.Height {
			continue
		}
		
		// Only show on walkable tiles
		if !g.state.Dungeon.IsWalkable(mcX, mcY) {
			continue
		}
		
		// Pick character based on position
		ch := spreadChars[(mcX+mcY)%3]
		// Deterministic color based on position and rotation
		colorIdx := (mcX + mcY + i) % 3
		mcStyle := tcell.StyleDefault.Foreground(colors[colorIdx]).Background(tcell.ColorBlack)
		g.screen.SetContent(offsetX+mcX, offsetY+mcY, ch, nil, mcStyle)
	}
}

func (g *Game) renderEndScreen(width, height int) {
	centerStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)

	var lines []string
	if g.state.Victory {
		lines = []string{
			"╔══════════════════════════════════════╗",
			"║            o VICTORY! o              ║",
			"║                                      ║",
			"║   You've conquered all the dungeons! ║",
			"║                                      ║",
			fmt.Sprintf("║   Levels Cleared: %d                  ║", g.state.Level),
			fmt.Sprintf("║   Enemies Killed: %-3d                ║", g.state.EnemiesKilled),
			"║                                      ║",
			"║      Press ENTER or SPACE to exit    ║",
			"║ (none of that vi :q nonsense to die) ",
			"╚══════════════════════════════════════╝",
		}
	} else {
		// Get custom death message based on what killed the player
		deathMsg := g.getDeathMessage()
		lines = []string{
			"╔══════════════════════════════════════╗",
			"║            x GAME OVER x             ║",
			"║                                      ║",
			fmt.Sprintf("║   %-36s ║", deathMsg),
			"║                                      ║",
			fmt.Sprintf("║   Levels Cleared: %d                  ║", g.state.Level-1),
			fmt.Sprintf("║   Enemies Killed: %-3d                ║", g.state.EnemiesKilled),
			"║                                      ║",
			"║      Press ENTER or SPACE to exit    ║",
			"║ (none of that vi :q nonsense to die) ║",
			"╚══════════════════════════════════════╝",
		}
	}

	startY := (height - len(lines)) / 2
	startX := (width - stringWidth(lines[0])) / 2 // Use first line (top border) for consistent alignment
	for i, line := range lines {
		col := 0
		for _, ch := range line {
			g.screen.SetContent(startX+col, startY+i, ch, nil, centerStyle)
			col++
		}
	}
}

func stringWidth(s string) int {
	return len([]rune(s))
}

func (g *Game) getDeathMessage() string {
	switch g.state.KilledBy {
	case "bug":
		return "In GitHub Dungeons... bug squashes YOU"
	case "merge_conflict":
		dayName := time.Now().Weekday().String()
		return fmt.Sprintf("Death by merge conflict. Just a typical %s.", dayName)
	case "scope_creep":
		return "Foiled by scope creep again!"
	default:
		return "The bugs and scope creeps won..."
	}
}
