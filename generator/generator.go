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
	Categories  []CategoryGroup
	TotalCount  int
}

func GenerateDailyReport(filename, date string, items []fetcher.NewsItem) error {
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

## 日报列表

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

	for _, file := range files {
		dateStr := strings.TrimSuffix(file, ".md")
		sb.WriteString(fmt.Sprintf("- [%s](daily/%s)\n", dateStr, file))
	}

	return os.WriteFile("README.md", []byte(sb.String()), 0644)
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
