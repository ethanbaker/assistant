package test

import (
	"context"
	"encoding/json"

	"github.com/ethanbaker/assistant/internal/api/modules/outreach"
)

func TestOutreach(ctx context.Context, params json.RawMessage) (*outreach.OutreachResponse, error) {
	return &outreach.OutreachResponse{
		Content: "test",
	}, nil
}
