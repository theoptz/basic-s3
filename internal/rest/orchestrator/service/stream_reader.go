package service

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/rs/zerolog"
	"io"

	"google.golang.org/grpc"

	"github.com/theoptz/basic-s3/proto"
)

type getServerStreamFunc func(part int) (grpc.ServerStreamingClient[proto.DownloadResponse], error)

type streamReader struct {
	buf              *bytes.Buffer
	stream           grpc.ServerStreamingClient[proto.DownloadResponse]
	getStreamForPart getServerStreamFunc
	totalParts       int
	currentPart      int
	eof              bool
	logger           zerolog.Logger
}

func (s *streamReader) Read(p []byte) (n int, err error) {
	if s.buf.Len() > 0 {
		return s.buf.Read(p)
	}

	if s.eof {
		return 0, io.EOF
	}

	chunk, err := s.getChunk()
	if err != nil {
		if errors.Is(err, io.EOF) {
			s.eof = true
			return 0, io.EOF
		}

		return 0, err
	}

	s.buf.Reset()

	if _, err = io.CopyN(s.buf, bytes.NewBuffer(chunk), int64(len(chunk))); err != nil && !errors.Is(err, io.EOF) {
		return 0, fmt.Errorf("failed to write chunk: %w", err)
	}

	return s.buf.Read(p)
}

func (s *streamReader) getChunk() (chunk []byte, err error) {
	if s.stream == nil {
		if err = s.setNextStream(); err != nil {
			return nil, err
		}
	}

	var res *proto.DownloadResponse
	for {
		res, err = s.stream.Recv()
		if err == nil {
			break
		} else if !errors.Is(err, io.EOF) {
			return nil, err
		}

		s.logger.Debug().Int("part", s.currentPart).Msg("part downloaded")
		err = s.setNextStream()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return nil, io.EOF
			}

			return nil, err
		}
	}

	if res != nil {
		return res.Chunk, nil
	}

	return nil, errors.New("empty response")
}

func (s *streamReader) setNextStream() error {
	if (s.currentPart + 1) >= s.totalParts {
		return io.EOF
	}

	var err error
	s.currentPart++

	s.stream, err = s.getStreamForPart(s.currentPart)
	if err != nil {
		return fmt.Errorf("failed to get stream for part %d: %w", s.currentPart, err)
	}

	return nil
}

func newStreamReader(getStreamForPart getServerStreamFunc, totalParts int, logger zerolog.Logger) *streamReader {
	return &streamReader{
		buf:              bytes.NewBuffer(make([]byte, 0, chunkSize)),
		getStreamForPart: getStreamForPart,
		totalParts:       totalParts,
		currentPart:      -1,
		eof:              false,
		logger:           logger,
	}
}
