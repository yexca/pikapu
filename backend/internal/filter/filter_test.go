package filter

import (
	"slices"
	"strings"
	"testing"

	"pikapu/internal/store"
)

func TestContainsKeyword(t *testing.T) {
	tests := []struct {
		text, kw string
		want     bool
	}{
		{"ai tools for writers", "ai", true},
		{"what the author said", "ai", false},
		{"ai-generated images", "ai", true},
		{"the ai's answer", "ai", true},
		{"openai ships a model", "ai", false},
		{"said ai, again", "ai", true}, // a later occurrence can match
		{"sponsored: example product", "sponsored", true},
		{"sponsored: example product", "sponsor", false},
		{"weekly digest #12", "digest #12", true},
		{"c++ tips", "c++", true},
		{"ai绘画工具", "ai", true}, // CJK next to Latin is a boundary
		{"广告位招租", "广告", true},  // CJK keywords match anywhere
		{"示例课程限时优惠", "优惠", true},
		{"本期没有推广内容", "广告", false},
		{"2026 roadmap", "2026", true},
		{"roadmap 20261", "2026", false},
	}
	for _, tt := range tests {
		if got := containsKeyword(tt.text, tt.kw); got != tt.want {
			t.Errorf("containsKeyword(%q, %q) = %v, want %v", tt.text, tt.kw, got, tt.want)
		}
	}
}

func TestCleanKeywords(t *testing.T) {
	got, ok := CleanKeywords([]string{"  Sponsored ", "", "sponsored", "open \n source", "广告"})
	if !ok || !slices.Equal(got, []string{"Sponsored", "open source", "广告"}) {
		t.Fatalf("got %q, %v", got, ok)
	}
	if _, ok := CleanKeywords([]string{" ", ""}); ok {
		t.Error("only blank keywords should be rejected")
	}
	if _, ok := CleanKeywords([]string{strings.Repeat("x", MaxKeywordLength+1)}); ok {
		t.Error("an over-long keyword should be rejected")
	}
	many := make([]string, MaxKeywords+1)
	for i := range many {
		many[i] = strings.Repeat("k", i+1)
	}
	if _, ok := CleanKeywords(many); ok {
		t.Error("too many keywords should be rejected")
	}
}

func TestTriage(t *testing.T) {
	sponsored := &store.Filter{Keywords: []string{"Sponsored"}, Action: store.FilterSkip}
	digest := &store.Filter{Keywords: []string{"digest"}, Action: store.FilterMarkRead}
	inBody := &store.Filter{Keywords: []string{"广告"}, MatchContent: true, Action: store.FilterMarkRead}
	onlyGo := &store.Filter{Keywords: []string{"Go", "Rust"}, MatchContent: true, Invert: true, Action: store.FilterSkip}

	tests := []struct {
		name    string
		filters []*store.Filter
		item    store.EntryInput
		want    store.Verdict
	}{
		{"no filters", nil, store.EntryInput{Title: "Sponsored post"}, store.Keep},
		{"title match skips", []*store.Filter{sponsored}, store.EntryInput{Title: "SPONSORED: a product"}, store.Drop},
		{"no match keeps", []*store.Filter{sponsored}, store.EntryInput{Title: "Example release notes"}, store.Keep},
		{"title-only rule ignores content", []*store.Filter{sponsored},
			store.EntryInput{Title: "Example", Content: "<p>Sponsored by Example</p>"}, store.Keep},
		{"content match marks read", []*store.Filter{inBody},
			store.EntryInput{Title: "示例文章", Content: "<p>本文含<b>广告</b>内容</p>"}, store.KeepRead},
		{"markup is not matched", []*store.Filter{{Keywords: []string{"strong"}, MatchContent: true, Action: store.FilterSkip}},
			store.EntryInput{Title: "Example", Content: "<p><strong>Bold</strong> text</p>"}, store.Keep},
		{"skip wins over mark read", []*store.Filter{digest, sponsored},
			store.EntryInput{Title: "Sponsored weekly digest"}, store.Drop},
		{"mark read alone", []*store.Filter{digest, sponsored},
			store.EntryInput{Title: "Weekly digest #12"}, store.KeepRead},
		{"invert keeps matching items", []*store.Filter{onlyGo},
			store.EntryInput{Title: "Notes", Content: "<p>Why Go generics</p>"}, store.Keep},
		{"invert skips the rest", []*store.Filter{onlyGo},
			store.EntryInput{Title: "Example interview", Content: "<p>Nothing relevant</p>"}, store.Drop},
		{"unknown action is ignored", []*store.Filter{{Keywords: []string{"example"}, Action: "archive"}},
			store.EntryInput{Title: "Example"}, store.Keep},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Compile(tt.filters).Triage(tt.item); got != tt.want {
				t.Errorf("Triage = %v, want %v", got, tt.want)
			}
		})
	}
}
