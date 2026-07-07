package outbox

import "github.com/prometheus/client_golang/prometheus"

var (
	outboxPublishedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "medconnect_booking_outbox_published_total",
			Help: "Total de eventos outbox publicados correctamente en Kafka.",
		},
	)
	outboxPublishFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "medconnect_booking_outbox_publish_failed_total",
			Help: "Total de intentos fallidos de publicacion de eventos outbox.",
		},
	)
	outboxFinalFailedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "medconnect_booking_outbox_final_failed_total",
			Help: "Total de eventos outbox marcados definitivamente como FAILED.",
		},
	)
	outboxPendingGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "medconnect_booking_outbox_pending",
			Help: "Cantidad de eventos outbox pendientes de publicacion.",
		},
	)
	outboxOldestPendingAgeGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "medconnect_booking_outbox_oldest_pending_age_seconds",
			Help: "Edad en segundos del evento outbox pendiente mas antiguo.",
		},
	)
	outboxFinalFailedGauge = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "medconnect_booking_outbox_final_failed",
			Help: "Cantidad de eventos outbox en estado FAILED.",
		},
	)
)

func init() {
	prometheus.MustRegister(
		outboxPublishedTotal,
		outboxPublishFailedTotal,
		outboxFinalFailedTotal,
		outboxPendingGauge,
		outboxOldestPendingAgeGauge,
		outboxFinalFailedGauge,
	)
}

func recordOutboxPublished() {
	outboxPublishedTotal.Inc()
}

func recordOutboxPublishFailed(final bool) {
	outboxPublishFailedTotal.Inc()
	if final {
		outboxFinalFailedTotal.Inc()
	}
}

func observeOutboxStats(stats Stats) {
	outboxPendingGauge.Set(float64(stats.PendingEvents))
	outboxOldestPendingAgeGauge.Set(stats.OldestPendingAge.Seconds())
	outboxFinalFailedGauge.Set(float64(stats.FinalFailedEvents))
}
