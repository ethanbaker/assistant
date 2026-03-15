// routes.go - defines constant route expressions
package routes

const (
	// API base
	API_BASE = "/api"
	PUBLIC_PREFIX = API_BASE + "/public"
	INTERNAL_PREFIX = API_BASE + "/internal"

	// Health endpoints
	GET_HEALTH = PUBLIC_PREFIX + "/health"

	// Agent endpoints
	AGENT_BASE = INTERNAL_PREFIX + "/agent"
	POST_CREATE_SESSION = AGENT_BASE + "/sessions"
	GET_SESSION_BY_UUID = AGENT_BASE + "/sessions/:uuid"
	POST_MESSAGE_TO_SESSION = AGENT_BASE + "/sessions/:uuid/message"
	DELETE_SESSION = AGENT_BASE + "/sessions/:uuid"
)
