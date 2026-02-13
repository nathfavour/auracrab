package schema

import (
	"encoding/json"
	"fmt"
	"regexp"

	"github.com/hjson/hjson-go/v4"
)

// --- Prompt Schema ---

type ProjectTopology struct {
	Files            []string `json:"files"`
	ModifiedRecently []string `json:"modified_recently"`
	Dependencies     []string `json:"dependencies"`
	Deltas           string   `json:"deltas,omitempty"`
}

type SystemTelemetry struct {
	OS          string  `json:"os"`
	CPUUsage    float64 `json:"cpu_usage"`
	MemoryUsage float64 `json:"memory_usage"`
	EnergyLevel float64 `json:"energy_level"` // 0.0 - 1.0 (manages heartbeat frequency)
}

type MemoryContext struct {
	RecentActions []string `json:"recent_actions"`
	LastFailures  []string `json:"last_failures"`
	EgoState      string   `json:"ego_state"`
	Mission       *MissionInfo `json:"mission,omitempty"`
}

type MissionInfo struct {
	Title         string `json:"title"`
	Goal          string `json:"goal"`
	TimeRemaining string `json:"time_remaining"`
	Progress      float64 `json:"progress"`
	TTC           string  `json:"estimated_ttc"`
	SubTasks      []SubTaskInfo `json:"sub_tasks,omitempty"`
}

type SubTaskInfo struct {
	ID           string   `json:"id"`
	Title        string   `json:"title"`
	Status       string   `json:"status"`
	Dependencies []string `json:"dependencies"`
}

type ToolManifest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Parameters  string `json:"parameters"` // JSON schema of params
}

type PromptPacket struct {
	Mode      string          `json:"mode"` // "analytical" or "casual"
	Project   ProjectTopology `json:"project"`
	System    SystemTelemetry `json:"system"`
	Memory    MemoryContext   `json:"memory"`
	Tools     []ToolManifest  `json:"tools"`
	Blueprint string          `json:"response_blueprint"`
}

// ToHjson converts the prompt packet to HJSON string
func (p *PromptPacket) ToHjson() (string, error) {
	data, err := json.Marshal(p)
	if err != nil {
		return "", err
	}
	
	var hjsonObj interface{}
	err = hjson.Unmarshal(data, &hjsonObj)
	if err != nil {
		return "", err
	}
	
	hjsonString, err := hjson.Marshal(hjsonObj)
	return string(hjsonString), err
}

// --- Response Schema ---

type Action struct {
	Tool           string                 `json:"tool"`
	Parameters     map[string]interface{} `json:"parameters"`
	AssuranceScore float64                `json:"assurance_score"` // 0.0 - 1.0
}

type ResponsePacket struct {
	Intent         string   `json:"intent"`
	Strategy       string   `json:"strategy"`
	Actions        []Action `json:"actions"`
	CasualMessage  string   `json:"casual_message,omitempty"` // For the "Taunting Friend" vibe
	Cooldown       int      `json:"cooldown_ms"`
	SelfCorrection string   `json:"self_correction,omitempty"`

	// Autonomous mission management
	MissionProgress *float64       `json:"mission_progress,omitempty"`
	EstimatedTTC    *string        `json:"estimated_ttc,omitempty"` // e.g. "2h45m"
	Finalize        bool           `json:"finalize,omitempty"`
	NewSubTasks     []NewSubTask   `json:"new_sub_tasks,omitempty"`
	UpdateSubTask   *UpdateSubTask `json:"update_sub_task,omitempty"`
}

type NewSubTask struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Dependencies []string `json:"dependencies"`
}

type UpdateSubTask struct {
	ID     string `json:"id"`
	Status string `json:"status"`
	Result string `json:"result,omitempty"`
}

func ParseResponse(data string) (*ResponsePacket, error) {
	// Use non-greedy match to find individual JSON objects
	re := regexp.MustCompile(`(?s)\{.*?\}`)
	matches := re.FindAllString(data, -1)
	
	if len(matches) == 0 {
		return nil, fmt.Errorf("no JSON found in response: %s", data)
	}

	// Try matches from last to first (latest is usually best)
	var lastErr error
	for i := len(matches) - 1; i >= 0; i-- {
		var resp ResponsePacket
		if err := json.Unmarshal([]byte(matches[i]), &resp); err == nil {
			return &resp, nil
		} else {
			lastErr = err
		}
	}

	return nil, fmt.Errorf("failed to parse any JSON block: %v. Raw: %s", lastErr, data)
}
