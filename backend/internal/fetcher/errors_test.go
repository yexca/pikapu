package fetcher

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/url"
	"testing"
)

func TestClassify(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"deadline", &url.Error{Op: "Get", URL: "https://feed.example.invalid", Err: context.DeadlineExceeded}, CodeTimeout},
		{"dns", &url.Error{Op: "Get", URL: "https://feed.example.invalid", Err: &net.DNSError{Name: "feed.example.invalid"}}, CodeDNS},
		{"canceled", context.Canceled, CodeCanceled},
		{"coded passthrough", fmt.Errorf("wrapped: %w", ErrNoFeed), CodeNotFound},
		{"unknown", errors.New("connection reset"), CodeNetwork},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := Classify(tc.err)
			if got.Code != tc.code {
				t.Fatalf("code = %q, want %q", got.Code, tc.code)
			}
			if got.Error() == "" {
				t.Fatal("empty message")
			}
		})
	}
}
