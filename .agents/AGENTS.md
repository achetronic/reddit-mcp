# AGENTS.md

This document helps AI agents work effectively in this repository.

## Project Overview

**Reddit MCP** is a Model Context Protocol server for Reddit trend analysis. Built in Go, it lets AI assistants discover what's trending on Reddit, find relevant communities, and analyze topic popularity — without authentication.

## Key Technologies

- **Language**: Go 1.24+
- **MCP Library**: `github.com/mark3labs/mcp-go` v0.44.0
- **JQ filtering**: `github.com/itchyny/gojq` — embedded jq for output filtering
- **Reddit API**: Public JSON endpoints, no auth required
- **Configuration**: YAML with environment variable expansion

## Code Organization

```
.
├── cmd/
│   └── main.go                  # Entrypoint
├── api/
│   └── config_types.go          # Configuration types
├── internal/
│   ├── config/
│   │   └── config.go            # YAML config loader
│   ├── globals/
│   │   └── globals.go           # ApplicationContext
│   ├── reddit/
│   │   └── client.go            # Reddit HTTP client + trend scoring
│   └── tools/
│       ├── tools.go             # ToolsManager + tool registration
│       ├── handlers.go          # Tool handler implementations
│       ├── helpers.go           # getArgs, getString, getInt, getStringSlice
│       └── jq.go                # runJQ helper using gojq
├── docs/
│   ├── config-stdio.yaml        # Stdio config example
│   └── config-http.yaml         # HTTP config example
└── .github/workflows/
    └── release.yaml             # CI/CD release pipeline
```

## Architecture

- No authentication — uses Reddit's public `.json` endpoints
- Always include a proper `User-Agent` or Reddit will 429 you
- Trend score formula: `velocity * (1 + engagement)` where:
  - `velocity = score / age_hours`
  - `engagement = num_comments / max(score, 1)`
- All tools that return posts accept a `jq_filter` parameter to reduce context usage

## Available Tools

- `get_trending_posts` — Trending posts from multiple subreddits, sorted by trend score
- `get_subreddit_pulse` — Current state of a community (hot/new/top/rising)
- `search_posts` — Search globally or within a subreddit
- `get_rising_posts` — Posts gaining momentum in r/all right now
- `discover_communities` — Find relevant subreddits for a topic
- `get_crossover_communities` — Subreddits covering multiple topics (ranked by overlap)
- `get_subreddit_info` — Stats and info about a specific subreddit

## JQ Filters

Use `jq_filter` to reduce token usage. Examples:

```
# Only title, score, subreddit and URL
[.[] | {title, score, subreddit, url}]

# Only posts with score > 100
[.[] | select(.score > 100)]

# Just titles
[.[] | .title]
```

## Adding New Tools

1. Add Reddit API method in `internal/reddit/client.go`
2. Add tool definition in `internal/tools/tools.go`
3. Add handler in `internal/tools/handlers.go` using `getArgs`, `getString`, etc.

## Common Issues

- **429 Too Many Requests**: Reddit rate-limits ~60 req/min without auth. Add delays between calls if hitting this.
- **Empty results**: Some subreddits are private or don't exist. Use `get_subreddit_info` to validate first.
- **jq filter error**: If the filter is invalid, the tool falls back to unfiltered output.

## Guidelines

1. Release notes must always be written in **English**
2. Plain language — no corporate speak, no jargon
