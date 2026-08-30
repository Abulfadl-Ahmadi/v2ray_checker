package collector

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/jszwec/csvutil"
)

type ChannelsType struct {
	URL             string `csv:"URL"`
	AllMessagesFlag bool   `csv:"AllMessagesFlag"`
}

type RawScrapedNode struct {
	Protocol string
	RawLink  string
	Source   string
}

var (
	httpClient = &http.Client{Timeout: 10 * time.Second}
	myregex    = map[string]string{
		"ss":     `(?m)(...ss:|^ss:)\/\/.+?(?:%3A%40|#|$)`,
		"vmess":  `(?m)vmess:\/\/.+`,
		"trojan": `(?m)trojan:\/\/.+?(?:%3A%40|#|$)`,
		"vless":  `(?m)vless:\/\/.+?(?:%3A%40|#|$)`,
	}
)

func ScrapeChannels(csvPath string, maxMessages int) ([]RawScrapedNode, error) {
	fileData, err := os.ReadFile(csvPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read channels csv: %w", err)
	}

	var channels []ChannelsType
	if err = csvutil.Unmarshal(fileData, &channels); err != nil {
		return nil, fmt.Errorf("failed to parse channels csv: %w", err)
	}

	var mu sync.Mutex
	var results []RawScrapedNode
	seen := make(map[string]bool)

	var wg sync.WaitGroup
	sem := make(chan struct{}, 5)

	for _, channel := range channels {
		wg.Add(1)
		sem <- struct{}{}

		go func(c ChannelsType) {
			defer wg.Done()
			defer func() { <-sem }()

			webURL := ChangeUrlToTelegramWebUrl(c.URL)
			channelName := extractChannelName(c.URL)

			resp, err := httpClient.Get(webURL)
			if err != nil || resp.StatusCode != 200 {
				if resp != nil {
					resp.Body.Close()
				}
				return
			}

			doc, err := goquery.NewDocumentFromReader(resp.Body)
			resp.Body.Close()
			if err != nil {
				return
			}

			extracted := crawlForV2ray(doc, webURL, c.AllMessagesFlag, channelName, maxMessages)

			mu.Lock()
			for _, item := range extracted {
				if !seen[item.RawLink] {
					seen[item.RawLink] = true
					results = append(results, item)
				}
			}
			mu.Unlock()
		}(channel)
	}

	wg.Wait()
	return results, nil
}

func ChangeUrlToTelegramWebUrl(input string) string {
	if !strings.Contains(input, "/s/") {
		index := strings.Index(input, "/t.me/")
		if index != -1 {
			return input[:index+len("/t.me/")] + "s/" + input[index+len("/t.me/"):]
		}
	}
	return input
}

func extractChannelName(urlStr string) string {
	urlStr = strings.TrimSuffix(urlStr, "/")
	parts := strings.Split(urlStr, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}
	return "unknown"
}

func crawlForV2ray(doc *goquery.Document, channelLink string, hasAllMessagesFlag bool, channelName string, maxMsg int) []RawScrapedNode {
	var nodes []RawScrapedNode

	doc.Find(".tgme_widget_message_text").Each(func(j int, s *goquery.Selection) {
		messageText, _ := s.Html()
		str := strings.ReplaceAll(messageText, "<br/>", "\n")
		docHtml, _ := goquery.NewDocumentFromReader(strings.NewReader(str))
		messageText = docHtml.Text()
		lines := strings.Split(strings.TrimSpace(messageText), "\n")

		for _, line := range lines {
			line = strings.TrimSpace(line)
			for proto, reg := range myregex {
				re := regexp.MustCompile(reg)
				matches := re.FindAllString(line, -1)
				for _, match := range matches {
					match = strings.TrimSpace(match)
					if match == "" {
						continue
					}
					match = strings.TrimSuffix(match, "#")
					match = strings.TrimSuffix(match, "%3A%40")
					nodes = append(nodes, RawScrapedNode{
						Protocol: proto,
						RawLink:  match,
						Source:   channelName,
					})
				}
			}
		}
	})

	return nodes
}
