package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/url"
	"os"
	"strings"
	"text/tabwriter"
	"time"
	"unicode"
)

const defaultDBPath = "data/crawler.db"

type crawlConfig struct {
	StartURL  string
	DBPath    string
	MaxPages  int
	Delay     time.Duration
	Timeout   time.Duration
	UserAgent string
}

type listConfig struct {
	DBPath string
	Limit  int
	Host   string
}

type showConfig struct {
	DBPath string
	ID     int64
	URL    string
}

type helpCommand struct{}

func main() {
	if err := run(os.Args[1:]); err != nil {
		log.Fatal(err)
	}
}

func run(args []string) error {
	command, err := parseCommand(args)
	if err != nil {
		return err
	}

	switch cfg := command.(type) {
	case crawlConfig:
		return runCrawl(cfg)
	case listConfig:
		return runList(cfg)
	case showConfig:
		return runShow(cfg)
	case helpCommand:
		fmt.Fprintln(os.Stdout, usageText())
		return nil
	default:
		return fmt.Errorf("未知命令")
	}
}

func parseCommand(args []string) (any, error) {
	if len(args) == 0 {
		return helpCommand{}, nil
	}

	switch args[0] {
	case "crawl":
		return parseCrawlFlags(args[1:])
	case "list":
		return parseListFlags(args[1:])
	case "show":
		return parseShowFlags(args[1:])
	case "-h", "--help", "help":
		return helpCommand{}, nil
	default:
		// 向后兼容旧用法: go run . -url ...
		return parseCrawlFlags(args)
	}
}

func parseCrawlFlags(args []string) (crawlConfig, error) {
	fs := flag.NewFlagSet("crawl", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cfg := crawlConfig{}
	fs.StringVar(&cfg.StartURL, "url", "", "要抓取的起始 URL，例如 https://example.com")
	fs.StringVar(&cfg.DBPath, "db", defaultDBPath, "SQLite 数据库文件路径")
	fs.IntVar(&cfg.MaxPages, "max-pages", 20, "最多抓取的页面数量")
	fs.DurationVar(&cfg.Delay, "delay", 500*time.Millisecond, "每次请求之间的等待时间")
	fs.DurationVar(&cfg.Timeout, "timeout", 10*time.Second, "单个请求超时时间")
	fs.StringVar(&cfg.UserAgent, "user-agent", "local-sqlite-crawler/1.0", "HTTP User-Agent")

	if err := fs.Parse(args); err != nil {
		return crawlConfig{}, err
	}

	if cfg.StartURL == "" {
		return crawlConfig{}, fmt.Errorf("必须提供 -url 参数")
	}
	if cfg.MaxPages <= 0 {
		return crawlConfig{}, fmt.Errorf("-max-pages 必须大于 0")
	}

	return cfg, nil
}

func parseListFlags(args []string) (listConfig, error) {
	fs := flag.NewFlagSet("list", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cfg := listConfig{}
	fs.StringVar(&cfg.DBPath, "db", defaultDBPath, "SQLite 数据库文件路径")
	fs.IntVar(&cfg.Limit, "limit", 20, "显示最近多少条记录")
	fs.StringVar(&cfg.Host, "host", "", "按域名过滤")

	if err := fs.Parse(args); err != nil {
		return listConfig{}, err
	}
	if cfg.Limit <= 0 {
		return listConfig{}, fmt.Errorf("-limit 必须大于 0")
	}

	return cfg, nil
}

func parseShowFlags(args []string) (showConfig, error) {
	fs := flag.NewFlagSet("show", flag.ContinueOnError)
	fs.SetOutput(os.Stderr)

	cfg := showConfig{}
	fs.StringVar(&cfg.DBPath, "db", defaultDBPath, "SQLite 数据库文件路径")
	fs.Int64Var(&cfg.ID, "id", 0, "按记录 ID 查看")
	fs.StringVar(&cfg.URL, "url", "", "按页面 URL 查看")

	if err := fs.Parse(args); err != nil {
		return showConfig{}, err
	}
	if cfg.ID <= 0 && cfg.URL == "" {
		return showConfig{}, fmt.Errorf("必须提供 -id 或 -url")
	}

	return cfg, nil
}

func runCrawl(cfg crawlConfig) error {
	startURL, err := normalizeSeedURL(cfg.StartURL)
	if err != nil {
		return fmt.Errorf("起始 URL 无效: %w", err)
	}

	return withInitializedStore(context.Background(), cfg.DBPath, func(ctx context.Context, store *SQLiteStore) error {
		crawler := NewCrawler(CrawlerConfig{
			StartURL:  startURL,
			MaxPages:  cfg.MaxPages,
			Delay:     cfg.Delay,
			Timeout:   cfg.Timeout,
			UserAgent: cfg.UserAgent,
			Store:     store,
		})

		result, err := crawler.Crawl(ctx)
		if err != nil {
			return err
		}

		fmt.Printf(
			"抓取完成: fetched=%d stored=%d skipped=%d db=%s host=%s\n",
			result.Fetched,
			result.Stored,
			result.Skipped,
			cfg.DBPath,
			hostLabel(startURL),
		)

		return nil
	})
}

func runList(cfg listConfig) error {
	return withInitializedStore(context.Background(), cfg.DBPath, func(ctx context.Context, store *SQLiteStore) error {
		pages, err := store.ListPages(ctx, ListPagesQuery{
			Limit: cfg.Limit,
			Host:  cfg.Host,
		})
		if err != nil {
			return fmt.Errorf("查询列表失败: %w", err)
		}
		if len(pages) == 0 {
			fmt.Println("没有记录")
			return nil
		}

		writer := tabwriter.NewWriter(os.Stdout, 0, 4, 2, ' ', 0)
		fmt.Fprintln(writer, "ID\tHOST\tSTATUS\tDEPTH\tCRAWLED_AT\tTITLE\tURL")
		for _, page := range pages {
			fmt.Fprintf(
				writer,
				"%d\t%s\t%d\t%d\t%s\t%s\t%s\n",
				page.ID,
				sanitizeForTerminal(page.Host),
				page.StatusCode,
				page.Depth,
				page.CrawledAt.Local().Format("2006-01-02 15:04:05"),
				clipForTable(sanitizeForTerminal(page.Title), 36),
				sanitizeForTerminal(page.URL),
			)
		}
		return writer.Flush()
	})
}

func runShow(cfg showConfig) error {
	return withInitializedStore(context.Background(), cfg.DBPath, func(ctx context.Context, store *SQLiteStore) error {
		var page PageDetails
		var err error
		if cfg.ID > 0 {
			page, err = store.GetPageByID(ctx, cfg.ID)
		} else {
			page, err = store.GetPageByURL(ctx, cfg.URL)
		}
		if err != nil {
			return err
		}

		fmt.Printf("ID: %d\n", page.ID)
		fmt.Printf("URL: %s\n", sanitizeForTerminal(page.URL))
		fmt.Printf("Host: %s\n", sanitizeForTerminal(page.Host))
		fmt.Printf("Status: %d\n", page.StatusCode)
		fmt.Printf("Content-Type: %s\n", sanitizeForTerminal(page.ContentType))
		fmt.Printf("Depth: %d\n", page.Depth)
		fmt.Printf("Crawled-At: %s\n", page.CrawledAt.Local().Format(time.RFC3339))
		fmt.Printf("Title: %s\n", sanitizeForTerminal(page.Title))
		fmt.Printf("Summary: %s\n", sanitizeForTerminal(page.Summary))
		fmt.Println("Body:")
		fmt.Println(sanitizeForTerminal(page.BodyText))

		return nil
	})
}

func withInitializedStore(ctx context.Context, dbPath string, fn func(context.Context, *SQLiteStore) error) error {
	store, err := OpenSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("打开数据库失败: %w", err)
	}
	defer store.Close()

	if err := store.Init(ctx); err != nil {
		return fmt.Errorf("初始化数据库失败: %w", err)
	}

	return fn(ctx, store)
}

func sanitizeForTerminal(value string) string {
	return strings.Map(func(r rune) rune {
		switch r {
		case '\n', '\r', '\t':
			return r
		}
		if unicode.IsPrint(r) {
			return r
		}
		return -1
	}, value)
}

func usageText() string {
	return `用法:
  go run . crawl -url https://example.com
  go run . list [-db data/crawler.db] [-limit 20] [-host example.com]
  go run . show [-db data/crawler.db] -id 1
  go run . show [-db data/crawler.db] -url https://example.com/page

兼容旧用法:
  go run . -url https://example.com`
}

func hostLabel(rawURL *url.URL) string {
	if rawURL == nil {
		return ""
	}
	return rawURL.Hostname()
}

func clipForTable(value string, limit int) string {
	value = strings.ReplaceAll(value, "\n", " ")
	value = strings.TrimSpace(value)
	if value == "" {
		return "-"
	}
	return truncateRunes(value, limit)
}
