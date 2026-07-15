package apikey

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrInvalidAPIKey       = limen.NewLimenError("invalid API key", http.StatusUnauthorized, nil)
	ErrAPIKeyRevoked       = limen.NewLimenError("API key revoked", http.StatusGone, nil)
	ErrAPIKeyNotAuthorized = limen.NewLimenError("You cannot carry out this action", http.StatusForbidden, nil)

	ErrAPIKeyDisabled          = limen.NewLimenError("API key disabled", http.StatusForbidden, nil)
	ErrAPIKeyExpired           = limen.NewLimenError("API key expired", http.StatusForbidden, nil)
	ErrInsufficientPermissions = limen.NewLimenError("Insufficient permissions", http.StatusForbidden, nil)
	ErrRateLimitExceeded       = limen.NewLimenError("Rate limit exceeded", http.StatusTooManyRequests, nil)
)
