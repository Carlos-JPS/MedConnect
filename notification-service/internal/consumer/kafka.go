package consumer

import (
	"context"
	"errors"
	"log"

	"github.com/MedConnect/notification-service/internal/processor"
	"github.com/twmb/franz-go/pkg/kgo"
)

type BookingEventConsumer struct {
	client    *kgo.Client
	processor *processor.Processor
	logger    *log.Logger
}

type DLQPublisher struct {
	client *kgo.Client
	topic  string
}

func NewBookingEventConsumer(brokers []string, topic string, group string, processor *processor.Processor, logger *log.Logger) (*BookingEventConsumer, error) {
	if len(brokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS no esta configurado")
	}
	if topic == "" {
		return nil, errors.New("BOOKING_EVENTS_TOPIC no esta configurado")
	}
	if group == "" {
		return nil, errors.New("KAFKA_CONSUMER_GROUP no esta configurado")
	}
	if logger == nil {
		logger = log.Default()
	}

	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID("medconnect-notification-service"),
		kgo.ConsumerGroup(group),
		kgo.ConsumeTopics(topic),
		kgo.DisableAutoCommit(),
	)
	if err != nil {
		return nil, err
	}

	return &BookingEventConsumer{client: client, processor: processor, logger: logger}, nil
}

func NewDLQPublisher(brokers []string, topic string) (*DLQPublisher, error) {
	if len(brokers) == 0 {
		return nil, errors.New("KAFKA_BROKERS no esta configurado")
	}
	if topic == "" {
		return nil, errors.New("BOOKING_EVENTS_DLQ_TOPIC no esta configurado")
	}
	client, err := kgo.NewClient(
		kgo.SeedBrokers(brokers...),
		kgo.ClientID("medconnect-notification-service-dlq"),
		kgo.RequiredAcks(kgo.AllISRAcks()),
	)
	if err != nil {
		return nil, err
	}
	return &DLQPublisher{client: client, topic: topic}, nil
}

func (c *BookingEventConsumer) Run(ctx context.Context) error {
	for {
		fetches := c.client.PollFetches(ctx)
		if ctx.Err() != nil {
			return nil
		}

		for _, fetchErr := range fetches.Errors() {
			c.logger.Printf("notification-service: error consumiendo Kafka topic=%s partition=%d: %v", fetchErr.Topic, fetchErr.Partition, fetchErr.Err)
		}

		iter := fetches.RecordIter()
		for !iter.Done() {
			record := iter.Next()
			message := processor.Message{
				Topic: record.Topic,
				Key:   record.Key,
				Value: record.Value,
			}
			if err := c.processor.Process(ctx, message); err != nil {
				c.logger.Printf("notification-service: no se pudo procesar event topic=%s partition=%d offset=%d: %v", record.Topic, record.Partition, record.Offset, err)
				continue
			}
			if err := c.client.CommitRecords(ctx, record); err != nil {
				return err
			}
		}
	}
}

func (c *BookingEventConsumer) Close() {
	if c.client != nil {
		c.client.Close()
	}
}

func (p *DLQPublisher) Publish(ctx context.Context, message processor.Message, reason string) error {
	record := &kgo.Record{
		Topic: p.topic,
		Key:   message.Key,
		Value: message.Value,
		Headers: []kgo.RecordHeader{
			{Key: "dlq_reason", Value: []byte(reason)},
			{Key: "source_topic", Value: []byte(message.Topic)},
		},
	}
	return p.client.ProduceSync(ctx, record).FirstErr()
}

func (p *DLQPublisher) Close() {
	if p.client != nil {
		p.client.Close()
	}
}
