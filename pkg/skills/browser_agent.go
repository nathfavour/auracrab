package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nathfavour/auracrab/pkg/connect"
	"github.com/nathfavour/auracrab/pkg/vibe"
)

type BrowserAgentSkill struct {
	MaxSteps int
}

func (s *BrowserAgentSkill) Name() string {
	return "browser_agent"
}

func (s *BrowserAgentSkill) Description() string {
	return "Autonomous browser agent that can perform multi-step tasks to achieve a goal"
}

func (s *BrowserAgentSkill) Manifest() []byte {
	return []byte(`{
		"parameters": {
			"type": "object",
			"properties": {
				"goal": {
					"type": "string",
					"description": "The high-level goal to achieve (e.g., 'Find the price of BTC on CoinMarketCap and tell me')"
				},
				"context": {
					"type": "string",
					"description": "Optional keyword to identify the right browser profile/window"
				},
				"max_steps": {
					"type": "integer",
					"default": 10
				}
			},
			"required": ["goal"]
		}
	}`)
}

type BrowserStepAction struct {
	Action    string `json:"action"`
	URL       string `json:"url,omitempty"`
	Selector  string `json:"selector,omitempty"`
	Text      string `json:"text,omitempty"`
	Condition string `json:"condition,omitempty"`
	Reasoning string `json:"reasoning"`
	Finished  bool   `json:"finished"`
}

func (s *BrowserAgentSkill) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var params struct {
		Goal     string `json:"goal"`
		Context  string `json:"context"`
		MaxSteps int    `json:"max_steps"`
	}
	if err := json.Unmarshal(args, &params); err != nil {
		return "", err
	}

	if params.MaxSteps == 0 {
		params.MaxSteps = 10
	}
	if s.MaxSteps > 0 && params.MaxSteps > s.MaxSteps {
		params.MaxSteps = s.MaxSteps
	}

	bc := connect.GetBrowserChannel()
	if bc == nil || !bc.IsActive() {
		return "", fmt.Errorf("no browser extension connected")
	}

	targetInstance := ""
	if params.Context != "" {
		client := bc.FindClientByTab(params.Context)
		if client != nil {
			targetInstance = client.InstanceID
		}
	}

	vibeClient := vibe.NewClient()
	history := []string{fmt.Sprintf("Goal: %s", params.Goal)}

	for i := 0; i < params.MaxSteps; i++ {
		// 1. Observe
		observation, err := s.observe(ctx, bc, targetInstance)
		if err != nil {
			return "", fmt.Errorf("step %d observation failed: %v", i, err)
		}

		// 2. Plan next step
		step, err := s.planNextStep(ctx, vibeClient, params.Goal, observation, history)
		if err != nil {
			return "", fmt.Errorf("step %d planning failed: %v", i, err)
		}

		history = append(history, fmt.Sprintf("Step %d: %s (Action: %s)", i+1, step.Reasoning, step.Action))

		if step.Finished {
			return fmt.Sprintf("Goal achieved: %s\n\nFinal Result: %s", params.Goal, step.Text), nil
		}

		// 3. Execute step
		result, err := s.executeStep(ctx, bc, targetInstance, step)
		if err != nil {
			history = append(history, fmt.Sprintf("Error: %v", err))
			continue
		}
		history = append(history, fmt.Sprintf("Result: %s", result))

		time.Sleep(1 * time.Second) // Small delay between steps
	}

	return fmt.Sprintf("Reached max steps (%d) without completing the goal: %s", params.MaxSteps, params.Goal), nil
}

func (s *BrowserAgentSkill) observe(ctx context.Context, bc *connect.BrowserChannel, target string) (string, error) {
	// Get current URL and interactive elements
	rawElements, err := bc.Request(ctx, target, "scrape:interactive")
	if err != nil {
		return bc.Request(ctx, target, "scrape")
	}
	return rawElements, nil
}

func (s *BrowserAgentSkill) planNextStep(ctx context.Context, client *vibe.Client, goal, observation string, history []string) (*BrowserStepAction, error) {
	prompt := fmt.Sprintf("You are an autonomous browser agent.\nGOAL: %s\n\nHISTORY:\n%s\n\nCURRENT PAGE OBSERVATION (Interactive Elements):\n%s\n\nYour task is to decide the next single action to take to reach the goal.\nAvailable actions:\n- open (url: string)\n- click (selector: string)\n- type (selector: string, text: string)\n- hover (selector: string)\n- wait (condition: string - e.g. \"selector:.class\" or \"2000\")\n- finished (text: string - use this when the goal is achieved, provide final answer in 'text')\n\nReturn ONLY a JSON object:\n{\n  \"action\": \"open|click|type|hover|wait|finished\",\n  \"url\": \"...\",\n  \"selector\": \"...\",\n  \"text\": \"...\",\n  \"condition\": \"...\",\n  \"reasoning\": \"why you are taking this step\",\n  \"finished\": true|false\n}", goal, strings.Join(history, "\n"), observation)

	res, err := client.Query(prompt, "plan")
	if err != nil {
		return nil, err
	}

	res = strings.TrimSpace(res)
	if strings.HasPrefix(res, "```json") {
		res = strings.TrimPrefix(res, "```json")
		res = strings.TrimSuffix(res, "```")
		res = strings.TrimSpace(res)
	}

	var step BrowserStepAction
	if err := json.Unmarshal([]byte(res), &step); err != nil {
		return nil, fmt.Errorf("failed to parse plan: %v. Raw: %s", err, res)
	}

	return &step, nil
}

func (s *BrowserAgentSkill) executeStep(ctx context.Context, bc *connect.BrowserChannel, target string, step *BrowserStepAction) (string, error) {
	var command string
	switch step.Action {
	case "open":
		command = "open " + step.URL
	case "click":
		command = "click " + step.Selector
	case "type":
		command = "type " + step.Selector + " " + step.Text
	case "hover":
		command = "hover " + step.Selector
	case "wait":
		command = "wait " + step.Condition
	case "finished":
		return "Goal achieved: " + step.Text, nil
	default:
		return "", fmt.Errorf("unknown action: %s", step.Action)
	}

	return bc.Request(ctx, target, command)
}
