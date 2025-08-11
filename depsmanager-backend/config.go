package depsmanager

type Config struct {
	HTTPPort    int    `envconfig:"HTTP_PORT" default:"8085"`
	DepsAddress string `envconfig:"DEPS_ADDRESS" default:"https://api.deps.dev"`
	SQLLiteConfig
}

type SQLLiteConfig struct {
	DBPath      string `envconfig:"DB_PATH" default:"./deps.db"`
	BusyTimeout int64  `envconfig:"BUSY_TIMEOUT" default:"5000"`
}
