package prometheus

import (
	"gickup/types"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var CountSourcesConfigured = promauto.NewCounter(prometheus.CounterOpts{
	Name: "gickup_sources_count",
	Help: "The count of sources configured",
})

var CountDestinationsConfigured = promauto.NewCounter(prometheus.CounterOpts{
	Name: "gickup_destinations_count",
	Help: "The count of destinations configured",
})

var JobsRun = promauto.NewCounter(prometheus.CounterOpts{
	Name: "gickup_jobs_run",
	Help: "The count of scheduled jobs run since process startup",
})

func Serve(conf types.PrometheusConfig) {
	log.Info().Str("listenAddr", conf.ListenAddr).Str("endpoint", conf.Endpoint).Msg("Starting Prometheus listener")

	http.Handle(conf.Endpoint, promhttp.Handler())
	http.ListenAndServe(conf.ListenAddr, nil)
}
