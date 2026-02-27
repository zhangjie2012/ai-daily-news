package summarizer

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"

	"ai-daily-news/fetcher"
)

type Summarizer struct {
	apiKey  string
	baseURL string
	model   string
	client  *http.Client
}

func NewSummarizer() *Summarizer {
	apiKey := os.Getenv("LLM_API_KEY")
	baseURL := os.Getenv("LLM_BASE_URL")
	model := os.Getenv("LLM_MODEL")

	if baseURL == "" {
		baseURL = "https://api.openai.com/v1"
	}
	if model == "" {
		model = "gpt-3.5-turbo" // 默认模型，可被环境变量覆盖
	}

	return &Summarizer{
		apiKey:  apiKey,
		baseURL: strings.TrimSuffix(baseURL, "/"),
		model:   model,
		client:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (s *Summarizer) Enabled() bool {
	return s.apiKey != ""
}

func (s *Summarizer) Summarize(title, content string) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("LLM_API_KEY not set")
	}

	prompt := fmt.Sprintf(`请用中文简要总结以下 AI 资讯。
要求：
1. 翻译成中文。
2. 提炼核心内容，保留关键技术术语（如 Transformer, Agent 等）。
3. 语气专业、客观。
4. 字数控制在 50-100 字以内。
5. 不要包含"这篇资讯介绍了..."等套话，直接陈述内容。

标题：%s
内容：%s`, title, content)

	reqBody := map[string]interface{}{
		"model": s.model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个专业的 AI 技术媒体编辑，擅长将英文技术资讯翻译并总结为简练的中文摘要。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.3,
		"max_tokens":  200,
	}

	jsonBody, _ := json.Marshal(reqBody)

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}

func (s *Summarizer) GenerateBriefing(items []fetcher.NewsItem) (string, error) {
	if !s.Enabled() {
		return "", fmt.Errorf("LLM_API_KEY not set")
	}

	var contentBuilder strings.Builder
	contentBuilder.WriteString("以下是今日 AI 资讯列表：\n\n")
	for i, item := range items {
		if i >= 30 {
			break
		}
		contentBuilder.WriteString(fmt.Sprintf("%d. [%s] %s - %s\n", i+1, item.Source, item.Title, item.Summary))
		if item.Summary != "" {
			contentBuilder.WriteString(fmt.Sprintf("   简介：%s\n", item.Summary))
		}
	}

	prompt := fmt.Sprintf(`请根据以下今日 AI 资讯列表，生成一份简洁的中文简报。

要求：
1. 总结今日 AI 领域的主要趋势和热点话题（3-5条）。
2. 每条趋势用一句话概括，突出关键信息。
3. 如果有重大新闻（如新模型发布、重要收购、突破性研究），请特别标注。
4. 语气专业、客观，适合技术人员阅读。
5. 总字数控制在 200-400 字。

%s

请直接输出简报内容，不要包含"以下是简报"等套话。`, contentBuilder.String())

	reqBody := map[string]interface{}{
		"model": s.model,
		"messages": []map[string]string{
			{"role": "system", "content": "你是一个专业的 AI 技术媒体编辑，擅长从大量资讯中提炼关键趋势和热点。"},
			{"role": "user", "content": prompt},
		},
		"temperature": 0.5,
		"max_tokens":  800,
	}

	jsonBody, _ := json.Marshal(reqBody)

	ctx, cancel := context.WithTimeout(context.Background(), 90*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, "POST", s.baseURL+"/chat/completions", bytes.NewBuffer(jsonBody))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+s.apiKey)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var result struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}

	if len(result.Choices) == 0 {
		return "", fmt.Errorf("no response from LLM")
	}

	return strings.TrimSpace(result.Choices[0].Message.Content), nil
}
