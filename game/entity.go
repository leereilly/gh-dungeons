package game

import "github.com/gdamore/tcell/v2"

type EntityType int

const (
	EntityPlayer EntityType = iota
	EntityMonster
	EntityPotion
	// Legacy types for backwards compatibility
	EntityBug
	EntityScopeCreep
)

type Entity struct {
	Type            EntityType
	X, Y            int
	HP              int
	MaxHP           int
	Damage          int
	Symbol          rune
	Name            string       // Monster name from YAML
	Color           tcell.Color  // Display color
	Speed           float64      // Movement speed multiplier (1.0 = normal, 0.5 = slow)
	Movement        MovementType // Movement pattern
	AttackRange     int          // 1 = melee, 2+ = ranged
	ExperienceValue int          // XP awarded on kill
	Abilities       []string     // Special abilities
	TurnAccumulator float64      // Tracks partial turns for slow monsters
}

func NewPlayer(x, y int) *Entity {
	return &Entity{
		Type:     EntityPlayer,
		X:        x,
		Y:        y,
		HP:       20,
		MaxHP:    20,
		Damage:   2,
		Symbol:   '@',
		Name:     "Player",
		Color:    tcell.ColorWhite,
		Speed:    1.0,
		Movement: MovementAny,
	}
}

// NewMonsterFromDef creates an entity from a YAML monster definition
func NewMonsterFromDef(def MonsterDef, x, y int) *Entity {
	token := 'm'
	if len(def.Token) > 0 {
		token = rune(def.Token[0])
	}
	return &Entity{
		Type:            EntityMonster,
		X:               x,
		Y:               y,
		HP:              def.HP,
		MaxHP:           def.HP,
		Damage:          def.Strength,
		Symbol:          token,
		Name:            def.Name,
		Color:           ColorFromString(def.Color),
		Speed:           def.Speed,
		Movement:        def.Movement,
		AttackRange:     def.AttackRange,
		ExperienceValue: def.ExperienceValue,
		Abilities:       def.Abilities,
		TurnAccumulator: 0,
	}
}

func NewBug(x, y int) *Entity {
	return &Entity{
		Type:     EntityBug,
		X:        x,
		Y:        y,
		HP:       1,
		MaxHP:    1,
		Damage:   1,
		Symbol:   'b',
		Name:     "Bug",
		Color:    tcell.ColorRed,
		Speed:    1.0,
		Movement: MovementAny,
	}
}

func NewScopeCreep(x, y int) *Entity {
	return &Entity{
		Type:     EntityScopeCreep,
		X:        x,
		Y:        y,
		HP:       3,
		MaxHP:    3,
		Damage:   2,
		Symbol:   's',
		Name:     "Scope Creep",
		Color:    tcell.ColorYellow,
		Speed:    1.0,
		Movement: MovementAny,
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
	return e.Type == EntityMonster || e.Type == EntityBug || e.Type == EntityScopeCreep
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
