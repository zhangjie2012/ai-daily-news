package fetcher

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// --- Hugging Face Daily Papers ---

type HuggingFaceFetcher struct {
	client *http.Client
}

func NewHuggingFaceFetcher() *HuggingFaceFetcher {
	return &HuggingFaceFetcher{client: &http.Client{Timeout: 30 * time.Second}}
}

func (f *HuggingFaceFetcher) Name() string {
	return "Hugging Face Daily Papers"
}

func (f *HuggingFaceFetcher) Fetch() ([]NewsItem, error) {
	// Hugging Face Daily Papers API
	url := "https://huggingface.co/api/daily_papers"
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var papers []struct {
		Title       string `json:"title"`
		PaperId     string `json:"paper_id"`
		PublishedAt string `json:"publishedAt"`
		Summary     string `json:"summary"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&papers); err != nil {
		return nil, err
	}

	var items []NewsItem
	// 取前 5 篇热门论文
	limit := 5
	if len(papers) < limit {
		limit = len(papers)
	}

	for _, p := range papers[:limit] {
		link := fmt.Sprintf("https://huggingface.co/papers/%s", p.PaperId)
		items = append(items, NewsItem{
			Title:       "📄 " + p.Title,
			Source:      "Hugging Face",
			URL:         link,
			Summary:     p.Summary,
			PublishedAt: time.Now(), // API 可能不返回具体时间，默认当天
			Category:    "学术论文",
		})
	}

	return items, nil
}

// --- arXiv ---

type ArxivFetcher struct {
	client *http.Client
}

func NewArxivFetcher() *ArxivFetcher {
	return &ArxivFetcher{client: &http.Client{Timeout: 30 * time.Second}}
}

func (f *ArxivFetcher) Name() string {
	return "arXiv AI"
}

func (f *ArxivFetcher) Fetch() ([]NewsItem, error) {
	// 查询 cs.AI (人工智能), cs.CL (计算语言学), cs.LG (机器学习)
	apiURL := "http://export.arxiv.org/api/query?search_query=cat:cs.AI+OR+cat:cs.CL+OR+cat:cs.LG&sortBy=submittedDate&sortOrder=descending&max_results=5"
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", apiURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var feed struct {
		Entry []struct {
			Title     string `xml:"title"`
			Id        string `xml:"id"`
			Summary   string `xml:"summary"`
			Published string `xml:"published"`
		} `xml:"entry"`
	}

	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	var items []NewsItem
	for _, entry := range feed.Entry {
		// 清理标题中的换行
		title := strings.ReplaceAll(strings.TrimSpace(entry.Title), "\n", " ")
		summary := strings.ReplaceAll(strings.TrimSpace(entry.Summary), "\n", " ")
		if len(summary) > 200 {
			summary = summary[:200] + "..."
		}
		
		pubTime, _ := time.Parse(time.RFC3339, entry.Published)

		items = append(items, NewsItem{
			Title:       "🎓 " + title,
			Source:      "arXiv",
			URL:         entry.Id,
			Summary:     summary,
			PublishedAt: pubTime,
			Category:    "学术论文",
		})
	}

	return items, nil
}

// --- RSS (Blog) ---

type RSSFetcher struct {
	client  *http.Client
	sources map[string]string
}

func NewRSSFetcher() *RSSFetcher {
	return &RSSFetcher{
		client: &http.Client{Timeout: 30 * time.Second},
		sources: map[string]string{
			"OpenAI Blog":             "https://openai.com/index.xml",
			"Anthropic Blog":          "https://www.anthropic.com/index.xml",
			"Google DeepMind":         "https://deepmind.google/blog/rss.xml",
			"Microsoft Research":      "https://www.microsoft.com/en-us/research/feed/",
			"Andrej Karpathy":         "https://karpathy.github.io/feed.xml",
			"The Verge (AI)":          "https://www.theverge.com/rss/ai-artificial-intelligence/index.xml",
		},
	}
}

func (f *RSSFetcher) Name() string {
	return "Tech Blogs (RSS)"
}

func (f *RSSFetcher) Fetch() ([]NewsItem, error) {
	var items []NewsItem
	
	for sourceName, rssURL := range f.sources {
		fetchedItems, err := f.fetchRSS(sourceName, rssURL)
		if err != nil {
			fmt.Printf("RSS error for %s: %v\n", sourceName, err)
			continue
		}
		items = append(items, fetchedItems...)
	}
	
	return items, nil
}

func (f *RSSFetcher) fetchRSS(sourceName, rssURL string) ([]NewsItem, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "GET", rssURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; AI-Daily-News-Bot/1.0)")

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// 简单的 XML 解析结构，兼容 RSS 和 Atom
	type Item struct {
		Title       string `xml:"title"`
		Link        string `xml:"link"` // RSS 2.0
		LinkAtom    struct {
			Href string `xml:"href,attr"`
		} `xml:"link"` // Atom
		Description string `xml:"description"`
		PubDate     string `xml:"pubDate"`
		Updated     string `xml:"updated"` // Atom
	}

	type Channel struct {
		Items []Item `xml:"item"` // RSS
		Entries []Item `xml:"entry"` // Atom
	}
	
	type Feed struct {
		Channel Channel `xml:"channel"` // RSS
		Entries []Item  `xml:"entry"`   // Atom (direct children)
	}

	var feed Feed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	// 合并 RSS 和 Atom 的条目
	allEntries := append(feed.Channel.Items, feed.Channel.Entries...)
	allEntries = append(allEntries, feed.Entries...)

	var items []NewsItem
	// 每个源只取最新的 2 条，避免刷屏
	limit := 2
	count := 0
	
	for _, entry := range allEntries {
		if count >= limit {
			break
		}

		link := entry.Link
		if link == "" {
			link = entry.LinkAtom.Href
		}
		
		// 简单的日期解析尝试
		pubTime := time.Now()
		dateStr := entry.PubDate
		if dateStr == "" {
			dateStr = entry.Updated
		}
		if dateStr != "" {
			// 尝试常见格式
			formats := []string{
				time.RFC1123,
				time.RFC1123Z,
				time.RFC3339,
				"Mon, 02 Jan 2006 15:04:05 -0700",
			}
			for _, format := range formats {
				if t, err := time.Parse(format, dateStr); err == nil {
					pubTime = t
					break
				}
			}
		}

		// 忽略太旧的新闻（超过 3 天）
		if time.Since(pubTime) > 72*time.Hour {
			continue
		}

		items = append(items, NewsItem{
			Title:       "📰 " + strings.TrimSpace(entry.Title),
			Source:      sourceName,
			URL:         strings.TrimSpace(link),
			Summary:     "", // RSS summary 经常包含 HTML，简单起见先置空，让用户点进去看
			PublishedAt: pubTime,
			Category:    "官方博客",
		})
		count++
	}

	return items, nil
}
