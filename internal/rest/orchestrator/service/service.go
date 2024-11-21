package service

import (
	"github.com/rs/zerolog"

	"github.com/theoptz/basic-s3/internal/rest/distributor"
	"github.com/theoptz/basic-s3/internal/rest/meta"
)

type Service struct {
	metaClient      meta.Meta
	partDistributor distributor.Distributor
	logger          zerolog.Logger

	chunkSize int
}

func New(
	metaClient meta.Meta,
	partDistributor distributor.Distributor,
	logger zerolog.Logger,
	chunkSize int,
) *Service {
	return &Service{
		metaClient:      metaClient,
		partDistributor: partDistributor,
		logger:          logger,
		chunkSize:       chunkSize,
	}
}
