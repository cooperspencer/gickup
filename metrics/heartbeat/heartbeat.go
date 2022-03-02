package heartbeat

import (
	"net/http"

	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog/log"
)

func Send(conf types.HeartbeatConfig) {
	log.Info().Str("url", conf.URL).Msg("sending heartbeat")
	_, err := http.Get(conf.URL)
	if err != nil {
		log.Fatal().Str("monitoring", "heartbeat").Msg(err.Error())
	}
}
