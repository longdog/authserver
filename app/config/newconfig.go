package config

import "os"

func NewConfig() *Config {
	return &Config{
		Port:       os.Getenv("LISTEN_PORT"),
		UrlPrefix:  os.Getenv("URL_PREFIX"),
		RpcSecret:  os.Getenv("RPC_SECRET"),
		JwtSecret:  os.Getenv("JWT_SECRET"),
		PgHost:     os.Getenv("PG_HOST"),
		PgPort:     os.Getenv("PG_PORT"),
		PgDb:       os.Getenv("PG_DB"),
		PgSchema:   os.Getenv("PG_SCHEMA"),
		PgUser:     os.Getenv("PG_USER"),
		PgPassword: os.Getenv("PG_PASSWORD"),
		SentryDsn:  os.Getenv("SENTRY_DSN"),
	}
}
