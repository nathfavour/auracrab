package skills

import (
	"context"
	"encoding/json"
)

// Skill defines a modular capability for Auracrab.
type Skill interface {
	Name() string
	Description() string
	Manifest() json.RawMessage
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

var registry = make(map[string]Skill)

func Register(s Skill) {
	registry[s.Name()] = s
}

func GetRegistry() map[string]Skill {
	return registry
}
