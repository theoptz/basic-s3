package server

import (
	"bytes"
	"errors"
	"fmt"
	"io"

	"google.golang.org/grpc"

	"github.com/rs/zerolog"

	"github.com/theoptz/basic-s3/internal/storage"
	"github.com/theoptz/basic-s3/proto"
)

const (
	chunkSize = 8 * 1024
)

type StorageServer struct {
	proto.UnimplementedStorageServer
	store  storage.Storage
	logger zerolog.Logger
}

func New(store storage.Storage, logger zerolog.Logger) *StorageServer {
	return &StorageServer{
		store:  store,
		logger: logger,
	}
}

func (s *StorageServer) Upload(stream grpc.ClientStreamingServer[proto.UploadRequest, proto.UploadResponse]) error {
	var fwr io.WriteCloser

	defer func() {
		if fwr != nil {
			_ = fwr.Close()
		}
	}()

	var fileReq storage.FileRequest
	initialized := false

	for {
		req, err := stream.Recv()
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return err
		}

		if !initialized {
			if req.Bucket == "" || req.Key == "" {
				return errors.New("invalid first chunk")
			}

			fileReq = storage.FileRequest{
				Bucket:  req.Bucket,
				Key:     req.Key,
				Version: int(req.Version),
				Part:    int(req.Part),
			}

			fwr, err = s.store.NewWriteCloser(&fileReq)
			if err != nil {
				return err
			}

			initialized = true
		}

		if _, err = io.CopyN(fwr, bytes.NewBuffer(req.Chunk), int64(len(req.Chunk))); err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("failed to copy chunk: %w", err)
		}
	}

	if !initialized {
		// empty stream
		return stream.SendAndClose(&proto.UploadResponse{})
	}

	return stream.SendAndClose(&proto.UploadResponse{})
}

func (s *StorageServer) Download(req *proto.DownloadRequest, stream grpc.ServerStreamingServer[proto.DownloadResponse]) error {
	if req.Bucket == "" || req.Key == "" {
		return errors.New("invalid request")
	}

	frd, err := s.store.NewReadCloser(&storage.FileRequest{
		Bucket:  req.Bucket,
		Key:     req.Key,
		Version: int(req.Version),
		Part:    int(req.Part),
	})
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}

	defer func() {
		if frd != nil {
			_ = frd.Close()
		}
	}()

	buf := make([]byte, chunkSize)

	var n int

	for {
		if n, err = frd.Read(buf); err != nil {
			if errors.Is(err, io.EOF) {
				break
			}

			return fmt.Errorf("failed to read chunk: %w", err)
		}

		buf = buf[:n]

		if err = stream.Send(&proto.DownloadResponse{
			Chunk: buf[:n],
		}); err != nil {
			return fmt.Errorf("failed to send chunk: %w", err)
		}
	}

	return nil
}
