package domain

// Handoff describes a single agent handoff exposed by the overseer agent.
type Handoff struct {
	ToolName        string
	ToolDescription string
	Agent           CustomAgent
}
