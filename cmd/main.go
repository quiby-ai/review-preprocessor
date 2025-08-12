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
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db := storage.MustInitPostgres(cfg.Postgres)
	repoRaw := storage.NewRawRepository(db)
	repoClean := storage.NewCleanRepository(db)

	prod := producer.NewKafkaProducer(cfg.Kafka)
	svc := service.NewPreprocessService(repoRaw, repoClean, prod, cfg.Processing)

	cons := consumer.NewKafkaConsumer(cfg.Kafka, svc)
	if err := cons.Run(ctx); err != nil {
		log.Fatalf("consumer exited with error: %v", err)
	}
}
