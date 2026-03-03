package generator

import (
	"bufio"
	"fmt"
	"os"
	"sort"
	"strings"
	"text/template"
	"time"

	"ai-daily-news/fetcher"
)

const reportTemplate = `# AI 资讯日报 - {{.Date}}

> 生成时间: {{.GeneratedAt}}

---

{{if .Briefing}}
## 📋 今日简报

{{.Briefing}}

---

{{end}}
{{range .Categories}}
## {{.Name}}

{{range .Items}}
### [{{.Title}}]({{.URL}})

- **来源**: {{.Source}}
- **分类**: {{.Category}}
{{if .Summary}}
- **简介**: {{.Summary}}
{{end}}

---
{{end}}
{{end}}

---

## 📊 今日统计

- 总计资讯数: {{.TotalCount}}
- 数据来源: Hacker News, Reddit, GitHub Trending, Product Hunt

---

*本日报由 AI Daily News 自动生成*
`

type CategoryGroup struct {
	Name  string
	Items []fetcher.NewsItem
}

type ReportData struct {
	Date        string
	GeneratedAt string
	Briefing    string
	Categories  []CategoryGroup
	TotalCount  int
}

func GenerateDailyReport(filename, date string, items []fetcher.NewsItem, briefing string) error {
	categoryMap := make(map[string][]fetcher.NewsItem)
	for _, item := range items {
		categoryMap[item.Category] = append(categoryMap[item.Category], item)
	}

	var categories []CategoryGroup
	for name, catItems := range categoryMap {
		sort.Slice(catItems, func(i, j int) bool {
			return catItems[i].PublishedAt.After(catItems[i].PublishedAt)
		})
		categories = append(categories, CategoryGroup{
			Name:  name,
			Items: catItems,
		})
	}

	sort.Slice(categories, func(i, j int) bool {
		return len(categories[i].Items) > len(categories[j].Items)
	})

	data := ReportData{
		Date:        date,
		GeneratedAt: time.Now().Format("2006-01-02 15:04:05"),
		Briefing:    briefing,
		Categories:  categories,
		TotalCount:  len(items),
	}

	tmpl, err := template.New("report").Parse(reportTemplate)
	if err != nil {
		return fmt.Errorf("解析模板失败: %v", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("创建文件失败: %v", err)
	}
	defer file.Close()

	if err := tmpl.Execute(file, data); err != nil {
		return fmt.Errorf("生成报告失败: %v", err)
	}

	return nil
}

func UpdateReadme(date, filename string) error {
	return UpdateReadmeIndex()
}

func generateReadmeHeader() string {
	return `# AI 资讯日报

> 自动追踪全球 AI 领域最新动态，每日更新。
> 内容涵盖：新模型、Agent、编程能力、开源项目等。

## 📅 日报列表

`
}

func CleanupOldReports(maxDays int) error {
	dailyDir := "daily"
	entries, err := os.ReadDir(dailyDir)
	if err != nil {
		return err
	}

	cutoff := time.Now().AddDate(0, 0, -maxDays)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if !strings.HasSuffix(name, ".md") {
			continue
		}

		dateStr := strings.TrimSuffix(name, ".md")
		fileDate, err := time.Parse("2006-01-02", dateStr)
		if err != nil {
			continue
		}

		if fileDate.Before(cutoff) {
			filePath := dailyDir + "/" + name
			if err := os.Remove(filePath); err != nil {
				fmt.Printf("删除旧报告失败 %s: %v\n", filePath, err)
			} else {
				fmt.Printf("已删除旧报告: %s\n", filePath)
			}
		}
	}

	return nil
}

func UpdateReadmeIndex() error {
	dailyDir := "daily"
	entries, err := os.ReadDir(dailyDir)
	if err != nil {
		return err
	}

	var files []string
	for _, entry := range entries {
		if !entry.IsDir() && strings.HasSuffix(entry.Name(), ".md") {
			files = append(files, entry.Name())
		}
	}

	sort.Sort(sort.Reverse(sort.StringSlice(files)))

	var sb strings.Builder
	sb.WriteString(generateReadmeHeader())

	// 生成日报列表
	for _, file := range files {
		dateStr := strings.TrimSuffix(file, ".md")
		sb.WriteString(fmt.Sprintf("- [%s](daily/%s)\n", dateStr, file))
	}

	sb.WriteString(generateReadmeFooter())

	return os.WriteFile("README.md", []byte(sb.String()), 0644)
}

func generateReadmeFooter() string {
	return `
## 🌐 资讯来源

本项目聚合了全球最权威的 AI 媒体、实验室博客和技术社区，力求提供最全最新的 AI 资讯：

### 🗞️ 综合科技 + AI（国外）
- **[MIT Technology Review](https://www.technologyreview.com/)**: 深度科技报道与分析
- **[VentureBeat AI](https://venturebeat.com/category/ai/)**: 企业级 AI 应用与趋势
- **[TechCrunch AI](https://techcrunch.com/category/artificial-intelligence/)**: 创业公司与投融资动态
- **[The Information](https://www.theinformation.com/)**: 独家深度科技新闻（部分内容）
- **[Ars Technica AI](https://arstechnica.com/ai/)**: 深度技术评论
- **[The Verge AI](https://www.theverge.com/ai-artificial-intelligence)**: 消费级 AI 产品新闻

### 🏛️ AI 官方博客（国外）
- **[OpenAI Blog](https://openai.com/blog)**: GPT 系列与 AGI 路线图
- **[Google DeepMind](https://deepmind.google/discover/blog/)**: Gemini, AlphaFold 等基础研究
- **[Anthropic News](https://www.anthropic.com/news)**: Claude 模型安全与更新
- **[Hugging Face Blog](https://huggingface.co/blog)**: 开源模型、数据集与教程
- **[Meta AI Blog](https://ai.meta.com/blog/)**: LLaMA 开源生态
- **[Microsoft Research](https://www.microsoft.com/en-us/research/blog/)**: 计算机科学前沿

### 🔬 AI 研究 / 技术趋势
- **[Hugging Face Daily Papers](https://huggingface.co/papers)**: 每日最热门 AI 论文
- **[arXiv CS.AI](https://arxiv.org/list/cs.AI/recent)**: 人工智能最新预印本
- **[arXiv CS.LG](https://arxiv.org/list/cs.LG/recent)**: 机器学习最新预印本
- **[arXiv CS.CL](https://arxiv.org/list/cs.CL/recent)**: 计算语言学最新预印本

### 💡 高质量聚合 / 社区（国外）
- **[Hacker News](https://news.ycombinator.com/)**: 极客视角的 AI 技术讨论
- **[Techmeme](https://www.techmeme.com/)**: 科技新闻聚合（必读）
- **[GitHub Trending](https://github.com/trending)**: 热门开源项目
- **[Reddit r/MachineLearning](https://www.reddit.com/r/MachineLearning/)**: 严肃学术讨论
- **[Reddit r/LocalLLaMA](https://www.reddit.com/r/LocalLLaMA/)**: 本地大模型实战

### 🇨🇳 中国 AI 资讯（国内）
- **[机器之心](https://www.jiqizhixin.com/)**: 中国领先的 AI 科技媒体
- **[量子位](https://www.qbitai.com/)**: 前沿 AI 技术报道
- **[新智元](https://www.jiqizhixin.com/)**: AI 产业资讯
- **[InfoQ 中文](https://www.infoq.cn/)**: 技术深度文章
- **[36氪 AI](https://36kr.com/)**: AI 创投动态
- **[掘金](https://juejin.cn/)**: 开发者技术社区
- **[V2EX](https://www.v2ex.com/)**: 创意工作者社区

## 🛠️ 实现原理

本项目基于 Go 语言开发，利用 GitHub Actions 实现全自动运行。

` + "```mermaid" + `
graph TD
    Daily[📅 Daily Trigger] --> Fetcher
    
    subgraph "Fetcher (Data Collection)"
        F1[Official Blogs]
        F2[Tech Media]
        F3[arXiv/Papers]
        F4[GitHub/HackerNews]
        F5[中国资讯源]
    end
    
    Fetcher --> RawItems[Raw News Items]
    
    subgraph "LLM Processing"
        RawItems --> Filter[AI Keyword Filter]
        Filter --> Deduplicate[Deduplication]
        Deduplicate --> Summarizer[DeepSeek/OpenAI Summarizer]
    end
    
    Summarizer --> Generator[Markdown Generator]
    Generator --> Report[📄 Daily Report]
    Generator --> README[📄 README.md]
    
    style Daily fill:#f9f,stroke:#333,stroke-width:2px
    style Summarizer fill:#bbf,stroke:#333,stroke-width:2px
    style Report fill:#bfb,stroke:#333,stroke-width:2px
` + "```" + `

## 🚀 配置说明

如果你想自己部署，请参考以下配置：

### 环境变量（可选）

` + "```bash" + `
# Product Hunt API Key（可选，获取产品资讯）
export PRODUCTHUNT_API_KEY=your_key

# LLM 配置（可选，用于生成中文摘要）
# 支持 OpenAI, DeepSeek, Moonshot 等兼容接口
export LLM_API_KEY=sk-xxxxxx
export LLM_BASE_URL=https://api.deepseek.com/v1  # 默认为 OpenAI
export LLM_MODEL=deepseek-chat                   # 默认为 gpt-3.5-turbo
` + "```" + `
`
}

func readExistingReadme() ([]string, error) {
	file, err := os.Open("README.md")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var lines []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		lines = append(lines, scanner.Text())
	}
	return lines, scanner.Err()
}
