package consumer

import (
	"context"
	"fmt"

	"github.com/quiby-ai/common/pkg/events"
	"github.com/quiby-ai/review-preprocessor/config"
	"github.com/quiby-ai/review-preprocessor/internal/service"
)

type PreprocessServiceProcessor struct {
	svc *service.PreprocessService
}

func (p *PreprocessServiceProcessor) Handle(ctx context.Context, payload any, sagaID string) error {
	if evt, ok := payload.(events.PrepareRequest); ok {
		return p.svc.Handle(ctx, evt, sagaID)
	}
	return fmt.Errorf("invalid payload type for preprocess service")
}

type KafkaConsumer struct {
	consumer *events.KafkaConsumer
}

func NewKafkaConsumer(cfg config.KafkaConfig, svc *service.PreprocessService) *KafkaConsumer {
	consumer := events.NewKafkaConsumer(cfg.Brokers, events.PipelinePrepareRequest, cfg.GroupID)
	processor := &PreprocessServiceProcessor{svc: svc}
	consumer.SetProcessor(processor)
	return &KafkaConsumer{consumer: consumer}
}

func (kc *KafkaConsumer) Run(ctx context.Context) error {
	return kc.consumer.Run(ctx)
}

func (kc *KafkaConsumer) Close() error {
	return kc.consumer.Close()
}
