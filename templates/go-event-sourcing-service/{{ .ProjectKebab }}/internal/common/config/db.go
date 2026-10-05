package config

import "fmt"

type DB struct {
	Username string `env:"DB_USERNAME,required"`
	Password string `env:"DB_PASSWORD,required"`
	Host     string `env:"DB_HOST,required"`
	HostPort string `env:"DB_HOST_PORT,required"`
	Name     string `env:"DB_NAME,required"`
}

func (c *DB) ConnectionString() string {
	return fmt.Sprintf(
		"postgres://%s:%s@%s:%s/%s",
		c.Username,
		c.Password,
		c.Host,
		c.HostPort,
		c.Name,
	)
}
