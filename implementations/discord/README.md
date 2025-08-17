# assistant-discord

A Discord bot in Go that connects to an AI assistant backend.

Environment variables:
- DISCORD_TOKEN: your bot token
- BACKEND_BASE_URL: e.g., https://assistant.example.com

Config fields (can also be set via your own persistence implementation in `config.go`):
- BotChannelID: channel where the bot will respond to plain messages
- ConversationParentChannelID: text channel where conversation threads will be created
- GuildID: optional guild to scope slash commands

Run:
```bash
GO111MODULE=on go run ./...
```
