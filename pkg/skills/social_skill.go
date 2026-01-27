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
	return "Automate social media posts across X, LinkedIn, Facebook, Instagram, and Threads"
}

func (s *SocialSkill) Manifest() []byte {
	return []byte(`{
		"parameters": {
			"type": "object",
			"properties": {
				"action": {
					"type": "string",
					"enum": ["post"]
				},
				"platforms": {
					"type": "array",
					"items": { "type": "string" }
				},
				"content": {
					"type": "string"
				}
			},
			"required": ["action", "platforms", "content"]
		}
	}`)
}

func (s *SocialSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Action    string   `json:"action"`
		Platforms []string `json:"platforms"`
		Content   string   `json:"content"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	switch params.Action {
	case "post":
		if len(params.Platforms) == 0 {
			return "", fmt.Errorf("missing platforms")
		}
		if params.Content == "" {
			return "", fmt.Errorf("missing content")
		}

		results := s.manager.PostToAll(ctx, params.Content, params.Platforms)
		return results, nil
	default:
		return "", fmt.Errorf("unknown social action: %s", params.Action)
	}
}
