package prometheus

import (
	"gickup/types"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog/log"
)

var CountSourcesConfigured = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "gickup_sources",
	Help: "The count of sources configured",
})

var CountDestinationsConfigured = promauto.NewGauge(prometheus.GaugeOpts{
	Name: "gickup_destinations",
	Help: "The count of destinations configured",
})

var JobsComplete = promauto.NewCounter(prometheus.CounterOpts{
	Name: "gickup_jobs_complete",
	Help: "The count of scheduled jobs completed since process startup",
})

var JobsStarted = promauto.NewCounter(prometheus.CounterOpts{
	Name: "gickup_jobs_started",
	Help: "The count of scheduled jobs started since process startup",
})

var SourceBackupsComplete = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "gickup_sources_complete",
	Help: "The count of source backups completed",
}, []string{"source_name"})

var DestinationBackupsComplete = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "gickup_destinations_complete",
	Help: "The count of destination to which a backup was written",
}, []string{"destination_type"})

func Serve(conf types.PrometheusConfig) {
	log.Info().Str("listenAddr", conf.ListenAddr).Str("endpoint", conf.Endpoint).Msg("Starting Prometheus listener")

	http.Handle(conf.Endpoint, promhttp.Handler())
	http.ListenAndServe(conf.ListenAddr, nil)
}
