package http

import (
	"net/http"

	"github.com/D00Movenok/BounceBack/internal/wrapper"
	"github.com/rs/zerolog"
)

func handleError(w http.ResponseWriter) {
	http.Error(w, "internal error", http.StatusInternalServerError)
}

func logRequest(e wrapper.Entity, logger zerolog.Logger) {
	m, _ := e.GetMethod()
	u, _ := e.GetURL()
	h, _ := e.GetHeaders()
	b, _ := e.GetBody()

	ua := h["User-Agent"]

	ev := logger.Info().
		Str("method", m).
		Stringer("url", u).
		Any("user-agent", ua)

	if zerolog.GlobalLevel() <= zerolog.DebugLevel {
		ev = ev.Any("headers", h)
	}

	if zerolog.GlobalLevel() <= zerolog.TraceLevel {
		ev = ev.Bytes("body", b)
	}

	ev.Msg("New request")
}
