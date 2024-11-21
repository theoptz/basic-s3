package service

import (
	"google.golang.org/grpc"

	storage "github.com/theoptz/basic-s3/proto"
)

type streamInfo struct {
	Bucket   string
	Key      string
	Version  int
	Part     int
	Size     int
	ClientID int
}

type streamWriter struct {
	stream grpc.ClientStreamingClient[storage.UploadRequest, storage.UploadResponse]
	info   streamInfo

	packetNumber int
}

func (w *streamWriter) Write(p []byte) (n int, err error) {
	defer func() {
		w.packetNumber++
	}()

	frame := storage.UploadRequest{
		Chunk: p,
	}
	if w.packetNumber == 0 {
		frame.Bucket = w.info.Bucket
		frame.Key = w.info.Key
		frame.Version = int32(w.info.Version)
		frame.Part = int32(w.info.Part)
		frame.Size = int32(w.info.Size)
	}

	err = w.stream.Send(&frame)
	if err != nil {
		return 0, err
	}

	return len(p), nil
}

func newStreamWriter(info streamInfo, stream grpc.ClientStreamingClient[storage.UploadRequest, storage.UploadResponse]) *streamWriter {
	return &streamWriter{
		stream: stream,
		info:   info,
	}
}
