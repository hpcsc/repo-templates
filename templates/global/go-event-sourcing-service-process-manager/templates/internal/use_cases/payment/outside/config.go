package outside

type Config struct {
	FraudServiceURL     string `env:"FRAUD_SERVICE_URL" envDefault:"http://localhost:4444/fraud"`
	SanctionsServiceURL string `env:"SANCTIONS_SERVICE_URL" envDefault:"http://localhost:4444/sanctions"`
	PaymentNetworkURL   string `env:"PAYMENT_NETWORK_URL" envDefault:"http://localhost:4444/network"`
	WebhookURL          string `env:"WEBHOOK_URL" envDefault:"http://localhost:3333/webhooks"`
}
