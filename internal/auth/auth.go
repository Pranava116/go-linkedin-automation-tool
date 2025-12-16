package auth

import (
	"context"
	"github.com/go-rod/rod"
)

// Authenticator interface for LinkedIn authentication
type Authenticator interface {
	Login(ctx context.Context, page *rod.Page) error
	IsLoggedIn(ctx context.Context, page *rod.Page) (bool, error)
	HandleChallenge(ctx context.Context, page *rod.Page) error
}

// Credentials represents login credentials
type Credentials struct {
	Username string
	Password string
}

// AuthManager implements Authenticator interface
type AuthManager struct {
	credentials Credentials
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(credentials Credentials) *AuthManager {
	return &AuthManager{
		credentials: credentials,
	}
}