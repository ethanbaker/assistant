# Search Agent Implementation

## Overview

The Search Agent provides internet search and information gathering capabilities for the assistant system. It serves as the team's internet knowledge base, handling real-time information lookup, website content extraction, and search result synthesis.

## Features

### Core Capabilities
- **Web Search**: Meta-search using SearXNG (aggregates Google, Bing, DuckDuckGo, etc.)
- **URL Content Fetching**: Extract and parse website content with HTML stripping
- **News Search**: Find recent news articles with time range filtering
- **Content Summarization**: Synthesize information from multiple search results
- **Multi-source Verification**: Cross-reference information across sources

### Tools Available
1. **`web_search`** - Search the internet for information
2. **`fetch_url`** - Fetch and extract text content from specific URLs
3. **`summarize_search_results`** - Summarize and synthesize multiple search results

## Architecture

```
Search Agent
├── agent.go              # Main agent logic and initialization
├── tools.go              # Tool definitions and handlers
└── SearXNG Integration   # Meta-search engine backend
```

## Dependencies

### External Services
- **SearXNG**: Open-source meta-search engine (runs in Docker)
  - Aggregates results from multiple search engines
  - Privacy-focused (no tracking)
  - Self-hosted for complete control

### Docker Services
The search agent requires SearXNG running in Docker. This is automatically configured in `docker-compose.yml`:

```yaml
searxng:
  image: searxng/searxng:latest
  ports:
    - "8081:8080"
  environment:
    - SEARXNG_SECRET=your-secret-key-here
  volumes:
    - ./config/searxng:/etc/searxng
```

## Configuration

### Environment Variables

Add to your environment configuration:

```bash
# Search Agent Configuration
SEARCH_SYSPROMPT_PATH=resources/prompts/search-agent.txt
SEARXNG_URL=http://searxng:8080  # or http://localhost:8081 for local development

# Optional: Use public instance as fallback
# SEARX_URL=https://search.sapti.me
```

### SearXNG Configuration

The SearXNG instance is configured via `config/searxng/settings.yml` with:
- Multiple search engines enabled (Google, Bing, DuckDuckGo, etc.)
- JSON API format support
- Privacy settings (no tracking, no logging)
- Reasonable rate limits

## Usage Examples

### Web Search
```json
{
  "tool": "web_search",
  "arguments": {
    "query": "latest AI developments 2024",
    "num_results": 10,
    "category": "general"
  }
}
```

### News Search
```json
{
  "tool": "web_search",
  "arguments": {
    "query": "climate change",
    "num_results": 5,
    "category": "news"
  }
}
```

### Fetch URL Content
```json
{
  "tool": "fetch_url",
  "arguments": {
    "url": "https://example.com/article",
    "extract_main_content": true
  }
}
```

### Summarize Results
```json
{
  "tool": "summarize_search_results",
  "arguments": {
    "results": [...],
    "focus_query": "environmental impact",
    "summary_style": "bullet_points"
  }
}
```

## Integration with Overseer

The Search Agent is integrated into the system via the Overseer Agent, which can hand off tasks requiring internet search:

```
User Query → Overseer Agent → Search Agent (if internet search needed)
```

The overseer uses the handoff: `handoff_to_search_agent` with description:
"Hand off to the Search Agent for web searches, fetching URL content, finding current information, or researching topics on the internet"

## Setup Instructions

1. **Start the services**:
   ```bash
   docker-compose up -d
   ```

2. **Verify SearXNG is running**:
   ```bash
   curl http://localhost:8081/search?q=test&format=json
   ```

3. **Configure environment variables** as described above

4. **Test the search agent** through the API or direct integration

## Development Notes

### Content Processing
- HTML content is automatically stripped to text
- Main content extraction removes navigation, ads, etc.
- Content length is limited to prevent overwhelming the model
- Basic text cleaning and formatting applied

### Rate Limiting
- SearXNG handles rate limiting to upstream search engines
- No additional rate limiting implemented in the agent
- Reasonable defaults for result counts (10 for web, 5 for news)

### Error Handling
- Graceful degradation if SearXNG is unavailable
- Timeout handling for slow searches/URL fetches
- Validation of URLs and search parameters
- Informative error messages for debugging

## Troubleshooting

### Common Issues

1. **SearXNG not accessible**
   - Check if Docker container is running: `docker ps`
   - Verify port mapping: `docker-compose logs searxng`
   - Check SEARXNG_URL environment variable

2. **Search requests timeout**
   - SearXNG may be slow to start initially
   - Check SearXNG logs: `docker-compose logs searxng`
   - Consider increasing HTTP client timeout

3. **No search results**
   - Verify SearXNG configuration in `config/searxng/settings.yml`
   - Check if search engines are enabled
   - Test direct SearXNG API calls

### Monitoring
- Monitor SearXNG health via: `http://localhost:8081/healthz`
- Check search agent logs for error patterns
- Monitor response times for search operations

## Future Enhancements

Potential improvements for the search agent:
- Advanced content extraction (readability algorithms)
- Image search capabilities
- Search result caching
- Multiple search engine fallbacks
- Search result ranking/relevance scoring
- Specialized search types (academic papers, code search, etc.)
