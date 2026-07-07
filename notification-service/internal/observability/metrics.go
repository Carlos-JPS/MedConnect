package observability

import (
	"log"
	"net/http"
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	notificationEventsProcessedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "medconnect_notification_events_processed_total",
			Help: "Total de eventos de reserva procesados por notification-service.",
		},
		[]string{"result"},
	)
	notificationDLQMessagesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "medconnect_notification_dlq_messages_total",
			Help: "Total de mensajes enviados a la DLQ por notification-service.",
		},
	)
)

func init() {
	prometheus.MustRegister(notificationEventsProcessedTotal, notificationDLQMessagesTotal)
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
			log.Printf("error en servidor de metricas de %s: %v", serviceName, err)
		}
	}()
}

func RecordNotificationProcessed(result string) {
	notificationEventsProcessedTotal.WithLabelValues(result).Inc()
}

func RecordDLQMessage() {
	notificationDLQMessagesTotal.Inc()
}
