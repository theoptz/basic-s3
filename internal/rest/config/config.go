package config

import (
	"github.com/jessevdk/go-flags"
)

type Config struct {
	Host     string   `long:"host" env:"HOST" description:"Host" default:"localhost"`
	Port     int      `long:"port" env:"PORT" description:"Port" default:"8080"`
	MetaFile string   `long:"meta file" env:"META_FILE" description:"Meta state file" default:"meta.json"`
	Storages []string `long:"storages" env:"STORAGES" env-delim:"," description:"Storages" default:"localhost:5555"`
	Weights  []int    `long:"weights" env:"WEIGHTS" env-delim:"," description:"Weight for storages" default:"1"`

	MaxConnections int `long:"max-connections" env:"MAX_CONNECTIONS" description:"Max connections" default:"1000"`
	MaxBodySize    int `long:"max-body-size" env:"MAX_BODY_SIZE" description:"Max body size" default:"1073741824"`
	ChunkSize      int `long:"chunk-size" env:"CHUNK_SIZE" description:"Chunk size" default:"8192"`
	MinPartSize    int `long:"min-part-size" env:"MIN_PART_SIZE" description:"Min part size" default:"8192"`
	MaxParts       int `long:"max-parts" env:"MAX_PARTS" description:"Max parts" default:"6"`
}

func FromEnv() (*Config, error) {
	var cfg Config

	parser := flags.NewParser(&cfg, flags.Default)
	if _, err := parser.Parse(); err != nil {
		return nil, err
	}

	return &cfg, nil
}
