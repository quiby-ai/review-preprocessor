package main

import (
	"context"
	"log"
	"os/signal"
	"syscall"

	"github.com/quiby-ai/review-preprocessor/config"
	"github.com/quiby-ai/review-preprocessor/internal/consumer"
	"github.com/quiby-ai/review-preprocessor/internal/producer"
	"github.com/quiby-ai/review-preprocessor/internal/service"
	"github.com/quiby-ai/review-preprocessor/internal/storage"
	"github.com/quiby-ai/review-preprocessor/internal/translate"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db := storage.MustInitPostgres(cfg.Postgres)
	repoRaw := storage.NewRawRepository(db)
	repoClean := storage.NewCleanRepository(db)

	prod := producer.NewProducer(cfg.Kafka)

	// translator selection (primary + optional fallback via cascade)
	var tr translate.Translator
	switch cfg.Processing.TranslateProvider {
	case "openai":
		primary := translate.NewOpenAIClient(cfg.OpenAI.Endpoint, cfg.OpenAI.Model, cfg.OpenAI.APIKey, cfg.Processing.TranslateTimeout)
		var fallback translate.Translator
		if cfg.Processing.TranslateFallbackEnabled {
			fb := translate.NewOpenAIClient(cfg.OpenAI.Endpoint, cfg.OpenAI.Endpoint, cfg.OpenAI.APIKey, cfg.Processing.TranslateTimeout)
			fallback = fb
		}
		tr = translate.NewCascade(primary, fallback, cfg.Processing.TranslateFallbackSample, cfg.Processing.TranslateFallbackAdequacyRatio)
	default:
		tr = translate.Noop{}
	}
	svc := service.NewPreprocessService(repoRaw, repoClean, prod, cfg.Processing, tr)

	cons := consumer.NewKafkaConsumer(cfg.Kafka, svc)
	if err := cons.Run(ctx); err != nil {
		log.Fatalf("consumer exited with error: %v", err)
	}
}
