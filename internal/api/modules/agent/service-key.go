package agent

// ValidateAPIKey checks if the provided API key is valid
// TODO: replace with SQL table of API keys
func validateAPIKey(apiKey string) bool {
	return apiKey == "2d78d012-29a7-4210-b427-3037e79dc33b"
}
