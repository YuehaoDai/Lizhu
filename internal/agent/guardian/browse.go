package guardian

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

// browseWebToolInfo 是供 LLM 调用的网页浏览工具定义。
var browseWebToolInfo = &schema.ToolInfo{
	Name: "browse_web",
	Desc: "抓取指定 URL 的网页内容，提取纯文本后返回。当用户提供具体网址、或需要查看某个页面的详细内容时使用。",
	ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"url": {
			Type:     schema.String,
			Desc:     "要抓取的完整 URL，必须以 http:// 或 https:// 开头",
			Required: true,
		},
	}),
}

// searchWebToolInfo 是供 LLM 调用的网页搜索工具定义。
var searchWebToolInfo = &schema.ToolInfo{
	Name: "search_web",
	Desc: "在互联网上搜索信息，返回相关页面的标题、链接和摘要。当用户询问需要实时信息、新闻、文档或任何需要联网查询的问题时使用。",
	ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"query": {
			Type:     schema.String,
			Desc:     "搜索关键词或问题，建议使用英文以获得更好的搜索结果",
			Required: true,
		},
	}),
}

const (
	browseMaxChars    = 4000
	browseHTTPTimeout = 12 * time.Second
	searchMaxResults  = 5
)

// fetchWebContent 抓取 url 对应的网页，提取纯文本后截断返回。
// 若直接抓取内容过少（JS 渲染页面），自动降级到 Jina Reader 服务重试。
func fetchWebContent(rawURL string) (string, error) {
	text, err := fetchAndExtract(rawURL)
	if err != nil || utf8.RuneCountInString(text) < 80 {
		// 降级：通过 Jina Reader 渲染后抓取
		jinaURL := "https://r.jina.ai/" + rawURL
		jinaText, jinaErr := fetchAndExtract(jinaURL)
		if jinaErr != nil {
			if err != nil {
				return "", fmt.Errorf("直接抓取失败（%v），Jina 降级也失败（%v）", err, jinaErr)
			}
			return "", fmt.Errorf("页面内容提取失败（可能是 JavaScript 渲染页面），Jina 降级失败: %v", jinaErr)
		}
		if utf8.RuneCountInString(jinaText) < 80 {
			return "", fmt.Errorf("页面内容为空，无法提取有效信息（已尝试直接抓取和 Jina 渲染两种方式）")
		}
		return fmt.Sprintf("【来源】%s（经 Jina Reader 渲染）\n\n%s", rawURL, jinaText), nil
	}
	return fmt.Sprintf("【来源】%s\n\n%s", rawURL, text), nil
}

// fetchAndExtract 发送 HTTP GET 请求并提取纯文本，不做降级处理。
func fetchAndExtract(rawURL string) (string, error) {
	client := &http.Client{Timeout: browseHTTPTimeout}
	req, err := http.NewRequest(http.MethodGet, rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("构建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LizhuBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,text/plain")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	text := extractText(string(body))
	text = strings.TrimSpace(text)

	if utf8.RuneCountInString(text) > browseMaxChars {
		runes := []rune(text)
		text = string(runes[:browseMaxChars]) + "\n\n[...内容过长，已截断]"
	}
	return text, nil
}

// searchWeb 调用 Brave Search API，返回格式化的搜索结果摘要。
func searchWeb(query, apiKey string) (string, error) {
	apiURL := "https://api.search.brave.com/res/v1/web/search?q=" +
		url.QueryEscape(query) + fmt.Sprintf("&count=%d", searchMaxResults)

	req, err := http.NewRequest(http.MethodGet, apiURL, nil)
	if err != nil {
		return "", fmt.Errorf("构建搜索请求失败: %w", err)
	}
	req.Header.Set("X-Subscription-Token", apiKey)
	req.Header.Set("Accept", "application/json")

	client := &http.Client{Timeout: browseHTTPTimeout}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("搜索请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("Brave Search API 返回 HTTP %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 256*1024))
	if err != nil {
		return "", fmt.Errorf("读取搜索响应失败: %w", err)
	}

	var result struct {
		Web struct {
			Results []struct {
				Title       string `json:"title"`
				URL         string `json:"url"`
				Description string `json:"description"`
			} `json:"results"`
		} `json:"web"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("解析搜索结果失败: %w", err)
	}

	if len(result.Web.Results) == 0 {
		return "未找到相关搜索结果。", nil
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("【搜索词】%s\n\n【搜索结果】\n\n", query))
	for i, r := range result.Web.Results {
		sb.WriteString(fmt.Sprintf("%d. **%s**\n   链接：%s\n   摘要：%s\n\n", i+1, r.Title, r.URL, r.Description))
	}
	sb.WriteString("如需查看某个页面的完整内容，可使用 browse_web 工具抓取对应链接。")
	return sb.String(), nil
}

var (
	reScriptStyle = regexp.MustCompile(`(?is)<(script|style|head|noscript)[^>]*>.*?</(script|style|head|noscript)>`)
	reHTMLTag     = regexp.MustCompile(`<[^>]+>`)
	reBlankLines  = regexp.MustCompile(`\n{3,}`)
)

// extractText 从 HTML 字符串中去除标签，提取可见纯文本。
func extractText(htmlStr string) string {
	// 先去掉 script/style/head/noscript 整块
	text := reScriptStyle.ReplaceAllString(htmlStr, "")
	// 再去掉剩余 HTML 标签
	text = reHTMLTag.ReplaceAllString(text, "\n")
	// 合并多余空行
	text = reBlankLines.ReplaceAllString(text, "\n\n")
	return text
}
