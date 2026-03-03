package fetcher

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

type NewsItem struct {
	Title       string
	Source      string
	URL         string
	Summary     string
	PublishedAt time.Time
	Category    string
}

type Fetcher interface {
	Fetch() ([]NewsItem, error)
	Name() string
}

func FetchAllNews() ([]NewsItem, error) {
	var allNews []NewsItem
	fetchers := []Fetcher{
		NewHuggingFaceFetcher(),
		NewArxivFetcher(),
		NewRSSFetcher(),
		NewHackerNewsFetcher(),
		NewRedditFetcher(),
		NewGitHubTrendingFetcher(),
		NewProductHuntFetcher(),
		NewV2EXFetcher(),
		NewJuejinFetcher(),
	}

	for _, f := range fetchers {
		items, err := f.Fetch()
		if err != nil {
			fmt.Printf("从 %s 获取资讯失败: %v\n", f.Name(), err)
			continue
		}
		allNews = append(allNews, items...)
	}

	allNews = filterAINews(allNews)
	allNews = deduplicateNews(allNews)

	if len(allNews) > 50 {
		allNews = allNews[:50]
	}

	return allNews, nil
}

func filterAINews(items []NewsItem) []NewsItem {
	keywords := []string{
		// 英文关键词
		"AI", "artificial intelligence", "machine learning", "ML",
		"GPT", "LLM", "large language model", "ChatGPT", "Claude",
		"agent", "Agent", "AGI", "deep learning", "neural network",
		"transformer", "diffusion", "stable diffusion", "Midjourney",
		"open source", "framework", "model", "coding", "developer",
		"GitHub", "Hugging Face", "OpenAI", "Anthropic", "Google AI",
		"Meta AI", "Microsoft AI", "Copilot", "Cursor",

		// 中文关键词
		"人工智能", "机器学习", "深度学习", "神经网络",
		"大模型", "语言模型", "生成式", "AIGC",
		"智能体", "Agent", "AGI", "GPT", "LLM",
		"文心一言", "通义千问", "智谱", "月之暗面", "Kimi",
		"字节", "豆包", "腾讯", "阿里", "百度",
		"DeepSeek", "深度求索", "零一万物", "MiniMax",
		"百川智能", "智源", "商汤", "旷视",
		"开源", "模型", "算法", "训练", "推理",
	}

	var filtered []NewsItem
	for _, item := range items {
		itemLower := strings.ToLower(item.Title + " " + item.Summary)
		for _, kw := range keywords {
			if strings.Contains(itemLower, strings.ToLower(kw)) {
				filtered = append(filtered, item)
				break
			}
		}
	}
	return filtered
}

func deduplicateNews(items []NewsItem) []NewsItem {
	seen := make(map[string]bool)
	var result []NewsItem
	for _, item := range items {
		key := strings.ToLower(item.Title)
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return result
}

type HackerNewsFetcher struct {
	client *http.Client
}

func NewHackerNewsFetcher() *HackerNewsFetcher {
	return &HackerNewsFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *HackerNewsFetcher) Name() string {
	return "Hacker News"
}

func (f *HackerNewsFetcher) Fetch() ([]NewsItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://hacker-news.firebaseio.com/v0/topstories.json", nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var storyIDs []int
	if err := json.NewDecoder(resp.Body).Decode(&storyIDs); err != nil {
		return nil, err
	}

	var items []NewsItem
	for i := 0; i < 30 && i < len(storyIDs); i++ {
		storyURL := fmt.Sprintf("https://hacker-news.firebaseio.com/v0/item/%d.json", storyIDs[i])
		req, _ := http.NewRequestWithContext(ctx, "GET", storyURL, nil)
		resp, err := f.client.Do(req)
		if err != nil {
			continue
		}

		var story struct {
			Title string `json:"title"`
			URL   string `json:"url"`
			Time  int64  `json:"time"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&story); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		if story.URL == "" {
			story.URL = fmt.Sprintf("https://news.ycombinator.com/item?id=%d", storyIDs[i])
		}

		items = append(items, NewsItem{
			Title:       story.Title,
			Source:      "Hacker News",
			URL:         story.URL,
			PublishedAt: time.Unix(story.Time, 0),
			Category:    "技术社区",
		})
	}

	return items, nil
}

type V2EXFetcher struct {
	client *http.Client
}

func NewV2EXFetcher() *V2EXFetcher {
	return &V2EXFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *V2EXFetcher) Name() string {
	return "V2EX"
}

func (f *V2EXFetcher) Fetch() ([]NewsItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	nodes := []string{"ai", "python", "programmer", "share"}
	var items []NewsItem

	for _, node := range nodes {
		url := fmt.Sprintf("https://www.v2ex.com/api/topics/hot.json?node_name=%s", node)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AI-Daily-News-Bot/1.0)")

		resp, err := f.client.Do(req)
		if err != nil {
			continue
		}

		var topics []struct {
			Title string `json:"title"`
			URL   string `json:"url"`
			Node  struct {
				Name string `json:"name"`
			} `json:"node"`
			Created int64 `json:"created"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&topics); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, topic := range topics {
			items = append(items, NewsItem{
				Title:       topic.Title,
				Source:      "V2EX/" + topic.Node.Name,
				URL:         topic.URL,
				PublishedAt: time.Unix(topic.Created, 0),
				Category:    "技术社区",
			})
		}
	}

	return items, nil
}

type JuejinFetcher struct {
	client *http.Client
}

func NewJuejinFetcher() *JuejinFetcher {
	return &JuejinFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *JuejinFetcher) Name() string {
	return "掘金"
}

func (f *JuejinFetcher) Fetch() ([]NewsItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	reqBody := map[string]interface{}{
		"id_type":  2,
		"sort_type": 200,
		"cate_id":   "6809637773935377416",
		"cursor":    "0",
		"limit":     20,
	}
	jsonBody, _ := json.Marshal(reqBody)

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.juejin.cn/recommend_api/v1/article/recommend_cate_feed", bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AI-Daily-News-Bot/1.0)")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Data []struct {
			ArticleID  string `json:"article_id"`
			Title      string `json:"title"`
			AuthorName string `json:"author_user_info"`
		} `json:"data"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var items []NewsItem
	for _, article := range result.Data {
		items = append(items, NewsItem{
			Title:       article.Title,
			Source:      "掘金",
			URL:         fmt.Sprintf("https://juejin.cn/post/%s", article.ArticleID),
			PublishedAt: time.Now(),
			Category:    "技术社区",
		})
	}

	return items, nil
}

type WeixinFetcher struct {
	client *http.Client
}

func NewWeixinFetcher() *WeixinFetcher {
	return &WeixinFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *WeixinFetcher) Name() string {
	return "微信公众号"
}

func (f *WeixinFetcher) Fetch() ([]NewsItem, error) {
	return []NewsItem{}, nil
}

type RedditFetcher struct {
	client *http.Client
}

func NewRedditFetcher() *RedditFetcher {
	return &RedditFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *RedditFetcher) Name() string {
	return "Reddit"
}

func (f *RedditFetcher) Fetch() ([]NewsItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	subreddits := []string{"MachineLearning", "artificial", "OpenAI", "LocalLLaMA", "singularity"}
	var items []NewsItem

	for _, subreddit := range subreddits {
		url := fmt.Sprintf("https://www.reddit.com/r/%s/hot.json?limit=10", subreddit)
		req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			continue
		}
		req.Header.Set("User-Agent", "AI-Daily-News-Bot/1.0")

		resp, err := f.client.Do(req)
		if err != nil {
			continue
		}

		var result struct {
			Data struct {
				Children []struct {
					Data struct {
						Title     string  `json:"title"`
						URL       string  `json:"url"`
						Created   float64 `json:"created_utc"`
						Selftext  string  `json:"selftext"`
						Permalink string  `json:"permalink"`
					} `json:"data"`
				} `json:"children"`
			} `json:"data"`
		}

		if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
			resp.Body.Close()
			continue
		}
		resp.Body.Close()

		for _, child := range result.Data.Children {
			postURL := child.Data.URL
			if postURL == "" || strings.Contains(postURL, "reddit.com") {
				postURL = "https://www.reddit.com" + child.Data.Permalink
			}

			summary := child.Data.Selftext
			if len(summary) > 200 {
				summary = summary[:200] + "..."
			}

			items = append(items, NewsItem{
				Title:       child.Data.Title,
				Source:      "Reddit r/" + subreddit,
				URL:         postURL,
				Summary:     summary,
				PublishedAt: time.Unix(int64(child.Data.Created), 0),
				Category:    "社区讨论",
			})
		}
	}

	return items, nil
}

type GitHubTrendingFetcher struct {
	client *http.Client
}

func NewGitHubTrendingFetcher() *GitHubTrendingFetcher {
	return &GitHubTrendingFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *GitHubTrendingFetcher) Name() string {
	return "GitHub Trending"
}

func (f *GitHubTrendingFetcher) Fetch() ([]NewsItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.gitterapp.com/repositories?language=&since=daily", nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var trending []struct {
		Name        string `json:"name"`
		Owner       string `json:"owner"`
		Description string `json:"description"`
		URL         string `json:"url"`
		Stars       int    `json:"stars"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&trending); err != nil {
		body, _ := io.ReadAll(resp.Body)
		fmt.Printf("GitHub API 响应解析失败: %v, body: %s\n", err, string(body[:min(200, len(body))]))
		return nil, err
	}

	var items []NewsItem
	for _, repo := range trending {
		items = append(items, NewsItem{
			Title:       fmt.Sprintf("%s/%s - %s", repo.Owner, repo.Name, repo.Description),
			Source:      "GitHub Trending",
			URL:         repo.URL,
			Summary:     fmt.Sprintf("⭐ %d stars - %s", repo.Stars, repo.Description),
			PublishedAt: time.Now(),
			Category:    "开源项目",
		})
	}

	return items, nil
}

type ProductHuntFetcher struct {
	client *http.Client
}

func NewProductHuntFetcher() *ProductHuntFetcher {
	return &ProductHuntFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *ProductHuntFetcher) Name() string {
	return "Product Hunt"
}

func (f *ProductHuntFetcher) Fetch() ([]NewsItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", "https://www.producthunt.com/v1/posts/all", nil)
	if err != nil {
		return nil, err
	}

	apiKey := os.Getenv("PRODUCTHUNT_API_KEY")
	if apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Product Hunt API 返回状态码: %d", resp.StatusCode)
	}

	var result struct {
		Posts []struct {
			Name      string `json:"name"`
			Tagline   string `json:"tagline"`
			URL       string `json:"redirect_url"`
			CreatedAt string `json:"created_at"`
		} `json:"posts"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var items []NewsItem
	for _, post := range result.Posts {
		items = append(items, NewsItem{
			Title:       post.Name + " - " + post.Tagline,
			Source:      "Product Hunt",
			URL:         post.URL,
			Summary:     post.Tagline,
			PublishedAt: time.Now(),
			Category:    "产品发布",
		})
	}

	return items, nil
}

type TechCrunchFetcher struct {
	client *http.Client
}

func NewTechCrunchFetcher() *TechCrunchFetcher {
	return &TechCrunchFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
	}
}

func (f *TechCrunchFetcher) Name() string {
	return "TechCrunch"
}

func (f *TechCrunchFetcher) Fetch() ([]NewsItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	apiKey := os.Getenv("NEWSAPI_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("未设置 NEWSAPI_KEY 环境变量")
	}

	u, _ := url.Parse("https://newsapi.org/v2/everything")
	q := u.Query()
	q.Set("q", "AI OR \"artificial intelligence\" OR \"machine learning\" OR LLM OR GPT")
	q.Set("sources", "techcrunch")
	q.Set("sortBy", "publishedAt")
	q.Set("pageSize", "20")
	q.Set("apiKey", apiKey)
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, "GET", u.String(), nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var result struct {
		Articles []struct {
			Title       string    `json:"title"`
			URL         string    `json:"url"`
			Description string    `json:"description"`
			PublishedAt time.Time `json:"publishedAt"`
		} `json:"articles"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	var items []NewsItem
	for _, article := range result.Articles {
		items = append(items, NewsItem{
			Title:       article.Title,
			Source:      "TechCrunch",
			URL:         article.URL,
			Summary:     article.Description,
			PublishedAt: article.PublishedAt,
			Category:    "科技新闻",
		})
	}

	return items, nil
}
