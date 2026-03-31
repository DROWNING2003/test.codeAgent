package main

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"unicode/utf8"
)

var (
	titleRe            = regexp.MustCompile(`(?is)<title[^>]*>(.*?)</title>`)
	scriptStyleRe      = regexp.MustCompile(`(?is)<script[^>]*>.*?</script>|<style[^>]*>.*?</style>|<noscript[^>]*>.*?</noscript>`)
	tagRe              = regexp.MustCompile(`(?s)<[^>]+>`)
	hrefRe             = regexp.MustCompile(`(?is)<a[^>]+href\s*=\s*(?:"([^"]*)"|'([^']*)'|([^\s>]+))`)
	repeatedWhitespace = regexp.MustCompile(`\s+`)
)

func extractPageData(baseURL *url.URL, rawHTML string) (string, string, string, []*url.URL) {
	title := extractTitle(rawHTML)
	bodyText := extractBodyText(rawHTML)
	summary := truncateRunes(bodyText, 240)
	links := extractLinks(baseURL, rawHTML)

	return title, summary, bodyText, links
}

func extractTitle(rawHTML string) string {
	matches := titleRe.FindStringSubmatch(rawHTML)
	if len(matches) < 2 {
		return ""
	}

	return normalizeSpace(html.UnescapeString(tagRe.ReplaceAllString(matches[1], " ")))
}

func extractBodyText(rawHTML string) string {
	noScript := scriptStyleRe.ReplaceAllString(rawHTML, " ")
	text := tagRe.ReplaceAllString(noScript, " ")
	text = html.UnescapeString(text)
	text = normalizeSpace(text)

	return truncateRunes(text, 20_000)
}

func extractLinks(baseURL *url.URL, rawHTML string) []*url.URL {
	matches := hrefRe.FindAllStringSubmatch(rawHTML, -1)
	seen := make(map[string]struct{}, len(matches))
	links := make([]*url.URL, 0, len(matches))

	for _, match := range matches {
		href := firstNonEmpty(match[1:]...)
		href = strings.TrimSpace(html.UnescapeString(href))
		if href == "" {
			continue
		}
		if strings.HasPrefix(href, "#") || strings.HasPrefix(strings.ToLower(href), "javascript:") {
			continue
		}

		parsed, err := url.Parse(href)
		if err != nil {
			continue
		}
		if baseURL != nil {
			parsed = baseURL.ResolveReference(parsed)
		}
		if parsed.Scheme != "http" && parsed.Scheme != "https" {
			continue
		}

		normalized := canonicalURL(parsed)
		if _, ok := seen[normalized]; ok {
			continue
		}
		seen[normalized] = struct{}{}
		links = append(links, parsed)
	}

	return links
}

func normalizeSpace(value string) string {
	return strings.TrimSpace(repeatedWhitespace.ReplaceAllString(value, " "))
}

func truncateRunes(value string, limit int) string {
	if limit <= 0 || utf8.RuneCountInString(value) <= limit {
		return value
	}

	runes := []rune(value)
	if limit <= 3 {
		return string(runes[:limit])
	}

	return string(runes[:limit-3]) + "..."
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
