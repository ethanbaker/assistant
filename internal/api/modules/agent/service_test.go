package agent

import (
"context"
"testing"

"github.com/ethanbaker/assistant/internal/domain"
"github.com/ethanbaker/assistant/internal/repositories/session"
"github.com/google/uuid"
"github.com/nlpodyssey/openai-agents-go/agents"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

// stubAgent satisfies domain.CustomAgent without an LLM connection.
type stubAgent struct{}

func (s *stubAgent) Agent() *agents.Agent              { return agents.New("stub") }
func (s *stubAgent) ID() string                        { return "stub" }
func (s *stubAgent) ShouldDryRun(_ context.Context) bool { return false }

func newTestService() *Service {
return NewService(ServiceConfig{
SessionRepository: session.NewInMemoryRepository(),
EntryAgent:        &stubAgent{},
})
}

func TestService_CreateSession(t *testing.T) {
ctx := context.Background()
svc := newTestService()

sess, err := svc.CreateSession(ctx, "user1")
require.NoError(t, err)
assert.NotEmpty(t, sess.SessionID(ctx))
}

func TestService_GetSession_ValidID(t *testing.T) {
ctx := context.Background()
svc := newTestService()

created, err := svc.CreateSession(ctx, "user1")
require.NoError(t, err)

got, err := svc.GetSession(ctx, created.SessionID(ctx))
require.NoError(t, err)
assert.Equal(t, created.SessionID(ctx), got.SessionID(ctx))
}

func TestService_GetSession_InvalidUUIDFormat(t *testing.T) {
ctx := context.Background()
svc := newTestService()

_, err := svc.GetSession(ctx, "not-a-uuid")
assert.Error(t, err)
assert.Contains(t, err.Error(), "invalid session ID format")
}

func TestService_GetSession_NotFound(t *testing.T) {
ctx := context.Background()
svc := newTestService()

_, err := svc.GetSession(ctx, uuid.New().String())
assert.Error(t, err)
}

func TestService_DeleteSession_ValidID(t *testing.T) {
ctx := context.Background()
svc := newTestService()

created, err := svc.CreateSession(ctx, "user1")
require.NoError(t, err)

deleted, err := svc.DeleteSession(ctx, created.SessionID(ctx))
require.NoError(t, err)
assert.Equal(t, created.SessionID(ctx), deleted.SessionID(ctx))

// Session should be gone
_, err = svc.GetSession(ctx, created.SessionID(ctx))
assert.Error(t, err)
}

func TestService_DeleteSession_InvalidUUIDFormat(t *testing.T) {
ctx := context.Background()
svc := newTestService()

_, err := svc.DeleteSession(ctx, "not-a-uuid")
assert.Error(t, err)
assert.Contains(t, err.Error(), "invalid session ID format")
}

func TestService_DeleteSession_NotFound(t *testing.T) {
ctx := context.Background()
svc := newTestService()

_, err := svc.DeleteSession(ctx, uuid.New().String())
assert.Error(t, err)
}

func TestService_GetItemCount_Empty(t *testing.T) {
ctx := context.Background()
svc := newTestService()

created, err := svc.CreateSession(ctx, "user1")
require.NoError(t, err)

count, err := svc.GetItemCount(ctx, created.SessionID(ctx))
require.NoError(t, err)
assert.Equal(t, 0, count)
}

func TestService_GetItemCount_InvalidUUID(t *testing.T) {
ctx := context.Background()
svc := newTestService()

_, err := svc.GetItemCount(ctx, "bad-uuid")
assert.Error(t, err)
assert.Contains(t, err.Error(), "invalid session ID format")
}

func TestService_GetSessionItems_LimitRespected(t *testing.T) {
ctx := context.Background()
repo := session.NewInMemoryRepository()
svc := NewService(ServiceConfig{
SessionRepository: repo,
EntryAgent:        &stubAgent{},
})

sess, err := svc.CreateSession(ctx, "user1")
require.NoError(t, err)

sessionID, _ := uuid.Parse(sess.SessionID(ctx))
for i := 0; i < 5; i++ {
require.NoError(t, repo.SaveItem(ctx, &domain.Item{SessionID: sessionID}))
}

items, err := svc.GetSessionItems(ctx, sess.SessionID(ctx), 3)
require.NoError(t, err)
assert.Len(t, items, 3)
}

func TestService_GetSessionItems_LimitLargerThanCount(t *testing.T) {
ctx := context.Background()
repo := session.NewInMemoryRepository()
svc := NewService(ServiceConfig{
SessionRepository: repo,
EntryAgent:        &stubAgent{},
})

sess, err := svc.CreateSession(ctx, "user1")
require.NoError(t, err)

sessionID, _ := uuid.Parse(sess.SessionID(ctx))
for i := 0; i < 2; i++ {
require.NoError(t, repo.SaveItem(ctx, &domain.Item{SessionID: sessionID}))
}

items, err := svc.GetSessionItems(ctx, sess.SessionID(ctx), 10)
require.NoError(t, err)
assert.Len(t, items, 2)
}

func TestService_GetSessionItems_InvalidUUID(t *testing.T) {
ctx := context.Background()
svc := newTestService()

_, err := svc.GetSessionItems(ctx, "bad-uuid", 5)
assert.Error(t, err)
assert.Contains(t, err.Error(), "invalid session ID format")
}
