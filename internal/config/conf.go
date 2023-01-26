package config

import (
	"github.com/kelseyhightower/envconfig"
)

const APP_CONF_PREFIX = "ASCRAPER"

type Config struct {
	Kafka struct {
		Username string `json:"username"`
		Password string `json:"password"`
		Topic    string `json:"topic"`
		Broker   string `json:"broker"`
	} `json:"kafka"`
}

func LoadConfig() (Config, error) {
	var conf Config
	err := envconfig.Process(APP_CONF_PREFIX, &conf)

	return conf, err
}
