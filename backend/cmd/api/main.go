package main

import (
	"context"
	"errors"
	"github.com/jackc/pgx/v5/pgxpool"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
	"waypoint/internal/auth"
	"waypoint/internal/httpapi"
	"waypoint/internal/places"
	"waypoint/internal/store"
	"waypoint/migrations"
)

func main() {
	if err := run(); err != nil {
		slog.Error("startup failed", "error", err)
		os.Exit(1)
	}
}
func run() error {
	slog.SetDefault(slog.New(slog.NewJSONHandler(os.Stdout, nil)))
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	startup, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		return errors.New("DATABASE_URL is required")
	}
	pool, err := pgxpool.New(startup, databaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err = pool.Ping(startup); err != nil {
		return err
	}
	if err = migrations.Up(startup, pool); err != nil {
		return err
	}
	origin := os.Getenv("CORS_ALLOWED_ORIGIN")
	verifier, err := auth.NewClerk(os.Getenv("CLERK_SECRET_KEY"), os.Getenv("AUTH_ISSUER"), os.Getenv("AUTH_AUDIENCE"), origin)
	if err != nil {
		return err
	}
	api := &httpapi.API{Store: &store.Store{Pool: pool}, Auth: verifier, Places: places.NewGoogle(os.Getenv("GOOGLE_MAPS_SERVER_API_KEY")), Origin: origin}
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	server := &http.Server{Addr: ":" + port, Handler: api.Router(), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 15 * time.Second, WriteTimeout: 30 * time.Second, IdleTimeout: 60 * time.Second, MaxHeaderBytes: 16 << 10}
	stopped := make(chan error, 1)
	go func() { slog.Info("API listening", "port", port); stopped <- server.ListenAndServe() }()
	select {
	case err = <-stopped:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
		shutdown, done := context.WithTimeout(context.Background(), 10*time.Second)
		defer done()
		return server.Shutdown(shutdown)
	}
	return nil
}
