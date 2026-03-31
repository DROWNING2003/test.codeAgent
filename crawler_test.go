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
