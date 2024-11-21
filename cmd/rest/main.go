package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os/signal"
	"syscall"

	"github.com/rs/zerolog/log"

	"github.com/theoptz/basic-s3/internal/rest/config"
	"github.com/theoptz/basic-s3/internal/rest/distributor/weight"
	"github.com/theoptz/basic-s3/internal/rest/meta/inmemory"
	"github.com/theoptz/basic-s3/internal/rest/orchestrator/service"
	"github.com/theoptz/basic-s3/internal/rest/server"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to parse config")
	}

	log.Debug().Any("config", cfg).Msg("Parsed config")

	if len(cfg.Storages) == 0 {
		log.Fatal().Msg("No storage configured")
	}

	metaStorage, err := inmemory.New(
		cfg.MetaFile,
		log.With().Str("pkg", "meta").Logger(),
	)
	if err != nil {
		log.Fatal().Err(err).Msg("failed to initialize meta storage")
	}

	partDistributor, err := weight.New(weight.DistributorConfig{
		Endpoints:   cfg.Storages,
		Weights:     cfg.Weights,
		MaxParts:    cfg.MaxParts,
		MinPartSize: cfg.MinPartSize,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("failed to create part distributor")
	}

	orchestrator := service.New(
		metaStorage,
		partDistributor,
		log.With().Str("pkg", "service").Logger(),
		cfg.ChunkSize,
	)

	srv := server.New(
		*cfg,
		orchestrator,
		log.With().Str("pkg", "server").Logger(),
	)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	go func() {
		defer stop()

		log.Info().Str("endpoint", fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)).Msg("listening")
		if stopErr := srv.Listen(); stopErr != nil && !errors.Is(stopErr, http.ErrServerClosed) {
			log.Error().Err(stopErr).Msg("failed to listen and serve")
		}
	}()

	<-ctx.Done()

	if err = srv.Shutdown(); err != nil {
		log.Error().Err(err).Msg("failed to shutdown server")
	}

	if err = metaStorage.Close(); err != nil {
		log.Error().Err(err).Msg("failed to close meta storage")
	}
}
