package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

type Config struct {
	Kafka      KafkaConfig
	Postgres   PostgresConfig
	Processing ProcessingConfig
	OpenAI     OpenAIConfig
}

type KafkaConfig struct {
	Brokers []string
	GroupID string
}

type PostgresConfig struct {
	DSN string
}

type ProcessingConfig struct {
	DefaultLang         string
	BatchSize           int
	PublishIDsLimit     int
	MaxReviewLen        int
	MinContentLen       int
	HTMLStrip           bool
	EmojiStrip          bool
	WhitespaceNormalize bool
	TimeoutPerBatch     time.Duration

	// contentfulness thresholds
	MinWords      int
	MinChars      int
	MinAlphaRatio float64
	SaveSkipped   bool

	// language detection / translation
	LangDetectMinConf   float64
	TranslateEnabled    bool
	TranslateTargetLang string
	TranslateBatchSize  int
	TranslateTimeout    time.Duration
	TranslateProvider   string

	// translation fallback
	TranslateFallbackEnabled       bool
	TranslateFallbackModel         string
	TranslateFallbackSample        int
	TranslateFallbackAdequacyRatio float64
}

type OpenAIConfig struct {
	APIKey   string
	Model    string
	Endpoint string
}

func Load() (*Config, error) {
	viper.SetConfigName("config")
	viper.SetConfigType("toml")
	viper.AddConfigPath("/")

	if err := viper.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	// Bind environment variables
	viper.BindEnv("OPENAI_API_KEY")
	viper.BindEnv("PG_DSN")

	var config = &Config{
		Kafka: KafkaConfig{
			Brokers: viper.GetStringSlice("kafka.brokers"),
			GroupID: viper.GetString("kafka.group_id"),
		},
		Postgres: PostgresConfig{
			DSN: viper.GetString("PG_DSN"),
		},
		Processing: ProcessingConfig{
			DefaultLang:         viper.GetString("processing.default_lang"),
			BatchSize:           viper.GetInt("processing.batch_size"),
			PublishIDsLimit:     viper.GetInt("processing.publish_ids_limit"),
			MaxReviewLen:        viper.GetInt("processing.max_review_len"),
			MinContentLen:       viper.GetInt("processing.min_content_len"),
			HTMLStrip:           viper.GetBool("processing.html_strip"),
			EmojiStrip:          viper.GetBool("processing.emoji_strip"),
			WhitespaceNormalize: viper.GetBool("processing.whitespace_normalize"),

			MinWords:      viper.GetInt("processing.min_words"),
			MinChars:      viper.GetInt("processing.min_chars"),
			MinAlphaRatio: viper.GetFloat64("processing.min_alpha_ratio"),
			SaveSkipped:   viper.GetBool("processing.save_skipped"),

			LangDetectMinConf:   viper.GetFloat64("processing.lang_detect_min_conf"),
			TranslateEnabled:    viper.GetBool("processing.translate_enabled"),
			TranslateTargetLang: viper.GetString("processing.translate_target_lang"),
			TranslateBatchSize:  viper.GetInt("processing.translate_batch_size"),
			TranslateProvider:   viper.GetString("processing.translate_provider"),

			TranslateFallbackEnabled:       viper.GetBool("processing.translate_fallback_enabled"),
			TranslateFallbackModel:         viper.GetString("processing.translate_fallback_model"),
			TranslateFallbackSample:        viper.GetInt("processing.translate_fallback_sample"),
			TranslateFallbackAdequacyRatio: viper.GetFloat64("processing.translate_fallback_adequacy_ratio"),
		},
		OpenAI: OpenAIConfig{
			APIKey:   viper.GetString("OPENAI_API_KEY"),
			Model:    viper.GetString("openai.model"),
			Endpoint: viper.GetString("openai.endpoint"),
		},
	}

	// seconds → durations mapping for convenience
	config.Processing.TimeoutPerBatch = time.Duration(viper.GetInt("processing.timeout_seconds")) * time.Second
	// translate timeout in seconds
	if v := viper.GetInt("processing.translate_timeout_seconds"); v > 0 {
		config.Processing.TranslateTimeout = time.Duration(v) * time.Second
	} else {
		config.Processing.TranslateTimeout = 15 * time.Second
	}

	return config, nil
}
