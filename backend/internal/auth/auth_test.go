package auth

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"pikapu/internal/store"
)

func TestPasswordHash(t *testing.T) {
	hash, err := HashPassword("synthetic-password")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(hash, "$argon2id$v=19$m=19456,t=2,p=1$") {
		t.Fatalf("unexpected hash format %q", hash)
	}
	if !VerifyPassword(hash, "synthetic-password") {
		t.Error("correct password rejected")
	}
	if VerifyPassword(hash, "synthetic-passwore") {
		t.Error("wrong password accepted")
	}
	other, _ := HashPassword("synthetic-password")
	if other == hash {
		t.Error("hashes of the same password share a salt")
	}
	for _, bad := range []string{"", "plain", "$argon2id$v=19$m=1,t=1,p=1$$", strings.Replace(hash, "m=19456", "m=99999999", 1)} {
		if VerifyPassword(bad, "synthetic-password") {
			t.Errorf("malformed hash %q accepted", bad)
		}
	}
}

func TestCredentialRules(t *testing.T) {
	if got, ok := NormalizeUsername("  admin "); !ok || got != "admin" {
		t.Errorf("NormalizeUsername trims: got %q, %v", got, ok)
	}
	for _, bad := range []string{"", "   ", "a\x00b", strings.Repeat("u", 65)} {
		if _, ok := NormalizeUsername(bad); ok {
			t.Errorf("username %q accepted", bad)
		}
	}
	if !ValidPassword("12345678") || !ValidPassword("密码密码密码密码") {
		t.Error("8-character passwords rejected")
	}
	if ValidPassword("1234567") || ValidPassword(strings.Repeat("p", 129)) {
		t.Error("password length limits not enforced")
	}
	if !SameUsername(" Admin", "admin") || SameUsername("admin2", "admin") {
		t.Error("usernames compare case-insensitively and exactly otherwise")
	}
}

func TestLimiterBacksOffPerClient(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := newLimiter(func() time.Time { return now })

	for i := 0; i < freeFailures-1; i++ {
		l.Fail("198.51.100.1")
	}
	if wait := l.Allow("198.51.100.1"); wait != 0 {
		t.Fatalf("blocked after %d failures: %v", freeFailures-1, wait)
	}
	l.Fail("198.51.100.1")
	if wait := l.Allow("198.51.100.1"); wait < 30*time.Second || wait > 31*time.Second {
		t.Fatalf("first block = %v, want about 30s", wait)
	}
	if wait := l.Allow("198.51.100.2"); wait != 0 {
		t.Fatalf("another client is blocked: %v", wait)
	}

	now = now.Add(31 * time.Second)
	if wait := l.Allow("198.51.100.1"); wait != 0 {
		t.Fatalf("still blocked after the wait: %v", wait)
	}
	l.Fail("198.51.100.1")
	if wait := l.Allow("198.51.100.1"); wait < time.Minute {
		t.Fatalf("second block = %v, want it doubled", wait)
	}

	l.Succeed("198.51.100.1")
	if wait := l.Allow("198.51.100.1"); wait != 0 {
		t.Fatalf("success did not reset the client: %v", wait)
	}
}

func TestLimiterGlobalBudget(t *testing.T) {
	now := time.Unix(1_700_000_000, 0)
	l := newLimiter(func() time.Time { return now })
	for i := 0; i < globalBurst; i++ {
		l.Fail("203.0.113." + string(rune('a'+i%26)) + string(rune('a'+i/26)))
	}
	if wait := l.Allow("192.0.2.1"); wait == 0 {
		t.Fatal("a fresh client is allowed after the global budget is spent")
	}
	now = now.Add(globalRefill)
	if wait := l.Allow("192.0.2.1"); wait != 0 {
		t.Fatalf("budget did not refill: %v", wait)
	}
}

func openStore(t *testing.T) *store.Store {
	t.Helper()
	st, err := store.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { st.Close() })
	return st
}

func TestEnsureAccount(t *testing.T) {
	ctx := context.Background()
	st := openStore(t)

	if created, err := EnsureAccount(ctx, st, "", ""); err != nil || created {
		t.Fatalf("no password: created=%v err=%v", created, err)
	}
	if _, err := EnsureAccount(ctx, st, "", "short"); err == nil {
		t.Fatal("a short bootstrap password was accepted")
	}
	if created, err := EnsureAccount(ctx, st, "", "synthetic-password"); err != nil || !created {
		t.Fatalf("created=%v err=%v", created, err)
	}
	acct, err := st.GetAccount(ctx)
	if err != nil || acct.Username != "admin" || !VerifyPassword(acct.PasswordHash, "synthetic-password") {
		t.Fatalf("unexpected account %+v, %v", acct, err)
	}
	if created, err := EnsureAccount(ctx, st, "other", "another-password"); err != nil || created {
		t.Fatalf("existing account replaced: created=%v err=%v", created, err)
	}
}

func TestResetPassword(t *testing.T) {
	ctx := context.Background()
	st := openStore(t)
	if _, _, err := ResetPassword(ctx, st); !errors.Is(err, ErrNoAccount) {
		t.Fatalf("err = %v, want ErrNoAccount", err)
	}
	if _, err := EnsureAccount(ctx, st, "reader", "synthetic-password"); err != nil {
		t.Fatal(err)
	}
	if _, err := st.CreateSession(ctx, []byte("token-hash"), "", "", time.Now().Add(time.Hour)); err != nil {
		t.Fatal(err)
	}
	username, password, err := ResetPassword(ctx, st)
	if err != nil || username != "reader" || len(password) < 16 {
		t.Fatalf("username=%q password=%q err=%v", username, password, err)
	}
	acct, _ := st.GetAccount(ctx)
	if !VerifyPassword(acct.PasswordHash, password) {
		t.Error("new password does not verify")
	}
	if list, _ := st.ListSessions(ctx, time.Now()); len(list) != 0 {
		t.Errorf("sessions survived the reset: %d", len(list))
	}
}
