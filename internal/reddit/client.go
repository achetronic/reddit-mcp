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

package reddit

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"time"
)

const (
	baseURL       = "https://www.reddit.com"
	defaultUA     = "MCP-TrendBot/1.0 (by /u/achetronic)"
	requestTimeout = 15 * time.Second
)

// Client is a Reddit API client (no auth, public JSON endpoints)
type Client struct {
	http      *http.Client
	userAgent string
}

// NewClient creates a new Reddit client
func NewClient(userAgent string) *Client {
	if userAgent == "" {
		userAgent = defaultUA
	}
	return &Client{
		http:      &http.Client{Timeout: requestTimeout},
		userAgent: userAgent,
	}
}

// Post represents a Reddit post with relevant fields
type Post struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Subreddit   string  `json:"subreddit"`
	Author      string  `json:"author"`
	Score       int     `json:"score"`
	NumComments int     `json:"num_comments"`
	UpvoteRatio float64 `json:"upvote_ratio"`
	URL         string  `json:"url"`
	Permalink   string  `json:"permalink"`
	CreatedUTC  float64 `json:"created_utc"`
	TrendScore  float64 `json:"trend_score,omitempty"`
	IsSelf      bool    `json:"is_self"`
}

// SubredditInfo represents info about a subreddit
type SubredditInfo struct {
	Name            string `json:"name"`
	DisplayName     string `json:"display_name"`
	Title           string `json:"title"`
	Description     string `json:"public_description"`
	Subscribers     int    `json:"subscribers"`
	ActiveUsers     int    `json:"active_user_count"`
	Over18          bool   `json:"over18"`
	URL             string `json:"url"`
}

// CommunityHit is a subreddit found during topic discovery
type CommunityHit struct {
	Name        string  `json:"name"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Subscribers int     `json:"subscribers"`
	ActiveUsers int     `json:"active_user_count"`
	HitCount    int     `json:"hit_count"`
	Topics      []string `json:"topics,omitempty"`
}

// doGet performs a GET request to the Reddit JSON API
func (c *Client) doGet(endpoint string, params url.Values) ([]byte, error) {
	reqURL := baseURL + endpoint
	if len(params) > 0 {
		reqURL += "?" + params.Encode()
	}

	req, err := http.NewRequest(http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("reddit API error (status %d): %s", resp.StatusCode, string(body))
	}

	return body, nil
}

// parsePosts extracts posts from a Reddit listing JSON response
func parsePosts(data []byte) ([]Post, error) {
	var listing struct {
		Data struct {
			Children []struct {
				Data Post `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(data, &listing); err != nil {
		return nil, fmt.Errorf("failed to parse listing: %w", err)
	}

	posts := make([]Post, 0, len(listing.Data.Children))
	for _, child := range listing.Data.Children {
		posts = append(posts, child.Data)
	}
	return posts, nil
}

// calcTrendScore calculates a custom trend score based on velocity and engagement
// trend_score = velocity * (1 + engagement)
// velocity = score / age_hours
// engagement = num_comments / max(score, 1)
func calcTrendScore(p *Post) float64 {
	ageHours := time.Since(time.Unix(int64(p.CreatedUTC), 0)).Hours()
	if ageHours < 1 {
		ageHours = 1
	}
	velocity := float64(p.Score) / ageHours
	engagement := float64(p.NumComments) / math.Max(float64(p.Score), 1)
	return velocity * (1 + engagement)
}

// GetSubredditPosts gets posts from a subreddit by sort (hot, new, top, rising)
func (c *Client) GetSubredditPosts(subreddit, sort, timeRange string, limit int) ([]Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	params := url.Values{
		"limit": {fmt.Sprintf("%d", limit)},
	}
	if sort == "top" && timeRange != "" {
		params.Set("t", timeRange)
	}

	endpoint := fmt.Sprintf("/r/%s/%s.json", subreddit, sort)
	body, err := c.doGet(endpoint, params)
	if err != nil {
		return nil, err
	}

	posts, err := parsePosts(body)
	if err != nil {
		return nil, err
	}

	for i := range posts {
		posts[i].TrendScore = calcTrendScore(&posts[i])
	}
	return posts, nil
}

// SearchPosts searches for posts globally or within a subreddit
func (c *Client) SearchPosts(query, subreddit, sortBy, timeRange string, limit int) ([]Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}

	params := url.Values{
		"q":     {query},
		"sort":  {sortBy},
		"limit": {fmt.Sprintf("%d", limit)},
	}
	if timeRange != "" {
		params.Set("t", timeRange)
	}

	var endpoint string
	if subreddit != "" {
		endpoint = fmt.Sprintf("/r/%s/search.json", subreddit)
		params.Set("restrict_sr", "1")
	} else {
		endpoint = "/search.json"
	}

	body, err := c.doGet(endpoint, params)
	if err != nil {
		return nil, err
	}

	posts, err := parsePosts(body)
	if err != nil {
		return nil, err
	}

	for i := range posts {
		posts[i].TrendScore = calcTrendScore(&posts[i])
	}
	return posts, nil
}

// GetRisingPosts gets posts rising in r/all
func (c *Client) GetRisingPosts(limit int) ([]Post, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	params := url.Values{"limit": {fmt.Sprintf("%d", limit)}}
	body, err := c.doGet("/r/all/rising.json", params)
	if err != nil {
		return nil, err
	}

	posts, err := parsePosts(body)
	if err != nil {
		return nil, err
	}
	for i := range posts {
		posts[i].TrendScore = calcTrendScore(&posts[i])
	}
	return posts, nil
}

// GetSubredditInfo returns info and stats about a subreddit
func (c *Client) GetSubredditInfo(subreddit string) (*SubredditInfo, error) {
	body, err := c.doGet(fmt.Sprintf("/r/%s/about.json", subreddit), nil)
	if err != nil {
		return nil, err
	}

	var resp struct {
		Data SubredditInfo `json:"data"`
	}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse subreddit info: %w", err)
	}
	return &resp.Data, nil
}

// DiscoverCommunities finds subreddits related to a topic
func (c *Client) DiscoverCommunities(topic string, limit int) ([]SubredditInfo, error) {
	if limit <= 0 || limit > 25 {
		limit = 10
	}
	params := url.Values{
		"q":     {topic},
		"limit": {fmt.Sprintf("%d", limit)},
	}
	body, err := c.doGet("/subreddits/search.json", params)
	if err != nil {
		return nil, err
	}

	var listing struct {
		Data struct {
			Children []struct {
				Data SubredditInfo `json:"data"`
			} `json:"children"`
		} `json:"data"`
	}
	if err := json.Unmarshal(body, &listing); err != nil {
		return nil, fmt.Errorf("failed to parse subreddits: %w", err)
	}

	result := make([]SubredditInfo, 0, len(listing.Data.Children))
	for _, child := range listing.Data.Children {
		if !child.Data.Over18 {
			result = append(result, child.Data)
		}
	}
	return result, nil
}

// GetCrossoverCommunities finds subreddits that appear across multiple topics,
// ranked by how many topics they cover
func (c *Client) GetCrossoverCommunities(topics []string, limitPerTopic int) ([]CommunityHit, error) {
	if limitPerTopic <= 0 {
		limitPerTopic = 10
	}

	hits := make(map[string]*CommunityHit)

	for _, topic := range topics {
		subs, err := c.DiscoverCommunities(topic, limitPerTopic)
		if err != nil {
			continue
		}
		for _, sub := range subs {
			key := sub.DisplayName
			if key == "" {
				key = sub.Name
			}
			if existing, ok := hits[key]; ok {
				existing.HitCount++
				existing.Topics = append(existing.Topics, topic)
			} else {
				hits[key] = &CommunityHit{
					Name:        sub.Name,
					Title:       sub.Title,
					Description: sub.Description,
					Subscribers: sub.Subscribers,
					ActiveUsers: sub.ActiveUsers,
					HitCount:    1,
					Topics:      []string{topic},
				}
			}
		}
	}

	result := make([]CommunityHit, 0, len(hits))
	for _, h := range hits {
		result = append(result, *h)
	}

	// Sort by hit count desc, then subscribers desc
	sort.Slice(result, func(i, j int) bool {
		if result[i].HitCount != result[j].HitCount {
			return result[i].HitCount > result[j].HitCount
		}
		return result[i].Subscribers > result[j].Subscribers
	})

	return result, nil
}

// TrendingByTopicResult holds the result for a single topic in GetTrendingByTopic
type TrendingByTopicResult struct {
	Topic string `json:"topic"`
	Posts []Post `json:"posts"`
}

// GetTrendingByTopic searches Reddit globally for multiple topics and returns
// top posts per topic sorted by trend score
func (c *Client) GetTrendingByTopic(topics []string, timeRange string, limit int) ([]TrendingByTopicResult, error) {
	if limit <= 0 || limit > 100 {
		limit = 25
	}
	var results []TrendingByTopicResult
	for _, topic := range topics {
		posts, err := c.SearchPosts(topic, "", "top", timeRange, limit)
		if err != nil {
			continue
		}
		sort.Slice(posts, func(i, j int) bool {
			return posts[i].TrendScore > posts[j].TrendScore
		})
		results = append(results, TrendingByTopicResult{
			Topic: topic,
			Posts: posts,
		})
	}
	return results, nil
}

// SentimentSnapshot holds a snapshot of posts at a point in time
type SentimentSnapshot struct {
	Period      string  `json:"period"`
	PostCount   int     `json:"post_count"`
	AvgScore    float64 `json:"avg_score"`
	AvgComments float64 `json:"avg_comments"`
	AvgTrend    float64 `json:"avg_trend_score"`
	TopPosts    []Post  `json:"top_posts"`
}

// SentimentCompare holds the comparison between recent and historical posts
type SentimentCompare struct {
	Topic      string            `json:"topic"`
	Recent     SentimentSnapshot `json:"recent"`
	Historical SentimentSnapshot `json:"historical"`
	Trending   bool              `json:"trending_up"`
}

// AnalyzeSentimentTrend compares top posts from the last 24h vs the previous N days
func (c *Client) AnalyzeSentimentTrend(topic string, days int) (*SentimentCompare, error) {
	if days <= 0 {
		days = 7
	}

	recentPosts, err := c.SearchPosts(topic, "", "top", "day", 25)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent posts: %w", err)
	}

	timeRange := "week"
	if days > 7 {
		timeRange = "month"
	}
	historicalPosts, err := c.SearchPosts(topic, "", "top", timeRange, 25)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch historical posts: %w", err)
	}

	return &SentimentCompare{
		Topic:      topic,
		Recent:     snapshotFromPosts("last_24h", recentPosts),
		Historical: snapshotFromPosts(fmt.Sprintf("last_%dd", days), historicalPosts),
		Trending:   avgTrendScore(recentPosts) > avgTrendScore(historicalPosts),
	}, nil
}

func snapshotFromPosts(period string, posts []Post) SentimentSnapshot {
	if len(posts) == 0 {
		return SentimentSnapshot{Period: period}
	}
	var totalScore, totalComments, totalTrend float64
	for _, p := range posts {
		totalScore += float64(p.Score)
		totalComments += float64(p.NumComments)
		totalTrend += p.TrendScore
	}
	n := float64(len(posts))
	top := posts
	if len(top) > 5 {
		top = top[:5]
	}
	return SentimentSnapshot{
		Period:      period,
		PostCount:   len(posts),
		AvgScore:    totalScore / n,
		AvgComments: totalComments / n,
		AvgTrend:    totalTrend / n,
		TopPosts:    top,
	}
}

func avgTrendScore(posts []Post) float64 {
	if len(posts) == 0 {
		return 0
	}
	var total float64
	for _, p := range posts {
		total += p.TrendScore
	}
	return total / float64(len(posts))
}
