package fetcher

import (
	"crypto/sha1"
	"encoding/hex"
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	ext "github.com/mmcdole/gofeed/extensions"

	"github.com/mmcdole/gofeed"

	"pikapu/internal/store"
)

const summaryLength = 200

var youtubeIDRe = regexp.MustCompile(`^https?://(?:www\.|m\.)?(?:youtube\.com/(?:watch\?v=|shorts/)|youtu\.be/)([A-Za-z0-9_-]{11})`)

// FeedMeta is the feed-level information worth storing.
type FeedMeta struct {
	Title       string
	SiteURL     string
	Description string
}

func Meta(feed *gofeed.Feed, feedURL string) FeedMeta {
	base, _ := url.Parse(feedURL)
	return FeedMeta{
		Title:       CleanText(feed.Title),
		SiteURL:     resolveURL(base, feed.Link),
		Description: PlainText(feed.Description, 300),
	}
}

// ConvertItems maps parsed feed items to storable entries.
func ConvertItems(feed *gofeed.Feed, feedURL string) []store.EntryInput {
	base, _ := url.Parse(feedURL)
	if feed.Link != "" {
		if u, err := base.Parse(feed.Link); err == nil {
			base = u
		}
	}
	now := time.Now().UTC()
	out := make([]store.EntryInput, 0, len(feed.Items))
	for _, it := range feed.Items {
		if it == nil {
			continue
		}
		out = append(out, convertItem(it, base, now))
	}
	return out
}

func convertItem(it *gofeed.Item, base *url.URL, now time.Time) store.EntryInput {
	link := resolveURL(base, it.Link)
	if link == "" && strings.HasPrefix(it.GUID, "http") {
		link = it.GUID
	}
	itemBase := base
	if u, err := url.Parse(link); err == nil && u.Host != "" {
		itemBase = u
	}

	rawContent := it.Content
	if strings.TrimSpace(rawContent) == "" {
		rawContent = it.Description
	}
	if strings.TrimSpace(rawContent) == "" {
		rawContent = mediaDescription(it)
	}
	if strings.TrimSpace(rawContent) == "" && it.ITunesExt != nil {
		rawContent = it.ITunesExt.Summary
	}
	content := dropTitleHeading(SanitizeHTML(rawContent, itemBase), CleanText(it.Title))
	if m := youtubeIDRe.FindStringSubmatch(link); m != nil {
		content = `<iframe src="https://www.youtube-nocookie.com/embed/` + m[1] +
			`" width="560" height="315" frameborder="0" allowfullscreen></iframe>` + content
	}

	summarySource := it.Description
	if strings.TrimSpace(summarySource) == "" {
		summarySource = rawContent
	}
	summary := PlainText(summarySource, summaryLength)

	title := CleanText(it.Title)
	if title == "" {
		title = truncate(summary, 80)
	}

	published := now
	if it.PublishedParsed != nil {
		published = *it.PublishedParsed
	} else if it.UpdatedParsed != nil {
		published = *it.UpdatedParsed
	}
	if published.After(now.Add(time.Hour)) {
		published = now
	}

	guid := strings.TrimSpace(it.GUID)
	if guid == "" {
		guid = link
	}
	if guid == "" {
		sum := sha1.Sum([]byte(it.Title + "|" + it.Published + "|" + truncate(it.Description, 200)))
		guid = "sha1:" + hex.EncodeToString(sum[:])
	}

	image := itemImage(it, itemBase)
	if image == "" {
		image = firstImage(content)
	}

	return store.EntryInput{
		GUID:        guid,
		URL:         link,
		Title:       title,
		Author:      itemAuthor(it),
		Summary:     summary,
		Content:     content,
		ImageURL:    image,
		PublishedAt: published.UTC(),
	}
}

func itemAuthor(it *gofeed.Item) string {
	for _, a := range it.Authors {
		if a == nil {
			continue
		}
		if name := CleanText(a.Name); name != "" {
			return name
		}
	}
	if it.DublinCoreExt != nil && len(it.DublinCoreExt.Creator) > 0 {
		return CleanText(strings.Join(it.DublinCoreExt.Creator, ", "))
	}
	if it.ITunesExt != nil && it.ITunesExt.Author != "" {
		return CleanText(it.ITunesExt.Author)
	}
	return ""
}

func itemImage(it *gofeed.Item, base *url.URL) string {
	if it.Image != nil && it.Image.URL != "" {
		return resolveURL(base, it.Image.URL)
	}
	if u := mediaImage(it.Extensions); u != "" {
		return resolveURL(base, u)
	}
	for _, enc := range it.Enclosures {
		if enc != nil && strings.HasPrefix(enc.Type, "image/") && enc.URL != "" {
			return resolveURL(base, enc.URL)
		}
	}
	if it.ITunesExt != nil && it.ITunesExt.Image != "" {
		return resolveURL(base, it.ITunesExt.Image)
	}
	return ""
}

// mediaImage looks for Media RSS thumbnails, including those nested in media:group.
func mediaImage(exts ext.Extensions) string {
	media, ok := exts["media"]
	if !ok {
		return ""
	}
	if u := mediaImageIn(media); u != "" {
		return u
	}
	for _, g := range media["group"] {
		if u := mediaImageIn(g.Children); u != "" {
			return u
		}
	}
	return ""
}

func mediaImageIn(m map[string][]ext.Extension) string {
	for _, t := range m["thumbnail"] {
		if u := t.Attrs["url"]; u != "" {
			return u
		}
	}
	for _, c := range m["content"] {
		if c.Attrs["medium"] == "image" || strings.HasPrefix(c.Attrs["type"], "image/") {
			if u := c.Attrs["url"]; u != "" {
				return u
			}
		}
		for _, t := range c.Children["thumbnail"] {
			if u := t.Attrs["url"]; u != "" {
				return u
			}
		}
	}
	return ""
}

// mediaDescription returns media:description (used by YouTube feeds) as HTML.
func mediaDescription(it *gofeed.Item) string {
	media, ok := it.Extensions["media"]
	if !ok {
		return ""
	}
	find := func(m map[string][]ext.Extension) string {
		for _, d := range m["description"] {
			if v := strings.TrimSpace(d.Value); v != "" {
				return v
			}
		}
		return ""
	}
	text := find(media)
	if text == "" {
		for _, g := range media["group"] {
			if text = find(g.Children); text != "" {
				break
			}
		}
	}
	if text == "" {
		return ""
	}
	if htmlTagRe.MatchString(text) {
		return text
	}
	return textToHTML(html.UnescapeString(text))
}
