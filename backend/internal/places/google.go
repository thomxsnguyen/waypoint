package places

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
	"waypoint/internal/domain"
)

type Provider interface {
	Search(context.Context, string, string) ([]domain.Place, error)
	Details(context.Context, string) (domain.Place, error)
}
type Google struct {
	Key    string
	Client *http.Client
}

func NewGoogle(key string) *Google { return &Google{key, &http.Client{Timeout: 8 * time.Second}} }

type googlePlace struct {
	ID          string `json:"id"`
	DisplayName struct {
		Text string `json:"text"`
	} `json:"displayName"`
	Address  string `json:"formattedAddress"`
	Location *struct {
		Latitude  float64 `json:"latitude"`
		Longitude float64 `json:"longitude"`
	} `json:"location"`
	URL          string `json:"googleMapsUri"`
	Attributions []struct {
		Provider string `json:"provider"`
		URI      string `json:"providerUri"`
	} `json:"attributions"`
}

func (p googlePlace) normalized() domain.Place {
	out := domain.Place{Provider: "google", ProviderPlaceID: p.ID, Name: p.DisplayName.Text, Address: p.Address, ExternalURL: p.URL, Attributions: []domain.Attribution{}}
	if p.Location != nil {
		out.Location = &domain.Location{Lat: p.Location.Latitude, Lng: p.Location.Longitude}
	}
	for _, a := range p.Attributions {
		out.Attributions = append(out.Attributions, domain.Attribution{Name: a.Provider, URL: a.URI})
	}
	return out
}

const fields = "id,displayName,formattedAddress,location,googleMapsUri,attributions"

func (g *Google) request(ctx context.Context, method, path, mask string, body any, out any) error {
	if g.Key == "" {
		return &domain.Error{Status: 503, Code: "PLACES_UNAVAILABLE", Message: "Place search is not configured. Your saved itinerary remains available."}
	}
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, "https://places.googleapis.com/v1/"+path, reader)
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Goog-Api-Key", g.Key)
	req.Header.Set("X-Goog-FieldMask", mask)
	res, err := g.Client.Do(req)
	if err != nil {
		return &domain.Error{Status: 503, Code: "PLACES_UNAVAILABLE", Message: "Google Places is temporarily unavailable. Try again."}
	}
	defer res.Body.Close()
	if res.StatusCode == 429 {
		return &domain.Error{Status: 429, Code: "PROVIDER_RATE_LIMIT", Message: "Google Places quota reached. Try again later."}
	}
	if res.StatusCode == 404 || res.StatusCode == 400 {
		return &domain.Error{Status: 502, Code: "PLACE_UNAVAILABLE", Message: "This place is unavailable from Google Places."}
	}
	if res.StatusCode != 200 {
		return &domain.Error{Status: 502, Code: "PROVIDER_ERROR", Message: "Google Places could not complete the request. Try again later."}
	}
	if err = json.NewDecoder(io.LimitReader(res.Body, 2<<20)).Decode(out); err != nil {
		return &domain.Error{Status: 502, Code: "PROVIDER_ERROR", Message: "Google Places returned an invalid response."}
	}
	return nil
}
func (g *Google) Search(ctx context.Context, q, destination string) ([]domain.Place, error) {
	var response struct {
		Places []googlePlace `json:"places"`
	}
	mask := "places." + strings.ReplaceAll(fields, ",", ",places.")
	err := g.request(ctx, http.MethodPost, "places:searchText", mask, map[string]any{"textQuery": fmt.Sprintf("%s in %s", q, destination), "pageSize": 20}, &response)
	if err != nil {
		return nil, err
	}
	out := []domain.Place{}
	for _, p := range response.Places {
		out = append(out, p.normalized())
	}
	return out, nil
}
func (g *Google) Details(ctx context.Context, id string) (domain.Place, error) {
	if err := domain.ValidatePlaceID(id); err != nil {
		return domain.Place{}, err
	}
	var p googlePlace
	err := g.request(ctx, http.MethodGet, "places/"+url.PathEscape(id), fields, nil, &p)
	return p.normalized(), err
}
