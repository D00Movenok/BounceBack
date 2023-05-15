package http

import (
	"net/http"

	"github.com/rs/zerolog"
)

func handleError(w http.ResponseWriter) {
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func logRequest(r *http.Request, logger zerolog.Logger) {
	logger.Debug().
		Str("method", r.Method).
		Stringer("url", r.URL).
		Msg("New request")
}
