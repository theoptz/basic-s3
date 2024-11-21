package service

import (
	"context"
	"errors"
	"fmt"
	"io"

	"github.com/hashicorp/go-multierror"

	"google.golang.org/grpc"

	"github.com/theoptz/basic-s3/internal/rest/meta"
	"github.com/theoptz/basic-s3/internal/rest/orchestrator"
	"github.com/theoptz/basic-s3/proto"
)

const (
	chunkSize = 8 * 1024
)

func (s *Service) Upload(ctx context.Context, req *orchestrator.UploadRequest, body io.Reader) (err error) {
	metaFile := &meta.File{
		Bucket: req.Bucket,
		Key:    req.Key,
	}

	fv, err := s.metaClient.NewVersion(ctx, metaFile, req.ContentType)
	if err != nil {
		return fmt.Errorf("failed to create meta file version: %w", err)
	}
	defer func() {
		var status meta.Status = meta.StatusReady
		if err != nil {
			status = meta.StatusError
		}

		if updErr := s.metaClient.UpdateStatus(ctx, metaFile, &meta.FileVersion{
			Version: fv.Version,
			Status:  status,
		}); updErr != nil {
			err = multierror.Append(err, updErr)
		}
	}()

	totalLength := req.ContentLength
	clientIds, partSize := s.partDistributor.GetPlan(req.ContentLength)
	totalParts := len(clientIds)

	diff := totalLength - partSize*totalParts
	firstPartSize := partSize + diff

	var total, n int64

	part := -1

	for i := 0; i < totalParts; i++ {
		part++

		var curPartSize int
		if i == 0 {
			curPartSize = firstPartSize
		} else {
			curPartSize = partSize
		}

		n, err = s.uploadPart(
			ctx,
			streamInfo{
				Bucket:   req.Bucket,
				Key:      req.Key,
				Version:  fv.Version,
				Part:     part,
				Size:     curPartSize,
				ClientID: clientIds[i],
			},
			body,
		)
		total += n

		if err != nil {
			return fmt.Errorf("failed to upload part: %w", err)
		}

		if err = s.metaClient.NewPart(ctx, metaFile, fv, &meta.Part{
			Index:   part,
			Servers: []int{clientIds[i]},
		}); err != nil {
			return fmt.Errorf("failed to save meta for part %d: %w", part, err)
		}

		s.logger.Debug().Int("part", part).Int64("size", n).Msg("part uploaded")
	}

	s.logger.Debug().Str("bucket", req.Bucket).
		Str("key", req.Key).Int("version", fv.Version).Int64("size", total).Msg("file uploaded")

	return nil
}

func (s *Service) uploadPart(ctx context.Context, info streamInfo, body io.Reader) (n int64, err error) {
	var stream grpc.ClientStreamingClient[proto.UploadRequest, proto.UploadResponse]

	storageClient, err := s.partDistributor.GetClientByID(info.ClientID)
	if err != nil {
		return 0, fmt.Errorf("failed to get storage client: %w", err)
	}

	stream, err = storageClient.Upload(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to start stream: %w", err)
	}

	wr := newStreamWriter(info, stream)

	var copied int64
	for n < int64(info.Size) {
		copied, err = io.CopyN(wr, body, chunkSize)
		n += copied

		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return n, fmt.Errorf("failed to copy chunk: %w", err)
		}
	}

	_, err = stream.CloseAndRecv()
	if err != nil {
		return n, fmt.Errorf("failed to close stream: %w", err)
	}

	return n, nil
}
