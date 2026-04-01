package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

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
- "whatsapp": Handles WhatsApp messages, chat analytics, sending messages, AND sending any PDT content (reports, briefings, summaries) via WhatsApp. Use this when users ask about WhatsApp chats, want to search conversations, OR want to SEND anything via WhatsApp. This agent can generate briefings/summaries and send reports directly.
- "scheduler": Handles creating, listing, enabling/disabling, deleting, and running scheduled agent tasks. Use this when users ask to schedule something, manage their schedules, automate tasks, set up recurring agents, or check schedule run history.

The user may write in Indonesian or English. Route based on intent, not language.
Keywords that suggest "briefing" agent: morning briefing, standup, persiapkan report, audit tiket, blocker, risiko, laporan pagi, briefing pagi.
Keywords that suggest "whatsapp" agent: whatsapp, wa, chat summary, pesan, kirim pesan, ringkasan chat, listener, group chat, send report, send briefing, kirim laporan, kirim ringkasan, share via wa, send to contact, send to group.
Keywords that suggest "scheduler" agent: schedule, jadwal, jadwalkan, automate, recurring, cron, timer, every morning, setiap pagi, scheduled task, my schedules, run history, disable schedule, enable schedule.
IMPORTANT: When user asks to SEND something via WhatsApp (report, briefing, summary, any content), ALWAYS route to "whatsapp" even if the content is about Jira, reports, or git.

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
				"enum": ["git", "jira", "report", "proof", "briefing", "whatsapp", "scheduler"],
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
	// Extract last user message for keyword fallback
	var userMessage string
	for i := len(messages) - 1; i >= 0; i-- {
		if messages[i].Role == "user" {
			userMessage = messages[i].Content
			break
		}
	}

	today := time.Now().Format("2006-01-02")
	routerMessages := append([]minimax.Message{{
		Role:    "system",
		Content: fmt.Sprintf("Today is %s.\n\n%s", today, routerSystemPrompt),
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
	log.Printf("[orchestrator] tool call: name=%s args=%s", tc.Function.Name, tc.Function.Arguments)

	var routing struct {
		AgentName  string `json:"agent_name"`
		Reason     string `json:"reason"`
		Properties string `json:"properties"` // MiniMax sometimes nests args here
	}
	if err := json.Unmarshal([]byte(tc.Function.Arguments), &routing); err != nil {
		log.Printf("[orchestrator] parse routing failed: %v, raw=%s", err, tc.Function.Arguments)
		return nil, fmt.Errorf("parse routing: %w", err)
	}

	// MiniMax wraps args as: {"properties": "{\"agent_name\":...}"}
	// Unwrap the nested JSON string if present
	if routing.AgentName == "" && routing.Properties != "" {
		var nested struct {
			AgentName string `json:"agent_name"`
			Reason    string `json:"reason"`
		}
		if json.Unmarshal([]byte(routing.Properties), &nested) == nil && nested.AgentName != "" {
			routing.AgentName = nested.AgentName
			routing.Reason = nested.Reason
			log.Printf("[orchestrator] unwrapped nested properties: agent=%s", routing.AgentName)
		}
	}

	if routing.AgentName == "" {
		// LLM returned empty agent — try keyword-based fallback
		routing.AgentName = detectAgentByKeyword(userMessage)
		routing.Reason = "keyword fallback"
		log.Printf("[orchestrator] empty agent from LLM, keyword fallback: %s", routing.AgentName)
	}

	if routing.AgentName == "" {
		content := "Maaf, saya tidak bisa menentukan agent yang tepat. Bisa ulangi pertanyaannya?"
		if err := writer.WriteContent(content); err != nil {
			return nil, err
		}
		return &LoopResult{FullResponse: content, Usage: resp.Usage}, nil
	}

	agent, ok := o.Agents[routing.AgentName]
	if !ok {
		return nil, fmt.Errorf("unknown agent: %s", routing.AgentName)
	}

	log.Printf("[orchestrator] routing to %s: %s", routing.AgentName, routing.Reason)

	return RunLoop(ctx, o.Client, agent, messages, writer)
}

// detectAgentByKeyword provides a fallback when the LLM fails to route.
func detectAgentByKeyword(msg string) string {
	lower := strings.ToLower(msg)

	// Order matters — more specific patterns first
	scheduleKeywords := []string{"schedule", "jadwal", "jadwalkan", "automate", "recurring", "cron", "scheduled task", "my schedules", "run history", "disable schedule"}
	for _, kw := range scheduleKeywords {
		if strings.Contains(lower, kw) {
			return "scheduler"
		}
	}

	waKeywords := []string{"whatsapp", "wa ", "kirim pesan", "send message", "send to", "kirim ke", "chat summary", "ringkasan chat", "send report", "send briefing", "kirim laporan"}
	for _, kw := range waKeywords {
		if strings.Contains(lower, kw) {
			return "whatsapp"
		}
	}

	reportKeywords := []string{"report", "laporan", "generate report", "daily report", "monthly report", "template", "laporan harian", "buat laporan", "generate daily"}
	for _, kw := range reportKeywords {
		if strings.Contains(lower, kw) {
			return "report"
		}
	}

	briefingKeywords := []string{"briefing", "standup", "blocker", "risiko", "audit"}
	for _, kw := range briefingKeywords {
		if strings.Contains(lower, kw) {
			return "briefing"
		}
	}

	jiraKeywords := []string{"jira", "sprint", "card", "ticket", "tiket", "issue", "backlog"}
	for _, kw := range jiraKeywords {
		if strings.Contains(lower, kw) {
			return "jira"
		}
	}

	gitKeywords := []string{"commit", "repository", "branch", "git", "repo", "push", "merge"}
	for _, kw := range gitKeywords {
		if strings.Contains(lower, kw) {
			return "git"
		}
	}

	proofKeywords := []string{"proof", "evidence", "bukti", "quality", "requirement"}
	for _, kw := range proofKeywords {
		if strings.Contains(lower, kw) {
			return "proof"
		}
	}

	return ""
}
