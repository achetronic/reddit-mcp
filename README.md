<p align="center">
  <img src="docs/images/header.svg" alt="Reddit MCP" width="800"/>
</p>

<p align="center">
  <em>A Model Context Protocol server that lets AI assistants explore Reddit trends,<br/>discover communities, and analyze what's hot — without authentication.</em>
</p>

<p align="center">
  <a href="#-what-can-it-do">What it does</a> •
  <a href="#-getting-started">Getting Started</a> •
  <a href="#-available-tools">Tools</a> •
  <a href="#-the-trend-score">Trend Score</a> •
  <a href="#-jq-filters">JQ Filters</a> •
  <a href="#-docker">Docker</a>
</p>

---

## 🎯 What can it do?

This MCP gives your AI assistant the ability to:

- **Discover** which subreddits are most relevant for any topic
- **Find** crossover communities that cover multiple topics at once
- **Track** what's trending right now across specific subreddits
- **Search** posts globally or within a community
- **Spot** rising content before it goes viral
- **Analyze** whether a topic is gaining or losing momentum over time

No API key. No OAuth. Just Reddit's public JSON endpoints.

---

## 🚀 Getting started

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

> ⚠️ Always set a proper `user_agent`. Reddit blocks requests with missing or generic user agents. Use your actual Reddit username.

#### HTTP transport (for production)

```yaml
server:
  name: "reddit-mcp"
  version: "0.1.0"
  transport:
    type: "http"
    http:
      host: ":8080"

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

## 🛠️ Available tools

### Discovery

| Tool | What it does |
|------|--------------|
| `discover_communities` | Find subreddits for a single topic, sorted by subscribers |
| `get_crossover_communities` | Find subreddits covering multiple topics at once, ranked by overlap |
| `get_subreddit_info` | Stats about a subreddit: subscribers, active users, description |

### Trending & Search

| Tool | What it does |
|------|--------------|
| `get_trending_posts` | Posts from known subreddits, merged and sorted by trend score |
| `get_topic_trends` | Global Reddit search across multiple topics at once |
| `get_subreddit_pulse` | Current state of a single community (hot/new/top/rising) |
| `search_posts` | Search posts by keyword, globally or within a subreddit |
| `get_rising_posts` | Posts gaining momentum in r/all right now |

### Analysis

| Tool | What it does |
|------|--------------|
| `analyze_sentiment_trend` | Compare topic momentum: last 24h vs last N days |

All tools that return posts accept a `jq_filter` parameter. **Always use it** to avoid burning context.

---

## 🔥 The trend score

Every post comes with a `trend_score` calculated as:

```
trend_score = velocity × (1 + engagement)
velocity    = score / age_hours
engagement  = num_comments / max(score, 1)
```

A post with 500 upvotes from 2 hours ago scores much higher than one with 2000 upvotes from 3 days ago. This gives a real signal of what's actually hot right now, not just what accumulated votes over time.

---

## 🔍 JQ filters

All tools that return posts accept a `jq_filter` parameter. This is the key to keeping output compact and avoiding context blowup.

**Always provide one.** The tool descriptions include recommended filters.

### Examples

```bash
# Compact list — title, score and link only
[.[] | {title, score, permalink}]

# Top 5 posts sorted by trend score
sort_by(-.trend_score) | .[0:5]

# Only high-scoring posts
[.[] | select(.score > 500)]

# For get_topic_trends — 3 posts per topic, minimal fields
[.[] | {topic, posts: [.posts[:3] | .[] | {title, score, trend_score, subreddit}]}]
```

If the filter is invalid, the tool falls back to full unfiltered output.

---

## 🗺️ Recommended workflow

When exploring multiple topics (e.g. kubernetes, cloud, golang, AI):

**Step 1 — Find the right communities:**
```
get_crossover_communities(
  topics: ["kubernetes", "cloud", "golang", "AI"],
  limit_per_topic: 5
)
```
This gives you subreddits ranked by how many of your topics they cover.

**Step 2 — Get trending posts from those communities:**
```
get_trending_posts(
  subreddits: ["devops", "kubernetes", "golang", "MachineLearning"],
  time_range: "week",
  limit: 5,
  jq_filter: "[.[:20] | .[] | {title, subreddit, score, trend_score, permalink}]"
)
```

Two calls. Compact output. High-quality results from actual communities.

---

## 🐳 Docker

```bash
docker build -t reddit-mcp .
docker run -v $(pwd)/config.yaml:/config/config.yaml reddit-mcp
```

---

## ⚠️ Rate limits

Reddit allows ~60 requests/minute without authentication. If you're making many calls, add a small delay between them. The server doesn't implement automatic backoff — handle it at the application level if needed.

---

## 🔧 Troubleshooting

### 429 Too Many Requests
You've hit Reddit's rate limit. Wait a minute and try again.

### Empty results
The subreddit might be private, very small, or the query too specific. Use `get_subreddit_info` to validate a community before fetching posts from it.

### jq filter error
Invalid jq syntax falls back to unfiltered output. Check the filter syntax if you're getting more data than expected.

### Reddit returns HTML instead of JSON
Your `user_agent` is being blocked. Make sure it's set and follows the format `AppName/Version (by /u/username)`.

---

## 🤝 Contributing

For AI agents working on this codebase, see [AGENTS.md](.agents/AGENTS.md).

---

## 📄 License

Apache 2.0
