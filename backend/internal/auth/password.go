// Package auth holds the admin account rules that do not depend on HTTP:
// password hashing, credential validation, sign-in rate limiting, and
// creating or resetting the account.
package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"

	"golang.org/x/crypto/argon2"
)

// Argon2id parameters (OWASP baseline: 19 MiB, 2 passes, 1 lane). They are
// encoded in every hash, so they can be raised later without a migration.
const (
	argonMemory  = 19 * 1024
	argonTime    = 2
	argonThreads = 1
	argonKeyLen  = 32
	argonSaltLen = 16
)

const (
	MinPasswordLength = 8
	MaxPasswordLength = 128
	MaxUsernameLength = 64
)

// HashPassword returns a PHC-formatted argon2id hash of password.
func HashPassword(password string) (string, error) {
	salt := make([]byte, argonSaltLen)
	if _, err := rand.Read(salt); err != nil {
		return "", err
	}
	key := argon2.IDKey([]byte(password), salt, argonTime, argonMemory, argonThreads, argonKeyLen)
	b64 := base64.RawStdEncoding
	return fmt.Sprintf("$argon2id$v=%d$m=%d,t=%d,p=%d$%s$%s",
		argon2.Version, argonMemory, argonTime, argonThreads,
		b64.EncodeToString(salt), b64.EncodeToString(key)), nil
}

// VerifyPassword reports whether password matches encoded. A malformed
// hash never matches.
func VerifyPassword(encoded, password string) bool {
	parts := strings.Split(encoded, "$")
	if len(parts) != 6 || parts[1] != "argon2id" {
		return false
	}
	var version int
	if _, err := fmt.Sscanf(parts[2], "v=%d", &version); err != nil || version != argon2.Version {
		return false
	}
	var memory, passes uint32
	var threads uint8
	if _, err := fmt.Sscanf(parts[3], "m=%d,t=%d,p=%d", &memory, &passes, &threads); err != nil {
		return false
	}
	// Refuse parameters that would let a tampered hash exhaust memory or CPU.
	if memory == 0 || memory > 1<<20 || passes == 0 || passes > 16 || threads == 0 {
		return false
	}
	b64 := base64.RawStdEncoding
	salt, err := b64.DecodeString(parts[4])
	if err != nil {
		return false
	}
	want, err := b64.DecodeString(parts[5])
	if err != nil || len(want) == 0 || len(want) > 64 {
		return false
	}
	got := argon2.IDKey([]byte(password), salt, passes, memory, threads, uint32(len(want)))
	return subtle.ConstantTimeCompare(got, want) == 1
}

// NormalizeUsername trims the username and reports whether it is 1-64
// characters without control characters.
func NormalizeUsername(username string) (string, bool) {
	username = strings.TrimSpace(username)
	if username == "" || utf8.RuneCountInString(username) > MaxUsernameLength {
		return "", false
	}
	for _, r := range username {
		if unicode.IsControl(r) {
			return "", false
		}
	}
	return username, true
}

// ValidPassword reports whether a new password is 8-128 characters.
func ValidPassword(password string) bool {
	n := utf8.RuneCountInString(password)
	return utf8.ValidString(password) && n >= MinPasswordLength && n <= MaxPasswordLength
}

// SameUsername compares usernames case-insensitively.
func SameUsername(a, b string) bool {
	return strings.EqualFold(strings.TrimSpace(a), b)
}

// RandomToken returns n random bytes encoded as unpadded base64url.
func RandomToken(n int) string {
	b := make([]byte, n)
	// crypto/rand.Read never returns an error on supported platforms.
	_, _ = rand.Read(b)
	return base64.RawURLEncoding.EncodeToString(b)
}
