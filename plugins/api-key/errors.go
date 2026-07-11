package apikey

import (
	"net/http"

	"github.com/thecodearcher/limen"
)

var (
	ErrAPIKeyNotFound      = limen.NewLimenError("API key not found", http.StatusNotFound, nil)
	ErrAPIKeyRevoked       = limen.NewLimenError("API key revoked", http.StatusGone, nil)
	ErrAPIKeyNotAuthorized = limen.NewLimenError("You cannot carry out this action", http.StatusForbidden, nil)
)
