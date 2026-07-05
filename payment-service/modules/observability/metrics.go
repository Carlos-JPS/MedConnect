package observability

import (
	"context"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

var (
	grpcServerRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "medconnect_grpc_server_requests_total",
			Help: "Total de requests gRPC atendidas por el servidor.",
		},
		[]string{"service", "method", "code"},
	)
	grpcServerRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "medconnect_grpc_server_request_duration_seconds",
			Help:    "Duración de requests gRPC atendidas por el servidor.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"service", "method", "code"},
	)
)

func init() {
	prometheus.MustRegister(grpcServerRequestsTotal, grpcServerRequestDuration)
}

func StartMetricsServer(serviceName string) {
	port := os.Getenv("METRICS_PORT")
	if port == "" {
		port = "9090"
	}

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{Addr: ":" + port, Handler: mux}
	go func() {
		log.Printf("%s metrics escuchando en %s", serviceName, server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("error en servidor de métricas de %s: %v", serviceName, err)
		}
	}()
}

func UnaryServerInterceptor(serviceName string) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req any, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (any, error) {
		start := time.Now()
		resp, err := handler(ctx, req)
		duration := time.Since(start)
		code := codes.OK.String()
		if err != nil {
			code = status.Code(err).String()
		}
		requestID := requestIDFromContext(ctx)

		method := methodName(info.FullMethod)
		labels := []string{serviceName, method, code}
		grpcServerRequestsTotal.WithLabelValues(labels...).Inc()
		grpcServerRequestDuration.WithLabelValues(labels...).Observe(duration.Seconds())
		log.Printf("service=%s request_id=%s method=%s code=%s duration=%s", serviceName, requestID, method, code, duration)
		return resp, err
	}
}

func methodName(fullMethod string) string {
	idx := strings.LastIndex(fullMethod, "/")
	if idx == -1 || idx == len(fullMethod)-1 {
		return "unknown"
	}
	return fullMethod[idx+1:]
}

func requestIDFromContext(ctx context.Context) string {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return "unknown"
	}
	vals := md.Get("x-request-id")
	if len(vals) == 0 || vals[0] == "" {
		return "unknown"
	}
	return vals[0]
}
