package interceptors

import (
	"context"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"google.golang.org/grpc"
)

var (
	requestDuration     *prometheus.HistogramVec
	lastRequestDuration prometheus.Gauge
)

const appName = "auth"

func init() {
	requestDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "users",
		Subsystem: "grpc",
		Name:      appName + "_request_duration_seconds",
		Help:      "Duration of API requests",
		Buckets:   prometheus.ExponentialBuckets(0.0001, 4, 12),
	}, []string{"status"})

	lastRequestDuration = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "users",
		Subsystem: "grpc",
		Name:      appName + "_last_request_duration_seconds",
	})
}

func MetricsInterceptor(ctx context.Context, req interface{}, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()
	next, err := handler(ctx, req)
	latency := time.Since(start)

	status := "success"
	if err != nil {
		status = "error"
	}

	requestDuration.WithLabelValues(status).Observe(latency.Seconds())
	lastRequestDuration.Set(latency.Seconds())

	return next, err
}
