package fetcher

import (
	"context"
	"errors"
	"net"
	"net/url"
)

// Failure codes for fetching a feed. They are stored with the feed and
// returned by the API so the UI can show a localized message; the English
// error text travels alongside as detail.
const (
	CodeInvalidURL = "invalid_url"
	CodeNotFound   = "feed_not_found"
	CodeTimeout    = "fetch_timeout"
	CodeDNS        = "fetch_dns"
	CodeHTTP       = "fetch_http"
	CodeParse      = "fetch_parse"
	CodeTooLarge   = "fetch_too_large"
	CodeCanceled   = "fetch_canceled"
	CodeNetwork    = "fetch_network"
)

// Error is a fetch failure with a stable code and an English description.
type Error struct {
	Code string
	Err  error
}

func (e *Error) Error() string { return e.Err.Error() }
func (e *Error) Unwrap() error { return e.Err }

var (
	ErrNoFeed     = &Error{Code: CodeNotFound, Err: errors.New("no RSS or Atom feed found")}
	ErrInvalidURL = &Error{Code: CodeInvalidURL, Err: errors.New("invalid URL")}
	errTooLarge   = &Error{Code: CodeTooLarge, Err: errors.New("response too large")}
)

// Classify reduces any fetch error to a failure code and a short English
// message without the request URL.
func Classify(err error) *Error {
	var fe *Error
	if errors.As(err, &fe) {
		return fe
	}
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	var dnsErr *net.DNSError
	var netErr net.Error
	switch {
	case errors.Is(err, context.DeadlineExceeded):
		return &Error{Code: CodeTimeout, Err: errors.New("request timed out")}
	case errors.As(err, &dnsErr):
		return &Error{Code: CodeDNS, Err: errors.New("DNS lookup failed: " + dnsErr.Name)}
	case errors.As(err, &netErr) && netErr.Timeout():
		return &Error{Code: CodeTimeout, Err: errors.New("request timed out")}
	case errors.Is(err, context.Canceled):
		return &Error{Code: CodeCanceled, Err: errors.New("request canceled")}
	}
	return &Error{Code: CodeNetwork, Err: err}
}
