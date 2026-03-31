package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "modernc.org/sqlite"
)

type PageRecord struct {
	URL         string
	Host        string
	Title       string
	Summary     string
	BodyText    string
	StatusCode  int
	ContentType string
	Depth       int
	CrawledAt   time.Time
}

type PageSummary struct {
	ID         int64
	URL        string
	Host       string
	Title      string
	StatusCode int
	Depth      int
	CrawledAt  time.Time
}

type PageDetails struct {
	ID          int64
	URL         string
	Host        string
	Title       string
	Summary     string
	BodyText    string
	StatusCode  int
	ContentType string
	Depth       int
	CrawledAt   time.Time
}

type ListPagesQuery struct {
	Limit int
	Host  string
}

type SQLiteStore struct {
	db *sql.DB
}

func OpenSQLiteStore(dbPath string) (*SQLiteStore, error) {
	dir := filepath.Dir(dbPath)
	if dir != "." && dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, err
		}
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}

	db.SetMaxOpenConns(1)
	db.SetConnMaxLifetime(0)
	db.SetMaxIdleConns(1)

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, err
	}

	return &SQLiteStore{db: db}, nil
}

func (s *SQLiteStore) Init(ctx context.Context) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	schema := `
CREATE TABLE IF NOT EXISTS pages (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    url TEXT NOT NULL UNIQUE,
    host TEXT NOT NULL,
    title TEXT,
    summary TEXT,
    body_text TEXT,
    status_code INTEGER NOT NULL,
    content_type TEXT,
    depth INTEGER NOT NULL DEFAULT 0,
    crawled_at DATETIME NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_pages_host ON pages(host);
CREATE INDEX IF NOT EXISTS idx_pages_crawled_at ON pages(crawled_at DESC);
`

	_, err := s.db.ExecContext(ctx, schema)
	return err
}

func (s *SQLiteStore) SavePage(ctx context.Context, record PageRecord) error {
	if s == nil || s.db == nil {
		return fmt.Errorf("数据库未初始化")
	}

	query := `
INSERT INTO pages (
    url, host, title, summary, body_text, status_code, content_type, depth, crawled_at
) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(url) DO UPDATE SET
    host = excluded.host,
    title = excluded.title,
    summary = excluded.summary,
    body_text = excluded.body_text,
    status_code = excluded.status_code,
    content_type = excluded.content_type,
    depth = excluded.depth,
    crawled_at = excluded.crawled_at;
`

	_, err := s.db.ExecContext(
		ctx,
		query,
		record.URL,
		record.Host,
		record.Title,
		record.Summary,
		record.BodyText,
		record.StatusCode,
		record.ContentType,
		record.Depth,
		record.CrawledAt,
	)

	return err
}

func (s *SQLiteStore) ListPages(ctx context.Context, query ListPagesQuery) ([]PageSummary, error) {
	if s == nil || s.db == nil {
		return nil, fmt.Errorf("数据库未初始化")
	}
	if query.Limit <= 0 {
		query.Limit = 20
	}

	baseQuery := `
SELECT id, url, host, title, status_code, depth, crawled_at
FROM pages
`

	args := make([]any, 0, 2)
	if query.Host != "" {
		baseQuery += "WHERE host = ?\n"
		args = append(args, strings.ToLower(query.Host))
	}
	baseQuery += "ORDER BY crawled_at DESC, id DESC LIMIT ?"
	args = append(args, query.Limit)

	rows, err := s.db.QueryContext(ctx, baseQuery, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	pages := make([]PageSummary, 0, query.Limit)
	for rows.Next() {
		var page PageSummary
		if err := rows.Scan(
			&page.ID,
			&page.URL,
			&page.Host,
			&page.Title,
			&page.StatusCode,
			&page.Depth,
			&page.CrawledAt,
		); err != nil {
			return nil, err
		}
		pages = append(pages, page)
	}

	return pages, rows.Err()
}

func (s *SQLiteStore) GetPageByID(ctx context.Context, id int64) (PageDetails, error) {
	if s == nil || s.db == nil {
		return PageDetails{}, fmt.Errorf("数据库未初始化")
	}

	const query = `
SELECT id, url, host, title, summary, body_text, status_code, content_type, depth, crawled_at
FROM pages
WHERE id = ?
LIMIT 1
`

	var page PageDetails
	err := s.db.QueryRowContext(ctx, query, id).Scan(
		&page.ID,
		&page.URL,
		&page.Host,
		&page.Title,
		&page.Summary,
		&page.BodyText,
		&page.StatusCode,
		&page.ContentType,
		&page.Depth,
		&page.CrawledAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return PageDetails{}, fmt.Errorf("未找到 ID=%d 的记录", id)
		}
		return PageDetails{}, err
	}

	return page, nil
}

func (s *SQLiteStore) GetPageByURL(ctx context.Context, rawURL string) (PageDetails, error) {
	if s == nil || s.db == nil {
		return PageDetails{}, fmt.Errorf("数据库未初始化")
	}

	lookupURL := strings.TrimSpace(rawURL)
	if parsed, err := normalizeSeedURL(lookupURL); err == nil {
		lookupURL = canonicalURL(parsed)
	}

	const query = `
SELECT id, url, host, title, summary, body_text, status_code, content_type, depth, crawled_at
FROM pages
WHERE url = ?
LIMIT 1
`

	var page PageDetails
	err := s.db.QueryRowContext(ctx, query, lookupURL).Scan(
		&page.ID,
		&page.URL,
		&page.Host,
		&page.Title,
		&page.Summary,
		&page.BodyText,
		&page.StatusCode,
		&page.ContentType,
		&page.Depth,
		&page.CrawledAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return PageDetails{}, fmt.Errorf("未找到 URL=%s 的记录", lookupURL)
		}
		return PageDetails{}, err
	}

	return page, nil
}

func (s *SQLiteStore) Close() error {
	if s == nil || s.db == nil {
		return nil
	}
	return s.db.Close()
}
