package main

import (
	"context"
	"fmt"
	"net"
	"os/signal"
	"syscall"

	"github.com/grpc-ecosystem/go-grpc-middleware/v2/interceptors/recovery"
	"github.com/rs/zerolog/log"

	"google.golang.org/grpc"

	"github.com/theoptz/basic-s3/internal/storage/config"
	"github.com/theoptz/basic-s3/internal/storage/filestorage"
	"github.com/theoptz/basic-s3/internal/storage/server"
	"github.com/theoptz/basic-s3/proto"
)

func main() {
	cfg, err := config.FromEnv()
	if err != nil {
		log.Fatal().Err(err).Msg("failed to load config from env")
	}

	log.Debug().Any("config", cfg).Msg("Parsed config")

	endpoint := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)

	listener, err := net.Listen("tcp", endpoint)
	if err != nil {
		log.Fatal().Err(err).Str("endpoint", endpoint).Msg("failed to listen")
	}

	storage := filestorage.New(cfg.Directory)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	grpcPanicRecoveryHandler := func(p any) error {
		log.Error().Any("panic", p).Stack().Msg("Panic")
		return nil
	}

	srv := grpc.NewServer(
		grpc.ChainUnaryInterceptor(
			recovery.UnaryServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
		),
		grpc.ChainStreamInterceptor(
			recovery.StreamServerInterceptor(recovery.WithRecoveryHandler(grpcPanicRecoveryHandler)),
		),
	)
	proto.RegisterStorageServer(
		srv,
		server.New(
			storage,
			log.With().Str("pkg", "grpc").Logger(),
		),
	)

	go func() {
		defer stop()

		log.Info().Str("endpoint", endpoint).Msg("listening")
		if err = srv.Serve(listener); err != nil {
			log.Error().Err(err).Msg("failed to serve")
		}
	}()

	<-ctx.Done()

	srv.GracefulStop()
}
