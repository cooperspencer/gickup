package heartbeat

import (
	"net/http"

	"github.com/cooperspencer/gickup/types"
	"github.com/rs/zerolog/log"
)

// Send pings the configured heartbeat URLs.
//
// On a successful backup run, conf.URLs is pinged. On a failed run,
// conf.FailureURLs is pinged instead, so monitoring services such as
// healthchecks.io that expose a dedicated "failure" endpoint can be
// notified immediately instead of only alerting once the regular
// heartbeat times out.
func Send(conf types.HeartbeatConfig, success bool) {
	urls := conf.URLs
	if !success {
		urls = conf.FailureURLs
	}

	for _, u := range urls {
		log.Info().Str("url", u).Bool("success", success).Msg("sending heartbeat")
		resp, err := http.Get(u) //nolint:noctx
		if err != nil {
			log.Error().Str("monitoring", "heartbeat").Msg(err.Error())
			continue
		}
		if err := resp.Body.Close(); err != nil {
			log.Error().Str("monitoring", "heartbeat").Str("url", u).Msg(err.Error())
		}
	}
}
