package game

import (
	"fmt"
	"os"
	"strings"

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

	screen.SetStyle(tcell.StyleDefault.Background(tcell.ColorBlack).Foreground(tcell.ColorWhite))
	screen.Clear()

	state := NewGameState(codeFiles, seed)

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

	// Styles
	wallStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	floorStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkGray).Background(tcell.ColorBlack)
	codeStyle := tcell.StyleDefault.Foreground(tcell.ColorDarkGray).Background(tcell.ColorBlack)
	playerStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack).Bold(true)
	enemyStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	potionStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack)
	doorStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack).Bold(true)
	fogStyle := tcell.StyleDefault.Foreground(tcell.Color240).Background(tcell.ColorBlack)

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
				g.screen.SetContent(x, y, ' ', nil, tcell.StyleDefault)
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
				// Show code character if available
				if len(codeLines) > 0 {
					lineIdx := y % len(codeLines)
					line := codeLines[lineIdx]
					if x < len(line) {
						ch = rune(line[x])
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

			g.screen.SetContent(x, y, ch, nil, style)
		}
	}

	// Render potions
	for _, potion := range g.state.Potions {
		if g.state.Visible[potion.Y][potion.X] {
			g.screen.SetContent(potion.X, potion.Y, potion.Symbol, nil, potionStyle)
		}
	}

	// Render enemies
	for _, enemy := range g.state.Enemies {
		if enemy.IsAlive() && g.state.Visible[enemy.Y][enemy.X] {
			g.screen.SetContent(enemy.X, enemy.Y, enemy.Symbol, nil, enemyStyle)
		}
	}

	// Render player
	g.screen.SetContent(g.state.Player.X, g.state.Player.Y, g.state.Player.Symbol, nil, playerStyle)

	// Render UI bar
	uiY := min(dungeon.Height, height-2)
	uiLine := fmt.Sprintf("HP: %d/%d | Level: %d/%d | Kills: %d | [q]uit",
		g.state.Player.HP, g.state.Player.MaxHP,
		g.state.Level, g.state.MaxLevel,
		g.state.EnemiesKilled)

	for i, ch := range uiLine {
		if i < width {
			g.screen.SetContent(i, uiY, ch, nil, floorStyle)
		}
	}

	// Render message
	if g.state.Message != "" {
		msgY := uiY + 1
		for i, ch := range g.state.Message {
			if i < width {
				g.screen.SetContent(i, msgY, ch, nil, floorStyle)
			}
		}
	}

	// Game over / Victory screen
	if g.state.GameOver || g.state.Victory {
		g.renderEndScreen(width, height)
	}
}

func (g *Game) renderEndScreen(width, height int) {
	centerStyle := tcell.StyleDefault.Foreground(tcell.ColorWhite).Background(tcell.ColorBlack).Bold(true)

	var lines []string
	if g.state.Victory {
		lines = []string{
			"╔══════════════════════════════════════╗",
			"║            🎉 VICTORY! 🎉            ║",
			"║                                      ║",
			"║   You've conquered all the dungeons! ║",
			"║                                      ║",
			fmt.Sprintf("║   Levels Cleared: %d                  ║", g.state.Level),
			fmt.Sprintf("║   Enemies Killed: %-3d                ║", g.state.EnemiesKilled),
			"║                                      ║",
			"║      Press ENTER or SPACE to exit    ║",
			"╚══════════════════════════════════════╝",
		}
	} else {
		lines = []string{
			"╔══════════════════════════════════════╗",
			"║            💀 GAME OVER 💀           ║",
			"║                                      ║",
			"║   The bugs and scope creeps won...   ║",
			"║                                      ║",
			fmt.Sprintf("║   Levels Cleared: %d                  ║", g.state.Level-1),
			fmt.Sprintf("║   Enemies Killed: %-3d                ║", g.state.EnemiesKilled),
			"║                                      ║",
			"║      Press ENTER or SPACE to exit    ║",
			"╚══════════════════════════════════════╝",
		}
	}

	startY := (height - len(lines)) / 2
	for i, line := range lines {
		startX := (width - stringWidth(line)) / 2
		col := 0
		for _, ch := range line {
			g.screen.SetContent(startX+col, startY+i, ch, nil, centerStyle)
			col++
		}
	}
}

func stringWidth(s string) int {
	return len([]rune(strings.TrimSpace(s))) + 2
}
