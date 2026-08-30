package utils

import (
	"github.com/caarlos0/env/v11"
	"github.com/go-playground/validator/v10"
	"github.com/shubomifashakin/go-kv/internal/models"
)

var Validator *validator.Validate

func init() {
	Validator= validator.New()
}


func ValidateServerConfig() (*models.ServerConfig,error) {
	var cfg models.ServerConfig
	err := env.Parse(&cfg)
	if err != nil {
		return nil,err
	}

	err=Validator.Struct(cfg)

	if err != nil {
		return nil,err
	}

	return &cfg,err
}