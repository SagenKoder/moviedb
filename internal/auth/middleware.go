package auth

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/auth0/go-jwt-middleware/v2"
	"github.com/auth0/go-jwt-middleware/v2/jwks"
	"github.com/auth0/go-jwt-middleware/v2/validator"
)

type User struct {
	Auth0ID string `json:"auth0_id"`
	Email   string `json:"email"`
	Name    string `json:"name"`
}

// CustomClaims contains custom data we want to extract from the token.
type CustomClaims struct {
	Email string `json:"email"`
	Name  string `json:"name"`
}

// Validate does nothing for this example, but we need
// it to satisfy validator.CustomClaims interface.
func (c CustomClaims) Validate(ctx context.Context) error {
	return nil
}

func NewMiddleware(domain, audience string) (*jwtmiddleware.JWTMiddleware, error) {
	issuerURL, err := url.Parse("https://" + domain + "/")
	if err != nil {
		return nil, fmt.Errorf("failed to parse the issuer url: %w", err)
	}

	provider := jwks.NewCachingProvider(issuerURL, 5*time.Minute)

	jwtValidator, err := validator.New(
		provider.KeyFunc,
		validator.RS256,
		issuerURL.String(),
		[]string{audience},
		validator.WithCustomClaims(
			func() validator.CustomClaims {
				return &CustomClaims{}
			},
		),
		validator.WithAllowedClockSkew(time.Minute),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create JWT validator: %w", err)
	}

	return jwtmiddleware.New(jwtValidator.ValidateToken), nil
}

func GetUserFromContext(ctx context.Context) (*User, error) {
	claims, ok := ctx.Value(jwtmiddleware.ContextKey{}).(*validator.ValidatedClaims)
	if !ok {
		return nil, fmt.Errorf("no claims found in context")
	}

	customClaims, ok := claims.CustomClaims.(*CustomClaims)
	if !ok {
		return nil, fmt.Errorf("invalid custom claims format")
	}

	return &User{
		Auth0ID: claims.RegisteredClaims.Subject,
		Email:   customClaims.Email,
		Name:    customClaims.Name,
	}, nil
}

func RequireAuth(middleware *jwtmiddleware.JWTMiddleware) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return middleware.CheckJWT(next)
	}
}