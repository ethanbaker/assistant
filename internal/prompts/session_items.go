package prompts

import (
	"context"
	"errors"
	"time"

	"github.com/nlpodyssey/openai-agents-go/memory"
	"github.com/openai/openai-go/v2/packages/param"
	"github.com/openai/openai-go/v2/responses"
)

const (
	timeFormat = "3:04 PM"
	dateFormat = "Monday, January 2, 2006"
)

var (
	// ErrSessionRequired is returned when a nil session is passed to an injector helper.
	ErrSessionRequired = errors.New("session is required")
	// ErrContentRequired is returned when an empty context message is passed.
	ErrContentRequired = errors.New("content is required")
)

// InjectMessageItem adds a single message item to a session.
func InjectMessageItem(ctx context.Context, session memory.Session, role responses.EasyInputMessageRole, content string) error {
	if session == nil {
		return ErrSessionRequired
	}
	if content == "" {
		return ErrContentRequired
	}

	message := responses.EasyInputMessageParam{Role: role, Type: "message"}
	message.Content.OfString = param.NewOpt(content)

	item := responses.ResponseInputItemUnionParam{OfMessage: &message}
	return session.AddItems(ctx, []memory.TResponseInputItem{item})
}

// InjectContextItem adds a developer-context message item to a session.
func InjectContextItem(ctx context.Context, session memory.Session, content string) error {
	return InjectMessageItem(ctx, session, responses.EasyInputMessageRoleDeveloper, content)
}

// InjectCurrentTimeContext adds the current time as a context item to a session.
func InjectCurrentTimeContext(ctx context.Context, session memory.Session, now time.Time) error {
	if err := InjectContextItem(ctx, session, "Current time: "+now.Format(timeFormat)); err != nil {
		return err
	}
	return InjectContextItem(ctx, session, "Current date: "+now.Format(dateFormat))
}
