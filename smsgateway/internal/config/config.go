package config

import (
	"fmt"

	"github.com/spf13/viper"
)

type Config struct {
	API API `mapstructure:"api"`
}

type API struct {
	Port string `mapstructure:"port"`
}

func Load() (cfg *Config, err error) {
	viper.SetConfigName("config")
	viper.SetConfigType("yml")
	viper.AddConfigPath("./config")

	err = viper.ReadInConfig()
	if err != nil {
		return cfg, fmt.Errorf("failed to load config: %w", err)
	}

	err = viper.Unmarshal(&cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
