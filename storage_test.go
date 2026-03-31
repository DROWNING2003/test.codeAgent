package main

import (
	"context"
	"path/filepath"
	"testing"
	"time"
)

func TestSQLiteStoreListAndGet(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "crawler.db")

	store, err := OpenSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLiteStore returned error: %v", err)
	}
	defer store.Close()

	if err := store.Init(ctx); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	record := PageRecord{
		URL:         "https://example.com/posts/1",
		Host:        "example.com",
		Title:       "First Post",
		Summary:     "summary",
		BodyText:    "body",
		StatusCode:  200,
		ContentType: "text/html; charset=utf-8",
		Depth:       1,
		CrawledAt:   time.Now().UTC().Truncate(time.Second),
	}

	if err := store.SavePage(ctx, record); err != nil {
		t.Fatalf("SavePage returned error: %v", err)
	}

	pages, err := store.ListPages(ctx, ListPagesQuery{
		Limit: 10,
		Host:  "example.com",
	})
	if err != nil {
		t.Fatalf("ListPages returned error: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("unexpected page count: got %d want %d", len(pages), 1)
	}

	pageByID, err := store.GetPageByID(ctx, pages[0].ID)
	if err != nil {
		t.Fatalf("GetPageByID returned error: %v", err)
	}
	if pageByID.URL != record.URL {
		t.Fatalf("unexpected URL by ID: got %q want %q", pageByID.URL, record.URL)
	}

	pageByURL, err := store.GetPageByURL(ctx, record.URL)
	if err != nil {
		t.Fatalf("GetPageByURL returned error: %v", err)
	}
	if pageByURL.Title != record.Title {
		t.Fatalf("unexpected title by URL: got %q want %q", pageByURL.Title, record.Title)
	}
}

func TestSQLiteStoreSavePageUpdatesExistingURLAndNormalizesLookup(t *testing.T) {
	ctx := context.Background()
	dbPath := filepath.Join(t.TempDir(), "crawler.db")

	store, err := OpenSQLiteStore(dbPath)
	if err != nil {
		t.Fatalf("OpenSQLiteStore returned error: %v", err)
	}
	defer store.Close()

	if err := store.Init(ctx); err != nil {
		t.Fatalf("Init returned error: %v", err)
	}

	record := PageRecord{
		URL:         "https://example.com/posts/1?x=1&y=2",
		Host:        "example.com",
		Title:       "First Post",
		Summary:     "summary",
		BodyText:    "body",
		StatusCode:  200,
		ContentType: "text/html; charset=utf-8",
		Depth:       1,
		CrawledAt:   time.Now().UTC().Truncate(time.Second),
	}

	if err := store.SavePage(ctx, record); err != nil {
		t.Fatalf("SavePage returned error: %v", err)
	}

	updated := record
	updated.Title = "Updated Post"
	updated.Summary = "updated summary"
	updated.StatusCode = 201
	updated.Depth = 3
	updated.CrawledAt = updated.CrawledAt.Add(time.Minute)

	if err := store.SavePage(ctx, updated); err != nil {
		t.Fatalf("SavePage update returned error: %v", err)
	}

	page, err := store.GetPageByURL(ctx, "Example.com/posts/1?y=2&x=1#ignored")
	if err != nil {
		t.Fatalf("GetPageByURL returned error: %v", err)
	}
	if page.Title != updated.Title {
		t.Fatalf("unexpected updated title: got %q want %q", page.Title, updated.Title)
	}
	if page.StatusCode != updated.StatusCode {
		t.Fatalf("unexpected updated status: got %d want %d", page.StatusCode, updated.StatusCode)
	}
	if page.Depth != updated.Depth {
		t.Fatalf("unexpected updated depth: got %d want %d", page.Depth, updated.Depth)
	}

	pages, err := store.ListPages(ctx, ListPagesQuery{
		Limit: 10,
		Host:  "EXAMPLE.COM",
	})
	if err != nil {
		t.Fatalf("ListPages returned error: %v", err)
	}
	if len(pages) != 1 {
		t.Fatalf("unexpected page count after update: got %d want %d", len(pages), 1)
	}
}
