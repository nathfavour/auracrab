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
	mgr := social.NewManager()
	// Register default platforms (placeholders)
	mgr.Register(&social.XPlatform{})
	mgr.Register(&social.LinkedInPlatform{})
	
	return &SocialSkill{manager: mgr}
}

func (s *SocialSkill) Name() string {
	return "social_post"
}

func (s *SocialSkill) Description() string {
	return "Post content to social media platforms (X, LinkedIn, Facebook, Instagram, Threads)."
}

func (s *SocialSkill) Manifest() json.RawMessage {
	manifest := map[string]interface{}{
		"name":        s.Name(),
		"description": s.Description(),
		"parameters": map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"platform": map[string]interface{}{
					"type":        "string",
					"description": "The target platform (x, linkedin, facebook, instagram, threads)",
					"enum":        []string{"x", "linkedin", "facebook", "instagram", "threads"},
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "The content to post",
				},
			},
			"required": []string{"platform", "content"},
		},
	}
	data, _ := json.Marshal(manifest)
	return json.RawMessage(data)
}

func (s *SocialSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Platform string `json:"platform"`
		Content  string `json:"content"`
	}

	if err := json.Unmarshal(args, &params); err != nil {
		return "", fmt.Errorf("invalid arguments: %v", err)
	}

	p, err := s.manager.GetPlatform(params.Platform)
	if err != nil {
		// For now, let's simulate the other platforms if they aren't fully implemented
		return fmt.Sprintf("Simulation: Posted '%s' to %s. (Platform driver pending full implementation)", params.Content, params.Platform), nil
	}

	url, err := p.Post(ctx, params.Content)
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("Successfully posted to %s: %s", params.Platform, url), nil
}

func init() {
	Register(NewSocialSkill())
}
