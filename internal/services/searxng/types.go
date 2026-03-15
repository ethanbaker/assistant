package searxng

// SearxResult represents a single search result from SearXNG
type SearxResult struct {
	URL     string `json:"url"`
	Title   string `json:"title"`
	Content string `json:"content"`
	Engine  string `json:"engine"`
}

// SearxResponse represents the full response from SearXNG
type SearxResponse struct {
	Query           string        `json:"query"`
	NumberOfResults int           `json:"number_of_results"`
	Results         []SearxResult `json:"results"`
}
