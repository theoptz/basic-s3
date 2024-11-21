package meta

import (
	"context"
	"errors"
	"strings"
)

type Meta interface {
	NewVersion(context.Context, *File, string) (*FileVersion, error)
	NewPart(context.Context, *File, *FileVersion, *Part) error
	UpdateStatus(context.Context, *File, *FileVersion) error
	GetVersion(context.Context, *File) (*FileVersion, error)
}

type File struct {
	Bucket string `json:"bucket"`
	Key    string `json:"key"`
}

func (f File) String() string {
	return strings.Join([]string{f.Bucket, f.Key}, "/")
}

func FileFromString(s string) (File, error) {
	parts := strings.SplitN(s, "/", 2)
	if len(parts) != 2 {
		return File{}, errors.New("invalid file format")
	}

	return File{Bucket: parts[0], Key: parts[1]}, nil
}

type FileVersion struct {
	Version     int    `json:"version"`
	ContentType string `json:"content_type"`
	Status      Status `json:"status"`
	Parts       []Part `json:"parts"`
}

type Part struct {
	Servers []int `json:"servers"`
	Index   int   `json:"index"`
}

type FilePart struct {
	Bucket  string
	Key     string
	Version int
	Part    int
}

type Status string

const (
	StatusLoading = "loading"
	StatusReady   = "ready"
	StatusError   = "error"
)
