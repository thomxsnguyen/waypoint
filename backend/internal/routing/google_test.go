package routing

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"reflect"
	"strings"
	"testing"
	"time"
	"waypoint/internal/domain"
)

type transport func(*http.Request) (*http.Response, error)

func (f transport) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

const validMatrix = `[{"originIndex":1,"destinationIndex":0,"condition":"ROUTE_EXISTS","duration":"120.1s"},{"condition":"ROUTE_EXISTS","duration":"0s"},{"originIndex":1,"destinationIndex":1,"condition":"ROUTE_EXISTS","duration":"0s"},{"destinationIndex":1,"condition":"ROUTE_EXISTS","duration":"600s"}]`

func mockRoutes(body string, status int) *Google {
	g := NewGoogle("test-key")
	g.Client.Transport = transport(func(*http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: status, Body: io.NopCloser(strings.NewReader(body))}, nil
	})
	return g
}

func TestGoogleMatrixContractAndProposal(t *testing.T) {
	for _, mode := range []TravelMode{Walk, Drive} {
		t.Run(string(mode), func(t *testing.T) {
			g := NewGoogle("test-key")
			g.Client.Transport = transport(func(r *http.Request) (*http.Response, error) {
				if r.URL.String() != "https://routes.googleapis.com/distanceMatrix/v2:computeRouteMatrix" || r.Method != "POST" {
					t.Fatal("wrong endpoint")
				}
				if r.Header.Get("X-Goog-Api-Key") != "test-key" || r.Header.Get("X-Goog-FieldMask") != matrixFields {
					t.Fatal("incorrect headers")
				}
				deadline, ok := r.Context().Deadline()
				if !ok || time.Until(deadline) > 10*time.Second {
					t.Fatal("missing bounded deadline")
				}
				var payload struct {
					Origins []struct {
						Waypoint struct {
							PlaceID string `json:"placeId"`
						} `json:"waypoint"`
					} `json:"origins"`
					Destinations []json.RawMessage `json:"destinations"`
					Mode         string            `json:"travelMode"`
					Preference   string            `json:"routingPreference"`
				}
				if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
					t.Fatal(err)
				}
				if len(payload.Origins) != 2 || len(payload.Destinations) != 2 || payload.Origins[0].Waypoint.PlaceID != "place-a" || payload.Mode != string(mode) {
					t.Fatalf("%+v", payload)
				}
				if (mode == Drive && payload.Preference != "TRAFFIC_UNAWARE") || (mode == Walk && payload.Preference != "") {
					t.Fatal("incorrect routing preference")
				}
				return &http.Response{StatusCode: 200, Body: io.NopCloser(strings.NewReader(validMatrix))}, nil
			})
			matrix, err := g.Matrix(context.Background(), []string{"place-a", "place-b"}, mode)
			if err != nil || !reflect.DeepEqual(matrix, [][]int{{0, 10}, {3, 0}}) {
				t.Fatalf("%v %v", matrix, err)
			}
			proposal, err := Propose(context.Background(), []Stop{{"a", 30, nil}, {"b", 30, nil}}, matrix, 540, 720)
			if err != nil || !proposal.Changed || proposal.Savings == nil || *proposal.Savings != 7 {
				t.Fatalf("%+v %v", proposal, err)
			}
		})
	}
}

func TestGoogleMatrixFailures(t *testing.T) {
	for _, tc := range []struct {
		name, body string
		status     int
		code       string
	}{
		{"quota", "private detail", 429, "ROUTES_RATE_LIMIT"},
		{"server", "private detail", 500, "ROUTES_PROVIDER_ERROR"},
		{"missing", "[]", 200, "ROUTES_INVALID_RESPONSE"},
		{"malformed", "{", 200, "ROUTES_INVALID_RESPONSE"},
		{"duplicate", strings.Replace(validMatrix, `"originIndex":1,"destinationIndex":0`, `"originIndex":0,"destinationIndex":0`, 1), 200, "ROUTES_INVALID_RESPONSE"},
		{"out of range", strings.Replace(validMatrix, `"originIndex":1`, `"originIndex":2`, 1), 200, "ROUTES_INVALID_RESPONSE"},
		{"negative duration", strings.Replace(validMatrix, "120.1s", "-1s", 1), 200, "ROUTES_INVALID_RESPONSE"},
		{"wrong units", strings.Replace(validMatrix, "120.1s", "2m", 1), 200, "ROUTES_INVALID_RESPONSE"},
		{"oversized", strings.Repeat(" ", (1<<20)+1), 200, "ROUTES_INVALID_RESPONSE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			_, err := mockRoutes(tc.body, tc.status).Matrix(context.Background(), []string{"a", "b"}, Walk)
			var de *domain.Error
			if !errors.As(err, &de) || de.Code != tc.code || strings.Contains(err.Error(), "private detail") {
				t.Fatalf("%v", err)
			}
		})
	}
}

func TestGoogleMatrixUnavailableLegs(t *testing.T) {
	for _, replacement := range []string{`"condition":"ROUTE_NOT_FOUND"`, `"status":{"code":7}`, `"condition":"ROUTE_EXISTS","duration":"86401s"`} {
		body := strings.Replace(validMatrix, `"condition":"ROUTE_EXISTS","duration":"120.1s"`, replacement, 1)
		matrix, err := mockRoutes(body, 200).Matrix(context.Background(), []string{"a", "b"}, Walk)
		if err != nil || matrix[1][0] != -1 || matrix[0][0] != 0 {
			t.Fatalf("%v %v", matrix, err)
		}
	}
}

func TestGoogleMatrixValidationAndCancellation(t *testing.T) {
	g := NewGoogle("")
	g.Client.Transport = transport(func(*http.Request) (*http.Response, error) { t.Fatal("unexpected network request"); return nil, nil })
	if _, err := g.Matrix(context.Background(), []string{"a", "b"}, Walk); err == nil {
		t.Fatal("missing key accepted")
	}
	for _, ids := range [][]string{nil, {"a"}} {
		if _, err := g.Matrix(context.Background(), ids, Walk); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := g.Matrix(context.Background(), []string{"a", "b"}, "TRANSIT"); err == nil {
		t.Fatal("unsupported mode")
	}
	if _, err := g.Matrix(context.Background(), make([]string, 21), Walk); err == nil {
		t.Fatal("over limit")
	}
	if _, err := g.Matrix(context.Background(), []string{"", "b"}, Walk); err == nil {
		t.Fatal("invalid ID")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := g.Matrix(ctx, nil, Walk); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
	g.Key = "test"
	ctx, cancel = context.WithCancel(context.Background())
	g.Client.Transport = transport(func(r *http.Request) (*http.Response, error) {
		cancel()
		<-r.Context().Done()
		return nil, r.Context().Err()
	})
	if _, err := g.Matrix(ctx, []string{"a", "b"}, Walk); !errors.Is(err, context.Canceled) {
		t.Fatal(err)
	}
}

func TestGoogleMatrixDoesNotFollowRedirect(t *testing.T) {
	calls := 0
	g := NewGoogle("test")
	g.Client.Transport = transport(func(*http.Request) (*http.Response, error) {
		calls++
		return &http.Response{StatusCode: 302, Header: http.Header{"Location": []string{"https://example.com/"}}, Body: io.NopCloser(strings.NewReader(""))}, nil
	})
	if _, err := g.Matrix(context.Background(), []string{"a", "b"}, Walk); err == nil {
		t.Fatal("accepted redirect")
	}
	if calls != 1 {
		t.Fatalf("followed redirect: %d requests", calls)
	}
}
