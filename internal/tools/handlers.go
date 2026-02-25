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
	"context"
	"sort"

	"reddit-mcp/internal/reddit"

	"github.com/mark3labs/mcp-go/mcp"
)

// HandleToolGetTrendingPosts handles the get_trending_posts tool
func (tm *ToolsManager) HandleToolGetTrendingPosts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	subreddits := getStringSlice(args, "subreddits")
	timeRange := getString(args, "time_range", "day")
	limit := getInt(args, "limit", 10)
	jqFilter := getString(args, "jq_filter", "")

	if len(subreddits) == 0 {
		return mcp.NewToolResultError("subreddits is required"), nil
	}
	if limit > 100 {
		limit = 100
	}

	// Fetch all subreddits and aggregate
	var allPosts []reddit.Post
	for _, sub := range subreddits {
		posts, err := tm.dependencies.RedditClient.GetSubredditPosts(sub, "top", timeRange, limit)
		if err != nil {
			continue
		}
		allPosts = append(allPosts, posts...)
	}

	// Sort by trend_score descending
	sort.Slice(allPosts, func(i, j int) bool {
		return allPosts[i].TrendScore > allPosts[j].TrendScore
	})

	output, err := applyJQ(allPosts, jqFilter)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(output), nil
}

// HandleToolGetSubredditPulse handles the get_subreddit_pulse tool
func (tm *ToolsManager) HandleToolGetSubredditPulse(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	subreddit := getString(args, "subreddit", "")
	sortBy := getString(args, "sort", "hot")
	timeRange := getString(args, "time_range", "day")
	limit := getInt(args, "limit", 25)
	jqFilter := getString(args, "jq_filter", "")

	if subreddit == "" {
		return mcp.NewToolResultError("subreddit is required"), nil
	}

	posts, err := tm.dependencies.RedditClient.GetSubredditPosts(subreddit, sortBy, timeRange, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, err := applyJQ(posts, jqFilter)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(output), nil
}

// HandleToolSearchPosts handles the search_posts tool
func (tm *ToolsManager) HandleToolSearchPosts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	query := getString(args, "query", "")
	subreddit := getString(args, "subreddit", "")
	sortBy := getString(args, "sort", "hot")
	timeRange := getString(args, "time_range", "week")
	limit := getInt(args, "limit", 25)
	jqFilter := getString(args, "jq_filter", "")

	if query == "" {
		return mcp.NewToolResultError("query is required"), nil
	}

	posts, err := tm.dependencies.RedditClient.SearchPosts(query, subreddit, sortBy, timeRange, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, err := applyJQ(posts, jqFilter)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(output), nil
}

// HandleToolGetRisingPosts handles the get_rising_posts tool
func (tm *ToolsManager) HandleToolGetRisingPosts(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	limit := getInt(args, "limit", 25)
	jqFilter := getString(args, "jq_filter", "")

	posts, err := tm.dependencies.RedditClient.GetRisingPosts(limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, err := applyJQ(posts, jqFilter)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(output), nil
}

// HandleToolDiscoverCommunities handles the discover_communities tool
func (tm *ToolsManager) HandleToolDiscoverCommunities(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	topic := getString(args, "topic", "")
	limit := getInt(args, "limit", 10)

	if topic == "" {
		return mcp.NewToolResultError("topic is required"), nil
	}

	communities, err := tm.dependencies.RedditClient.DiscoverCommunities(topic, limit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, err := applyJQ(communities, "")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(output), nil
}

// HandleToolGetCrossoverCommunities handles the get_crossover_communities tool
func (tm *ToolsManager) HandleToolGetCrossoverCommunities(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	topics := getStringSlice(args, "topics")
	limitPerTopic := getInt(args, "limit_per_topic", 10)

	if len(topics) == 0 {
		return mcp.NewToolResultError("topics is required"), nil
	}

	communities, err := tm.dependencies.RedditClient.GetCrossoverCommunities(topics, limitPerTopic)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, err := applyJQ(communities, "")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(output), nil
}

// HandleToolGetSubredditInfo handles the get_subreddit_info tool
func (tm *ToolsManager) HandleToolGetSubredditInfo(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	args := getArgs(request)
	subreddit := getString(args, "subreddit", "")

	if subreddit == "" {
		return mcp.NewToolResultError("subreddit is required"), nil
	}

	info, err := tm.dependencies.RedditClient.GetSubredditInfo(subreddit)
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	output, err := applyJQ(info, "")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}
	return mcp.NewToolResultText(output), nil
}
