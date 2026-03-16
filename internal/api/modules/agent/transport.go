package agent

import (
	"time"

	"github.com/google/uuid"
)

// SessionTranscript represents searchable session content
type SessionTranscript struct {
	SessionID uuid.UUID `json:"session_id"`
	Data      string    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

// AttachJobExecutionContextRequest defines payload for adding outreach execution context.
type AttachJobExecutionContextRequest struct {
	JobExecutionIDs []int `json:"job_execution_ids" binding:"required"`
}
