// tools.go handles registering tools for the SearchAgent
package search

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/nlpodyssey/openai-agents-go/agents"
	"github.com/openai/openai-go/v2/packages/param"
)

// Limit content length to prevent overwhelming the model
const maxContentLength = 10000

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

// registerTools registers the search-related tools
func (sa *SearchAgent) registerTools() {
	// Web search tool
	webSearchTool := agents.FunctionTool{
		Name:        "web_search",
		Description: "Search the internet for information using a meta-search engine",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"query": map[string]any{
					"type":        "string",
					"description": "The search query to execute",
				},
				"num_results": map[string]any{
					"type":        "integer",
					"description": "Maximum number of results to return (optional, defaults to 10)",
					"minimum":     1,
					"maximum":     50,
					"default":     10,
				},
				"category": map[string]any{
					"type":        "string",
					"description": "Search category: 'general', 'news', 'images', or 'map' (optional, defaults to 'general')",
					"enum":        []string{"general", "news", "images", "map"},
					"default":     "general",
				},
			},
			"additionalProperties": false,
			"required":             []string{"query", "num_results", "category"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleWebSearch(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// URL fetch tool
	fetchURLTool := agents.FunctionTool{
		Name:        "fetch_url",
		Description: "Fetch and extract text content from a specific URL",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"url": map[string]any{
					"type":        "string",
					"description": "The URL to fetch content from",
				},
				"extract_main_content": map[string]any{
					"type":        "boolean",
					"description": "Whether to extract only main content (removes ads, navigation, etc.) or full HTML text (optional, defaults to true)",
					"default":     true,
				},
			},
			"additionalProperties": false,
			"required":             []string{"url", "extract_main_content"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleFetchURL(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Summarize search results tool
	summarizeResultsTool := agents.FunctionTool{
		Name:        "summarize_search_results",
		Description: "Summarize and synthesize information from multiple search results",
		ParamsJSONSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"results": map[string]any{
					"type": "array",
					"items": map[string]any{
						"type": "object",
						"properties": map[string]any{
							"url": map[string]any{
								"type": "string",
							},
							"title": map[string]any{
								"type": "string",
							},
							"content": map[string]any{
								"type": "string",
							},
						},
						"additionalProperties": false,
						"required":             []string{"url", "title", "content"},
					},
					"description": "Array of search results to summarize",
				},
				"focus_query": map[string]any{
					"type":        "string",
					"description": "Specific question or topic to focus the summary on (optional)",
					"default":     "",
				},
				"summary_style": map[string]any{
					"type":        "string",
					"description": "Summary style: 'brief', 'detailed', or 'bullet_points' (optional, defaults to 'detailed')",
					"enum":        []string{"brief", "detailed", "bullet_points"},
					"default":     "detailed",
				},
			},
			"additionalProperties": false,
			"required":             []string{"results", "focus_query", "summary_style"},
		},
		StrictJSONSchema: param.NewOpt(true),
		OnInvokeTool: func(ctx context.Context, arguments string) (any, error) {
			return sa.handleSummarizeResults(ctx, arguments)
		},
		IsEnabled: agents.FunctionToolEnabled(),
	}

	// Register all tools with the agent
	sa.agent.Tools = []agents.Tool{
		webSearchTool,
		fetchURLTool,
		summarizeResultsTool,
	}
}

// handleWebSearch performs a web search using SearXNG
func (sa *SearchAgent) handleWebSearch(_ context.Context, arguments string) (map[string]any, error) {
	// Unmarshal parameters
	var params struct {
		Query      string `json:"query"`
		NumResults int    `json:"num_results"`
		Category   string `json:"category"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate parameters
	if params.Query == "" {
		return nil, fmt.Errorf("query parameter is required")
	}
	if params.NumResults <= 0 || params.NumResults > 50 {
		return nil, fmt.Errorf("num_results must be between 1 and 50")
	}
	if params.Category == "" {
		return nil, fmt.Errorf("category parameter is required")
	}

	validCategories := map[string]bool{
		"general": true,
		"news":    true,
		"images":  true,
		"videos":  true,
	}
	if val, ok := validCategories[params.Category]; !ok || !val {
		return nil, fmt.Errorf("invalid category: %s", params.Category)
	}

	// Get SearXNG instance URL from config
	searxURL := sa.config.Get("SEARXNG_URL")
	if searxURL == "" {
		return nil, fmt.Errorf("SEARXNG_URL not configured")
	}

	// Construct search URL
	searchURL := fmt.Sprintf("%s/search?q=%s&format=json&category_%s=1",
		searxURL,
		url.QueryEscape(params.Query),
		params.Category,
	)

	// Make the request
	resp, err := sa.httpClient.Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("search request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("search request failed with status: %d", resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Parse JSON response
	var searxResp SearxResponse
	if err := json.Unmarshal(body, &searxResp); err != nil {
		return nil, fmt.Errorf("failed to parse search results: %w", err)
	}

	// Limit results to requested number
	if len(searxResp.Results) > params.NumResults {
		searxResp.Results = searxResp.Results[:params.NumResults]
	}

	return map[string]any{
		"query":           params.Query,
		"num_results":     len(searxResp.Results),
		"total_found":     searxResp.NumberOfResults,
		"search_engine":   "SearXNG (Meta-search)",
		"results":         searxResp.Results,
		"search_category": params.Category,
	}, nil
}

// handleFetchURL fetches content from a specific URL
func (sa *SearchAgent) handleFetchURL(ctx context.Context, arguments string) (map[string]any, error) {
	// Unmarshal parameters
	var params struct {
		URL                string `json:"url"`
		ExtractMainContent bool   `json:"extract_main_content"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate parameters
	if params.URL == "" {
		return nil, fmt.Errorf("url parameter is required")
	}

	// Validate URL
	_, err := url.Parse(params.URL)
	if err != nil {
		return nil, fmt.Errorf("invalid url: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", params.URL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set a reasonable user agent
	req.Header.Set("User-Agent", "SearchAgent/1.0 (Web Content Fetcher)")

	// Make the request
	resp, err := sa.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch request failed with status: %d", resp.StatusCode)
	}

	// Read response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	content := string(body)

	// Extract main content if requested
	if params.ExtractMainContent {
		content = sa.extractMainContent(content)
	} else {
		content = sa.stripHTML(content)
	}

	// Truncate content if too long
	if len(content) > maxContentLength {
		content = content[:maxContentLength] + "\n\n[Content truncated...]"
	}

	return map[string]any{
		"url":            params.URL,
		"status_code":    resp.StatusCode,
		"content_type":   resp.Header.Get("Content-Type"),
		"title":          sa.extractTitle(string(body)),
		"content":        content,
		"content_length": len(content),
		"extracted_main": params.ExtractMainContent,
	}, nil
}

// handleSummarizeResults summarizes multiple search results
func (sa *SearchAgent) handleSummarizeResults(_ context.Context, arguments string) (map[string]any, error) {
	// Unmarshal parameters
	var params struct {
		Results []struct {
			URL     string `json:"url"`
			Title   string `json:"title"`
			Content string `json:"content"`
		} `json:"results"`
		FocusQuery   string `json:"focus_query"`
		SummaryStyle string `json:"summary_style"`
	}

	if err := json.Unmarshal([]byte(arguments), &params); err != nil {
		return nil, fmt.Errorf("invalid arguments: %w", err)
	}

	// Validate parameters
	if len(params.Results) == 0 {
		return map[string]any{
			"summary": "No results provided to summarize.",
			"sources": []string{},
			"style":   params.SummaryStyle,
		}, nil
	}

	validSummaryStyles := map[string]bool{
		"brief":         true,
		"detailed":      true,
		"bullet_points": true,
	}
	if valid, ok := validSummaryStyles[params.SummaryStyle]; !ok || !valid {
		return nil, fmt.Errorf("invalid summary_style: %s", params.SummaryStyle)
	}

	// Build summary based on style
	var summary strings.Builder
	var sources []string

	if params.FocusQuery != "" {
		summary.WriteString(fmt.Sprintf("Summary focused on: %s\n\n", params.FocusQuery))
	}

	switch params.SummaryStyle {
	case "brief":
		summary.WriteString("Key findings:\n")
		for i, result := range params.Results {
			if i >= 3 { // Limit to top 3 for brief summary
				break
			}
			summary.WriteString(fmt.Sprintf("• %s\n", sa.extractKeyPoint(result.Content)))
			sources = append(sources, result.URL)
		}

	case "bullet_points":
		summary.WriteString("Key points from search results:\n\n")
		for _, result := range params.Results {
			summary.WriteString(fmt.Sprintf("## %s\n", result.Title))
			points := sa.extractBulletPoints(result.Content)
			for _, point := range points {
				summary.WriteString(fmt.Sprintf("• %s\n", point))
			}
			summary.WriteString(fmt.Sprintf("Source: %s\n\n", result.URL))
			sources = append(sources, result.URL)
		}

	default: // detailed
		summary.WriteString("Detailed synthesis of search results:\n\n")
		for i, result := range params.Results {
			summary.WriteString(fmt.Sprintf("### Source %d: %s\n", i+1, result.Title))
			summary.WriteString(fmt.Sprintf("%s\n\n", sa.cleanAndSummarizeContent(result.Content)))
			sources = append(sources, result.URL)
		}
	}

	return map[string]any{
		"summary":     summary.String(),
		"sources":     sources,
		"style":       params.SummaryStyle,
		"focus_query": params.FocusQuery,
		"num_sources": len(sources),
	}, nil
}

// Helper functions for content processing

func (sa *SearchAgent) stripHTML(content string) string {
	// Simple HTML tag removal
	re := regexp.MustCompile(`<[^>]*>`)
	content = re.ReplaceAllString(content, " ")

	// Clean up whitespace
	re = regexp.MustCompile(`\s+`)
	content = re.ReplaceAllString(content, " ")

	return strings.TrimSpace(content)
}

func (sa *SearchAgent) extractTitle(html string) string {
	re := regexp.MustCompile(`<title[^>]*>([^<]*)</title>`)
	matches := re.FindStringSubmatch(html)
	if len(matches) > 1 {
		return strings.TrimSpace(matches[1])
	}
	return "No title found"
}

func (sa *SearchAgent) extractMainContent(html string) string {
	// Remove script and style tags
	re := regexp.MustCompile(`<(?:script|style)[^>]*>.*?</(?:script|style)>`)
	html = re.ReplaceAllString(html, "")

	// Look for main content indicators
	mainSelectors := []string{
		`<main[^>]*>(.*?)</main>`,
		`<article[^>]*>(.*?)</article>`,
		`<div[^>]*class="[^"]*content[^"]*"[^>]*>(.*?)</div>`,
		`<div[^>]*class="[^"]*main[^"]*"[^>]*>(.*?)</div>`,
	}

	for _, selector := range mainSelectors {
		re := regexp.MustCompile(`(?s)` + selector)
		matches := re.FindStringSubmatch(html)
		if len(matches) > 1 {
			return sa.stripHTML(matches[1])
		}
	}

	// Fallback to full body content
	return sa.stripHTML(html)
}

// Simple heuristic functions for summarization
func (sa *SearchAgent) extractKeyPoint(content string) string {
	// Simple extraction of first meaningful sentence
	sentences := strings.Split(content, ". ")
	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 20 && len(sentence) < 200 {
			return sentence + "."
		}
	}

	// Fallback to first 100 characters
	if len(content) > 100 {
		return content[:100] + "..."
	}
	return content
}

// Extract bullet points from content
func (sa *SearchAgent) extractBulletPoints(content string) []string {
	// Split content into sentences and extract key ones
	sentences := strings.Split(content, ". ")
	var points []string

	for _, sentence := range sentences {
		sentence = strings.TrimSpace(sentence)
		if len(sentence) > 30 && len(sentence) < 300 {
			points = append(points, sentence)
			if len(points) >= 5 {
				break
			}
		}
	}

	return points
}

// Clean and summarize content for detailed summaries
func (sa *SearchAgent) cleanAndSummarizeContent(content string) string {
	content = strings.TrimSpace(content)

	// Split into paragraphs and take first few meaningful ones
	paragraphs := strings.Split(content, "\n")
	var cleanParagraphs []string

	for _, para := range paragraphs {
		para = strings.TrimSpace(para)
		if len(para) > 50 { // Only include substantial paragraphs
			cleanParagraphs = append(cleanParagraphs, para)
			if len(cleanParagraphs) >= 3 { // Limit to first 3 paragraphs
				break
			}
		}
	}

	result := strings.Join(cleanParagraphs, "\n\n")

	// Limit overall length
	if len(result) > 1000 {
		result = result[:1000] + "..."
	}

	return result
}
