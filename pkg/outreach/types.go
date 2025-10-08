package outreach

import "github.com/ethanbaker/assistant/pkg/utils"

/** Loaded types for outreach module */

// Task return type
type TaskReturn struct {
	Content string `json:"content"`        // Content/message from the outreach task
	Data    any    `json:"data,omitempty"` // Optional extra data from the outreach task
}

// Task function type
type TaskRunFunction func(cfg *utils.Config) *TaskReturn

// Task represents a single outreach task that can be scheduled and executed
type Task struct {
	Key           string         `json:"key" yaml:"key"`                       // Unique identifier for the outreach task
	ClientIds     []string       `json:"client_ids" yaml:"client_ids"`         // List of client IDs to use for this task
	Params        map[string]any `json:"params" yaml:"params"`                 // Parameters for the outreach task (must include 'client_ids' array)
	Cadence       CadenceType    `json:"cadence" yaml:"cadence"`               // Cadence type for scheduling (cron, sunrise, sunset)
	CadenceParams map[string]any `json:"cadence_params" yaml:"cadence_params"` // Parameters specific to the cadence type

	Run TaskRunFunction `json:"-" yaml:"-"` // Internal function to execute the task (set when loaded)
}

/** DTOs for outreach module */

// RegisterRequest sent by an implementation to register itself with the outreach service
type RegisterRequest struct {
	CallbackUrl  string `json:"callback_url"`  // HTTP endpoint where outreach requests will be sent
	ClientSecret string `json:"client_secret"` // Secret for signing requests (optional)
	ClientId     string `json:"client_id"`     // Unique identifier for the implementation
}

/** Responses from outreach implementations */

// Response sent from an outreach task to the manager
// This represents the result of executing an outreach task
type Response struct {
	Status string `json:"status"` // Status of the outreach request (success, error, etc.)

	Key           string         `json:"key"`            // Key of the outreach task that generated this response
	IdempotencyId string         `json:"idempotency_id"` // Idempotency ID of the request that generated this response
	Params        map[string]any `json:"params"`         // Parameters from the original outreach task

	Content string `json:"content"`        // Optional message from the implementation
	Data    any    `json:"data,omitempty"` // Optional extra data from the implementation

	Clients []struct {
		Id          string `json:"id"`           // Id of the client implementation
		CallbackUrl string `json:"callback_url"` // Callback URL of the client implementation
	} `json:"clients"` // List of clients, in priority order, to send the response to
}
