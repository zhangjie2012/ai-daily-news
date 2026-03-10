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
			// AI 官方博客
			"OpenAI Blog":        "https://openai.com/index.xml",
			"Anthropic Blog":     "https://www.anthropic.com/index.xml",
			"Google DeepMind":    "https://deepmind.google/blog/rss.xml",
			"Hugging Face Blog":  "https://huggingface.co/blog/feed.xml",
			"Microsoft Research": "https://www.microsoft.com/en-us/research/feed/",
			"Andrej Karpathy":    "https://karpathy.github.io/feed.xml",

			// 综合科技 + AI
			"MIT Tech Review": "https://www.technologyreview.com/feed/",
			"VentureBeat AI":  "https://venturebeat.com/category/ai/feed/",
			"TechCrunch AI":   "https://techcrunch.com/category/artificial-intelligence/feed/",
			"Ars Technica AI": "https://arstechnica.com/ai/feed/",
			"The Verge AI":    "https://www.theverge.com/rss/ai-artificial-intelligence/index.xml",

			// 聚合社区
			"Techmeme": "https://www.techmeme.com/feed.xml",

			// 中国 AI 公司博客
			"字节跳动技术团队":  "https://juejin.cn/rss/byte-tech?sort=newest",
			"腾讯 AI Lab": "https://ai.tencent.com/ailab/rss.xml",
			"阿里巴巴技术":    "https://developer.aliyun.com/rss/article",

			// 中国科技媒体
		"机器之心":     "https://www.jiqizhixin.com/rss",
		"量子位":      "https://www.qbitai.com/feed",
		"新智元":      "https://www.jiqizhixin.com/rss",
		"InfoQ 中文": "https://www.infoq.cn/feed",
		"36氪 AI":   "https://36kr.com/feed",

		// 互联网资讯（融资、市值、商业）
		"TechCrunch": "https://techcrunch.com/feed/",
		"The Information": "https://www.theinformation.com/feed",
		"PitchBook": "https://pitchbook.com/news/feed",
		"Crunchbase News": "https://news.crunchbase.com/feed/",
		"CB Insights": "https://www.cbinsights.com/research/feed",
		"Bloomberg Technology": "https://feeds.bloomberg.com/bloomberg/markets.rss",
		"Reuters Technology": "https://www.reutersagency.com/feed/?taxonomy=markets&post_type=reuters-best",
		"Financial Times": "https://www.ft.com/technology?format=rss",
		"CNBC": "https://www.cnbc.com/id/19854910/device/rss/rss.html",
		"MarketWatch": "https://feeds.marketwatch.com/marketwatch/topstories/",

		// 中国商业资讯
		"虎嗅": "https://www.huxiu.com/rss/0.xml",
		"钛媒体": "https://www.tmtpost.com/feed",
		"IT桔子": "https://www.itjuzi.com/feed",
		"投资界": "https://www.pedaily.cn/rss/",
		"创业邦": "https://www.cyzone.cn/feed",
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

	// 1. 定义 RSS 结构
	type RItem struct {
		Title   string `xml:"title"`
		Link    string `xml:"link"`
		PubDate string `xml:"pubDate"`
	}

	// 2. 定义 Atom 结构
	type AEntry struct {
		Title string `xml:"title"`
		Link  struct {
			Href string `xml:"href,attr"`
		} `xml:"link"`
		Updated string `xml:"updated"`
	}

	// 3. 通用 Feed
	type UniversalFeed struct {
		// RSS 2.0
		Channel struct {
			Items []RItem `xml:"item"`
		} `xml:"channel"`
		// Atom
		Entries []AEntry `xml:"entry"`
	}

	var feed UniversalFeed
	if err := xml.NewDecoder(resp.Body).Decode(&feed); err != nil {
		return nil, err
	}

	var items []NewsItem
	limit := 2
	count := 0

	processItem := func(title, link, dateStr string) {
		if count >= limit {
			return
		}

		pubTime := time.Now()
		if dateStr != "" {
			formats := []string{
				time.RFC1123,
				time.RFC1123Z,
				time.RFC3339,
				"Mon, 02 Jan 2006 15:04:05 -0700",
				"Mon, 02 Jan 2006 15:04:05 GMT",
			}
			for _, format := range formats {
				if t, err := time.Parse(format, dateStr); err == nil {
					pubTime = t
					break
				}
			}
		}

		if time.Since(pubTime) > 72*time.Hour {
			return
		}

		items = append(items, NewsItem{
			Title:       "📰 " + strings.TrimSpace(title),
			Source:      sourceName,
			URL:         strings.TrimSpace(link),
			PublishedAt: pubTime,
			Category:    "官方博客",
		})
		count++
	}

	for _, item := range feed.Channel.Items {
		processItem(item.Title, item.Link, item.PubDate)
	}
	for _, entry := range feed.Entries {
		processItem(entry.Title, entry.Link.Href, entry.Updated)
	}

	return items, nil
}
