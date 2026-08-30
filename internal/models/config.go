package models

type ServerConfig struct {
	Port   string `env:"PORT" validate:"required"`
	AppEnv string `env:"APP_ENV" validate:"required"`
}