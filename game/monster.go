package game

import (
	_ "embed"
	"math/rand"

	"github.com/gdamore/tcell/v2"
	"gopkg.in/yaml.v3"
)

//go:embed monsters.yaml
var monstersYAML []byte

// MovementType defines how a monster can move
type MovementType string

const (
	MovementStraight   MovementType = "straight"   // Only cardinal directions
	MovementDiagonal   MovementType = "diagonal"   // Only diagonal directions
	MovementAny        MovementType = "any"        // Any direction
	MovementStationary MovementType = "stationary" // Cannot move
	MovementHorizontal MovementType = "horizontal" // Only left or right
)

// MonsterDef represents a monster definition from YAML
type MonsterDef struct {
	Name            string       `yaml:"name"`
	Description     string       `yaml:"description"`
	Token           string       `yaml:"token"`
	Color           string       `yaml:"color"`
	HP              int          `yaml:"hp"`
	Strength        int          `yaml:"strength"`
	Speed           float64      `yaml:"speed"`
	Movement        MovementType `yaml:"movement"`
	AttackRange     int          `yaml:"attack_range"`
	ExperienceValue int          `yaml:"experience_value"`
	Abilities       []string     `yaml:"abilities"`
	Unique          bool         `yaml:"unique"` // If true, exactly one spawns per level
}

// MonsterConfig holds all monster definitions
type MonsterConfig struct {
	Monsters []MonsterDef `yaml:"monsters"`
}

// MonsterRegistry holds loaded monster definitions
type MonsterRegistry struct {
	monsters     map[string]MonsterDef
	names        []string // for random selection (non-unique only)
	uniqueNames  []string // unique monsters (one per level)
}

var globalRegistry *MonsterRegistry

// GetMonsterRegistry returns the global monster registry, loading if needed
func GetMonsterRegistry() *MonsterRegistry {
	if globalRegistry == nil {
		globalRegistry = loadMonsterRegistry()
	}
	return globalRegistry
}

func loadMonsterRegistry() *MonsterRegistry {
	var config MonsterConfig
	if err := yaml.Unmarshal(monstersYAML, &config); err != nil {
		// Fall back to empty registry on error
		return &MonsterRegistry{monsters: make(map[string]MonsterDef)}
	}

	registry := &MonsterRegistry{
		monsters: make(map[string]MonsterDef),
	}
	for _, m := range config.Monsters {
		registry.monsters[m.Name] = m
		if m.Unique {
			registry.uniqueNames = append(registry.uniqueNames, m.Name)
		} else {
			registry.names = append(registry.names, m.Name)
		}
	}
	return registry
}

// GetMonster returns a monster definition by name
func (r *MonsterRegistry) GetMonster(name string) (MonsterDef, bool) {
	m, ok := r.monsters[name]
	return m, ok
}

// GetRandomMonster returns a random monster definition
func (r *MonsterRegistry) GetRandomMonster(rng *rand.Rand) MonsterDef {
	if len(r.names) == 0 {
		// Fallback if no monsters loaded
		return MonsterDef{
			Name:     "Bug",
			Token:    "b",
			HP:       1,
			Strength: 1,
			Speed:    1.0,
			Movement: MovementAny,
		}
	}
	return r.monsters[r.names[rng.Intn(len(r.names))]]
}

// GetAllMonsters returns all monster definitions
func (r *MonsterRegistry) GetAllMonsters() []MonsterDef {
	result := make([]MonsterDef, 0, len(r.names)+len(r.uniqueNames))
	for _, name := range r.names {
		result = append(result, r.monsters[name])
	}
	for _, name := range r.uniqueNames {
		result = append(result, r.monsters[name])
	}
	return result
}

// GetUniqueMonsters returns all monster definitions marked as unique (one per level)
func (r *MonsterRegistry) GetUniqueMonsters() []MonsterDef {
	result := make([]MonsterDef, 0, len(r.uniqueNames))
	for _, name := range r.uniqueNames {
		result = append(result, r.monsters[name])
	}
	return result
}

// ColorFromString converts a color name to tcell.Color
func ColorFromString(name string) tcell.Color {
	switch name {
	case "red":
		return tcell.ColorRed
	case "green":
		return tcell.ColorGreen
	case "blue":
		return tcell.ColorBlue
	case "yellow":
		return tcell.ColorYellow
	case "cyan":
		return tcell.ColorAqua
	case "magenta":
		return tcell.ColorFuchsia
	case "white":
		return tcell.ColorWhite
	case "orange":
		return tcell.ColorOrange
	case "purple":
		return tcell.ColorPurple
	case "gray", "grey":
		return tcell.ColorGray
	default:
		return tcell.ColorWhite
	}
}
