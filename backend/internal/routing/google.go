package routing

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"regexp"
	"time"
	"waypoint/internal/domain"
)

type TravelMode string

const (
	Walk  TravelMode = "WALK"
	Drive TravelMode = "DRIVE"
)

// MatrixProvider returns directed minutes, with -1 for unavailable legs.
// Callers must authorize the trip and budget n*n provider elements before use.
type MatrixProvider interface {
	Matrix(context.Context, []string, TravelMode) ([][]int, error)
}

type Google struct {
	Key    string
	Client *http.Client
}

func NewGoogle(key string) *Google {
	return &Google{Key: key, Client: &http.Client{Timeout: 10 * time.Second}}
}

const matrixFields = "originIndex,destinationIndex,status,condition,duration"

var durationPattern = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,9})?s$`)

func routeError(status int, code, message string) error {
	return &domain.Error{Status: status, Code: code, Message: message}
}
func invalidMatrix() error {
	return routeError(502, "ROUTES_INVALID_RESPONSE", "Google Routes returned an invalid travel-time matrix.")
}

func (g *Google) Matrix(ctx context.Context, ids []string, mode TravelMode) ([][]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if mode != Walk && mode != Drive {
		return nil, domain.Invalid("Choose walking or driving for this routing version.")
	}
	n := len(ids)
	if n > 20 {
		return nil, domain.Invalid("Route estimates support at most 20 activities.")
	}
	for _, id := range ids {
		if err := domain.ValidatePlaceID(id); err != nil {
			return nil, err
		}
	}
	matrix := make([][]int, n)
	for i := range matrix {
		matrix[i] = make([]int, n)
		for j := range matrix[i] {
			if i != j {
				matrix[i][j] = -1
			}
		}
	}
	if n < 2 {
		return matrix, nil
	}
	if g.Key == "" {
		return nil, routeError(503, "ROUTES_UNAVAILABLE", "Travel-time estimates are not configured.")
	}
	type waypoint struct {
		PlaceID string `json:"placeId"`
	}
	type point struct {
		Waypoint waypoint `json:"waypoint"`
	}
	points := make([]point, n)
	for i, id := range ids {
		points[i] = point{waypoint{id}}
	}
	payload := struct {
		Origins      []point    `json:"origins"`
		Destinations []point    `json:"destinations"`
		Mode         TravelMode `json:"travelMode"`
		Preference   string     `json:"routingPreference,omitempty"`
	}{Origins: points, Destinations: points, Mode: mode}
	if mode == Drive {
		payload.Preference = "TRAFFIC_UNAWARE"
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://routes.googleapis.com/distanceMatrix/v2:computeRouteMatrix", bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", g.Key)
	req.Header.Set("X-Goog-FieldMask", matrixFields)
	client := http.Client{Timeout: 10 * time.Second}
	if g.Client != nil {
		client = *g.Client
	}
	client.CheckRedirect = func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }
	res, err := client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		return nil, routeError(503, "ROUTES_UNAVAILABLE", "Travel-time estimates are temporarily unavailable.")
	}
	defer res.Body.Close()
	if res.StatusCode == 429 {
		return nil, routeError(429, "ROUTES_RATE_LIMIT", "Google Routes quota reached. Try again later.")
	}
	if res.StatusCode != http.StatusOK {
		return nil, routeError(502, "ROUTES_PROVIDER_ERROR", "Google Routes could not complete the request.")
	}
	const maxBody = 1 << 20
	body, err := io.ReadAll(io.LimitReader(res.Body, maxBody+1))
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if err != nil || len(body) > maxBody {
		return nil, invalidMatrix()
	}
	var elements []struct {
		Origin      int `json:"originIndex"`
		Destination int `json:"destinationIndex"`
		Status      struct {
			Code int `json:"code"`
		} `json:"status"`
		Condition string `json:"condition"`
		Duration  string `json:"duration"`
	}
	if json.Unmarshal(body, &elements) != nil || len(elements) != n*n {
		return nil, invalidMatrix()
	}
	seen := make([]bool, n*n)
	for _, element := range elements {
		i, j := element.Origin, element.Destination
		// Protobuf JSON may omit a zero-valued index; default zero is intentional.
		if i < 0 || i >= n || j < 0 || j >= n || seen[i*n+j] {
			return nil, invalidMatrix()
		}
		seen[i*n+j] = true
		if element.Status.Code != 0 || element.Condition == "ROUTE_NOT_FOUND" {
			continue
		}
		if element.Condition != "ROUTE_EXISTS" || !durationPattern.MatchString(element.Duration) {
			return nil, invalidMatrix()
		}
		duration, err := time.ParseDuration(element.Duration)
		if err != nil {
			return nil, invalidMatrix()
		}
		if i == j {
			continue
		}
		if duration > 24*time.Hour {
			continue
		}
		matrix[i][j] = int((duration + time.Minute - 1) / time.Minute)
	}
	return matrix, nil
}

var _ MatrixProvider = (*Google)(nil)
