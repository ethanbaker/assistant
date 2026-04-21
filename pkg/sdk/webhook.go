package sdk

// WebhookPayload represents a payload delivered to a webhook from outreach
type WebhookPayload struct {
	ID      uint   `json:"id"`
	JobID   int    `json:"job_id"`
	JobName string `json:"job_name"`
	Content string `json:"content"`
	Data    any    `json:"data,omitempty"`
}
