package server

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/rs/zerolog"

	"github.com/theoptz/basic-s3/internal/rest/common"
	"github.com/theoptz/basic-s3/internal/rest/config"
	"github.com/theoptz/basic-s3/internal/rest/orchestrator"
)

const (
	defaultShutdownTimeout = 15 * time.Second
	defaultConcurrency     = 1000
	defaultBodyLimit       = 1 * 1024 * 1024
)

type Server struct {
	endpoint string

	app *fiber.App
	cfg fiber.Config

	service orchestrator.Orchestrator
	logger  zerolog.Logger
}

func (s *Server) Listen() error {
	s.app = fiber.New(s.cfg)

	s.app.Use(recover.New())

	s.app.Put("/:bucket/:key", s.handleUpload)
	s.app.Get("/:bucket/:key", s.handleDownload)

	return s.app.Listen(s.endpoint)
}

func (s *Server) Shutdown() error {
	if s.app != nil {
		return s.app.ShutdownWithTimeout(defaultShutdownTimeout)
	}

	return nil
}

func New(cfg config.Config, service orchestrator.Orchestrator, logger zerolog.Logger) *Server {
	conf := getDefaultConfig()
	if cfg.MaxConnections != 0 {
		conf.Concurrency = cfg.MaxConnections
	}
	if cfg.MaxBodySize != 0 {
		conf.BodyLimit = cfg.MaxBodySize
	}

	conf.ErrorHandler = makeErrorHandler(logger)

	return &Server{
		endpoint: fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		cfg:      conf,
		service:  service,
		logger:   logger,
	}
}

func (s *Server) handleUpload(ctx fiber.Ctx) (err error) {
	bucket, key, err := s.getBucketAndKeyFromContext(ctx)
	if err != nil {
		return err
	}

	contentLength, err := strconv.Atoi(ctx.Get("content-length", "0"))
	if err != nil {
		return fmt.Errorf("%w: invalid content-length provided: %w", common.ErrBadRequest, err)
	} else if contentLength == 0 {
		return fmt.Errorf("%w: invalid content length provided", common.ErrBadRequest)
	}

	err = s.service.Upload(
		ctx.Context(),
		&orchestrator.UploadRequest{
			Bucket:        bucket,
			Key:           key,
			ContentLength: contentLength,
			ContentType:   ctx.Get("Content-Type", "text/plain"),
		},
		ctx.Context().RequestBodyStream(),
	)

	if err != nil {
		return fmt.Errorf("upload failed: %w", err)
	}

	return nil
}

func (s *Server) handleDownload(ctx fiber.Ctx) error {
	bucket, key, err := s.getBucketAndKeyFromContext(ctx)
	if err != nil {
		return err
	}

	contentType, streamWriter, err := s.service.Download(ctx.Context(), &orchestrator.DownloadRequest{
		Bucket: bucket,
		Key:    key,
	})
	if err != nil {
		return fmt.Errorf("download failed: %w", err)
	}

	ctx.Response().Header.Set("Content-Type", contentType)
	ctx.Context().SetBodyStreamWriter(streamWriter)

	return nil
}

func (s *Server) getBucketAndKeyFromContext(ctx fiber.Ctx) (string, string, error) {
	bucket := ctx.Params("bucket")
	if bucket == "" {
		return "", "", fmt.Errorf("%w: empty bucket provided", common.ErrBadRequest)
	}
	key := ctx.Params("key")
	if key == "" {
		return "", "", fmt.Errorf("%w: empty key provided", common.ErrBadRequest)
	}

	return bucket, key, nil
}

func getDefaultConfig() fiber.Config {
	return fiber.Config{
		BodyLimit:         defaultBodyLimit,
		Concurrency:       defaultConcurrency,
		StreamRequestBody: true,
	}
}

func makeErrorHandler(logger zerolog.Logger) fiber.ErrorHandler {
	return func(ctx fiber.Ctx, err error) error {
		code := http.StatusInternalServerError

		switch {
		case errors.Is(err, common.ErrNotFound):
			code = http.StatusNotFound
		case errors.Is(err, common.ErrBadRequest):
			code = http.StatusBadRequest
		default:
			var e *fiber.Error
			if errors.As(err, &e) {
				code = e.Code
			}
		}

		if code == http.StatusInternalServerError {
			logger.Error().Err(err).Msg("Internal server error")
		}

		ctx.Status(code)
		return nil
	}
}
