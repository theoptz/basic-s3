package service

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"math/rand/v2"

	"github.com/valyala/fasthttp"

	"google.golang.org/grpc"

	"github.com/theoptz/basic-s3/internal/rest/meta"
	"github.com/theoptz/basic-s3/internal/rest/orchestrator"
	"github.com/theoptz/basic-s3/proto"
)

func (s *Service) Download(ctx context.Context, req *orchestrator.DownloadRequest) (string, fasthttp.StreamWriter, error) {
	metaFile := &meta.File{
		Bucket: req.Bucket,
		Key:    req.Key,
	}

	fv, err := s.metaClient.GetVersion(ctx, metaFile)
	if err != nil {
		return "", nil, fmt.Errorf("file not found: %w", err)
	} else if len(fv.Parts) == 0 {
		return "", nil, fmt.Errorf("file has no parts")
	}

	clients := make([]proto.StorageClient, len(fv.Parts))

	for i := 0; i < len(fv.Parts); i++ {
		if len(fv.Parts[i].Servers) == 0 {
			return "", nil, fmt.Errorf("failed to locate part server")
		}

		randId := fv.Parts[i].Servers[rand.IntN(len(fv.Parts[i].Servers))]
		cl, err := s.partDistributor.GetClientByID(randId)
		if err != nil {
			return "", nil, fmt.Errorf("failed to get storage client: %w", err)
		}

		clients[i] = cl
	}

	reader := newStreamReader(func(i int) (grpc.ServerStreamingClient[proto.DownloadResponse], error) {
		return clients[i].Download(ctx, &proto.DownloadRequest{
			Bucket:  metaFile.Bucket,
			Key:     metaFile.Key,
			Version: int32(fv.Version),
			Part:    int32(i),
		})
	}, len(fv.Parts), s.logger)

	return fv.ContentType, s.makeBodyStreamWriter(reader), nil
}

func (s *Service) makeBodyStreamWriter(reader io.Reader) fasthttp.StreamWriter {
	return func(writer *bufio.Writer) {
		var err error

		var total, n int64

		for {
			if n, err = io.CopyN(writer, reader, int64(s.chunkSize)); err != nil && !errors.Is(err, io.EOF) {
				return
			}
			total += n

			isEOF := errors.Is(err, io.EOF)

			if err = writer.Flush(); err != nil {
				return
			}

			if isEOF {
				break
			}
		}

		s.logger.Debug().Int64("size", total).Msg("file downloaded")
	}
}
