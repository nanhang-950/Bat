package fn

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

const (
	defaultAIHostURL = "https://generativelanguage.googleapis.com/v1beta/models"
	defaultAIModel   = "gemini-2.0-flash"
	aiRequestTimeout = 45 * time.Second
	maxPromptResults = 400
)

type aiConfig struct {
	HostURL string
	Model   string
	APIKey  string
}

func ProcessWebSocketData(results []ScanResult) ([]string, error) {
	if len(results) == 0 {
		return nil, nil
	}

	cfg := loadAIConfig()
	if err := cfg.validate(); err != nil {
		return nil, err
	}

	answer, err := requestGeminiContent(cfg, resultsToPrompt(results))
	if err != nil {
		return nil, err
	}

	return compactLines(strings.Split(strings.TrimSpace(answer), "\n")), nil
}

func requestGeminiContent(cfg aiConfig, question string) (string, error) {
	reqBody := map[string]interface{}{
		"system_instruction": map[string]interface{}{
			"parts": []map[string]string{{
				"text": "根据以下内网测绘结果，评估该内网的安全性，先分点说明风险与建议，最后给出一段总结。",
			}},
		},
		"contents": []map[string]interface{}{{
			"role": "user",
			"parts": []map[string]string{{
				"text": question,
			}},
		}},
		"generationConfig": map[string]interface{}{
			"temperature":     0.5,
			"topK":            1,
			"maxOutputTokens": 2048,
		},
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("编码 Gemini 请求失败: %w", err)
	}

	endpoint := strings.TrimRight(cfg.HostURL, "/")
	if strings.Contains(endpoint, ":generateContent") {
		sep := "?"
		if strings.Contains(endpoint, "?") {
			sep = "&"
		}
		endpoint += sep + "key=" + url.QueryEscape(cfg.APIKey)
	} else {
		endpoint += "/" + cfg.Model + ":generateContent?key=" + url.QueryEscape(cfg.APIKey)
	}

	req, err := http.NewRequest(http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return "", fmt.Errorf("创建 Gemini 请求失败: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := (&http.Client{Timeout: aiRequestTimeout}).Do(req)
	if err != nil {
		return "", fmt.Errorf("调用 Gemini 失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("读取 Gemini 响应失败: %w", err)
	}

	var data struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
		Error *struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		} `json:"error"`
	}

	if err := json.Unmarshal(body, &data); err != nil {
		if resp.StatusCode >= http.StatusBadRequest {
			return "", fmt.Errorf("Gemini 接口返回异常 code=%d, body=%s", resp.StatusCode, string(body))
		}
		return "", fmt.Errorf("解析 Gemini 响应失败: %w", err)
	}

	if data.Error != nil {
		return "", fmt.Errorf("Gemini 接口返回错误 code=%d: %s", data.Error.Code, data.Error.Message)
	}
	if resp.StatusCode >= http.StatusBadRequest {
		return "", fmt.Errorf("Gemini 接口返回异常 code=%d, body=%s", resp.StatusCode, string(body))
	}

	var answer strings.Builder
	for _, candidate := range data.Candidates {
		for _, part := range candidate.Content.Parts {
			answer.WriteString(part.Text)
		}
		if answer.Len() > 0 {
			break
		}
	}

	if strings.TrimSpace(answer.String()) == "" {
		return "", fmt.Errorf("Gemini 未返回有效内容")
	}

	return answer.String(), nil
}

func resultsToPrompt(results []ScanResult) string {
	if len(results) == 0 {
		return ""
	}

	uniqueHosts := make(map[string]struct{})
	for _, result := range results {
		uniqueHosts[result.IP] = struct{}{}
	}

	limit := len(results)
	if limit > maxPromptResults {
		limit = maxPromptResults
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("扫描摘要: 存活并开放端口的主机 %d 台，开放端口 %d 个。\n", len(uniqueHosts), len(results)))
	for i := 0; i < limit; i++ {
		result := results[i]
		osName := result.OS
		if osName == "" {
			osName = "Unknown"
		}
		sb.WriteString(fmt.Sprintf("IP=%s OS=%s Port=%d Service=%s\n", result.IP, osName, result.Port, result.Protocol))
	}

	if limit < len(results) {
		sb.WriteString(fmt.Sprintf("其余 %d 条开放端口记录已省略，请结合总量与样本综合评估风险。\n", len(results)-limit))
	}

	return sb.String()
}

func compactLines(lines []string) []string {
	var result []string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		result = append(result, line)
	}
	return result
}

func loadAIConfig() aiConfig {
	return aiConfig{
		HostURL: getenvDefault("LLM_API_HOST_URL", defaultAIHostURL),
		Model:   getenvDefault("LLM_MODEL", defaultAIModel),
		APIKey:  strings.TrimSpace(os.Getenv("LLM_API_KEY")),
	}
}

func (c aiConfig) validate() error {
	if c.HostURL == "" {
		return fmt.Errorf("AI Host 未配置")
	}
	if c.Model == "" {
		return fmt.Errorf("AI Model 未配置")
	}
	if c.APIKey == "" {
		return fmt.Errorf("未配置 LLM_API_KEY")
	}
	return nil
}

func getenvDefault(key, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}
