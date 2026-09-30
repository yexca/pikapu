package auth

import (
	"sync"
	"time"
)

// Limiter slows down password guessing. Each client (usually an IP
// address) gets a few free failures, then must wait before trying again,
// with the wait doubling after every further failure. A global budget also
// caps failures across all clients so many addresses cannot share the work.
// State is in memory: Pikapu is a single instance, and a restart only
// resets the waits.
type Limiter struct {
	mu      sync.Mutex
	now     func() time.Time
	clients map[string]*client

	tokens float64
	filled time.Time
}

type client struct {
	failures int
	last     time.Time
	until    time.Time
}

const (
	freeFailures = 5
	firstBlock   = 30 * time.Second
	maxBlock     = 15 * time.Minute
	// A client is forgotten after this long without a failure.
	forgetAfter = time.Hour

	globalBurst  = 50
	globalRefill = 6 * time.Second // one failure every 6 s once the burst is spent
	pruneAbove   = 1024
)

func NewLimiter() *Limiter {
	return newLimiter(time.Now)
}

func newLimiter(now func() time.Time) *Limiter {
	return &Limiter{now: now, clients: map[string]*client{}, tokens: globalBurst, filled: now()}
}

// Allow returns 0 when key may attempt a sign-in, otherwise how long it
// must wait.
func (l *Limiter) Allow(key string) time.Duration {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.refill(now)
	var wait time.Duration
	if c := l.clients[key]; c != nil && now.Before(c.until) {
		wait = c.until.Sub(now)
	}
	if l.tokens < 1 {
		wait = max(wait, time.Duration((1-l.tokens)*float64(globalRefill)))
	}
	if wait > 0 {
		// Round up so a Retry-After of whole seconds is never too early.
		wait = wait.Truncate(time.Second) + time.Second
	}
	return wait
}

// Fail records a failed attempt by key.
func (l *Limiter) Fail(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	l.refill(now)
	l.tokens = max(l.tokens-1, 0)

	c := l.clients[key]
	if c == nil || now.Sub(c.last) > forgetAfter {
		if len(l.clients) >= pruneAbove {
			l.prune(now)
		}
		c = &client{}
		l.clients[key] = c
	}
	c.failures++
	c.last = now
	if n := c.failures - freeFailures; n >= 0 {
		block := maxBlock
		if n < 10 {
			block = min(firstBlock<<n, maxBlock)
		}
		c.until = now.Add(block)
	}
}

// Succeed forgets key's failures after a successful attempt.
func (l *Limiter) Succeed(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	delete(l.clients, key)
}

func (l *Limiter) refill(now time.Time) {
	if elapsed := now.Sub(l.filled); elapsed > 0 {
		l.tokens = min(l.tokens+float64(elapsed)/float64(globalRefill), globalBurst)
	}
	l.filled = now
}

func (l *Limiter) prune(now time.Time) {
	for k, c := range l.clients {
		if now.Sub(c.last) > forgetAfter && !now.Before(c.until) {
			delete(l.clients, k)
		}
	}
}
