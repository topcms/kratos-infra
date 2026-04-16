package kafka

import (
	"context"
	"time"

	kafkago "github.com/segmentio/kafka-go"
)

type ProducerConfig struct {
	Brokers      []string
	Topic        string
	Balancer     kafkago.Balancer
	BatchTimeout time.Duration
}

type Producer struct {
	writer *kafkago.Writer
}

func NewProducer(cfg ProducerConfig) *Producer {
	balancer := cfg.Balancer
	if balancer == nil {
		balancer = &kafkago.LeastBytes{}
	}

	return &Producer{
		writer: &kafkago.Writer{
			Addr:         kafkago.TCP(cfg.Brokers...),
			Topic:        cfg.Topic,
			Balancer:     balancer,
			BatchTimeout: cfg.BatchTimeout,
		},
	}
}

func (p *Producer) Publish(ctx context.Context, key, value []byte) error {
	return p.writer.WriteMessages(ctx, kafkago.Message{
		Key:   key,
		Value: value,
		Time:  time.Now(),
	})
}

func (p *Producer) Close() error {
	return p.writer.Close()
}
