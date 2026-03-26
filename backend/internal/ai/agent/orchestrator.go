package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/cds-id/pdt/backend/internal/ai/minimax"
)

type Orchestrator struct {
	Client *minimax.Client
	Agents map[string]Agent
}

func NewOrchestrator(client *minimax.Client, agents ...Agent) *Orchestrator {
	agentMap := make(map[string]Agent)
	for _, a := range agents {
		agentMap[a.Name()] = a
	}
	return &Orchestrator{
		Client: client,
		Agents: agentMap,
	}
}

const routerSystemPrompt = `You are a routing assistant for PDT (Personal Development Tracker). Your job is to determine which specialist agent should handle the user's request.

Available agents:
- "git": Handles questions about commits, repositories, branches, and code activity.
- "jira": Handles questions about Jira sprints, cards, issues, and linking commits to cards.
- "report": Handles report generation (daily/monthly), listing reports, and report templates.
- "proof": Handles finding evidence in Jira comments, detecting quality issues in cards, and checking requirement coverage. Use this when users ask about what someone said, want proof of decisions, or want to find quality problems.
- "briefing": Handles morning briefing preparation, sprint auditing for risks, and blocker analysis. Use this when users ask to prepare for standup, audit their cards, find blockers, or identify risky tickets that could be questioned.

The user may write in Indonesian or English. Route based on intent, not language.
Keywords that suggest "briefing" agent: morning briefing, standup, persiapkan report, audit tiket, blocker, risiko, laporan pagi, briefing pagi.

If the user's message is a simple greeting or general question not related to any agent, respond directly without routing.

For all other messages, use the route_to_agent tool to delegate to the appropriate agent.`

var routerTool = minimax.Tool{
	Name:        "route_to_agent",
	Description: "Route the user's message to a specialist agent",
	InputSchema: json.RawMessage(`{
		"type": "object",
		"properties": {
			"agent_name": {
				"type": "string",
				"enum": ["git", "jira", "report", "proof", "briefing"],
				"description": "The specialist agent to route to"
			},
			"reason": {
				"type": "string",
				"description": "Brief reason for routing to this agent"
			}
		},
		"required": ["agent_name", "reason"]
	}`),
}

func (o *Orchestrator) HandleMessage(ctx context.Context, messages []minimax.Message, writer StreamWriter) (*LoopResult, error) {
	routerMessages := append([]minimax.Message{{
		Role:    "system",
		Content: routerSystemPrompt,
	}}, messages...)

	req := minimax.ChatRequest{
		Messages:    routerMessages,
		Tools:       []minimax.Tool{routerTool},
		Temperature: 0.3,
	}

	resp, err := o.Client.Chat(req)
	if err != nil {
		return nil, fmt.Errorf("router call: %w", err)
	}

	if len(resp.Choices) == 0 {
		return nil, fmt.Errorf("router returned no choices")
	}

	choice := resp.Choices[0]

	if len(choice.Delta.ToolCalls) == 0 {
		content := choice.Delta.Content
		if err := writer.WriteContent(content); err != nil {
			return nil, err
		}
		return &LoopResult{FullResponse: content, Usage: resp.Usage}, nil
	}

	tc := choice.Delta.ToolCalls[0]
	var routing struct {
		AgentName string `json:"agent_name"`
		Reason    string `json:"reason"`
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &routing); err != nil {
		return nil, fmt.Errorf("parse routing: %w", err)
	}

	agent, ok := o.Agents[routing.AgentName]
	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", routing.AgentName)
	}

	log.Printf("[orchestrator] routing to %s: %s", routing.AgentName, routing.Reason)

	return RunLoop(ctx, o.Client, agent, messages, writer)
}
