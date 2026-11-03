package auth

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/clerk/clerk-sdk-go/v2"
	jose "github.com/go-jose/go-jose/v4"
	josejwt "github.com/go-jose/go-jose/v4/jwt"
)

func TestClerkVerification(t *testing.T) {
	private, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	rotated, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatal(err)
	}
	publicKeys := jose.JSONWebKeySet{Keys: []jose.JSONWebKey{{Key: &private.PublicKey, KeyID: "waypoint-test-1", Algorithm: "RS256", Use: "sig"}, {Key: &rotated.PublicKey, KeyID: "waypoint-test-2", Algorithm: "RS256", Use: "sig"}}}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(publicKeys)
	}))
	defer server.Close()
	previous := clerk.GetBackend()
	defer clerk.SetBackend(previous)
	clerk.SetBackend(clerk.NewBackend(&clerk.BackendConfig{URL: clerk.String(server.URL), Key: clerk.String("test-only")}))
	verifier := &Clerk{Issuer: "https://waypoint-test.clerk.accounts.dev", Audience: "waypoint", Origin: "http://localhost:3000"}
	now := time.Now().Unix()
	sign := func(key *rsa.PrivateKey, kid string, claims map[string]any) string {
		t.Helper()
		signer, err := jose.NewSigner(jose.SigningKey{Algorithm: jose.RS256, Key: key}, (&jose.SignerOptions{}).WithType("JWT").WithHeader("kid", kid))
		if err != nil {
			t.Fatal(err)
		}
		token, err := josejwt.Signed(signer).Claims(claims).Serialize()
		if err != nil {
			t.Fatal(err)
		}
		return token
	}
	for _, tc := range []struct {
		name   string
		change func(map[string]any)
		key    *rsa.PrivateKey
		kid    string
		valid  bool
	}{
		{"valid", func(map[string]any) {}, private, "waypoint-test-1", true},
		{"rotated key", func(map[string]any) {}, rotated, "waypoint-test-2", true},
		{"wrong signature", func(map[string]any) {}, rotated, "waypoint-test-1", false},
		{"expired", func(c map[string]any) { c["exp"] = now - 60 }, private, "waypoint-test-1", false},
		{"not yet valid", func(c map[string]any) { c["nbf"] = now + 600 }, private, "waypoint-test-1", false},
		{"wrong issuer", func(c map[string]any) { c["iss"] = "https://other.clerk.accounts.dev" }, private, "waypoint-test-1", false},
		{"wrong origin", func(c map[string]any) { c["azp"] = "https://attacker.example" }, private, "waypoint-test-1", false},
		{"wrong audience", func(c map[string]any) { c["aud"] = "other" }, private, "waypoint-test-1", false},
		{"no expiry", func(c map[string]any) { delete(c, "exp") }, private, "waypoint-test-1", false},
		{"no identity", func(c map[string]any) { delete(c, "sub") }, private, "waypoint-test-1", false},
		{"not a session", func(c map[string]any) { delete(c, "sid") }, private, "waypoint-test-1", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			claims := map[string]any{"iss": verifier.Issuer, "sub": "traveler", "sid": "sess_test", "azp": verifier.Origin, "aud": "waypoint", "exp": now + 300, "nbf": now - 60}
			tc.change(claims)
			subject, err := verifier.Verify(context.Background(), sign(tc.key, tc.kid, claims))
			if (err == nil) != tc.valid {
				t.Fatalf("subject=%q error=%v", subject, err)
			}
			if tc.valid && subject != "traveler" {
				t.Fatal(subject)
			}
		})
	}
	if _, err = verifier.Verify(context.Background(), "not-a-token"); err == nil {
		t.Fatal("accepted unverified identity")
	}
}
