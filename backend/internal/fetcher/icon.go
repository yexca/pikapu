package fetcher

import (
	"bytes"
	"context"
	"encoding/base64"
	"errors"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"

	"github.com/PuerkitoBio/goquery"
)

var ErrNoIcon = errors.New("no icon found")

// FetchIcon finds a favicon for the site behind a feed.
func (f *Fetcher) FetchIcon(ctx context.Context, siteURL, feedURL string) ([]byte, string, error) {
	page := siteURL
	if page == "" {
		u, err := url.Parse(feedURL)
		if err != nil {
			return nil, "", err
		}
		page = u.Scheme + "://" + u.Host + "/"
	}

	var candidates []string
	if resp, err := f.get(ctx, page, "text/html,*/*;q=0.8", "", "", maxPageSize); err == nil {
		page = resp.URL
		if base, err := url.Parse(resp.URL); err == nil {
			candidates = findIconLinks(resp.Body, base)
		}
	}
	for _, p := range []string{page, feedURL} {
		if u, err := url.Parse(p); err == nil && u.Host != "" {
			candidates = append(candidates, u.Scheme+"://"+u.Host+"/favicon.ico")
		}
	}

	tried := map[string]bool{}
	for _, c := range candidates {
		if tried[c] || ctx.Err() != nil {
			continue
		}
		tried[c] = true
		if data, mime, err := f.fetchImage(ctx, c); err == nil {
			return data, mime, nil
		}
	}
	return nil, "", ErrNoIcon
}

func findIconLinks(body []byte, base *url.URL) []string {
	doc, err := goquery.NewDocumentFromReader(bytes.NewReader(body))
	if err != nil {
		return nil
	}
	type candidate struct {
		href  string
		score int
	}
	var list []candidate
	doc.Find("link[rel][href]").Each(func(_ int, s *goquery.Selection) {
		rel := strings.Fields(strings.ToLower(s.AttrOr("rel", "")))
		isIcon, isTouch := false, false
		for _, r := range rel {
			switch r {
			case "icon":
				isIcon = true
			case "apple-touch-icon", "apple-touch-icon-precomposed":
				isTouch = true
			}
		}
		if !isIcon && !isTouch {
			return
		}
		href := resolveURL(base, s.AttrOr("href", ""))
		if href == "" {
			return
		}
		score := 2
		switch {
		case isTouch:
			score = 3
		case strings.Contains(s.AttrOr("type", ""), "svg") || strings.HasSuffix(strings.ToLower(href), ".svg"):
			score = 4
		case iconSize(s.AttrOr("sizes", "")) >= 32:
			score = 5
		case iconSize(s.AttrOr("sizes", "")) > 0:
			score = 1
		}
		list = append(list, candidate{href, score})
	})
	sort.SliceStable(list, func(i, j int) bool { return list[i].score > list[j].score })
	out := make([]string, len(list))
	for i, c := range list {
		out[i] = c.href
	}
	return out
}

func iconSize(sizes string) int {
	best := 0
	for _, s := range strings.Fields(strings.ToLower(sizes)) {
		w, _, ok := strings.Cut(s, "x")
		if !ok {
			continue
		}
		if n, err := strconv.Atoi(w); err == nil && n > best {
			best = n
		}
	}
	return best
}

func (f *Fetcher) fetchImage(ctx context.Context, src string) ([]byte, string, error) {
	if strings.HasPrefix(src, "data:") {
		return decodeDataURI(src)
	}
	resp, err := f.get(ctx, src, "image/*,*/*;q=0.8", "", "", maxIconSize)
	if err != nil {
		return nil, "", err
	}
	if len(resp.Body) == 0 {
		return nil, "", ErrNoIcon
	}
	return detectImageType(resp.Body, resp.ContentType)
}

func detectImageType(data []byte, header string) ([]byte, string, error) {
	sniffed := http.DetectContentType(data)
	switch {
	case strings.HasPrefix(sniffed, "image/"):
		return data, sniffed, nil
	case bytes.Contains(bytes.ToLower(data[:min(len(data), 1024)]), []byte("<svg")):
		return data, "image/svg+xml", nil
	case strings.HasPrefix(header, "image/") && !strings.HasPrefix(sniffed, "text/html"):
		mime, _, _ := strings.Cut(header, ";")
		return data, strings.TrimSpace(mime), nil
	}
	return nil, "", ErrNoIcon
}

func decodeDataURI(src string) ([]byte, string, error) {
	meta, payload, ok := strings.Cut(strings.TrimPrefix(src, "data:"), ",")
	if !ok {
		return nil, "", ErrNoIcon
	}
	var data []byte
	if strings.HasSuffix(meta, ";base64") {
		var err error
		if data, err = base64.StdEncoding.DecodeString(payload); err != nil {
			return nil, "", ErrNoIcon
		}
	} else {
		unescaped, err := url.PathUnescape(payload)
		if err != nil {
			return nil, "", ErrNoIcon
		}
		data = []byte(unescaped)
	}
	if len(data) == 0 || len(data) > maxIconSize {
		return nil, "", ErrNoIcon
	}
	mime, _, _ := strings.Cut(meta, ";")
	return detectImageType(data, mime)
}
