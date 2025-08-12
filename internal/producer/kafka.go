package producer

import (
	"context"
	"encoding/json"

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
			Topic:    cfg.TopicClusterReviews,
			Balancer: &kafka.LeastBytes{},
		},
		cfg: cfg,
	}
}

type ClusterReviewsEvent struct {
	JobID string   `json:"job_id"`
	AppID string   `json:"app_id"`
	IDs   []string `json:"ids"`
	Count int      `json:"count"`
}

func (p *KafkaProducer) PublishClusterEvent(ctx context.Context, evt ClusterReviewsEvent) error {
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
