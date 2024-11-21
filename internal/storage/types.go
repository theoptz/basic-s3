package storage

import "io"

type FileRequest struct {
	Bucket  string
	Key     string
	Version int
	Part    int
}

type Storage interface {
	NewWriteCloser(*FileRequest) (io.WriteCloser, error)
	NewReadCloser(*FileRequest) (io.ReadCloser, error)
}
