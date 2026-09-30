package main

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	"pikapu/internal/api"
	"pikapu/internal/auth"
	"pikapu/internal/buildinfo"
	"pikapu/internal/config"
	"pikapu/internal/fetcher"
	"pikapu/internal/service"
	"pikapu/internal/store"
	"pikapu/web"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "pikapu:", err)
		os.Exit(2)
	}

	if len(os.Args) > 1 {
		switch os.Args[1] {
		case "healthcheck":
			// Lets the container probe itself without curl/wget.
			os.Exit(healthcheck(cfg.Addr))
		case "reset-password":
			os.Exit(resetPassword(cfg))
		default:
			fmt.Fprintf(os.Stderr, "pikapu: unknown command %q (commands: healthcheck, reset-password)\n", os.Args[1])
			os.Exit(2)
		}
	}

	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo}))
	if err := run(cfg, log); err != nil {
		log.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run(cfg config.Config, log *slog.Logger) error {
	if err := os.MkdirAll(cfg.DataDir, 0o755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	st, err := store.Open(filepath.Join(cfg.DataDir, "pikapu.db"))
	if err != nil {
		return fmt.Errorf("open database: %w", err)
	}
	defer st.Close()

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if cfg.LegacyPassword {
		log.Warn("PIKAPU_PASSWORD is no longer supported and is ignored; " +
			"set PIKAPU_ADMIN_USERNAME and PIKAPU_ADMIN_PASSWORD or use the setup token")
	}
	if cfg.Development {
		log.Warn("development mode: sign-in is disabled and anyone who can reach this port has full access; " +
			"never use PIKAPU_MODE=development on a reachable instance")
	}
	created, err := auth.EnsureAccount(ctx, st, cfg.AdminUsername, cfg.AdminPassword)
	if err != nil {
		return fmt.Errorf("PIKAPU_ADMIN_PASSWORD: %w", err)
	}
	if created {
		log.Info("admin account created from the environment; you can remove PIKAPU_ADMIN_PASSWORD now")
	} else if cfg.AdminPassword != "" {
		log.Info("PIKAPU_ADMIN_PASSWORD is ignored because the admin account already exists")
	}

	svc := service.New(ctx, st, fetcher.New(), log)
	handler, err := api.New(ctx, st, svc, api.Options{
		Development:    cfg.Development,
		TrustedProxies: cfg.TrustedProxies,
	}, web.Dist(), log)
	if err != nil {
		return err
	}

	srv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           handler,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}

	go svc.Run()

	errCh := make(chan error, 1)
	go func() {
		log.Info("pikapu listening", "version", buildinfo.Version, "addr", cfg.Addr, "data", cfg.DataDir, "development", cfg.Development)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		return err
	case <-ctx.Done():
	}

	log.Info("shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return srv.Shutdown(shutdownCtx)
}

// resetPassword sets a random admin password and signs out every session.
// Run it inside the container: `docker exec pikapu pikapu reset-password`.
func resetPassword(cfg config.Config) int {
	st, err := store.Open(filepath.Join(cfg.DataDir, "pikapu.db"))
	if err != nil {
		fmt.Fprintln(os.Stderr, "pikapu: open database:", err)
		return 1
	}
	defer st.Close()
	username, password, err := auth.ResetPassword(context.Background(), st)
	if errors.Is(err, auth.ErrNoAccount) {
		fmt.Fprintln(os.Stderr, "pikapu: no admin account yet; open Pikapu and use the setup token from the log")
		return 1
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "pikapu:", err)
		return 1
	}
	fmt.Printf("Password reset. All sessions were signed out.\nUsername: %s\nPassword: %s\n", username, password)
	return 0
}

func healthcheck(addr string) int {
	host := addr
	if strings.HasPrefix(host, ":") {
		host = "127.0.0.1" + host
	}
	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Get("http://" + host + "/api/healthz")
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return 1
	}
	return 0
}
