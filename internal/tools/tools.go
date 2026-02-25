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
		mcp.WithDescription("Get trending posts from one or more subreddits, sorted by custom trend score (velocity × engagement). Results include trend_score so you can rank across subreddits."),
		mcp.WithArray("subreddits",
			mcp.Required(),
			mcp.Description("List of subreddit names to fetch (e.g. ['kubernetes', 'golang', 'devops'])"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range for top posts: hour, day, week, month, year, all (default: day)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Posts per subreddit (default: 10, max: 100)"),
		),
		mcp.WithString("jq_filter",
			mcp.Description("Optional jq filter to reduce output. E.g. '[.[] | {title, score, subreddit, trend_score}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetTrendingPosts)

	// get_subreddit_pulse - Current state of a community
	tool = mcp.NewTool("get_subreddit_pulse",
		mcp.WithDescription("Get the current pulse of a subreddit: top posts by hot/new/top/rising with trend scores. Useful for understanding what's active in a specific community right now."),
		mcp.WithString("subreddit",
			mcp.Required(),
			mcp.Description("Subreddit name (without r/)"),
		),
		mcp.WithString("sort",
			mcp.Description("Sort order: hot, new, top, rising (default: hot)"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range when sort=top: hour, day, week, month, year, all (default: day)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of posts (default: 25, max: 100)"),
		),
		mcp.WithString("jq_filter",
			mcp.Description("Optional jq filter to reduce output. E.g. '[.[] | {title, score, num_comments, url}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetSubredditPulse)

	// search_posts - Search posts globally or within a subreddit
	tool = mcp.NewTool("search_posts",
		mcp.WithDescription("Search Reddit posts by keyword, globally or within a specific subreddit. Returns posts sorted by relevance or hot, with trend scores."),
		mcp.WithString("query",
			mcp.Required(),
			mcp.Description("Search query"),
		),
		mcp.WithString("subreddit",
			mcp.Description("Optional: restrict search to this subreddit"),
		),
		mcp.WithString("sort",
			mcp.Description("Sort: relevance, hot, top, new, comments (default: hot)"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range: hour, day, week, month, year, all (default: week)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Number of results (default: 25, max: 100)"),
		),
		mcp.WithString("jq_filter",
			mcp.Description("Optional jq filter. E.g. '[.[] | {title, subreddit, score, url}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolSearchPosts)

	// get_rising_posts - Posts rising in r/all right now
	tool = mcp.NewTool("get_rising_posts",
		mcp.WithDescription("Get posts currently rising across all of Reddit (r/all/rising). These are posts gaining momentum fast — good signal for what's about to trend."),
		mcp.WithNumber("limit",
			mcp.Description("Number of posts (default: 25, max: 100)"),
		),
		mcp.WithString("jq_filter",
			mcp.Description("Optional jq filter. E.g. '[.[] | {title, subreddit, score, url}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetRisingPosts)

	// discover_communities - Find subreddits for a topic
	tool = mcp.NewTool("discover_communities",
		mcp.WithDescription("Find the most relevant subreddits for a given topic. Returns communities sorted by subscriber count with description for relevance validation."),
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
		mcp.WithDescription("Find subreddits that cover multiple topics at once. A subreddit appearing for 3 topics is more valuable than one appearing for 1. Results are ranked by topic overlap count, then subscriber count."),
		mcp.WithArray("topics",
			mcp.Required(),
			mcp.Description("List of topics to cross-reference (e.g. ['kubernetes', 'devops', 'golang'])"),
		),
		mcp.WithNumber("limit_per_topic",
			mcp.Description("Communities to search per topic (default: 10)"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetCrossoverCommunities)

	// get_subreddit_info - Info and stats about a subreddit
	tool = mcp.NewTool("get_subreddit_info",
		mcp.WithDescription("Get detailed info and stats about a specific subreddit: subscribers, active users, description, etc. Useful to validate if a community is relevant before diving in."),
		mcp.WithString("subreddit",
			mcp.Required(),
			mcp.Description("Subreddit name (without r/)"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetSubredditInfo)

	// get_topic_trends - Search Reddit globally for multiple topics
	tool = mcp.NewTool("get_topic_trends",
		mcp.WithDescription("Search Reddit globally for one or more topics and return top posts per topic sorted by trend score. Useful to quickly compare what's being said about multiple subjects across the whole platform."),
		mcp.WithArray("topics",
			mcp.Required(),
			mcp.Description("List of topics to search (e.g. ['kubernetes', 'golang', 'AI'])"),
		),
		mcp.WithString("time_range",
			mcp.Description("Time range: hour, day, week, month, year, all (default: week)"),
		),
		mcp.WithNumber("limit",
			mcp.Description("Posts per topic (default: 10, max: 100)"),
		),
		mcp.WithString("jq_filter",
			mcp.Description("Optional jq filter to reduce output. E.g. '[.[] | {topic, posts: [.posts[] | {title, score, trend_score}]}]'"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolGetTopicTrends)

	// analyze_sentiment_trend - Compare topic momentum over time
	tool = mcp.NewTool("analyze_sentiment_trend",
		mcp.WithDescription("Compare top posts about a topic from the last 24h vs the last N days. Returns avg score, comments, trend score and whether the topic is gaining or losing momentum. Good for deciding if now is a good time to post about something."),
		mcp.WithString("topic",
			mcp.Required(),
			mcp.Description("Topic to analyze"),
		),
		mcp.WithNumber("days",
			mcp.Description("Number of days to compare against (default: 7)"),
		),
	)
	tm.dependencies.McpServer.AddTool(tool, tm.HandleToolAnalyzeSentimentTrend)
}
