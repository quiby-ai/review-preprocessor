package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Kafka      KafkaConfig      `mapstructure:"kafka"`
	Postgres   PostgresConfig   `mapstructure:"postgres"`
	Processing ProcessingConfig `mapstructure:"processing"`
}

type KafkaConfig struct {
	Brokers             []string `mapstructure:"brokers"`
	GroupID             string   `mapstructure:"group_id"`
	TopicPrepareReviews string   `mapstructure:"topic_prepare_reviews"`
	TopicClusterReviews string   `mapstructure:"topic_cluster_reviews"`
	TopicFailed         string   `mapstructure:"topic_failed"`
}

type PostgresConfig struct {
	DSN string `mapstructure:"dsn"`
}

type ProcessingConfig struct {
	DefaultLang         string        `mapstructure:"default_lang"`
	BatchSize           int           `mapstructure:"batch_size"`
	PublishIDsLimit     int           `mapstructure:"publish_ids_limit"`
	MaxReviewLen        int           `mapstructure:"max_review_len"`
	MinContentLen       int           `mapstructure:"min_content_len"`
	HTMLStrip           bool          `mapstructure:"html_strip"`
	EmojiStrip          bool          `mapstructure:"emoji_strip"`
	WhitespaceNormalize bool          `mapstructure:"whitespace_normalize"`
	TimeoutPerBatch     time.Duration `mapstructure:"timeout_per_batch"`
}

func LoadConfig() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath(".")
	viper.AddConfigPath("./config")

	setDefaults()

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// seconds → durations mapping for convenience
	cfg.Processing.TimeoutPerBatch = time.Duration(viper.GetInt("processing.timeout_seconds")) * time.Second

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}
	return &cfg, nil
}

func setDefaults() {
	viper.SetDefault("kafka.brokers", []string{"localhost:9092"})
	viper.SetDefault("kafka.group_id", "review-preprocessor")
	viper.SetDefault("kafka.topic_prepare_reviews", "pipeline.prepare_reviews")
	viper.SetDefault("kafka.topic_cluster_reviews", "pipeline.cluster_reviews")
	viper.SetDefault("kafka.topic_failed", "pipeline.failed")

	viper.SetDefault("postgres.dsn", "postgres://user:pass@localhost:5432/reviews?sslmode=disable")

	viper.SetDefault("processing.default_lang", "en")
	viper.SetDefault("processing.batch_size", 200)
	viper.SetDefault("processing.publish_ids_limit", 500)
	viper.SetDefault("processing.max_review_len", 8000)
	viper.SetDefault("processing.min_content_len", 10)
	viper.SetDefault("processing.html_strip", true)
	viper.SetDefault("processing.emoji_strip", true)
	viper.SetDefault("processing.whitespace_normalize", true)
	viper.SetDefault("processing.timeout_seconds", 30)
}

func (c *Config) Validate() error {
	if err := c.Kafka.Validate(); err != nil {
		return fmt.Errorf("kafka config: %w", err)
	}
	if err := c.Postgres.Validate(); err != nil {
		return fmt.Errorf("postgres config: %w", err)
	}
	return nil
}

func (k *KafkaConfig) Validate() error {
	if len(k.Brokers) == 0 {
		return fmt.Errorf("at least one broker is required")
	}
	if k.GroupID == "" {
		return fmt.Errorf("group_id is required")
	}
	if k.TopicPrepareReviews == "" {
		return fmt.Errorf("topic_prepare_reviews is required")
	}
	if k.TopicClusterReviews == "" {
		return fmt.Errorf("topic_cluster_reviews is required")
	}
	if k.TopicFailed == "" {
		return fmt.Errorf("topic_failed is required")
	}
	return nil
}

func (p *PostgresConfig) Validate() error {
	if p.DSN == "" {
		return fmt.Errorf("dsn is required")
	}
	return nil
}
