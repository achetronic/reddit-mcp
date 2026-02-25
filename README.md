# Reddit MCP

A Model Context Protocol server that lets AI assistants explore Reddit trends, discover communities, and analyze what's hot — without authentication.

Built in Go. No OAuth. No API key. Just Reddit's public JSON endpoints.

---

## What can it do?

- Find trending posts across multiple subreddits at once
- Search for posts about any topic, globally or within a community
- Discover which subreddits are most relevant for a given topic
- Find communities that cover multiple topics at once (the valuable ones)
- See what's rising right now across all of Reddit
- Get stats and info about any subreddit

All results include a custom **trend score** (`velocity × engagement`) so you can rank posts by actual momentum, not just raw votes.

Tools that return posts support a **jq_filter** parameter so the AI can filter output and avoid burning through context.

---

## Getting started

### 1. Configure

Create a `config.yaml`:

```yaml
server:
  name: "reddit-mcp"
  version: "0.1.0"
  transport:
    type: "stdio"

reddit:
  user_agent: "MCP-TrendBot/1.0 (by /u/yourusername)"
```

See `docs/config-stdio.yaml` and `docs/config-http.yaml` for full examples.

### 2. Build and run

```bash
go mod tidy
make build
./bin/reddit-mcp -config config.yaml
```

---

## Available tools

| Tool | What it does |
|------|--------------|
| `get_trending_posts` | Trending posts from one or more subreddits, sorted by trend score |
| `get_subreddit_pulse` | Current state of a community (hot/new/top/rising) |
| `search_posts` | Search posts by keyword, globally or within a subreddit |
| `get_rising_posts` | Posts gaining momentum in r/all right now |
| `discover_communities` | Find relevant subreddits for a topic |
| `get_crossover_communities` | Subreddits covering multiple topics (ranked by overlap) |
| `get_subreddit_info` | Stats and info about a specific subreddit |

---

## The trend score

Every post comes with a `trend_score` calculated as:

```
trend_score = velocity × (1 + engagement)
velocity    = score / age_hours
engagement  = num_comments / max(score, 1)
```

A post with 500 upvotes from 2 hours ago scores much higher than one with 2000 upvotes from 3 days ago. Combined with comment engagement, this gives a real signal of what's actually hot right now.

---

## JQ filters

All tools that return posts accept a `jq_filter` parameter. This lets you trim the output to only what you need, saving context tokens.

Examples:

```
# Only title, score and URL
[.[] | {title, score, url}]

# Only posts with more than 100 upvotes
[.[] | select(.score > 100)]

# Top 5 by trend score
sort_by(-.trend_score) | .[0:5]
```

If the filter is invalid, the tool falls back to full output.

---

## Crossover communities

`get_crossover_communities` is the most useful tool for topic research. Given a list of topics, it finds subreddits that appear across multiple of them:

```
topics: [kubernetes, devops, golang]

Result:
- r/devops      → appears in all 3  → hit_count: 3
- r/kubernetes  → appears in 2      → hit_count: 2
- r/golang      → appears in 1      → hit_count: 1
```

The more topics a subreddit covers, the more valuable it is as a monitoring target.

---

## Rate limits

Reddit allows ~60 requests/minute without auth. If you're making many calls, add a small delay between them. The server doesn't implement automatic backoff.

---

## Docker

```bash
docker build -t reddit-mcp .
docker run -v $(pwd)/config.yaml:/config/config.yaml reddit-mcp
```

---

## License

Apache 2.0
