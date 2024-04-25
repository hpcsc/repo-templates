package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
)

type Config struct {
	Port string `env:"PORT" envDefault:"3333"`
}

func Load() (*Config, error) {
	var c Config
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %v", err)
	}
	return &c, nil
}
