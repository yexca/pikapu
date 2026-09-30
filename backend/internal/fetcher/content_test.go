package fetcher

import (
	"net/url"
	"strings"
	"testing"
)

func TestSanitizeHTML(t *testing.T) {
	base, _ := url.Parse("https://example.com/posts/1")
	cases := []struct {
		name     string
		in       string
		want     []string
		dontWant []string
	}{
		{
			name:     "strips scripts and handlers",
			in:       `<p onclick="x()">hi</p><script>alert(1)</script>`,
			want:     []string{"<p>hi</p>"},
			dontWant: []string{"script", "onclick"},
		},
		{
			name: "resolves relative urls and adds lazy loading",
			in:   `<p><a href="/about">a</a><img src="img/a.png"></p>`,
			want: []string{
				`href="https://example.com/about"`,
				`src="https://example.com/posts/img/a.png"`,
				`loading="lazy"`,
				`target="_blank"`,
			},
		},
		{
			name:     "restores lazy images",
			in:       `<img src="data:image/gif;base64,R0lGOD" data-src="https://cdn.example.com/real.jpg">`,
			want:     []string{`src="https://cdn.example.com/real.jpg"`},
			dontWant: []string{"R0lGOD"},
		},
		{
			name:     "drops tracking pixels",
			in:       `<p>text</p><img src="https://t.example.com/p.gif" width="1" height="1">`,
			dontWant: []string{"<img"},
		},
		{
			name: "keeps allowed embeds",
			in:   `<iframe src="https://www.youtube.com/embed/abc123"></iframe>`,
			want: []string{`<iframe src="https://www.youtube.com/embed/abc123"`},
		},
		{
			name:     "turns unknown embeds into links",
			in:       `<iframe src="https://evil.example.com/x"></iframe>`,
			want:     []string{`<a href="https://evil.example.com/x"`},
			dontWant: []string{"<iframe"},
		},
		{
			name: "wraps plain text in paragraphs",
			in:   "line one\nline two\n\nsecond a < b",
			want: []string{"<p>line one<br>line two</p>", "<p>second a &lt; b</p>"},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := SanitizeHTML(tc.in, base)
			for _, w := range tc.want {
				if !strings.Contains(got, w) {
					t.Errorf("missing %q in %q", w, got)
				}
			}
			for _, w := range tc.dontWant {
				if strings.Contains(got, w) {
					t.Errorf("unexpected %q in %q", w, got)
				}
			}
		})
	}
}

func TestPlainText(t *testing.T) {
	got := PlainText("<p>Hello <b>world</b></p><p>Second&nbsp;para</p>", 100)
	if got != "Hello world Second para" && got != "Hello world Second para" {
		t.Errorf("got %q", got)
	}
	if got := PlainText("一二三四五六", 3); got != "一二三…" {
		t.Errorf("truncate: got %q", got)
	}
}

func TestCleanText(t *testing.T) {
	cases := map[string]string{
		"Tom &amp; Jerry":         "Tom & Jerry",
		"<b>Bold</b> title":       "Bold title",
		"  spaced \n  out  ":      "spaced out",
		"a < b and c > d":         "a < b and c > d",
		"Use the &lt;div&gt; tag": "Use the <div> tag",
	}
	for in, want := range cases {
		if got := CleanText(in); got != want {
			t.Errorf("CleanText(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestDropTitleHeading(t *testing.T) {
	in := `<h1>Blog</h1><h1>My Post</h1><p>Body</p>`
	got := dropTitleHeading(in, "My Post")
	if strings.Contains(got, "My Post") || !strings.Contains(got, "<h1>Blog</h1>") {
		t.Errorf("got %q", got)
	}
	if got := dropTitleHeading(in, "Other"); got != in {
		t.Errorf("unrelated title changed content: %q", got)
	}
}

func TestNormalizeURL(t *testing.T) {
	cases := map[string]string{
		"example.com/feed":                "https://example.com/feed",
		" http://feed.example.com/rss#x ": "http://feed.example.com/rss",
		"feed://example.com/rss":          "https://example.com/rss",
		"feed:https://example.com":        "https://example.com",
	}
	for in, want := range cases {
		got, err := NormalizeURL(in)
		if err != nil || got != want {
			t.Errorf("NormalizeURL(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	for _, bad := range []string{"", "ftp://x.com", "https://"} {
		if _, err := NormalizeURL(bad); err == nil {
			t.Errorf("NormalizeURL(%q) should fail", bad)
		}
	}
}

func TestFindFeedLinks(t *testing.T) {
	base, _ := url.Parse("https://example.com/blog/")
	page := `<html><head>
		<link rel="stylesheet" href="/s.css">
		<link rel="alternate" type="application/rss+xml" href="feed.xml">
		<link rel="alternate" type="application/atom+xml" href="https://other.com/atom">
		<link rel="alternate" hreflang="en" href="/en/">
	</head></html>`
	got := findFeedLinks([]byte(page), base)
	want := []string{"https://example.com/blog/feed.xml", "https://other.com/atom"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("got %v, want %v", got, want)
	}
}

// The app's CSP only allows frames from EmbedOrigins, so every embed the
// sanitizer keeps must come from one of them.
func TestEmbedOriginsCoverIframeAllowlist(t *testing.T) {
	embeds := []string{
		"https://www.youtube.com/embed/x",
		"https://www.youtube-nocookie.com/embed/x",
		"https://youtube.com/embed/x",
		"https://player.vimeo.com/video/1",
		"https://player.bilibili.com/player.html?bvid=x",
	}
	for _, src := range embeds {
		if !iframeRe.MatchString(src) {
			t.Errorf("%s: not allowed by iframeRe; update this test", src)
			continue
		}
		u, _ := url.Parse(src)
		origin := u.Scheme + "://" + u.Host
		found := false
		for _, o := range EmbedOrigins {
			found = found || o == origin
		}
		if !found {
			t.Errorf("%s is allowed by iframeRe but missing from EmbedOrigins", origin)
		}
	}
}
