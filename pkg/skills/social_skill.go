package skills

import (
	"context"
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

func (s *SocialSkill) Execute(ctx context.Context, action string, args map[string]interface{}) (string, error) {
	switch action {
	case "post":
		platforms, ok := args["platforms"].([]interface{})
		if !ok {
			return "", fmt.Errorf("missing platforms (list of strings)")
		}
		content, ok := args["content"].(string)
		if !ok {
			return "", fmt.Errorf("missing content")
		}

		var platformStrings []string
		for _, p := range platforms {
			if ps, ok := p.(string); ok {
				platformStrings = append(platformStrings, ps)
			}
		}

		results := s.manager.PostToAll(ctx, content, platformStrings)
		return results, nil
	default:
		return "", fmt.Errorf("unknown social action: %s", action)
	}
}
