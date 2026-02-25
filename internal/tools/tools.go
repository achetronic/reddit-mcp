// Copyright 2024 Alby Hernández
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package tools

import (
	"encoding/json"
	"fmt"
	"reddit-mcp/internal/globals"
	"reddit-mcp/internal/reddit"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

// ToolsManagerDependencies holds the dependencies for the tools manager
type ToolsManagerDependencies struct {
	AppCtx       *globals.ApplicationContext
	McpServer    *server.MCPServer
	RedditClient *reddit.Client
}

// ToolsManager manages the MCP tools registration
type ToolsManager struct {
	dependencies ToolsManagerDependencies
}

// NewToolsManager creates a new ToolsManager
func NewToolsManager(deps ToolsManagerDependencies) *ToolsManager {
	return &ToolsManager{dependencies: deps}
}

// applyJQ applies a jq filter to any value and returns the filtered JSON string.
// If filter is empty or ".", returns the original marshalled JSON.
func applyJQ(data any, filter string) (string, error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}
	if filter == "" || filter == "." {
		return string(raw), nil
	}

	// Parse JSON back to generic interface for jq processing
	var parsed any
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", fmt.Errorf("failed to unmarshal for jq: %w", err)
	}

	result, err := runJQ(filter, parsed)
	if err != nil {
		return string(raw), nil // fallback to unfiltered on jq error
	}
	return result, nil
}

// AddTools registers all Reddit tools into the MCP server
func (tm *ToolsManager) AddTools() {

	// get_trending_posts - Get trending posts from one or more subreddits
	tool := mcp.NewTool("get_trending_posts",
		mcp.WithDescription(`Get trending posts from a list of known subreddits, merged and sorted by trend_score (velocity × engagement).
Use this when you already know which communities are relevant.
Always use jq_filter to avoid burning context — recommended: '[.[:20] | .[] | {title, subreddit, score, trend_score, permalink}]'
For topic discovery first, use get_crossover_communities to find the right subreddits, then call this.`),
		mcp.WithArray("subreddits",
			mcp.Required(),
			mcp.Description("List of subreddit names (without r/). E.g. ['kubernetes', 'golang', 'devops']"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range: hour, day, week, month, year, all (default: day)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Posts per subreddit (default: 10, max: 100). Keep low (5-10) to save tokens."),
		),
		mcp.WithString("jq_filter",
			mcp.Description("jq filter to reduce output. Always provide this. Recommended: '[.[:20] | .[] | {title, subreddit, score, trend_score, permalink}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetTrendingPosts)

	// get_subreddit_pulse - Current state of a community
	tool = mcp.NewTool("get_subreddit_pulse",
		mcp.WithDescription(`Get the current state of a specific subreddit: top posts sorted by hot/new/top/rising with trend scores.
Use this to deep-dive into a single community after discovering it.
Always use jq_filter. Recommended: '[.[:10] | .[] | {title, score, num_comments, trend_score, permalink}]'`),
		mcp.WithString("subreddit",
			mcp.Required(),
			mcp.Description("Subreddit name (without r/)"),
		),
		mcp.WithString("sort",
			mcp.Description("Sort: hot (default), new, top, rising"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range when sort=top: hour, day, week, month, year, all (default: day)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of posts (default: 25, max: 100). Keep at 10-15 to save tokens."),
		),
		mcp.WithString("jq_filter",
			mcp.Description("jq filter to reduce output. Always provide this. Recommended: '[.[:10] | .[] | {title, score, num_comments, trend_score, permalink}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetSubredditPulse)

	// search_posts - Search posts globally or within a subreddit
	tool = mcp.NewTool("search_posts",
		mcp.WithDescription(`Search Reddit posts by keyword. Works globally or restricted to a subreddit.
Use this for specific queries or to find posts about a precise topic.
Always use jq_filter. Recommended: '[.[:10] | .[] | {title, subreddit, score, trend_score, permalink}]'
For broad topic discovery across multiple subjects, prefer get_topic_trends instead.`),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query. Supports Reddit operators like 'golang site:github.com' or 'kubernetes OR k8s'"),
		),
		mcp.WithString("subreddit",
			mcp.Description("Optional: restrict search to this subreddit"),
		),
		mcp.WithString("sort",
			mcp.Description("Sort: relevance, hot, top (default: hot)"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range: hour, day, week, month, year, all (default: week)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of results (default: 25, max: 100). Keep at 10-15 to save tokens."),
		),
		mcp.WithString("jq_filter",
			mcp.Description("jq filter to reduce output. Always provide this. Recommended: '[.[:10] | .[] | {title, subreddit, score, trend_score, permalink}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolSearchPosts)

	// get_rising_posts - Posts rising in r/all right now
	tool = mcp.NewTool("get_rising_posts",
		mcp.WithDescription(`Get posts currently gaining momentum in r/all/rising — early signal of what's about to trend.
This is cross-subreddit and real-time. No topic filter, so use jq_filter to focus on what matters.
Always use jq_filter. Recommended: '[.[:15] | .[] | {title, subreddit, score, trend_score, permalink}]'`),
		mcp.WithNumber("limit",
			mcp.Description("Number of posts (default: 25, max: 100)"),
		),
		mcp.WithString("jq_filter",
			mcp.Description("jq filter to reduce output. Always provide this. Recommended: '[.[:15] | .[] | {title, subreddit, score, trend_score, permalink}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetRisingPosts)

	// discover_communities - Find subreddits for a single topic
	tool = mcp.NewTool("discover_communities",
		mcp.WithDescription(`Find subreddits related to a single topic, sorted by subscriber count.
Use this to explore where a topic lives on Reddit.
For multiple topics at once, use get_crossover_communities instead — it's more efficient and ranks by cross-topic relevance.`),
		mcp.WithString("topic",
			mcp.Required(),
			mcp.Description("Topic to search for (e.g. 'kubernetes', 'machine learning', 'golang')"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Max communities to return (default: 10, max: 25)"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolDiscoverCommunities)

	// get_crossover_communities - Subreddits covering multiple topics
	tool = mcp.NewTool("get_crossover_communities",
		mcp.WithDescription(`Find subreddits that appear across multiple topics at once.
A subreddit with hit_count=3 covers 3 of your topics — it's the most valuable one to monitor.
Results sorted by hit_count desc, then subscribers desc.
Ideal first step when researching multiple topics: call this, then feed the top subreddits into get_trending_posts.
Recommended flow: get_crossover_communities → get_trending_posts with jq_filter.`),
		mcp.WithArray("topics",
			mcp.Required(),
			mcp.Description("List of topics to cross-reference (e.g. ['kubernetes', 'devops', 'golang', 'AI'])"),
		),
		mcp.WithNumber("limit_per_topic",
			mcp.Description("Communities to search per topic (default: 10). Keep at 5-10."),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetCrossoverCommunities)

	// get_subreddit_info - Info and stats about a subreddit
	tool = mcp.NewTool("get_subreddit_info",
		mcp.WithDescription(`Get stats about a subreddit: subscribers, active users right now, description, NSFW flag.
Use this to validate a community before committing API calls to it.
Active users (active_user_count) is the best signal of real-time activity.`),
		mcp.WithString("subreddit",
			mcp.Required(),
			mcp.Description("Subreddit name (without r/)"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetSubredditInfo)

	// get_topic_trends - Search Reddit globally for multiple topics
	tool = mcp.NewTool("get_topic_trends",
		mcp.WithDescription(`Search Reddit globally for multiple topics at once and return top posts per topic sorted by trend_score.
This is the fastest way to get a broad overview of what's being discussed about several subjects.
Always use jq_filter to avoid huge output. Recommended: '[.[] | {topic, posts: [.posts[:3] | .[] | {title, score, trend_score, subreddit}]}]'
For higher quality results (real community context), use get_crossover_communities + get_trending_posts instead.`),
		mcp.WithArray("topics",
			mcp.Required(),
			mcp.Description("List of topics to search globally (e.g. ['kubernetes', 'golang', 'AI agents'])"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range: hour, day, week, month, year, all (default: week)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Posts per topic (default: 10, max: 100). Keep at 5-10 to save tokens."),
		),
		mcp.WithString("jq_filter",
			mcp.Description("jq filter to reduce output. Always provide this. Recommended: '[.[] | {topic, posts: [.posts[:3] | .[] | {title, score, trend_score, subreddit}]}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetTopicTrends)

	// analyze_sentiment_trend - Compare topic momentum over time
	tool = mcp.NewTool("analyze_sentiment_trend",
		mcp.WithDescription(`Compare how a topic is performing now (last 24h) vs the last N days.
Returns avg_score, avg_comments, avg_trend_score for both periods, top 5 posts each, and a trending_up boolean.
Use this to decide if now is a good moment to post about a topic, or to detect if interest is rising or fading.
Output is already compact — no jq_filter needed.`),
		mcp.WithString("topic",
			mcp.Required(),
			mcp.Description("Topic to analyze"),
		),
		mcp.WithNumber("days",
			mcp.Description("Days to compare against (default: 7). Use 30 for monthly trend."),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolAnalyzeSentimentTrend)
}
