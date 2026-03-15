package agent

import (
"bytes"
"encoding/json"
"net/http"
"net/http/httptest"
"testing"

"github.com/ethanbaker/assistant/internal/repositories/session"
"github.com/gin-gonic/gin"
"github.com/google/uuid"
"github.com/stretchr/testify/assert"
"github.com/stretchr/testify/require"
)

func init() {
gin.SetMode(gin.TestMode)
}

func newTestRouter() (*gin.Engine, *Handler) {
svc := NewService(ServiceConfig{
SessionRepository: session.NewInMemoryRepository(),
EntryAgent:        &stubAgent{},
})
h := NewHandler(HandlerConfig{Service: svc})

r := gin.New()
r.POST("/sessions", h.CreateSession)
r.GET("/sessions/:uuid", h.GetSession)
r.DELETE("/sessions/:uuid", h.DeleteSession)
return r, h
}

func TestHandler_CreateSession_Success(t *testing.T) {
r, _ := newTestRouter()

body, _ := json.Marshal(map[string]string{"user_id": "user1"})
req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader(body))
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()

r.ServeHTTP(w, req)

assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_CreateSession_MissingBody(t *testing.T) {
r, _ := newTestRouter()

req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader([]byte(`{}`)))
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()

r.ServeHTTP(w, req)

assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_CreateSession_InvalidJSON(t *testing.T) {
r, _ := newTestRouter()

req := httptest.NewRequest(http.MethodPost, "/sessions", bytes.NewReader([]byte(`not-json`)))
req.Header.Set("Content-Type", "application/json")
w := httptest.NewRecorder()

r.ServeHTTP(w, req)

assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSession_Success(t *testing.T) {
r, h := newTestRouter()

// Create a session first via the service directly
ctx := httptest.NewRequest(http.MethodGet, "/", nil).Context()
sess, err := h.Service.CreateSession(ctx, "user1")
require.NoError(t, err)

req := httptest.NewRequest(http.MethodGet, "/sessions/"+sess.SessionID(ctx), nil)
w := httptest.NewRecorder()

r.ServeHTTP(w, req)

assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_GetSession_InvalidUUID(t *testing.T) {
r, _ := newTestRouter()

req := httptest.NewRequest(http.MethodGet, "/sessions/not-a-uuid", nil)
w := httptest.NewRecorder()

r.ServeHTTP(w, req)

assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_GetSession_NotFound(t *testing.T) {
r, _ := newTestRouter()

req := httptest.NewRequest(http.MethodGet, "/sessions/"+uuid.New().String(), nil)
w := httptest.NewRecorder()

r.ServeHTTP(w, req)

assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteSession_Success(t *testing.T) {
r, h := newTestRouter()

ctx := httptest.NewRequest(http.MethodDelete, "/", nil).Context()
sess, err := h.Service.CreateSession(ctx, "user1")
require.NoError(t, err)

req := httptest.NewRequest(http.MethodDelete, "/sessions/"+sess.SessionID(ctx), nil)
w := httptest.NewRecorder()

r.ServeHTTP(w, req)

assert.Equal(t, http.StatusOK, w.Code)
}

func TestHandler_DeleteSession_NotFound(t *testing.T) {
r, _ := newTestRouter()

req := httptest.NewRequest(http.MethodDelete, "/sessions/"+uuid.New().String(), nil)
w := httptest.NewRecorder()

r.ServeHTTP(w, req)

assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestHandler_DeleteSession_InvalidUUID(t *testing.T) {
r, _ := newTestRouter()

req := httptest.NewRequest(http.MethodDelete, "/sessions/not-a-uuid", nil)
w := httptest.NewRecorder()

r.ServeHTTP(w, req)

assert.Equal(t, http.StatusBadRequest, w.Code)
}
