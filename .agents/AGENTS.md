# AGENTS.md

## Project Overview

**Reddit MCP** is a Model Context Protocol server for Reddit trend analysis. Built in Go, it lets AI assistants discover trending content, find communities, and analyze topic momentum — without any authentication. Uses Reddit's public `.json` endpoints exclusively.

**License**: Apache 2.0. All `.go` files carry the copyright header (Copyright 2024 Alby Hernández).

## Key Technologies

| Technology | Details |
|------------|---------|
| Language | Go 1.24+ — pure Go, `CGO_ENABLED=0`, no C dependencies |
| MCP | `github.com/mark3labs/mcp-go` v0.44.0 |
| JQ | `github.com/itchyny/gojq` v0.12.17 — embedded, no external binary |
| Reddit API | Public `.json` endpoints, no OAuth, no API keys |
| Config | YAML via `gopkg.in/yaml.v3`, env vars expanded with `os.ExpandEnv` |
| Logging | `log/slog` with JSON handler to stderr |

## Commands

| Command | What it does |
|---------|-------------|
| `make build` | Build to `bin/reddit-mcp` (`CGO_ENABLED=0`, ldflags strip + version inject) |
| `make build-all` | Cross-compile: linux/amd64, linux/arm64, darwin/arm64, windows/amd64 |
| `make run` | Build then run with `-config config.yaml` |
| `make test` | `go test -v ./...` (no tests exist yet) |
| `make fmt` | `go fmt ./...` |
| `make vet` | `go vet ./...` |
| `make tidy` | `go mod tidy` |
| `make clean` | Remove `bin/` |

Version is injected via `-ldflags "-w -s -X main.Version=$(VERSION)"` where VERSION defaults to `git describe --tags --always --dirty` or `"dev"`. Note: `main.Version` is referenced in ldflags but not declared in `cmd/main.go` — it exists only when injected at build time.

**Running locally**:
```bash
make build
./bin/reddit-mcp -config docs/config-stdio.yaml   # stdio transport
./bin/reddit-mcp -config docs/config-http.yaml     # HTTP on :8080
```

**Docker**:
```bash
docker build -t reddit-mcp .
docker run -v $(pwd)/config.yaml:/config/config.yaml reddit-mcp
```

The Dockerfile uses a two-stage build (golang:1.24-alpine → alpine:3.19), runs as non-root `appuser`, and defaults to `-config /config/config.yaml`.

## Code Organization

```
.
├── cmd/
│   └── main.go                  # Entrypoint: config → reddit client → MCP server → transport
├── api/
│   └── config_types.go          # Config types: Configuration, ServerConfig, TransportConfig, HTTPConfig, RedditConfig
├── internal/
│   ├── config/
│   │   └── config.go            # ReadFile(): reads YAML, expands env vars via os.ExpandEnv
│   ├── globals/
│   │   └── globals.go           # ApplicationContext: Context + Logger (*slog.Logger) + Config
│   ├── reddit/
│   │   └── client.go            # Reddit HTTP client, all API methods, domain types (Post, SubredditInfo, etc.)
│   └── tools/
│       ├── tools.go             # ToolsManager + AddTools() — all 9 tool registrations + applyJQ helper
│       ├── handlers.go          # One HandleTool* method per tool
│       ├── helpers.go           # Argument extraction: getArgs, getString, getInt, getStringSlice
│       └── jq.go                # runJQ(): gojq-based jq filter execution
├── docs/
│   ├── config-stdio.yaml        # Example: stdio transport
│   ├── config-http.yaml         # Example: HTTP transport on :8080
│   └── images/header.svg        # README header
└── .github/workflows/
    └── release.yaml             # CI: multi-platform binaries + Docker to ghcr.io
```

## Architecture

### Startup Flow (`cmd/main.go`)

1. `globals.NewApplicationContext()` — parses `-config` flag (default: `config.yaml`), reads YAML config, creates JSON slog logger to stderr
2. `reddit.NewClient(userAgent)` — HTTP client with 15s timeout; falls back to `"MCP-TrendBot/1.0 (by /u/achetronic)"` if user_agent is empty
3. `server.NewMCPServer(name, version)` — MCP server with tool capabilities enabled
4. `tools.NewToolsManager(deps).AddTools()` — registers all 9 tools with descriptions and handlers
5. Transport switch on `server.transport.type`:
   - `"http"` → `StreamableHTTPServer` on configured host, mounted at `/mcp` endpoint, 30s heartbeat, stateful sessions, 10s read header timeout
   - default → `ServeStdio`

### Reddit Client (`internal/reddit/client.go`)

- Base URL: `https://www.reddit.com`
- All requests routed through `doGet(endpoint, params)` — sets `User-Agent` header, checks status codes
- Request timeout: 15 seconds
- Parses Reddit's standard listing response: `data.children[].data`
- Trend score computed on every post: `velocity * (1 + engagement)`
  - `velocity = score / age_hours` (age floored to 1 hour)
  - `engagement = num_comments / max(score, 1)`
- `DiscoverCommunities` filters out NSFW subreddits; other methods do not
- Methods that loop over topics/subreddits (`GetCrossoverCommunities`, `GetTrendingByTopic`) silently skip failures with `continue`

### Tool System (`internal/tools/`)

Every handler follows the same pattern:
1. `getArgs(request)` → extracts `map[string]any` from MCP request
2. Typed extraction: `getString(args, key, default)`, `getInt(args, key, default)`, `getStringSlice(args, key)`
3. Validate required fields → `mcp.NewToolResultError(msg)` on failure
4. Call Reddit client method
5. `applyJQ(data, jqFilter)` → marshal to JSON → run jq filter → on jq error, fall back to unfiltered output silently
6. Return `mcp.NewToolResultText(output)`

### Configuration (`api/config_types.go`)

```yaml
server:
  name: "reddit-mcp"
  version: "0.1.0"
  transport:
    type: "stdio"           # or "http"
    http:
      host: ":8080"         # only when type=http
reddit:
  user_agent: "AppName/1.0 (by /u/username)"
```

Environment variables in YAML values are expanded before parsing (`$VAR` and `${VAR}` syntax).

## Available MCP Tools

### Discovery
| Tool | Handler | Client Method | Has jq_filter |
|------|---------|---------------|:---:|
| `discover_communities` | `HandleToolDiscoverCommunities` | `DiscoverCommunities` | no |
| `get_crossover_communities` | `HandleToolGetCrossoverCommunities` | `GetCrossoverCommunities` | no |
| `get_subreddit_info` | `HandleToolGetSubredditInfo` | `GetSubredditInfo` | no |

### Trending & Search
| Tool | Handler | Client Method | Has jq_filter |
|------|---------|---------------|:---:|
| `get_trending_posts` | `HandleToolGetTrendingPosts` | `GetSubredditPosts` (loop + merge) | yes |
| `get_topic_trends` | `HandleToolGetTopicTrends` | `GetTrendingByTopic` | yes |
| `get_subreddit_pulse` | `HandleToolGetSubredditPulse` | `GetSubredditPosts` | yes |
| `search_posts` | `HandleToolSearchPosts` | `SearchPosts` | yes |
| `get_rising_posts` | `HandleToolGetRisingPosts` | `GetRisingPosts` | yes |

### Analysis
| Tool | Handler | Client Method | Has jq_filter |
|------|---------|---------------|:---:|
| `analyze_sentiment_trend` | `HandleToolAnalyzeSentimentTrend` | `AnalyzeSentimentTrend` | no |

Tools with `jq_filter` accept an optional jq expression to reduce output. Invalid filters silently fall back to full unfiltered JSON.

## Adding a New Tool

Three files, always:

1. **`internal/reddit/client.go`** — Add a method on `*Client`. Follow existing patterns: validate limits, call `doGet`, parse with `parsePosts` or custom struct, apply `calcTrendScore` on posts.

2. **`internal/tools/tools.go`** in `AddTools()` — Register with `mcp.NewTool(name, ...)`. Include a thorough multi-line description with recommended `jq_filter`. Bind handler with `tm.dependencies.McpServer.AddTool(tool, tm.HandleTool*)`.

3. **`internal/tools/handlers.go`** — Add `HandleTool*` method on `*ToolsManager`. Follow the exact 6-step pattern described above. Use only the helpers from `helpers.go` for arg extraction.

## Naming Conventions

| Element | Convention | Example |
|---------|-----------|---------|
| Files | snake_case | `config_types.go` |
| Packages | single lowercase word | `tools`, `reddit`, `globals` |
| Types | PascalCase | `ToolsManager`, `SubredditInfo` |
| Exported functions | PascalCase | `NewClient`, `ReadFile` |
| Handler methods | `HandleTool` + PascalCase name | `HandleToolGetTrendingPosts` |
| MCP tool names | snake_case | `get_trending_posts` |
| JSON struct tags | snake_case, matching Reddit API | `json:"num_comments"` |
| Error wrapping | `fmt.Errorf("context: %w", err)` | consistent across codebase |
| Module name | `reddit-mcp` | matches binary name |

## Testing

No `_test.go` files exist. `make test` runs `go test -v ./...` but there's nothing to test currently. When adding tests:
- Place `*_test.go` alongside source files (Go convention)
- Reddit client: mock with `httptest.NewServer` returning fixture JSON
- Tool handlers: the `reddit.Client` is a concrete struct (not an interface), so either introduce an interface or use integration-style tests with a mock HTTP server

## CI/CD

`.github/workflows/release.yaml` triggers on:
- GitHub Release `published` event
- Manual `workflow_dispatch` (requires version tag, optional `use_main` flag)

Pipeline:
1. **build-binaries** — Matrix build: linux/amd64, linux/arm64 (ubuntu), darwin/arm64 (macos-14), windows/amd64 (ubuntu)
2. **build-docker** — Multi-arch `linux/amd64,linux/arm64`, pushed to `ghcr.io` with `VERSION` + `latest` tags, GHA cache
3. **upload-release** — Downloads all artifacts, generates SHA256 checksums, uploads to GitHub Release via `softprops/action-gh-release@v2`

## Gotchas

- **429 rate limits**: Reddit allows ~60 req/min unauthenticated. No backoff is implemented — callers must add delays between tool calls.
- **HTML instead of JSON**: Reddit blocks requests with missing or generic `User-Agent`. Format must be `AppName/Version (by /u/username)`.
- **Silent jq fallback**: Invalid jq filters don't produce errors — the tool returns the full unfiltered JSON. If output is unexpectedly large, check the filter syntax.
- **Silent error swallowing**: `GetCrossoverCommunities`, `GetTrendingByTopic`, and `HandleToolGetTrendingPosts` skip failed subreddit/topic fetches with `continue` — no partial error reporting.
- **JSON numbers as float64**: MCP protocol sends all numbers as `float64`. The `getInt` helper in `helpers.go` casts. Always use these helpers, never assert types directly on `args`.
- **NSFW filtering is inconsistent**: Only `DiscoverCommunities` filters NSFW subreddits (checks `Over18`). All other methods return NSFW content if present.
- **No pagination**: Every endpoint fetches a single page. Reddit's `limit` parameter caps at 100. `DiscoverCommunities` further caps at 25.
- **Default limits differ**: `get_trending_posts` handler defaults to 10 posts/subreddit; `get_subreddit_pulse`, `search_posts`, and `get_rising_posts` default to 25. The Reddit client methods all default to 25 internally.
- **Empty results**: Subreddit may be private, quarantined, banned, or non-existent. Use `get_subreddit_info` to validate before fetching posts.
- **`get_trending_posts` always uses `"top"` sort**: The handler hardcodes `sort="top"` when calling `GetSubredditPosts`, regardless of any user intent.

## Commit Conventions

- Commits are authored as **Magec** (`magec@magec.dev`)
- Commit style observed in history: `type: description` (e.g., `feat: add get_topic_trends`, `docs: update README`)
- Release notes must be written in **English**
- Plain language — no corporate speak, no jargon
- The `.agents/` directory is intentionally tracked in git (other AI tool dirs like `.cursor/`, `.claude/` are gitignored)
