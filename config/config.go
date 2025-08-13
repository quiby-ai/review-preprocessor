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
	OpenAI     OpenAIConfig     `mapstructure:"openai"`
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

	// contentfulness thresholds
	MinWords      int     `mapstructure:"min_words"`
	MinChars      int     `mapstructure:"min_chars"`
	MinAlphaRatio float64 `mapstructure:"min_alpha_ratio"`
	SaveSkipped   bool    `mapstructure:"save_skipped"`

	// language detection / translation
	LangDetectMinConf   float64       `mapstructure:"lang_detect_min_conf"`
	TranslateEnabled    bool          `mapstructure:"translate_enabled"`
	TranslateTargetLang string        `mapstructure:"translate_target_lang"`
	TranslateBatchSize  int           `mapstructure:"translate_batch_size"`
	TranslateTimeout    time.Duration `mapstructure:"translate_timeout"`
	TranslateProvider   string        `mapstructure:"translate_provider"`

	// translation fallback
	TranslateFallbackEnabled       bool    `mapstructure:"translate_fallback_enabled"`
	TranslateFallbackModel         string  `mapstructure:"translate_fallback_model"`
	TranslateFallbackSample        int     `mapstructure:"translate_fallback_sample"`
	TranslateFallbackAdequacyRatio float64 `mapstructure:"translate_fallback_adequacy_ratio"`
}

type OpenAIConfig struct {
	APIKey   string `mapstructure:"api_key"`
	Model    string `mapstructure:"model"`
	Endpoint string `mapstructure:"endpoint"`
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
	// translate timeout in seconds
	if v := viper.GetInt("processing.translate_timeout_seconds"); v > 0 {
		cfg.Processing.TranslateTimeout = time.Duration(v) * time.Second
	} else {
		cfg.Processing.TranslateTimeout = 15 * time.Second
	}

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

	// contentfulness defaults
	viper.SetDefault("processing.min_words", 4)
	viper.SetDefault("processing.min_chars", 20)
	viper.SetDefault("processing.min_alpha_ratio", 0.35)
	viper.SetDefault("processing.save_skipped", true)

	// language / translation defaults
	viper.SetDefault("processing.lang_detect_min_conf", 0.70)
	viper.SetDefault("processing.translate_enabled", true)
	viper.SetDefault("processing.translate_target_lang", "en")
	viper.SetDefault("processing.translate_batch_size", 20)
	viper.SetDefault("processing.translate_timeout_seconds", 15)
	viper.SetDefault("processing.translate_provider", "openai")

	// fallback defaults
	viper.SetDefault("processing.translate_fallback_enabled", true)
	viper.SetDefault("processing.translate_fallback_model", "gpt-5-mini")
	viper.SetDefault("processing.translate_fallback_sample", 5)
	viper.SetDefault("processing.translate_fallback_adequacy_ratio", 0.5)

	// openai defaults
	viper.SetDefault("openai.api_key", "")
	viper.SetDefault("openai.model", "gpt-5-nano")
	viper.SetDefault("openai.endpoint", "https://api.openai.com/v1/chat/completions")
}

func (c *Config) Validate() error {
	if err := c.Kafka.Validate(); err != nil {
		return fmt.Errorf("kafka config: %w", err)
	}
	if err := c.Postgres.Validate(); err != nil {
		return fmt.Errorf("postgres config: %w", err)
	}
	if err := c.Processing.Validate(); err != nil {
		return fmt.Errorf("processing config: %w", err)
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

func (p *ProcessingConfig) Validate() error {
	if p.BatchSize <= 0 {
		return fmt.Errorf("batch_size must be > 0")
	}
	if p.TranslateEnabled {
		if p.TranslateTargetLang == "" {
			return fmt.Errorf("translate_target_lang is required when translate_enabled=true")
		}
		if p.TranslateBatchSize <= 0 {
			return fmt.Errorf("translate_batch_size must be > 0")
		}
	}
	return nil
}
