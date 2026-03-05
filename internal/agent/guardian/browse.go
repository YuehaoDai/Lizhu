package guardian

import (
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cloudwego/eino/schema"
)

// browseWebToolInfo 是供 LLM 调用的网页浏览工具定义。
var browseWebToolInfo = &schema.ToolInfo{
	Name: "browse_web",
	Desc: "抓取指定 URL 的网页内容，提取纯文本后返回。当修行者提供网址、或问题需要参考网页实时内容时使用。",
	ParamsOneOf: schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"url": {
			Type:     schema.String,
			Desc:     "要抓取的完整 URL，必须以 http:// 或 https:// 开头",
			Required: true,
		},
	}),
}

const (
	browseMaxChars  = 4000
	browseHTTPTimeout = 12 * time.Second
)

// fetchWebContent 抓取 url 对应的网页，提取纯文本后截断返回。
func fetchWebContent(url string) (string, error) {
	client := &http.Client{Timeout: browseHTTPTimeout}
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return "", fmt.Errorf("构建请求失败: %w", err)
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; LizhuBot/1.0)")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("网络请求失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return "", fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// 最多读取 512KB，防止超大页面
	body, err := io.ReadAll(io.LimitReader(resp.Body, 512*1024))
	if err != nil {
		return "", fmt.Errorf("读取响应失败: %w", err)
	}

	text := extractText(string(body))
	text = strings.TrimSpace(text)

	// 按 rune 截断，避免切断多字节字符
	if utf8.RuneCountInString(text) > browseMaxChars {
		runes := []rune(text)
		text = string(runes[:browseMaxChars]) + "\n\n[...内容过长，已截断]"
	}

	return fmt.Sprintf("【来源】%s\n\n%s", url, text), nil
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
