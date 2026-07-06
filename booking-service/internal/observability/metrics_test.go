package observability

import (
	"testing"
	"time"

	dto "github.com/prometheus/client_model/go"
)

func TestRecordSagaTransitionIncrementsCounter(t *testing.T) {
	metric := sagaTransitionsTotal.WithLabelValues("COMPLETED", "CONFIRM_BOOKING", "NOT_REQUIRED")
	before := readCounterValue(t, metric)

	RecordSagaTransition("COMPLETED", "CONFIRM_BOOKING", "NOT_REQUIRED")

	after := readCounterValue(t, metric)
	if after != before+1 {
		t.Fatalf("expected saga transition counter to increment by 1, before=%v after=%v", before, after)
	}
}

func TestRecordSagaCompensationIncrementsCounter(t *testing.T) {
	metric := sagaCompensationsTotal.WithLabelValues("completed")
	before := readCounterValue(t, metric)

	RecordSagaCompensation("completed")

	after := readCounterValue(t, metric)
	if after != before+1 {
		t.Fatalf("expected saga compensation counter to increment by 1, before=%v after=%v", before, after)
	}
}

func TestObserveSagaDurationRecordsObservation(t *testing.T) {
	metric := sagaDurationSeconds.WithLabelValues("COMPLETED")
	before := readHistogramCount(t, metric)

	ObserveSagaDuration("COMPLETED", 250*time.Millisecond)

	after := readHistogramCount(t, metric)
	if after != before+1 {
		t.Fatalf("expected saga duration sample count to increment, before=%d after=%d", before, after)
	}
}

type writableMetric interface {
	Write(*dto.Metric) error
}

func readCounterValue(t *testing.T, metric writableMetric) float64 {
	t.Helper()

	collected := &dto.Metric{}
	if err := metric.Write(collected); err != nil {
		t.Fatalf("write counter metric: %v", err)
	}
	return collected.GetCounter().GetValue()
}

func readHistogramCount(t *testing.T, metric any) uint64 {
	t.Helper()

	writable, ok := metric.(writableMetric)
	if !ok {
		t.Fatalf("metric does not implement Write")
	}
	collected := &dto.Metric{}
	if err := writable.Write(collected); err != nil {
		t.Fatalf("write histogram metric: %v", err)
	}
	return collected.GetHistogram().GetSampleCount()
}
