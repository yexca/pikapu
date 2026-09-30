package config

import (
	"fmt"
	"net/netip"
	"os"
	"strings"
)

// Config holds process-level settings read from the environment.
// Settings that users change at runtime (refresh interval, retention)
// live in the database instead.
type Config struct {
	Addr    string
	DataDir string
	// Development disables sign-in. Production is the default.
	Development bool
	// AdminUsername and AdminPassword create the admin account on first
	// start. They are ignored once an account exists.
	AdminUsername string
	AdminPassword string
	// TrustedProxies are the reverse proxies whose X-Forwarded-For header
	// is believed when determining the client address.
	TrustedProxies []netip.Prefix
	// LegacyPassword is set when the removed PIKAPU_PASSWORD is still
	// configured, so startup can point to its replacement.
	LegacyPassword bool
}

func Load() (Config, error) {
	cfg := Config{
		Addr:           env("PIKAPU_ADDR", ":7660"),
		DataDir:        env("PIKAPU_DATA_DIR", "./data"),
		AdminUsername:  strings.TrimSpace(os.Getenv("PIKAPU_ADMIN_USERNAME")),
		AdminPassword:  os.Getenv("PIKAPU_ADMIN_PASSWORD"),
		LegacyPassword: os.Getenv("PIKAPU_PASSWORD") != "",
	}
	switch mode := strings.ToLower(env("PIKAPU_MODE", "production")); mode {
	case "production":
	case "development":
		cfg.Development = true
	default:
		return cfg, fmt.Errorf("PIKAPU_MODE must be production or development, not %q", mode)
	}
	proxies, err := parsePrefixes(os.Getenv("PIKAPU_TRUSTED_PROXIES"))
	if err != nil {
		return cfg, fmt.Errorf("PIKAPU_TRUSTED_PROXIES: %w", err)
	}
	cfg.TrustedProxies = proxies
	return cfg, nil
}

// parsePrefixes reads a comma- or space-separated list of IP addresses and
// CIDR ranges.
func parsePrefixes(s string) ([]netip.Prefix, error) {
	var out []netip.Prefix
	for _, field := range strings.FieldsFunc(s, func(r rune) bool { return r == ',' || r == ' ' }) {
		if strings.Contains(field, "/") {
			p, err := netip.ParsePrefix(field)
			if err != nil {
				return nil, err
			}
			out = append(out, p.Masked())
			continue
		}
		a, err := netip.ParseAddr(field)
		if err != nil {
			return nil, err
		}
		a = a.Unmap()
		out = append(out, netip.PrefixFrom(a, a.BitLen()))
	}
	return out, nil
}

func env(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}
