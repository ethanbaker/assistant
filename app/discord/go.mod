module github.com/ethanbaker/assistant/app/discord

go 1.24.3

require (
	github.com/bwmarrin/discordgo v0.29.0
	github.com/ethanbaker/assistant v0.0.0-00010101000000-000000000000
)

require (
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	golang.org/x/crypto v0.39.0 // indirect
	golang.org/x/sys v0.33.0 // indirect
)

replace github.com/ethanbaker/assistant => ../../
