package paymentgateway

import "time"

type Config struct {
	Enable     bool          `mapstructure:"enable"`
	BaseURL    string        `mapstructure:"base_url"`
	Timeout    time.Duration `mapstructure:"timeout"`
	MaxRetries int           `mapstructure:"max_retries"`
	APIKey     string        `mapstructure:"api_key"`
}
