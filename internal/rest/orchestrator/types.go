package orchestrator

import (
	"context"
	"io"

	"github.com/valyala/fasthttp"
)

type UploadRequest struct {
	Bucket        string
	Key           string
	ContentLength int
	ContentType   string
}

type DownloadRequest struct {
	Bucket string
	Key    string
}

type Orchestrator interface {
	Upload(context.Context, *UploadRequest, io.Reader) error
	Download(context.Context, *DownloadRequest) (string, fasthttp.StreamWriter, error)
}
