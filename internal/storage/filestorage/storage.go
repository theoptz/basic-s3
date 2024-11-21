package filestorage

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	"github.com/theoptz/basic-s3/internal/storage"
)

const (
	partExtension = ".bin"
)

type FileStorage struct {
	dir string
}

func New(dir string) *FileStorage {
	return &FileStorage{dir: dir}
}

func (s *FileStorage) NewWriteCloser(req *storage.FileRequest) (io.WriteCloser, error) {
	if req == nil {
		return nil, errors.New("empty request")
	}

	dir, filename := getDirAndFilename(s.dir, req)

	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return nil, fmt.Errorf("mkdirall: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return nil, fmt.Errorf("create file: %w", err)
	}

	return file, nil
}

func (s *FileStorage) NewReadCloser(req *storage.FileRequest) (io.ReadCloser, error) {
	if req == nil {
		return nil, errors.New("empty request")
	}

	_, filename := getDirAndFilename(s.dir, req)
	file, err := os.Open(filename)
	if err != nil {
		return nil, fmt.Errorf("open file: %w", err)
	}

	return file, nil
}

func getDirAndFilename(storageDir string, req *storage.FileRequest) (string, string) {
	filename := path.Join(
		storageDir,
		req.Bucket,
		req.Key,
		strconv.Itoa(req.Version),
		strconv.Itoa(req.Part)+partExtension,
	)

	return path.Dir(filename), filename
}
