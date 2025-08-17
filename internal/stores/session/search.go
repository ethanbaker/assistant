package session

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
