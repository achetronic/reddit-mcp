# AGENTS.md

This document helps AI agents work effectively in this repository.

## Project Overview

**Reddit MCP** is a Model Context Protocol server for Reddit trend analysis. Built in Go, it lets AI assistants discover what's trending on Reddit, find relevant communities, and analyze topic popularity — without authentication.

## Key Technologies

- **Language**: Go 1.24+
- **MCP Library**: `github.com/mark3labs/mcp-go` v0.44.0
- **JQ filtering**: `github.com/itchyny/gojq` — embedded jq, no binary dependency
- **Reddit API**: Public `.json` endpoints, no auth required
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
│   │   └── config.go            # YAML config loader with env expansion
│   ├── globals/
│   │   └── globals.go           # ApplicationContext (config + logger)
│   ├── reddit/
│   │   └── client.go            # Reddit HTTP client, trend scoring, all API methods
│   └── tools/
│       ├── tools.go             # ToolsManager, tool registration, applyJQ helper
│       ├── handlers.go          # One handler per tool
│       ├── helpers.go           # getArgs, getString, getInt, getStringSlice
│       └── jq.go                # runJQ using gojq
├── docs/
│   ├── config-stdio.yaml        # Stdio config example
│   ├── config-http.yaml         # HTTP config example
│   └── images/
│       └── header.svg           # README header image
└── .github/workflows/
    └── release.yaml             # CI/CD — binaries + Docker image
```

## Architecture

- No authentication — uses Reddit's public `.json` endpoints
- Always set a proper `User-Agent` or Reddit will 429 you
- Trend score: `velocity * (1 + engagement)` where:
  - `velocity = score / age_hours`
  - `engagement = num_comments / max(score, 1)`
- All tools that return posts accept `jq_filter` — **always use it** to reduce context

## Available Tools

### Discovery
- `discover_communities` — Subreddits for a single topic, sorted by subscribers
- `get_crossover_communities` — Subreddits covering multiple topics, ranked by hit_count then subscribers
- `get_subreddit_info` — Stats: subscribers, active_user_count, description, NSFW flag

### Trending & Search
- `get_trending_posts` — Posts from known subreddits, merged and sorted by trend_score
- `get_topic_trends` — Global Reddit search across multiple topics at once
- `get_subreddit_pulse` — Current state of a single community (hot/new/top/rising)
- `search_posts` — Keyword search, global or restricted to a subreddit
- `get_rising_posts` — Posts gaining momentum in r/all right now

### Analysis
- `analyze_sentiment_trend` — Compares last 24h vs last N days: avg_score, avg_comments, avg_trend_score, trending_up flag

## Recommended Flow (multi-topic research)

```
1. get_crossover_communities(topics: [...], limit_per_topic: 5)
   → returns subreddits ranked by how many topics they cover

2. get_trending_posts(subreddits: [...top results...], limit: 5,
   jq_filter: "[.[:20] | .[] | {title, subreddit, score, trend_score, permalink}]")
   → compact, high-quality posts from real communities
```

For quick overviews: `get_topic_trends` with jq_filter in one call.

## JQ Filters

Always provide `jq_filter` on tools that return posts. Recommended defaults are in each tool's description. Examples:

```
# Compact list
[.[] | {title, score, trend_score, permalink}]

# Top N by trend score
sort_by(-.trend_score) | .[0:5]

# For get_topic_trends
[.[] | {topic, posts: [.posts[:3] | .[] | {title, score, trend_score, subreddit}]}]
```

If the filter is invalid, the tool falls back to full unfiltered output — no error thrown.

## Adding New Tools

1. Add the Reddit API method in `internal/reddit/client.go`
2. Register the tool in `internal/tools/tools.go` with a clear description and recommended jq_filter
3. Add the handler in `internal/tools/handlers.go` using `getArgs`, `getString`, etc.

## Common Issues

- **429**: Rate limit (~60 req/min). Add delays between calls.
- **Empty results**: Subreddit may be private or not exist. Validate with `get_subreddit_info` first.
- **jq filter error**: Falls back to unfiltered output silently.
- **Reddit returns HTML**: User-Agent is being blocked. Set it properly in config.

## Guidelines

1. Release notes must always be written in **English**
2. Plain language — no corporate speak, no jargon
3. Always document the recommended `jq_filter` in tool descriptions
