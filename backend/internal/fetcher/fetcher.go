package fetcher

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/mmcdole/gofeed"
)

const (
	userAgent   = "Mozilla/5.0 (compatible; Pikapu/1.0; RSS Reader)"
	feedAccept  = "application/rss+xml, application/atom+xml, application/feed+json, application/xml;q=0.9, text/xml;q=0.9, */*;q=0.8"
	maxFeedSize = 20 << 20
	maxPageSize = 4 << 20
	maxIconSize = 512 << 10
)

type Fetcher struct {
	client *http.Client
}

func New() *Fetcher {
	transport := http.DefaultTransport.(*http.Transport).Clone()
	transport.MaxIdleConnsPerHost = 4
	transport.ResponseHeaderTimeout = 20 * time.Second
	return &Fetcher{client: &http.Client{Timeout: 30 * time.Second, Transport: transport}}
}

type response struct {
	URL          string
	Body         []byte
	ContentType  string
	ETag         string
	LastModified string
	NotModified  bool
}

func (f *Fetcher) get(ctx context.Context, rawURL, accept, etag, lastModified string, limit int64) (*response, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", userAgent)
	req.Header.Set("Accept", accept)
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	if lastModified != "" {
		req.Header.Set("If-Modified-Since", lastModified)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	out := &response{
		URL:          resp.Request.URL.String(),
		ContentType:  resp.Header.Get("Content-Type"),
		ETag:         resp.Header.Get("ETag"),
		LastModified: resp.Header.Get("Last-Modified"),
	}
	if resp.StatusCode == http.StatusNotModified {
		out.NotModified = true
		return out, nil
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, &Error{Code: CodeHTTP, Err: fmt.Errorf("HTTP %s", resp.Status)}
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, err
	}
	if int64(len(body)) > limit {
		return nil, errTooLarge
	}
	out.Body = body
	return out, nil
}

type FeedResult struct {
	Feed         *gofeed.Feed
	URL          string
	ETag         string
	LastModified string
	NotModified  bool
}

func parseFeed(body []byte) (*gofeed.Feed, error) {
	return gofeed.NewParser().Parse(bytes.NewReader(body))
}

// FetchFeed downloads and parses a feed, sending conditional request headers
// when etag or lastModified are known.
func (f *Fetcher) FetchFeed(ctx context.Context, feedURL, etag, lastModified string) (*FeedResult, error) {
	resp, err := f.get(ctx, feedURL, feedAccept, etag, lastModified, maxFeedSize)
	if err != nil {
		return nil, err
	}
	if resp.NotModified {
		return &FeedResult{URL: feedURL, ETag: etag, LastModified: lastModified, NotModified: true}, nil
	}
	feed, err := parseFeed(resp.Body)
	if err != nil {
		return nil, &Error{Code: CodeParse, Err: fmt.Errorf("could not parse feed: %w", err)}
	}
	return &FeedResult{Feed: feed, URL: resp.URL, ETag: resp.ETag, LastModified: resp.LastModified}, nil
}

// Discover resolves user input, which may be a feed URL or a regular web
// page, to a working feed.
func (f *Fetcher) Discover(ctx context.Context, input string) (*FeedResult, error) {
	pageURL, err := NormalizeURL(input)
	if err != nil {
		return nil, err
	}
	resp, err := f.get(ctx, pageURL, feedAccept+", text/html;q=0.8", "", "", maxFeedSize)
	if err != nil {
		return nil, err
	}
	if feed, err := parseFeed(resp.Body); err == nil {
		return &FeedResult{Feed: feed, URL: resp.URL, ETag: resp.ETag, LastModified: resp.LastModified}, nil
	}

	base, err := url.Parse(resp.URL)
	if err != nil {
		return nil, err
	}
	candidates := findFeedLinks(resp.Body, base)
	for _, p := range []string{"/feed", "/rss", "/feed.xml", "/rss.xml", "/atom.xml", "/index.xml"} {
		candidates = append(candidates, base.ResolveReference(&url.URL{Path: p}).String())
	}

	tried := map[string]bool{resp.URL: true}
	for _, c := range candidates {
		if tried[c] {
			continue
		}
		tried[c] = true
		if ctx.Err() != nil {
			break
		}
		if res, err := f.FetchFeed(ctx, c, "", ""); err == nil {
			return res, nil
		}
	}
	return nil, ErrNoFeed
}

var feedLinkTypes = []string{
	"application/rss+xml", "application/atom+xml", "application/feed+json",
	"application/json", "application/rdf+xml", "application/xml", "text/xml",
}

func findFeedLinks(body []byte, base *url.URL) []string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	if href, ok := doc.Find("base[href]").First().Attr("href"); ok {
		if u, err := base.Parse(href); err == nil {
			base = u
		}
	}
	var out []string
	doc.Find(`link[rel~="alternate"][href]`).Each(func(_ int, s *goquery.Selection) {
		typ := strings.ToLower(strings.TrimSpace(s.AttrOr("type", "")))
		for _, t := range feedLinkTypes {
			if typ == t {
				if u := resolveURL(base, s.AttrOr("href", "")); u != "" {
					out = append(out, u)
				}
				return
			}
		}
	})
	return out
}

// NormalizeURL turns loose user input like "example.com/feed" into an
// absolute http(s) URL.
func NormalizeURL(input string) (string, error) {
	s := strings.TrimSpace(input)
	switch {
	case strings.HasPrefix(s, "feed://"):
		s = "https://" + strings.TrimPrefix(s, "feed://")
	case strings.HasPrefix(s, "feed:"):
		s = strings.TrimPrefix(s, "feed:")
	}
	if !strings.Contains(s, "://") {
		s = "https://" + s
	}
	u, err := url.Parse(s)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return "", ErrInvalidURL
	}
	u.Fragment = ""
	return u.String(), nil
}

func resolveURL(base *url.URL, ref string) string {
	ref = strings.TrimSpace(ref)
	if ref == "" {
		return ""
	}
	if base == nil {
		return ref
	}
	u, err := base.Parse(ref)
	if err != nil {
		return ref
	}
	return u.String()
}
