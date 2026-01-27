package copilot

import (
"context"
"encoding/json"
"fmt"
"log"

sdk "github.com/github/copilot-sdk/go"
"github.com/nathfavour/auracrab/pkg/core"
"github.com/nathfavour/auracrab/pkg/security"
"github.com/nathfavour/auracrab/pkg/skills"
)

type Agent struct {
	client *sdk.Client
}

func NewAgent() *Agent {
	return &Agent{
		client: sdk.NewClient(&sdk.ClientOptions{
			LogLevel: "error",
		}),
	}
}

func (a *Agent) Start(ctx context.Context) error {
	if err := a.client.Start(); err != nil {
		return err
	}

	startTaskTool := sdk.DefineTool("start_task", "Start a persistent autonomous task in Auracrab",
func(params struct {
Task string `json:"task" jsonschema:"Description of the task to perform"`
}, inv sdk.ToolInvocation) (any, error) {
butler := core.GetButler()
			task, err := butler.StartTask(context.Background(), params.Task)
			if err != nil {
				return nil, err
			}
			return fmt.Sprintf("Task started successfully. ID: %s", task.ID), nil
		})

	auditTool := sdk.DefineTool("run_security_audit", "Run a deep security audit on the local system",
func(params struct{}, inv sdk.ToolInvocation) (any, error) {
report, err := security.RunAudit()
			if err != nil {
				return nil, err
			}
			return report, nil
		})

	statusTool := sdk.DefineTool("check_butler_status", "Check the current status and health of the Auracrab Butler",
func(params struct{}, inv sdk.ToolInvocation) (any, error) {
butler := core.GetButler()
			return butler.GetStatus(), nil
		})

	agentTools := []sdk.Tool{startTaskTool, auditTool, statusTool}
	for _, s := range skills.GetRegistry() {
		skill := s
		var manifestMap map[string]interface{}
		_ = json.Unmarshal(skill.Manifest(), &manifestMap)
		
		agentTools = append(agentTools, sdk.Tool{
Name:        skill.Name(),
			Description: skill.Description(),
			Parameters:  manifestMap["parameters"].(map[string]interface{}),
			Handler: func(inv sdk.ToolInvocation) (sdk.ToolResult, error) {
				argsData, _ := json.Marshal(inv.Arguments)
				res, err := skill.Execute(context.Background(), argsData)
				if err != nil {
					return sdk.ToolResult{Error: err.Error(), ResultType: "error"}, nil
				}
				return sdk.ToolResult{TextResultForLLM: res, ResultType: "success"}, nil
			},
		})
	}

	session, err := a.client.CreateSession(&sdk.SessionConfig{
		Model: "gpt-4o",
		Tools: agentTools,
		SystemMessage: &sdk.SystemMessageConfig{
			Content: "You are the Auracrab Digital Butler, a system-intimate AI agent. You help users manage persistent tasks and maintain system health.",
		},
	})
	if err != nil {
		return err
	}

	log.Println("Auracrab Copilot Agent session created.")
	
	done := make(chan bool)
	session.On(func(event sdk.SessionEvent) {
if event.Type == sdk.SessionIdle {
// Keep alive if needed
}
})

	return nil
}
