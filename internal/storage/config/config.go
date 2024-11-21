package config

import "github.com/jessevdk/go-flags"

type Config struct {
	Host      string `long:"host" env:"HOST" description:"Host" default:"localhost"`
	Port      int    `long:"port" env:"PORT" description:"Port" default:"5555"`
	Directory string `long:"directory" env:"DIRECTORY" description:"Directory" default:"./files"`
}

func FromEnv() (*Config, error) {
	var cfg Config

	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.Parse(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
