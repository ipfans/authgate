package config

import (
	"github.com/ipfans/authgate/routers"
	"github.com/ipfans/components/v2/configuration"
	"github.com/knadh/koanf/parsers/yaml"
)

type Config struct {
	Addr   string         `koanf:"addr"`
	Routes routers.Config `koanf:"routes"`
}

func LoadConfig() (Config, error) {
	var cfg Config
	err := configuration.Load(
		&cfg,
		configuration.WithConfigFile("config.yaml", yaml.Parser()),
	)
	return cfg, err
}
