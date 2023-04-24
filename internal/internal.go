package internal

import (
	"math"
	"time"

	"fmt"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/txsvc/stdlib/v2"
)

func StartPrometheusListener() {
	// prometheus endpoint setup
	promHost := stdlib.GetString("prom_host", "0.0.0.0:2112")
	promMetricsPath := stdlib.GetString("prom_metrics_path", "/metrics")

	// start the metrics listener
	go func() {
		fmt.Printf(" --> starting metrics endpoint '%s' on '%s'\n", promMetricsPath, promHost)

		http.Handle(promMetricsPath, promhttp.Handler())
		http.ListenAndServe(promHost, nil)
	}()
}

func Duration(d time.Duration, dicimal int) time.Duration {
	shift := int(math.Pow10(dicimal))

	units := []time.Duration{time.Second, time.Millisecond, time.Microsecond, time.Nanosecond}
	for _, u := range units {
		if d > u {
			div := u / time.Duration(shift)
			if div == 0 {
				break
			}
			d = d / div * div
			break
		}
	}
	return d
}
