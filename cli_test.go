package main

import (
	"strings"
	"testing"
)

func TestParseCommandBackwardsCompatibleCrawl(t *testing.T) {
	command, err := parseCommand([]string{"-url", "https://example.com"})
	if err != nil {
		t.Fatalf("parseCommand returned error: %v", err)
	}

	cfg, ok := command.(crawlConfig)
	if !ok {
		t.Fatalf("unexpected command type: %T", command)
	}
	if cfg.StartURL != "https://example.com" {
		t.Fatalf("unexpected StartURL: %q", cfg.StartURL)
	}
}

func TestParseCommandList(t *testing.T) {
	command, err := parseCommand([]string{"list", "-limit", "5", "-host", "example.com"})
	if err != nil {
		t.Fatalf("parseCommand returned error: %v", err)
	}

	cfg, ok := command.(listConfig)
	if !ok {
		t.Fatalf("unexpected command type: %T", command)
	}
	if cfg.Limit != 5 {
		t.Fatalf("unexpected limit: got %d want %d", cfg.Limit, 5)
	}
	if cfg.Host != "example.com" {
		t.Fatalf("unexpected host: %q", cfg.Host)
	}
}

func TestParseCommandHelp(t *testing.T) {
	command, err := parseCommand([]string{"help"})
	if err != nil {
		t.Fatalf("parseCommand returned error: %v", err)
	}

	if _, ok := command.(helpCommand); !ok {
		t.Fatalf("unexpected command type: %T", command)
	}
}

func TestParseCrawlFlagsRejectsNonPositiveMaxPages(t *testing.T) {
	_, err := parseCrawlFlags([]string{"-url", "https://example.com", "-max-pages", "0"})
	if err == nil {
		t.Fatal("expected parseCrawlFlags to reject non-positive max-pages")
	}
	if !strings.Contains(err.Error(), "-max-pages 必须大于 0") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseShowFlagsRequiresLookupTarget(t *testing.T) {
	_, err := parseShowFlags([]string{})
	if err == nil {
		t.Fatal("expected parseShowFlags to require -id or -url")
	}
	if !strings.Contains(err.Error(), "必须提供 -id 或 -url") {
		t.Fatalf("unexpected error: %v", err)
	}
}
