package opml

import (
	"encoding/xml"
	"errors"
	"io"
	"strings"
	"time"

	"golang.org/x/net/html/charset"
)

// ErrInvalid reports a file that is not parseable OPML.
var ErrInvalid = errors.New("could not parse OPML file")

type document struct {
	XMLName xml.Name `xml:"opml"`
	Version string   `xml:"version,attr"`
	Head    head     `xml:"head"`
	Body    body     `xml:"body"`
}

type head struct {
	Title       string `xml:"title"`
	DateCreated string `xml:"dateCreated,omitempty"`
}

type body struct {
	Outlines []outline `xml:"outline"`
}

type outline struct {
	Text     string    `xml:"text,attr"`
	Title    string    `xml:"title,attr,omitempty"`
	Type     string    `xml:"type,attr,omitempty"`
	XMLURL   string    `xml:"xmlUrl,attr,omitempty"`
	HTMLURL  string    `xml:"htmlUrl,attr,omitempty"`
	Outlines []outline `xml:"outline"`
}

// Subscription is one feed found in (or written to) an OPML file.
type Subscription struct {
	Title    string
	FeedURL  string
	SiteURL  string
	Category string
}

// Group is a named folder of subscriptions; an empty name means top level.
type Group struct {
	Name          string
	Subscriptions []Subscription
}

func Parse(r io.Reader) ([]Subscription, error) {
	dec := xml.NewDecoder(r)
	dec.CharsetReader = charset.NewReaderLabel
	dec.Strict = false
	var doc document
	if err := dec.Decode(&doc); err != nil {
		return nil, ErrInvalid
	}
	var out []Subscription
	var walk func(items []outline, category string)
	walk = func(items []outline, category string) {
		for _, o := range items {
			if u := strings.TrimSpace(o.XMLURL); u != "" {
				title := strings.TrimSpace(o.Title)
				if title == "" {
					title = strings.TrimSpace(o.Text)
				}
				out = append(out, Subscription{Title: title, FeedURL: u, SiteURL: strings.TrimSpace(o.HTMLURL), Category: category})
				continue
			}
			name := strings.TrimSpace(o.Text)
			if name == "" {
				name = strings.TrimSpace(o.Title)
			}
			if name == "" {
				name = category
			}
			walk(o.Outlines, name)
		}
	}
	walk(doc.Body.Outlines, "")
	return out, nil
}

func Write(w io.Writer, title string, groups []Group) error {
	doc := document{
		Version: "2.0",
		Head:    head{Title: title, DateCreated: time.Now().UTC().Format(time.RFC1123Z)},
	}
	for _, g := range groups {
		subs := make([]outline, 0, len(g.Subscriptions))
		for _, s := range g.Subscriptions {
			subs = append(subs, outline{Text: s.Title, Title: s.Title, Type: "rss", XMLURL: s.FeedURL, HTMLURL: s.SiteURL})
		}
		if g.Name == "" {
			doc.Body.Outlines = append(doc.Body.Outlines, subs...)
			continue
		}
		doc.Body.Outlines = append(doc.Body.Outlines, outline{Text: g.Name, Title: g.Name, Outlines: subs})
	}
	if _, err := io.WriteString(w, xml.Header); err != nil {
		return err
	}
	enc := xml.NewEncoder(w)
	enc.Indent("", "  ")
	return enc.Encode(doc)
}
