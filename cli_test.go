package main

import "testing"

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
