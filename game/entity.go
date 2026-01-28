package game

type EntityType int

const (
	EntityPlayer EntityType = iota
	EntityBug
	EntityScopeCreep
	EntityPotion
)

type Entity struct {
	Type   EntityType
	X, Y   int
	HP     int
	MaxHP  int
	Damage int
	Symbol rune
}

func NewPlayer(x, y int) *Entity {
	return &Entity{
		Type:   EntityPlayer,
		X:      x,
		Y:      y,
		HP:     20,
		MaxHP:  20,
		Damage: 2,
		Symbol: '@',
	}
}

func NewBug(x, y int) *Entity {
	return &Entity{
		Type:   EntityBug,
		X:      x,
		Y:      y,
		HP:     1,
		MaxHP:  1,
		Damage: 1,
		Symbol: 'b',
	}
}

func NewScopeCreep(x, y int) *Entity {
	return &Entity{
		Type:   EntityScopeCreep,
		X:      x,
		Y:      y,
		HP:     3,
		MaxHP:  3,
		Damage: 2,
		Symbol: 's',
	}
}

func NewPotion(x, y int) *Entity {
	return &Entity{
		Type:   EntityPotion,
		X:      x,
		Y:      y,
		Symbol: '+',
	}
}

func (e *Entity) IsAlive() bool {
	return e.HP > 0
}

func (e *Entity) TakeDamage(dmg int) {
	e.HP -= dmg
	if e.HP < 0 {
		e.HP = 0
	}
}

func (e *Entity) Heal(amount int) {
	e.HP += amount
	if e.HP > e.MaxHP {
		e.HP = e.MaxHP
	}
}

func (e *Entity) IsEnemy() bool {
	return e.Type == EntityBug || e.Type == EntityScopeCreep
}

func (e *Entity) DistanceTo(other *Entity) int {
	dx := e.X - other.X
	dy := e.Y - other.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	if dx > dy {
		return dx
	}
	return dy
}

func (e *Entity) IsAdjacent(other *Entity) bool {
	dx := e.X - other.X
	dy := e.Y - other.Y
	if dx < 0 {
		dx = -dx
	}
	if dy < 0 {
		dy = -dy
	}
	return dx <= 1 && dy <= 1 && (dx+dy > 0)
}
