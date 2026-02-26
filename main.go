package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"ai-daily-news/fetcher"
	"ai-daily-news/generator"
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

	dailyDir := "daily"
	if err := os.MkdirAll(dailyDir, 0755); err != nil {
		log.Fatalf("创建目录失败: %v", err)
	}

	filename := fmt.Sprintf("%s/%s.md", dailyDir, today)
	if err := generator.GenerateDailyReport(filename, today, newsItems); err != nil {
		log.Fatalf("生成日报失败: %v", err)
	}

	if err := generator.UpdateReadme(today, filename); err != nil {
		log.Fatalf("更新 README 失败: %v", err)
	}

	log.Printf("AI 资讯日报生成完成: %s", filename)
}
