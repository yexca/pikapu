package config

import "testing"

func TestLoadModeDefaultsToProduction(t *testing.T) {
	t.Setenv("PIKAPU_MODE", "")
	cfg, err := Load()
	if err != nil || cfg.Development {
		t.Fatalf("development=%v err=%v, want production", cfg.Development, err)
	}

	t.Setenv("PIKAPU_MODE", "Development")
	if cfg, err := Load(); err != nil || !cfg.Development {
		t.Fatalf("development=%v err=%v", cfg.Development, err)
	}

	t.Setenv("PIKAPU_MODE", "dev")
	if _, err := Load(); err == nil {
		t.Fatal("an unknown mode was accepted")
	}
}

func TestParsePrefixes(t *testing.T) {
	got, err := parsePrefixes("10.0.0.0/8, 192.0.2.7 ::1")
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"10.0.0.0/8", "192.0.2.7/32", "::1/128"}
	if len(got) != len(want) {
		t.Fatalf("got %v", got)
	}
	for i := range want {
		if got[i].String() != want[i] {
			t.Errorf("prefix %d = %s, want %s", i, got[i], want[i])
		}
	}
	if _, err := parsePrefixes("proxy.example.com"); err == nil {
		t.Error("a host name was accepted")
	}
}
