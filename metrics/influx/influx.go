package influx

import (
	"sync"
	"time"

	"gickup/types"

	influxdb2 "github.com/influxdata/influxdb-client-go/v2"
	"github.com/influxdata/influxdb-client-go/v2/api"
	"github.com/rs/zerolog/log"
)

var once sync.Once
var writeApi api.WriteAPI
var client influxdb2.Client

func Api() api.WriteAPI {
	if writeApi == nil || client == nil {
		log.Error().Msg("InfluxDB2 API retrieved before initialization. This is a bug. Continuing nonetheless.")
	}
	return writeApi
}

func Setup(config types.InfluxDb2Config) {
	once.Do(func() {
		client = influxdb2.NewClient(config.Url, config.Token)
		writeApi = client.WriteAPI(config.Org, config.Bucket)
	})
}

func Teardown() {
	client.Close()
}

func DoIt() {
	p := influxdb2.NewPoint("stat",
		map[string]string{"unit": "temperature"},
		map[string]interface{}{"avg": 24.5, "max": 45},
		time.Now())
	// Write point immediately
	influx := Api()
	influx.WritePoint(p)
	// Ensures background processes finishes
	client.Close()
}
