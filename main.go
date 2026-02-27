package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"

	"ai-daily-news/fetcher"
	"ai-daily-news/generator"
	"ai-daily-news/summarizer"
)

func main() {
	today := time.Now().Format("2006-01-02")
	log.Printf("开始生成 %s 的 AI 资讯日报...", today)

	newsItems, err := fetcher.FetchAllNews()
	if err != nil {
		log.Fatalf("获取资讯失败: %v", err)
	}

	if len(newsItems) == 0 {
		log.Println("未获取到任何资讯，退出")
		return
	}

	var briefing string
	sm := summarizer.NewSummarizer()
	if sm.Enabled() {
		log.Println("LLM API 已配置，开始生成中文摘要...")
		processSummaries(sm, newsItems)

		log.Println("正在生成今日简报...")
		if b, err := sm.GenerateBriefing(newsItems); err != nil {
			log.Printf("生成简报失败: %v", err)
		} else {
			briefing = b
			log.Println("简报生成成功")
		}
	} else {
		log.Println("LLM API 未配置，跳过摘要生成步骤")
	}

	dailyDir := "daily"
	if err := os.MkdirAll(dailyDir, 0755); err != nil {
		log.Fatalf("创建目录失败: %v", err)
	}

	filename := fmt.Sprintf("%s/%s.md", dailyDir, today)
	if err := generator.GenerateDailyReport(filename, today, newsItems, briefing); err != nil {
		log.Fatalf("生成日报失败: %v", err)
	}

	if err := generator.UpdateReadme(today, filename); err != nil {
		log.Fatalf("更新 README 失败: %v", err)
	}

	log.Printf("AI 资讯日报生成完成: %s", filename)
}

func processSummaries(sm *summarizer.Summarizer, items []fetcher.NewsItem) {
	var wg sync.WaitGroup
	semaphore := make(chan struct{}, 5) // 限制并发数为 5

	for i := range items {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			semaphore <- struct{}{}        // 获取信号量
			defer func() { <-semaphore }() // 释放信号量

			item := &items[idx]
			content := item.Summary
			if content == "" {
				content = item.Title
			}

			// 如果已经是中文且较短，可能不需要摘要，但在 AI 领域，通常英文居多
			// 这里简单地全部调用，让 LLM 决定是否需要翻译或润色
			summary, err := sm.Summarize(item.Title, content)
			if err == nil && summary != "" {
				item.Summary = summary
				log.Printf("摘要生成成功: %s", item.Title)
			} else {
				log.Printf("摘要生成失败 [%s]: %v", item.Title, err)
			}
		}(i)
	}

	wg.Wait()
}
