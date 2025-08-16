package producer

import (
	"context"
	"encoding/json"

	"github.com/quiby-ai/common/pkg/events"
	"github.com/quiby-ai/review-preprocessor/config"
	"github.com/segmentio/kafka-go"
)

type KafkaProducer struct {
	writer *kafka.Writer
	cfg    config.KafkaConfig
}

func NewKafkaProducer(cfg config.KafkaConfig) *KafkaProducer {
	return &KafkaProducer{
		writer: &kafka.Writer{
			Addr:     kafka.TCP(cfg.Brokers...),
			Topic:    events.PipelinePrepareCompleted,
			Balancer: &kafka.LeastBytes{},
		},
		cfg: cfg,
	}
}

func (p *KafkaProducer) PublishEvent(ctx context.Context, evt events.PrepareCompleted) error {
	b, err := json.Marshal(evt)
	if err != nil {
		return err
	}
	return p.writer.WriteMessages(ctx, kafka.Message{Value: b})
}

func (p *KafkaProducer) Close() error {
	if p.writer != nil {
		return p.writer.Close()
	}
	return nil
}
