package kafka

import (
	"context"
	"errors"

	"github.com/MedConnect/booking-service/internal/outbox"
	"github.com/twmb/franz-go/pkg/kgo"
)

type Publisher struct {
	client *kgo.Client
}

func NewPublisher(brokers []string) (*Publisher, error) {
	if len(brokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS no esta configurado")
	}
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID("medconnect-booking-outbox"),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, err
	}
	return &Publisher{client: client}, nil
}

func (p *Publisher) Publish(ctx context.Context, event outbox.Event) error {
	record := &kgo.Record{
		Topic: event.Topic,
		Key:   []byte(event.EventKey),
		Value: event.Payload,
		Headers: []kgo.RecordHeader{
			{Key: "event_id", Value: []byte(event.EventID)},
			{Key: "event_type", Value: []byte(event.EventType)},
			{Key: "aggregate_id", Value: []byte(event.AggregateID)},
		},
	}
	return p.client.ProduceSync(ctx, record).FirstErr()
}

func (p *Publisher) Close() {
	if p.client != nil {
		p.client.Close()
	}
}
