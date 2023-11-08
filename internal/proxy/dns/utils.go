package dns

import (
	"github.com/miekg/dns"
	"github.com/rs/zerolog"
)

func logRequest(r *dns.Msg, logger zerolog.Logger) {
	arr := zerolog.Arr()
	for _, q := range r.Question {
		d := zerolog.Dict().
			Str("type", dns.TypeToString[q.Qtype]).
			Str("name", q.Name)
		arr.Dict(d)
	}

	logger.Info().
		Array("requests", arr).
		Msg("New request")
}
