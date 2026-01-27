package game

import (
	"fmt"
	"os"

	"github.com/gdamore/tcell/v2"
)

type Game struct {
	screen tcell.Screen
	state  *GameState
}

func New() (*Game, error) {
	// Find code files in current directory
	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}

	codeFiles, err := findCodeFiles(cwd, 60, 5)
	if err != nil {
		return nil, fmt.Errorf("scanning code files: %w", err)
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

	return &Game{
		screen: screen,
		state:  state,
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
			switch ev.Key() {
			case tcell.KeyUp:
				dy = -1
			case tcell.KeyDown:
				dy = 1
			case tcell.KeyLeft:
				dx = -1
			case tcell.KeyRight:
				dx = 1
			default:
				switch ev.Rune() {
				case 'h', 'a':
					dx = -1
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
				case 'n': // diagonal down-right
					dx, dy = 1, 1
				}
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

	// Styles
	wallStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	floorStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkGray)
	codeStyle := tcell.StyleDefault.Foreground(tcell.Color238)
	playerStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	enemyStyle := tcell.StyleDefault.Foreground(tcell.ColorRed)
	potionStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite)
	doorStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Bold(true)
	fogStyle := tcell.StyleDefault.Foreground(tcell.Color240)

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
					style = fogStyle
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

			g.screen.SetContent(offsetX+x, offsetY+y, ch, nil, style)
		}
	}

	// Render potions
	for _, potion := range g.state.Potions {
		if g.state.Visible[potion.Y][potion.X] {
			g.screen.SetContent(offsetX+potion.X, offsetY+potion.Y, potion.Symbol, nil, potionStyle)
		}
	}

	// Render enemies
	for _, enemy := range g.state.Enemies {
		if enemy.IsAlive() && g.state.Visible[enemy.Y][enemy.X] {
			g.screen.SetContent(offsetX+enemy.X, offsetY+enemy.Y, enemy.Symbol, nil, enemyStyle)
		}
	}

	// Render player
	g.screen.SetContent(offsetX+g.state.Player.X, offsetY+g.state.Player.Y, g.state.Player.Symbol, nil, playerStyle)

	// Render UI bar
	uiY := offsetY + dungeon.Height
	uiLine := fmt.Sprintf("HP: %d/%d | Level: %d/%d | Kills: %d | [q]uit",
		g.state.Player.HP, g.state.Player.MaxHP,
		g.state.Level, g.state.MaxLevel,
		g.state.EnemiesKilled)

	for i, ch := range uiLine {
		if offsetX+i < width {
			g.screen.SetContent(offsetX+i, uiY, ch, nil, floorStyle)
		}
	}

	// Render message
	if g.state.Message != "" {
		msgY := uiY + 1
		for i, ch := range g.state.Message {
			if offsetX+i < width {
				g.screen.SetContent(offsetX+i, msgY, ch, nil, floorStyle)
			}
		}
	}

	// Game over / Victory screen
	if g.state.GameOver || g.state.Victory {
		g.renderEndScreen(width, height)
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
		lines = []string{
			"╔══════════════════════════════════════╗",
			"║            x GAME OVER x             ║",
			"║                                      ║",
			"║   The bugs and scope creeps won...   ║",
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
