package main

import "testing"

func TestNormalizeSeedURL(t *testing.T) {
	rawURL, err := normalizeSeedURL("example.com")
	if err != nil {
		t.Fatalf("normalizeSeedURL returned error: %v", err)
	}

	if got, want := rawURL.String(), "https://example.com"; got != want {
		t.Fatalf("unexpected normalized URL: got %q want %q", got, want)
	}
}

func TestCanonicalURL(t *testing.T) {
	rawURL, err := normalizeSeedURL("https://Example.com/a/../b?q=2&z=1#part")
	if err != nil {
		t.Fatalf("normalizeSeedURL returned error: %v", err)
	}

	if got, want := canonicalURL(rawURL), "https://example.com/b?q=2&z=1"; got != want {
		t.Fatalf("unexpected canonical URL: got %q want %q", got, want)
	}
}

func TestExtractPageData(t *testing.T) {
	rawURL, err := normalizeSeedURL("https://example.com/posts/welcome")
	if err != nil {
		t.Fatalf("normalizeSeedURL returned error: %v", err)
	}

	htmlDocument := `
<!doctype html>
<html>
  <head>
    <title>Welcome &amp; Hello</title>
    <style>body { display:none; }</style>
  </head>
  <body>
    <h1>First Post</h1>
    <p>This is a test page.</p>
    <a href="/next">next</a>
    <a href="https://example.com/about#team">about</a>
  </body>
</html>
`

	title, summary, bodyText, links := extractPageData(rawURL, htmlDocument)

	if title != "Welcome & Hello" {
		t.Fatalf("unexpected title: %q", title)
	}
	if bodyText == "" {
		t.Fatal("expected extracted body text")
	}
	if summary == "" {
		t.Fatal("expected summary")
	}
	if len(links) != 2 {
		t.Fatalf("unexpected link count: got %d want %d", len(links), 2)
	}
	if got, want := canonicalURL(links[0]), "https://example.com/next"; got != want {
		t.Fatalf("unexpected first link: got %q want %q", got, want)
	}
}

func TestNormalizeSeedURLRejectsUnsupportedScheme(t *testing.T) {
	if _, err := normalizeSeedURL("ftp://example.com"); err == nil {
		t.Fatal("expected normalizeSeedURL to reject unsupported schemes")
	}
}

func TestExtractLinksFiltersAndDeduplicates(t *testing.T) {
	rawURL, err := normalizeSeedURL("https://example.com/docs/index")
	if err != nil {
		t.Fatalf("normalizeSeedURL returned error: %v", err)
	}

	htmlDocument := `
<a href="/guide?q=2&z=1#part">guide</a>
<a href="https://example.com/guide?z=1&q=2">duplicate</a>
<a href="../about">about</a>
<a href="#local">fragment</a>
<a href="javascript:void(0)">js</a>
<a href="mailto:test@example.com">mail</a>
`

	links := extractLinks(rawURL, htmlDocument)
	if len(links) != 2 {
		t.Fatalf("unexpected link count: got %d want %d", len(links), 2)
	}

	if got, want := canonicalURL(links[0]), "https://example.com/guide?q=2&z=1"; got != want {
		t.Fatalf("unexpected first link: got %q want %q", got, want)
	}
	if got, want := canonicalURL(links[1]), "https://example.com/about"; got != want {
		t.Fatalf("unexpected second link: got %q want %q", got, want)
	}
}

func TestClipForTable(t *testing.T) {
	if got, want := clipForTable(" \n ", 10), "-"; got != want {
		t.Fatalf("unexpected blank table value: got %q want %q", got, want)
	}

	if got, want := clipForTable("line one\nline two", 10), "line on..."; got != want {
		t.Fatalf("unexpected clipped table value: got %q want %q", got, want)
	}
}
