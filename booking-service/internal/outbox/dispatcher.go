package outbox

import (
	"context"
	"log"
	"time"
)

const (
	defaultBatchSize         = 50
	defaultPollInterval      = 500 * time.Millisecond
	defaultInitialRetryDelay = time.Second
	defaultMaxRetryDelay     = 30 * time.Second
)

type DispatcherConfig struct {
	BatchSize         int
	PollInterval      time.Duration
	InitialRetryDelay time.Duration
	MaxRetryDelay     time.Duration
}

type Dispatcher struct {
	store             Store
	publisher         Publisher
	batchSize         int
	pollInterval      time.Duration
	initialRetryDelay time.Duration
	maxRetryDelay     time.Duration
	logger            *log.Logger
	clock             func() time.Time
}

func NewDispatcher(store Store, publisher Publisher, cfg DispatcherConfig, logger *log.Logger) *Dispatcher {
	if cfg.BatchSize <= 0 {
		cfg.BatchSize = defaultBatchSize
	}
	if cfg.PollInterval <= 0 {
		cfg.PollInterval = defaultPollInterval
	}
	if cfg.InitialRetryDelay <= 0 {
		cfg.InitialRetryDelay = defaultInitialRetryDelay
	}
	if cfg.MaxRetryDelay <= 0 {
		cfg.MaxRetryDelay = defaultMaxRetryDelay
	}
	if logger == nil {
		logger = log.Default()
	}
	return &Dispatcher{
		store:             store,
		publisher:         publisher,
		batchSize:         cfg.BatchSize,
		pollInterval:      cfg.PollInterval,
		initialRetryDelay: cfg.InitialRetryDelay,
		maxRetryDelay:     cfg.MaxRetryDelay,
		logger:            logger,
		clock:             func() time.Time { return time.Now().UTC() },
	}
}

func (d *Dispatcher) Run(ctx context.Context) {
	ticker := time.NewTicker(d.pollInterval)
	defer ticker.Stop()

	for {
		processed, err := d.DispatchBatch(ctx)
		if err != nil && ctx.Err() == nil {
			d.logger.Printf("outbox dispatcher: %v", err)
		}
		if processed >= d.batchSize {
			continue
		}

		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
		}
	}
}

func (d *Dispatcher) DispatchBatch(ctx context.Context) (int, error) {
	events, err := d.store.FetchPendingOutboxEvents(ctx, d.batchSize)
	if err != nil {
		return 0, err
	}

	processed := 0
	for _, event := range events {
		if ctx.Err() != nil {
			return processed, ctx.Err()
		}

		if err := d.publisher.Publish(ctx, event); err != nil {
			nextAttemptAt := d.clock().Add(d.retryDelay(event.Attempts))
			if markErr := d.store.MarkOutboxFailed(ctx, event.EventID, nextAttemptAt, err.Error()); markErr != nil {
				return processed, markErr
			}
			d.logger.Printf("outbox dispatcher: evento %s no publicado, reintento en %s: %v", event.EventID, nextAttemptAt.Format(time.RFC3339), err)
			processed++
			continue
		}

		if err := d.store.MarkOutboxPublished(ctx, event.EventID, d.clock()); err != nil {
			return processed, err
		}
		processed++
	}

	return processed, nil
}

func (d *Dispatcher) retryDelay(attempts int) time.Duration {
	delay := d.initialRetryDelay
	for i := 0; i < attempts; i++ {
		delay *= 2
		if delay >= d.maxRetryDelay {
			return d.maxRetryDelay
		}
	}
	if delay > d.maxRetryDelay {
		return d.maxRetryDelay
	}
	return delay
}
