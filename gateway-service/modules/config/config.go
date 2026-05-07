package config

import "os"

type Config struct {
	HTTPPort           string
	PaymentServiceAddr string
}

func Load() *Config {
	return &Config{
		HTTPPort:           os.Getenv("HTTP_PORT"),
		PaymentServiceAddr: os.Getenv("PAYMENT_SERVICE_ADDR"),
	}
}
