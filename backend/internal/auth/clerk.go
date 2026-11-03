package auth

import (
	"context"
	"errors"
	"github.com/clerk/clerk-sdk-go/v2"
	"github.com/clerk/clerk-sdk-go/v2/jwt"
	"slices"
	"strings"
	"time"
)

type Verifier interface {
	Verify(context.Context, string) (string, error)
}
type Clerk struct{ Issuer, Audience, Origin string }

func NewClerk(secret, issuer, audience, origin string) (*Clerk, error) {
	if secret == "" || issuer == "" || origin == "" {
		return nil, errors.New("CLERK_SECRET_KEY, AUTH_ISSUER and CORS_ALLOWED_ORIGIN are required")
	}
	if !strings.HasPrefix(issuer, "https://") {
		return nil, errors.New("AUTH_ISSUER must use HTTPS")
	}
	clerk.SetKey(secret)
	return &Clerk{issuer, audience, origin}, nil
}
func (c *Clerk) Verify(ctx context.Context, token string) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	claims, err := jwt.Verify(ctx, &jwt.VerifyParams{Token: token, AuthorizedPartyHandler: func(azp string) bool { return azp == c.Origin }})
	if err != nil {
		return "", errors.New("invalid session")
	}
	if claims.Issuer != c.Issuer || claims.Subject == "" || claims.SessionID == "" || claims.Expiry == nil || claims.NotBefore == nil {
		return "", errors.New("invalid session claims")
	}
	if c.Audience != "" && !slices.Contains(claims.Audience, c.Audience) {
		return "", errors.New("invalid audience")
	}
	return claims.Subject, nil
}
