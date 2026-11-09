package config

type NATS struct {
	URL string `env:"NATS_URL" envDefault:"nats://127.0.0.1:4222"`
}
