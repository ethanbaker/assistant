# Personal AI Assistant

A modular, personalized AI assistant system built with Go that integrates with your daily workflows (Notion, Email, GitHub, etc.).

## Features

- **Modular Agent Architecture**: Each integration is a separate agent with its own tools and capabilities
- **Persistent Memory**: Session transcripts and key facts stored in MySQL
- **Intent-based Routing**: Automatically routes queries to the appropriate agent
- **Security-focused**: Scoped tool access with dry-run capabilities
- **Extensible**: Easy to add new agents and tools

## Architecture

The system consists of:

- **Core packages** (`pkg/`): Shared interfaces, session management, memory storage
- **Agent implementations** (`internal/`): Specific integrations (GitHub, Notion, Email, Memory)
- **Orchestrator** (`cmd/main.go`): Routes queries and manages agents

## Quick Start

### 1. Setup Database

Create a MySQL database and update the connection string in `.env`:

```sql
CREATE DATABASE assistant_db;
-- Tables will be created automatically on first run
```

### 2. Configure Environment

Copy the example configuration files:

```bash
cp .env.example .env
cp .env.memory-agent.example .env.memory-agent
cp .env.github-agent.example .env.github-agent
cp .env.notion-agent.example .env.notion-agent
cp .env.email-agent.example .env.email-agent
```

Update the files with your actual API keys and credentials.

### 3. Install Dependencies

```bash
go mod download
```

### 4. Run the Assistant

```bash
go run cmd/main.go
```

## Configuration

Each agent can have its own `.env.<agent-name>` file that overrides global settings. The configuration loading follows this priority:

1. Agent-specific config (`.env.memory-agent`)
2. Global config (`.env`)
3. Environment variables

### Required Configuration

- `DATABASE_URL`: MySQL connection string
- `OPENAI_API_KEY`: OpenAI API key for each agent
- Agent-specific tokens (GitHub, Notion, email credentials)

## Agent Capabilities

### Memory Agent
- Search conversation transcripts
- Store and retrieve key facts
- List all stored information

### GitHub Agent
- List repositories
- Get repository information
- List pull requests
- Create issues

### Notion Agent
- Search pages
- Create and update pages
- List databases

### Email Agent
- Read inbox emails
- Send emails
- Search email history
- Mark emails as read/unread

## Usage Examples

The assistant automatically routes your queries to the appropriate agent:

- "Remember that my birthday is March 15th" → Memory Agent
- "What are my open pull requests?" → GitHub Agent  
- "Create a note about today's meeting" → Notion Agent
- "Check my recent emails" → Email Agent

## Development

### Adding New Agents

1. Create a new directory in `internal/`
2. Implement the `CustomAgent` interface
3. Register the agent in `cmd/main.go`
4. Add intent routing logic

### Adding New Tools

Tools are registered within each agent's `registerTools()` method. Each tool needs:

- Function definition (name, description, parameters)
- Handler function that implements the tool logic

## Security

- All tools support dry-run mode for safe testing
- Database operations are parameterized to prevent SQL injection
- Tool access is scoped per agent
- Full audit trail of all interactions

## Database Schema

The system automatically creates these tables:

- `sessions`: Conversation sessions
- `messages`: Individual messages in sessions  
- `tool_calls`: Tool executions with input/output
- `key_facts`: Stored facts and preferences

## License

MIT License - see LICENSE file for details.
