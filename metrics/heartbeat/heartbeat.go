package heartbeat

import (
	"net/http"

	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog/log"
)

func Send(conf types.HeartbeatConfig) {
	for _, u := range conf.URLs {
		log.Info().Str("url", u).Msg("sending heartbeat")
		_, err := http.Get(u)
		if err != nil {
			log.Fatal().Str("monitoring", "heartbeat").Msg(err.Error())
		}
	}
}
