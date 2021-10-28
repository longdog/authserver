package config

type Config struct {
	Port,
	UrlPrefix,
	RpcSecret,
	JwtSecret,
	PgHost,
	PgPort,
	PgDb,
	PgSchema,
	PgUser,
	SentryDsn,
	PgPassword string
}
