//go:build e2e

// This executable exists only for deterministic browser tests. It is never
// built by the production Dockerfile or cmd/api. Production has no auth bypass.
package main

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
	"waypoint/internal/domain"
	"waypoint/internal/httpapi"
	"waypoint/internal/store"
	"waypoint/migrations"
)

type testAuth struct{}

func (testAuth) Verify(_ context.Context, token string) (string, error) {
	if token == "e2e-user-a" || token == "e2e-user-b" {
		return token, nil
	}
	return "", errors.New("invalid test session")
}

type testPlaces struct{}

var fixtures = []domain.Place{
	{Provider: "google", ProviderPlaceID: "tokyo-museum", Name: "Tokyo National Museum", Address: "Ueno Park, Tokyo", Location: &domain.Location{Lat: 35.7188, Lng: 139.7765}, ExternalURL: "https://maps.google.com/", Attributions: []domain.Attribution{}},
	{Provider: "google", ProviderPlaceID: "sensoji", Name: "Sensō-ji", Address: "Asakusa, Tokyo", Location: &domain.Location{Lat: 35.7148, Lng: 139.7967}, ExternalURL: "https://maps.google.com/", Attributions: []domain.Attribution{}},
}

func (testPlaces) Search(ctx context.Context, q, destination string) ([]domain.Place, error) {
	switch q {
	case "outage":
		return nil, &domain.Error{Status: 503, Code: "PLACES_UNAVAILABLE", Message: "Google Places is temporarily unavailable. Try again."}
	case "nothing":
		return []domain.Place{}, nil
	}
	return fixtures, nil
}
func (testPlaces) Details(ctx context.Context, id string) (domain.Place, error) {
	for _, p := range fixtures {
		if p.ProviderPlaceID == id {
			return p, nil
		}
	}
	return domain.Place{}, errors.New("unavailable")
}
func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}
func run() error {
	raw := os.Getenv("TEST_DATABASE_URL")
	if raw == "" {
		return errors.New("TEST_DATABASE_URL is required")
	}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	cfg, err := pgxpool.ParseConfig(raw)
	if err != nil {
		return err
	}
	if !strings.HasSuffix(cfg.ConnConfig.Database, "_test") {
		return errors.New("E2E requires a database name ending in _test")
	}
	admin, err := pgxpool.New(ctx, raw)
	if err != nil {
		return err
	}
	defer admin.Close()
	schema := "e2e_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err = admin.Exec(ctx, `CREATE SCHEMA `+schema); err != nil {
		return err
	}
	defer func() { _, _ = admin.Exec(context.Background(), `DROP SCHEMA `+schema+` CASCADE`) }()
	cfg.ConnConfig.RuntimeParams["search_path"] = schema
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return err
	}
	defer pool.Close()
	if err = migrations.Up(ctx, pool); err != nil {
		return err
	}
	api := &httpapi.API{Store: &store.Store{Pool: pool}, Auth: testAuth{}, Places: testPlaces{}, Origin: "http://localhost:3100"}
	server := &http.Server{Addr: "127.0.0.1:18080", Handler: api.Router(), ReadHeaderTimeout: 5 * time.Second}
	errs := make(chan error, 1)
	go func() { fmt.Println("E2E API ready on 18080"); errs <- server.ListenAndServe() }()
	select {
	case err = <-errs:
		return err
	case <-ctx.Done():
		shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		return server.Shutdown(shutdown)
	}
}
