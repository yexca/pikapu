// Package filter matches new entries against the user's keyword filters.
//
// Keywords are case-insensitive. A keyword edge that is a letter or digit of
// a space-separated script must sit at a word boundary, so "AI" matches
// "AI tools" and "AI-generated" but not "said". Chinese, Japanese, and Korean
// have no such boundaries, so keywords in those scripts match anywhere.
package filter

import (
	"math"
	"strings"
	"unicode"
	"unicode/utf8"

	"pikapu/internal/fetcher"
	"pikapu/internal/store"
)

const (
	MaxKeywords      = 50
	MaxKeywordLength = 100
)

// CleanKeywords trims and collapses whitespace, drops empty entries and
// case-insensitive duplicates, and reports whether the result is usable:
// 1 to MaxKeywords keywords of at most MaxKeywordLength characters each.
func CleanKeywords(in []string) ([]string, bool) {
	out := make([]string, 0, len(in))
	seen := map[string]bool{}
	for _, kw := range in {
		kw = strings.Join(strings.Fields(kw), " ")
		if kw == "" {
			continue
		}
		if utf8.RuneCountInString(kw) > MaxKeywordLength {
			return nil, false
		}
		if key := strings.ToLower(kw); !seen[key] {
			seen[key] = true
			out = append(out, kw)
		}
	}
	return out, len(out) > 0 && len(out) <= MaxKeywords
}

type rule struct {
	keywords []string // lower-cased
	content  bool
	invert   bool
	verdict  store.Verdict
}

// Set is a compiled list of filters.
type Set struct {
	rules []rule
}

// Compile prepares filters for matching. Filters with an unknown action are
// ignored.
func Compile(filters []*store.Filter) *Set {
	s := &Set{}
	for _, f := range filters {
		r := rule{content: f.MatchContent, invert: f.Invert}
		switch f.Action {
		case store.FilterMarkRead:
			r.verdict = store.KeepRead
		case store.FilterSkip:
			r.verdict = store.Drop
		default:
			continue
		}
		for _, kw := range f.Keywords {
			if kw = strings.ToLower(strings.TrimSpace(kw)); kw != "" {
				r.keywords = append(r.keywords, kw)
			}
		}
		if len(r.keywords) > 0 {
			s.rules = append(s.rules, r)
		}
	}
	return s
}

// Empty reports whether the set has no rules.
func (s *Set) Empty() bool { return s == nil || len(s.rules) == 0 }

// Triage returns Drop if any skip rule applies, otherwise KeepRead if any
// mark-as-read rule applies, otherwise Keep.
func (s *Set) Triage(it store.EntryInput) store.Verdict {
	if s.Empty() {
		return store.Keep
	}
	title := strings.ToLower(it.Title)
	var body *string
	verdict := store.Keep
	for _, r := range s.rules {
		if verdict == r.verdict {
			continue
		}
		matched := containsAny(title, r.keywords)
		if !matched && r.content {
			if body == nil {
				text := strings.ToLower(fetcher.PlainText(it.Content, math.MaxInt32))
				body = &text
			}
			matched = containsAny(*body, r.keywords)
		}
		if matched == r.invert {
			continue
		}
		if r.verdict == store.Drop {
			return store.Drop
		}
		verdict = r.verdict
	}
	return verdict
}

func containsAny(text string, keywords []string) bool {
	for _, kw := range keywords {
		if containsKeyword(text, kw) {
			return true
		}
	}
	return false
}

// containsKeyword reports whether kw occurs in text with word boundaries
// where the keyword's edges require them. Both are lower-case.
func containsKeyword(text, kw string) bool {
	first, _ := utf8.DecodeRuneInString(kw)
	last, _ := utf8.DecodeLastRuneInString(kw)
	needStart, needEnd := isWordRune(first), isWordRune(last)

	for from := 0; from <= len(text); {
		i := strings.Index(text[from:], kw)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(kw)
		before, _ := utf8.DecodeLastRuneInString(text[:start])
		after, _ := utf8.DecodeRuneInString(text[end:])
		if (!needStart || !isWordRune(before)) && (!needEnd || !isWordRune(after)) {
			return true
		}
		_, size := utf8.DecodeRuneInString(text[start:])
		from = start + size
	}
	return false
}

// isWordRune reports whether r is part of a word in a script that separates
// words with spaces.
func isWordRune(r rune) bool {
	if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
		return false
	}
	return !unicode.In(r, unicode.Han, unicode.Hiragana, unicode.Katakana, unicode.Hangul)
}
