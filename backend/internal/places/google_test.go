package places

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"waypoint/internal/domain"
)

type roundTrip func(*http.Request) (*http.Response, error)

func (f roundTrip) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
func TestGoogleContract(t *testing.T) {
	g := NewGoogle("test-only-key")
	g.Client.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
		if r.URL.Host != "places.googleapis.com" || r.URL.Path != "/v1/places:searchText" || r.Method != "POST" {
			t.Fatalf("unexpected target %s %s", r.Method, r.URL)
		}
		if r.Header.Get("X-Goog-FieldMask") != "places.id,places.displayName,places.formattedAddress,places.location,places.googleMapsUri,places.attributions" {
			t.Fatal("incorrect field mask")
		}
		body, _ := io.ReadAll(r.Body)
		if !strings.Contains(string(body), "museum in Tokyo") {
			t.Fatalf("missing destination: %s", body)
		}
		return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"places":[{"id":"test-id","displayName":{"text":"Museum"},"formattedAddress":"Tokyo","location":{"latitude":35,"longitude":139},"googleMapsUri":"https://maps.google.com/","attributions":[{"provider":"Example","providerUri":"https://example.com"}]}]}`))}, nil
	})
	out, err := g.Search(context.Background(), "museum", "Tokyo")
	if err != nil {
		t.Fatal(err)
	}
	if len(out) != 1 || out[0].Name != "Museum" || out[0].Location.Lat != 35 || len(out[0].Attributions) != 1 {
		t.Fatalf("bad normalization: %+v", out)
	}
}
func TestProviderFailures(t *testing.T) {
	for _, tc := range []struct {
		status int
		body   string
		code   string
	}{{200, `{}`, ""}, {429, `{}`, "PROVIDER_RATE_LIMIT"}, {404, `{}`, "PLACE_UNAVAILABLE"}, {500, `{}`, "PROVIDER_ERROR"}, {200, `broken`, "PROVIDER_ERROR"}} {
		g := NewGoogle("test")
		g.Client.Transport = roundTrip(func(r *http.Request) (*http.Response, error) {
			return &http.Response{StatusCode: tc.status, Body: io.NopCloser(strings.NewReader(tc.body))}, nil
		})
		out, err := g.Search(context.Background(), "museum", "Tokyo")
		if tc.code == "" {
			if err != nil || len(out) != 0 {
				t.Fatal(out, err)
			}
		} else {
			var e *domain.Error
			if !errors.As(err, &e) || e.Code != tc.code {
				t.Fatalf("%d: %v", tc.status, err)
			}
		}
	}
	g := NewGoogle("test")
	g.Client.Transport = roundTrip(func(r *http.Request) (*http.Response, error) { return nil, context.DeadlineExceeded })
	if _, err := g.Details(context.Background(), "test"); err == nil {
		t.Fatal("expected network failure")
	}
}
