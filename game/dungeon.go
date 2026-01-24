package game

import (
	"math/rand"
)

const (
	MinRoomSize = 6
	MaxRoomSize = 15
)

type Room struct {
	X, Y, W, H int
}

func (r Room) Center() (int, int) {
	return r.X + r.W/2, r.Y + r.H/2
}

func (r Room) Contains(x, y int) bool {
	return x >= r.X && x < r.X+r.W && y >= r.Y && y < r.Y+r.H
}

type BSPNode struct {
	X, Y, W, H int
	Left       *BSPNode
	Right      *BSPNode
	Room       *Room
}

func NewBSPNode(x, y, w, h int) *BSPNode {
	return &BSPNode{X: x, Y: y, W: w, H: h}
}

func (n *BSPNode) Split(rng *rand.Rand, depth int) {
	if depth <= 0 {
		return
	}

	// Decide split direction based on shape
	horizontal := rng.Float32() > 0.5
	if float32(n.W)/float32(n.H) >= 1.25 {
		horizontal = false
	} else if float32(n.H)/float32(n.W) >= 1.25 {
		horizontal = true
	}

	maxSize := n.H - MinRoomSize
	if !horizontal {
		maxSize = n.W - MinRoomSize
	}

	if maxSize <= MinRoomSize {
		return
	}

	split := rng.Intn(maxSize-MinRoomSize) + MinRoomSize

	if horizontal {
		n.Left = NewBSPNode(n.X, n.Y, n.W, split)
		n.Right = NewBSPNode(n.X, n.Y+split, n.W, n.H-split)
	} else {
		n.Left = NewBSPNode(n.X, n.Y, split, n.H)
		n.Right = NewBSPNode(n.X+split, n.Y, n.W-split, n.H)
	}

	n.Left.Split(rng, depth-1)
	n.Right.Split(rng, depth-1)
}

func (n *BSPNode) CreateRooms(rng *rand.Rand) {
	if n.Left != nil && n.Right != nil {
		n.Left.CreateRooms(rng)
		n.Right.CreateRooms(rng)
		return
	}

	// Leaf node - create a room
	// Ensure we have enough space for a room
	if n.W < MinRoomSize+2 || n.H < MinRoomSize+2 {
		return
	}

	maxW := min(MaxRoomSize, n.W-2)
	maxH := min(MaxRoomSize, n.H-2)
	if maxW < MinRoomSize {
		maxW = MinRoomSize
	}
	if maxH < MinRoomSize {
		maxH = MinRoomSize
	}

	roomW := MinRoomSize
	if maxW > MinRoomSize {
		roomW = rng.Intn(maxW-MinRoomSize+1) + MinRoomSize
	}
	roomH := MinRoomSize
	if maxH > MinRoomSize {
		roomH = rng.Intn(maxH-MinRoomSize+1) + MinRoomSize
	}

	roomX := n.X + 1
	if n.W-roomW-1 > 1 {
		roomX = n.X + rng.Intn(n.W-roomW-1) + 1
	}
	roomY := n.Y + 1
	if n.H-roomH-1 > 1 {
		roomY = n.Y + rng.Intn(n.H-roomH-1) + 1
	}

	n.Room = &Room{X: roomX, Y: roomY, W: roomW, H: roomH}
}

func (n *BSPNode) GetRooms() []*Room {
	if n.Room != nil {
		return []*Room{n.Room}
	}

	var rooms []*Room
	if n.Left != nil {
		rooms = append(rooms, n.Left.GetRooms()...)
	}
	if n.Right != nil {
		rooms = append(rooms, n.Right.GetRooms()...)
	}
	return rooms
}

func (n *BSPNode) GetRoom() *Room {
	if n.Room != nil {
		return n.Room
	}
	if n.Left != nil {
		if r := n.Left.GetRoom(); r != nil {
			return r
		}
	}
	if n.Right != nil {
		return n.Right.GetRoom()
	}
	return nil
}

type Dungeon struct {
	Width    int
	Height   int
	Tiles    [][]Tile
	Rooms    []*Room
	CodeFile *CodeFile
}

type Tile int

const (
	TileWall Tile = iota
	TileFloor
	TileDoor
)

func GenerateDungeon(width, height int, rng *rand.Rand, codeFile *CodeFile) *Dungeon {
	d := &Dungeon{
		Width:    width,
		Height:   height,
		Tiles:    make([][]Tile, height),
		CodeFile: codeFile,
	}

	for y := 0; y < height; y++ {
		d.Tiles[y] = make([]Tile, width)
		for x := 0; x < width; x++ {
			d.Tiles[y][x] = TileWall
		}
	}

	// BSP generation
	root := NewBSPNode(0, 0, width, height)
	root.Split(rng, 4)
	root.CreateRooms(rng)

	d.Rooms = root.GetRooms()

	// Carve rooms
	for _, room := range d.Rooms {
		for y := room.Y; y < room.Y+room.H; y++ {
			for x := room.X; x < room.X+room.W; x++ {
				if y >= 0 && y < height && x >= 0 && x < width {
					d.Tiles[y][x] = TileFloor
				}
			}
		}
	}

	// Connect rooms with corridors
	connectRooms(root, d, rng)

	return d
}

func connectRooms(node *BSPNode, d *Dungeon, rng *rand.Rand) {
	if node.Left == nil || node.Right == nil {
		return
	}

	connectRooms(node.Left, d, rng)
	connectRooms(node.Right, d, rng)

	leftRoom := node.Left.GetRoom()
	rightRoom := node.Right.GetRoom()

	if leftRoom == nil || rightRoom == nil {
		return
	}

	x1, y1 := leftRoom.Center()
	x2, y2 := rightRoom.Center()

	// L-shaped corridor
	if rng.Float32() > 0.5 {
		d.carveHorizontalCorridor(x1, x2, y1)
		d.carveVerticalCorridor(y1, y2, x2)
	} else {
		d.carveVerticalCorridor(y1, y2, x1)
		d.carveHorizontalCorridor(x1, x2, y2)
	}
}

func (d *Dungeon) carveHorizontalCorridor(x1, x2, y int) {
	if x1 > x2 {
		x1, x2 = x2, x1
	}
	for x := x1; x <= x2; x++ {
		if y >= 0 && y < d.Height && x >= 0 && x < d.Width {
			d.Tiles[y][x] = TileFloor
		}
	}
}

func (d *Dungeon) carveVerticalCorridor(y1, y2, x int) {
	if y1 > y2 {
		y1, y2 = y2, y1
	}
	for y := y1; y <= y2; y++ {
		if y >= 0 && y < d.Height && x >= 0 && x < d.Width {
			d.Tiles[y][x] = TileFloor
		}
	}
}

func (d *Dungeon) IsWalkable(x, y int) bool {
	if x < 0 || x >= d.Width || y < 0 || y >= d.Height {
		return false
	}
	return d.Tiles[y][x] != TileWall
}

func (d *Dungeon) PlaceDoor(rng *rand.Rand) (int, int) {
	// Place door in the last room
	if len(d.Rooms) == 0 {
		return d.Width / 2, d.Height / 2
	}
	room := d.Rooms[len(d.Rooms)-1]
	x := room.X + rng.Intn(room.W-2) + 1
	y := room.Y + rng.Intn(room.H-2) + 1
	d.Tiles[y][x] = TileDoor
	return x, y
}
