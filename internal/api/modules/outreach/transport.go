package outreach

import "encoding/json"

// CreateJobRequest is the request payload for creating an outreach job.
type CreateJobRequest struct {
	Name       string          `json:"name" binding:"required"`
	Schedule   json.RawMessage `json:"schedule" binding:"required"`
	Handler    string          `json:"handler" binding:"required"`
	Parameters json.RawMessage `json:"parameters"`
	Active     *bool           `json:"active"`
}

// CreateJobResponse is returned after successful job creation.
type CreateJobResponse struct {
	JobID int `json:"job_id"`
}

// RegisterClientRequest is the request payload for creating an outreach client.
type RegisterClientRequest struct {
	Name       string `json:"name" binding:"required"`
	WebhookURL string `json:"webhook_url" binding:"required"`
}

// RegisterClientResponse is returned after successful client creation.
type RegisterClientResponse struct {
	ClientID int    `json:"client_id"`
	APIKey   string `json:"api_key"`
}

// SubscribeRequest is the request payload for creating a job subscription.
type SubscribeRequest struct {
	JobName  string `json:"job_name" binding:"required"`
	Priority int    `json:"priority" binding:"required"`
}

// SubscribeResponse is returned after successful subscription.
type SubscribeResponse struct {
	SubscriptionID int `json:"subscription_id"`
}

// UnsubscribeRequest is the request payload for disabling a job subscription.
type UnsubscribeRequest struct {
	JobName string `json:"job_name" binding:"required"`
}
