package config

import (
	"fmt"

	"github.com/caarlos0/env/v6"
)

func Load[T any]() (*T, error) {
	var c T
	if err := env.Parse(&c); err != nil {
		return nil, fmt.Errorf("failed to parse configuration: %v", err)
	}
	return &c, nil
}
