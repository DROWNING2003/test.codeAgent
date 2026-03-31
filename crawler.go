package main

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"path"
	"strings"
	"time"
)

type CrawlerConfig struct {
	StartURL  *url.URL
	MaxPages  int
	Delay     time.Duration
	Timeout   time.Duration
	UserAgent string
	Store     *SQLiteStore
}

type CrawlResult struct {
	Fetched int
	Stored  int
	Skipped int
}

type Crawler struct {
	startURL    *url.URL
	allowedHost string
	maxPages    int
	delay       time.Duration
	userAgent   string
	store       *SQLiteStore
	client      *http.Client
}

type crawlTask struct {
	url   *url.URL
	depth int
}

func NewCrawler(cfg CrawlerConfig) *Crawler {
	return &Crawler{
		startURL:    cloneURL(cfg.StartURL),
		allowedHost: strings.ToLower(cfg.StartURL.Hostname()),
		maxPages:    cfg.MaxPages,
		delay:       cfg.Delay,
		userAgent:   cfg.UserAgent,
		store:       cfg.Store,
		client: &http.Client{
			Timeout: cfg.Timeout,
			CheckRedirect: func(req *http.Request, via []*http.Request) error {
				if len(via) >= 10 {
					return fmt.Errorf("redirect 太多，已停止")
				}
				return nil
			},
		},
	}
}

func (c *Crawler) Crawl(ctx context.Context) (CrawlResult, error) {
	result := CrawlResult{}
	seen := map[string]struct{}{
		canonicalURL(c.startURL): {},
	}
	queue := []crawlTask{{url: cloneURL(c.startURL), depth: 0}}

	for len(queue) > 0 && result.Fetched < c.maxPages {
		task := queue[0]
		queue = queue[1:]

		record, discovered, skipped, err := c.fetchPage(ctx, task)
		if err != nil {
			log.Printf("抓取失败: url=%s error=%v", canonicalURL(task.url), err)
			result.Skipped++
		} else if skipped {
			result.Skipped++
		} else {
			result.Fetched++
			if err := c.store.SavePage(ctx, *record); err != nil {
				return result, fmt.Errorf("写入数据库失败: %w", err)
			}
			result.Stored++
			log.Printf("已保存: depth=%d status=%d url=%s", record.Depth, record.StatusCode, record.URL)
		}

		for _, nextURL := range discovered {
			normalized := canonicalURL(nextURL)
			if _, ok := seen[normalized]; ok {
				continue
			}
			seen[normalized] = struct{}{}
			queue = append(queue, crawlTask{
				url:   nextURL,
				depth: task.depth + 1,
			})
		}

		if len(queue) == 0 || result.Fetched >= c.maxPages || c.delay <= 0 {
			continue
		}

		timer := time.NewTimer(c.delay)
		select {
		case <-ctx.Done():
			timer.Stop()
			return result, ctx.Err()
		case <-timer.C:
		}
	}

	return result, nil
}

func (c *Crawler) fetchPage(ctx context.Context, task crawlTask) (*PageRecord, []*url.URL, bool, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, canonicalURL(task.url), nil)
	if err != nil {
		return nil, nil, false, err
	}
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.client.Do(req)
	if err != nil {
		return nil, nil, false, err
	}
	defer resp.Body.Close()

	finalURL := resp.Request.URL
	if !c.isAllowed(finalURL) {
		return nil, nil, true, nil
	}

	contentType := resp.Header.Get("Content-Type")
	if !strings.Contains(strings.ToLower(contentType), "text/html") {
		return nil, nil, true, nil
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	if err != nil {
		return nil, nil, false, err
	}

	title, summary, bodyText, links := extractPageData(finalURL, string(body))

	discovered := make([]*url.URL, 0, len(links))
	for _, nextURL := range links {
		if !c.isAllowed(nextURL) {
			continue
		}
		discovered = append(discovered, nextURL)
	}

	record := &PageRecord{
		URL:         canonicalURL(finalURL),
		Host:        strings.ToLower(finalURL.Hostname()),
		Title:       title,
		Summary:     summary,
		BodyText:    bodyText,
		StatusCode:  resp.StatusCode,
		ContentType: contentType,
		Depth:       task.depth,
		CrawledAt:   time.Now().UTC(),
	}

	return record, discovered, false, nil
}

func (c *Crawler) isAllowed(rawURL *url.URL) bool {
	if rawURL == nil {
		return false
	}
	if rawURL.Scheme != "http" && rawURL.Scheme != "https" {
		return false
	}
	return strings.EqualFold(rawURL.Hostname(), c.allowedHost)
}

func normalizeSeedURL(raw string) (*url.URL, error) {
	candidate := strings.TrimSpace(raw)
	if candidate == "" {
		return nil, fmt.Errorf("URL 不能为空")
	}
	if !strings.Contains(candidate, "://") {
		candidate = "https://" + candidate
	}

	parsed, err := url.Parse(candidate)
	if err != nil {
		return nil, err
	}
	if parsed.Host == "" {
		return nil, fmt.Errorf("缺少 host")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, fmt.Errorf("只支持 http/https")
	}

	return parsed, nil
}

func canonicalURL(rawURL *url.URL) string {
	if rawURL == nil {
		return ""
	}

	cloned := cloneURL(rawURL)
	cloned.Fragment = ""
	cloned.Host = strings.ToLower(cloned.Host)

	if cloned.Path == "" {
		cloned.Path = "/"
	} else {
		cloned.Path = path.Clean(cloned.Path)
		if !strings.HasPrefix(cloned.Path, "/") {
			cloned.Path = "/" + cloned.Path
		}
	}

	query := cloned.Query()
	cloned.RawQuery = query.Encode()

	return cloned.String()
}

func cloneURL(rawURL *url.URL) *url.URL {
	if rawURL == nil {
		return nil
	}
	copied := *rawURL
	return &copied
}
