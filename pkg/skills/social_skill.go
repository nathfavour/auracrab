package skills

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/nathfavour/auracrab/pkg/social"
)

type SocialSkill struct {
	manager *social.Manager
}

func NewSocialSkill() *SocialSkill {
	return &SocialSkill{
		manager: social.GetManager(),
	}
}

func (s *SocialSkill) Name() string {
	return "social"
}

func (s *SocialSkill) Description() string {
	return "Automate social media posts across platforms"
}

func (s *SocialSkill) Manifest() []byte {
	return []byte(`{
		"parameters": {
			"type": "object",
			"properties": {
				"action": { "type": "string", "enum": ["post"] },
				"content": { "type": "string" },
				"platforms": { "type": "array", "items": { "type": "string" } }
			},
			"required": ["action", "content"]
		}
	}`)
}

func (s *SocialSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Action    string   `json:"action"`
		Content   string   `json:"content"`
		Platforms []string `json:"platforms"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	switch params.Action {
	case "post":
		results := s.manager.PostToAll(ctx, params.Content, params.Platforms)
		return results, nil
	default:
		return "", fmt.Errorf("unknown social action: %s", params.Action)
	}
}