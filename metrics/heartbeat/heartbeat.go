package heartbeat

import (
	"net/http"

	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog/log"
)

func Send(conf types.HeartbeatConfig) {
	for _, u := range conf.URLs {
		log.Info().Str("url", u).Msg("sending heartbeat")
		resp, err := http.Get(u) //nolint:noctx
		if err != nil {
			log.Error().Str("monitoring", "heartbeat").Msg(err.Error())
			continue
		}
		resp.Body.Close()
	}
}
