package fetcher

import (
	"html"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/PuerkitoBio/goquery"
	"github.com/microcosm-cc/bluemonday"
)

var (
	htmlTagRe = regexp.MustCompile(`</?[a-zA-Z][a-zA-Z0-9-]*(\s[^<>]*)?/?>`)
	iframeRe  = regexp.MustCompile(`^https://(www\.youtube\.com/embed/|www\.youtube-nocookie\.com/embed/|youtube\.com/embed/|player\.vimeo\.com/video/|player\.bilibili\.com/player\.html)`)
	policy    = newPolicy()
)

func newPolicy() *bluemonday.Policy {
	p := bluemonday.UGCPolicy()
	p.AllowElements("picture", "figure", "figcaption", "mark", "details", "summary", "time")
	p.AllowAttrs("loading", "decoding", "srcset", "sizes").OnElements("img")
	p.AllowAttrs("src", "srcset", "sizes", "media", "type").OnElements("source")
	p.AllowAttrs("src", "poster", "controls", "preload", "width", "height", "loop", "muted", "playsinline").OnElements("video")
	p.AllowAttrs("src", "controls", "preload", "loop").OnElements("audio")
	p.AllowAttrs("src").Matching(iframeRe).OnElements("iframe")
	p.AllowAttrs("width", "height", "allowfullscreen", "frameborder", "allow").OnElements("iframe")
	p.AllowDataURIImages()
	p.RequireNoReferrerOnLinks(true)
	p.AddTargetBlankToFullyQualifiedLinks(true)
	return p
}

// lazyImageAttrs are attributes commonly used by lazy-loading scripts to hold
// the real image source while src points at a placeholder.
var lazyImageAttrs = []string{"data-src", "data-original", "data-lazy-src", "data-actualsrc", "data-echo", "data-url"}

// SanitizeHTML resolves relative URLs against base, restores lazily loaded
// images, and strips anything unsafe for rendering in the reader.
func SanitizeHTML(raw string, base *url.URL) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	if !htmlTagRe.MatchString(raw) {
		return textToHTML(raw)
	}

	doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw))
	if err == nil {
		rewrite(doc, base)
		if body, err := doc.Find("body").Html(); err == nil {
			raw = body
		}
	}
	return strings.TrimSpace(policy.Sanitize(raw))
}

func rewrite(doc *goquery.Document, base *url.URL) {
	doc.Find("img").Each(func(_ int, s *goquery.Selection) {
		src := strings.TrimSpace(s.AttrOr("src", ""))
		if src == "" || strings.HasPrefix(src, "data:") || strings.Contains(src, "placeholder") || strings.Contains(src, "blank.gif") {
			for _, attr := range lazyImageAttrs {
				if v := strings.TrimSpace(s.AttrOr(attr, "")); v != "" {
					src = v
					break
				}
			}
		}
		if v := s.AttrOr("data-srcset", ""); v != "" {
			s.SetAttr("srcset", v)
		}
		if isTrackingPixel(s) || src == "" {
			s.Remove()
			return
		}
		s.SetAttr("src", resolveURL(base, src))
		if v, ok := s.Attr("srcset"); ok {
			s.SetAttr("srcset", resolveSrcset(base, v))
		}
		s.SetAttr("loading", "lazy")
		s.SetAttr("decoding", "async")
	})

	resolveAttr(doc, "a[href]", "href", base)
	resolveAttr(doc, "source[src]", "src", base)
	resolveAttr(doc, "video[src]", "src", base)
	resolveAttr(doc, "video[poster]", "poster", base)
	resolveAttr(doc, "audio[src]", "src", base)
	doc.Find("source[srcset]").Each(func(_ int, s *goquery.Selection) {
		s.SetAttr("srcset", resolveSrcset(base, s.AttrOr("srcset", "")))
	})

	doc.Find("iframe").Each(func(_ int, s *goquery.Selection) {
		src := strings.TrimSpace(s.AttrOr("src", ""))
		if strings.HasPrefix(src, "//") {
			src = "https:" + src
		}
		src = resolveURL(base, src)
		if iframeRe.MatchString(src) {
			s.SetAttr("src", src)
			return
		}
		if src == "" {
			s.Remove()
			return
		}
		// Unknown embeds become a plain link rather than disappearing silently.
		s.ReplaceWithHtml(`<p><a href="` + html.EscapeString(src) + `">` + html.EscapeString(src) + `</a></p>`)
	})
}

func resolveAttr(doc *goquery.Document, selector, attr string, base *url.URL) {
	doc.Find(selector).Each(func(_ int, s *goquery.Selection) {
		s.SetAttr(attr, resolveURL(base, s.AttrOr(attr, "")))
	})
}

func resolveSrcset(base *url.URL, srcset string) string {
	parts := strings.Split(srcset, ",")
	for i, p := range parts {
		fields := strings.Fields(p)
		if len(fields) == 0 {
			continue
		}
		fields[0] = resolveURL(base, fields[0])
		parts[i] = strings.Join(fields, " ")
	}
	return strings.Join(parts, ", ")
}

func isTrackingPixel(s *goquery.Selection) bool {
	w, _ := strconv.Atoi(s.AttrOr("width", ""))
	h, _ := strconv.Atoi(s.AttrOr("height", ""))
	_, hasW := s.Attr("width")
	_, hasH := s.Attr("height")
	return hasW && hasH && w <= 1 && h <= 1
}

func textToHTML(text string) string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	var b strings.Builder
	for _, para := range strings.Split(text, "\n\n") {
		para = strings.TrimSpace(para)
		if para == "" {
			continue
		}
		b.WriteString("<p>")
		b.WriteString(strings.ReplaceAll(html.EscapeString(para), "\n", "<br>"))
		b.WriteString("</p>")
	}
	return b.String()
}

// PlainText extracts readable text from an HTML fragment, truncated to max runes.
func PlainText(raw string, max int) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	text := raw
	if htmlTagRe.MatchString(raw) {
		if doc, err := goquery.NewDocumentFromReader(strings.NewReader(raw)); err == nil {
			doc.Find("script, style, noscript, iframe").Remove()
			doc.Find("br").ReplaceWithHtml(" ")
			doc.Find("p, div, li, h1, h2, h3, h4, h5, h6, tr, td, blockquote, pre, figure, figcaption").AppendHtml(" ")
			text = doc.Text()
		}
	} else {
		text = html.UnescapeString(text)
	}
	return truncate(strings.Join(strings.Fields(text), " "), max)
}

// CleanText strips stray markup and entities from short strings like titles.
func CleanText(s string) string {
	if strings.ContainsAny(s, "<&") {
		s = htmlTagRe.ReplaceAllString(s, "")
		s = html.UnescapeString(s)
	}
	return strings.Join(strings.Fields(s), " ")
}

func truncate(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return strings.TrimSpace(string(r[:max])) + "…"
}

// dropTitleHeading removes a heading that merely repeats the entry title,
// which some feeds include at the top of their full-text content.
func dropTitleHeading(content, title string) string {
	if title == "" || !strings.Contains(content, "<h") {
		return content
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return content
	}
	removed := false
	doc.Find("h1, h2").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		if CleanText(s.Text()) == title {
			s.Remove()
			removed = true
			return false
		}
		return true
	})
	if !removed {
		return content
	}
	out, err := doc.Find("body").Html()
	if err != nil {
		return content
	}
	return strings.TrimSpace(out)
}

// firstImage returns the first usable image URL in already-sanitized HTML.
func firstImage(content string) string {
	if !strings.Contains(content, "<img") {
		return ""
	}
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(content))
	if err != nil {
		return ""
	}
	var out string
	doc.Find("img[src]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		src := s.AttrOr("src", "")
		if strings.HasPrefix(src, "http") && !strings.HasSuffix(strings.ToLower(src), ".svg") {
			out = src
			return false
		}
		return true
	})
	return out
}
