package auth

import (
	"context"
	"errors"
	"fmt"

	"pikapu/internal/store"
)

// ErrNoAccount means the instance has not been set up yet.
var ErrNoAccount = errors.New("no admin account has been set up")

// EnsureAccount creates the admin account from bootstrap credentials when
// none exists yet. It reports whether an account was created; an existing
// account is never changed.
func EnsureAccount(ctx context.Context, st *store.Store, username, password string) (bool, error) {
	if password == "" {
		return false, nil
	}
	if username == "" {
		username = "admin"
	}
	name, ok := NormalizeUsername(username)
	if !ok {
		return false, fmt.Errorf("admin username must be 1-%d characters without control characters", MaxUsernameLength)
	}
	if !ValidPassword(password) {
		return false, fmt.Errorf("admin password must be %d-%d characters", MinPasswordLength, MaxPasswordLength)
	}
	if exists, err := st.HasAccount(ctx); err != nil || exists {
		return false, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return false, err
	}
	switch err := st.CreateAccount(ctx, name, hash); {
	case errors.Is(err, store.ErrConflict):
		return false, nil
	case err != nil:
		return false, err
	}
	return true, nil
}

// ResetPassword gives the admin account a new random password and signs
// out every session. It returns the username and the new password.
func ResetPassword(ctx context.Context, st *store.Store) (string, string, error) {
	acct, err := st.GetAccount(ctx)
	if errors.Is(err, store.ErrNotFound) {
		return "", "", ErrNoAccount
	}
	if err != nil {
		return "", "", err
	}
	password := RandomToken(15) // 20 characters
	hash, err := HashPassword(password)
	if err != nil {
		return "", "", err
	}
	if err := st.UpdateAccount(ctx, acct.Username, hash, 0); err != nil {
		return "", "", err
	}
	return acct.Username, password, nil
}
