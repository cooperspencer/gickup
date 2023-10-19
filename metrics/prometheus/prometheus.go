package prometheus

import (
	"net/http"

	"github.com/cooperspencer/gickup/types"
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

var CountReposDiscovered = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "gickup_repos_discovered",
	Help: "The count of sources configured",
}, []string{"source_name", "config_number"})

var JobsComplete = promauto.NewCounter(prometheus.CounterOpts{
	Name: "gickup_jobs_complete",
	Help: "The count of scheduled jobs completed since process startup",
})

var JobsStarted = promauto.NewCounter(prometheus.CounterOpts{
	Name: "gickup_jobs_started",
	Help: "The count of scheduled jobs started since process startup",
})

var JobDuration = promauto.NewSummary(prometheus.SummaryOpts{
	Name: "gickup_job_duration",
	Help: "The duration of scheduled jobs started since process startup",
})

var SourceBackupsComplete = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "gickup_sources_complete",
	Help: "The count of source backups completed",
}, []string{"source_name"})

var DestinationBackupsComplete = promauto.NewCounterVec(prometheus.CounterOpts{
	Name: "gickup_destinations_complete",
	Help: "The count of destination to which a backup was written",
}, []string{"destination_type"})

var RepoSuccess = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "gickup_repo_success",
	Help: "See if backup was successful",
}, []string{"hoster", "repository", "owner", "type", "path"})

var RepoTime = promauto.NewGaugeVec(prometheus.GaugeOpts{
	Name: "gickup_repo_time",
	Help: "How long did the task take",
}, []string{"hoster", "repository", "owner", "type", "path"})

func Serve(conf types.PrometheusConfig) {
	log.Info().
		Str("listenAddr", conf.ListenAddr).
		Str("endpoint", conf.Endpoint).
		Msg("Starting Prometheus listener")

	http.Handle(conf.Endpoint, promhttp.Handler())
	err := http.ListenAndServe(conf.ListenAddr, nil)
	log.Fatal().
		Str("listenAddr", conf.ListenAddr).
		Str("endpoint", conf.Endpoint).
		Msg(err.Error())
}
