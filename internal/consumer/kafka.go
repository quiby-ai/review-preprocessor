package consumer

import (
	"context"
	"encoding/json"
	"log"

	"github.com/quiby-ai/common/pkg/events"
	"github.com/quiby-ai/review-preprocessor/config"
	"github.com/quiby-ai/review-preprocessor/internal/service"
	"github.com/segmentio/kafka-go"
)

type KafkaConsumer struct {
	reader *kafka.Reader
	svc    *service.PreprocessService
}

func NewKafkaConsumer(cfg config.KafkaConfig, svc *service.PreprocessService) *KafkaConsumer {
	reader := kafka.NewReader(kafka.ReaderConfig{
		Brokers: cfg.Brokers,
		Topic:   events.PipelinePrepareRequest,
		GroupID: cfg.GroupID,
	})
	return &KafkaConsumer{reader: reader, svc: svc}
}

func (kc *KafkaConsumer) Run(ctx context.Context) error {
	for {
		m, err := kc.reader.ReadMessage(ctx)
		if err != nil {
			return err
		}
		var evt events.PrepareRequest
		if err := json.Unmarshal(m.Value, &evt); err != nil {
			log.Printf("invalid message: %v", err)
			continue
		}
		if err := kc.svc.Handle(ctx, evt); err != nil {
			log.Printf("handle error: %v", err)
		}
	}
}

func (kc *KafkaConsumer) Close() error {
	if kc.reader != nil {
		return kc.reader.Close()
	}
	return nil
}
