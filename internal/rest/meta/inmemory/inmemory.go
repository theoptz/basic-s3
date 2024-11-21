package inmemory

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/hashicorp/go-multierror"
	"github.com/rs/zerolog"

	"github.com/theoptz/basic-s3/internal/rest/common"
	"github.com/theoptz/basic-s3/internal/rest/meta"
)

type Meta struct {
	state  map[string][]meta.FileVersion
	file   string
	logger zerolog.Logger
	closed bool
	mu     sync.RWMutex
}

func New(filename string, logger zerolog.Logger) (res *Meta, err error) {
	f, err := os.OpenFile(filename, os.O_CREATE, 0666)
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = multierror.Append(err, closeErr)
		}
	}()

	by, err := io.ReadAll(f)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	state := make(map[string][]meta.FileVersion)
	if len(by) > 0 {
		err = json.Unmarshal(by, &state)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal file: %w", err)
		}
	}

	return &Meta{
		file:   filename,
		state:  state,
		logger: logger,
	}, nil
}

func (m *Meta) NewVersion(ctx context.Context, file *meta.File, contentType string) (*meta.FileVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if file == nil {
		return nil, fmt.Errorf("%w: no file provided", common.ErrBadRequest)
	}

	fv := meta.FileVersion{
		Status:      meta.StatusLoading,
		ContentType: contentType,
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	filename := file.String()

	_, ok := m.state[filename]
	if !ok {
		m.state[filename] = make([]meta.FileVersion, 0, 8)
	}

	fv.Version = len(m.state[filename])
	m.state[filename] = append(m.state[filename], fv)

	m.logger.Debug().
		Str("bucket", file.Bucket).
		Str("key", file.Key).
		Int("version", fv.Version).
		Msg("new version created")

	return &fv, nil
}

func (m *Meta) UpdateStatus(ctx context.Context, f *meta.File, fv *meta.FileVersion) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if f == nil {
		return fmt.Errorf("%w: no file provided", common.ErrBadRequest)
	} else if fv == nil {
		return fmt.Errorf("%w: no file version provided", common.ErrBadRequest)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	versions, ok := m.state[f.String()]
	if !ok {
		return fmt.Errorf("%w: file not found", common.ErrNotFound)
	}

	for i := range versions {
		if versions[i].Version == fv.Version {
			if !canChangeStatus(versions[i].Status, fv.Status) {
				return fmt.Errorf("%w: can't update final status", common.ErrBadRequest)
			}

			versions[i].Status = fv.Status
			return nil
		}
	}

	return fmt.Errorf("%w: file version not found", common.ErrNotFound)
}

func (m *Meta) NewPart(ctx context.Context, f *meta.File, fv *meta.FileVersion, p *meta.Part) error {
	if err := ctx.Err(); err != nil {
		return err
	}

	if f == nil {
		return fmt.Errorf("%w: no file provided", common.ErrBadRequest)
	} else if fv == nil {
		return fmt.Errorf("%w: no file version provided", common.ErrBadRequest)
	} else if p == nil {
		return fmt.Errorf("%w: no part provided", common.ErrBadRequest)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	versions, ok := m.state[f.String()]
	if !ok {
		return fmt.Errorf("%w: file not found", common.ErrNotFound)
	}

	if len(versions) <= fv.Version {
		return fmt.Errorf("%w: file version not found", common.ErrNotFound)
	}

	if p.Index != len(versions[fv.Version].Parts) {
		return fmt.Errorf("%w: invalid part index", common.ErrBadRequest)
	}

	versions[fv.Version].Parts = append(versions[fv.Version].Parts, *p)

	return nil
}

func (m *Meta) GetVersion(ctx context.Context, f *meta.File) (*meta.FileVersion, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	if f == nil {
		return nil, fmt.Errorf("%w: no file provided", common.ErrBadRequest)
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	file, ok := m.state[f.String()]
	if !ok {
		return nil, fmt.Errorf("%w: file not found", common.ErrNotFound)
	}

	if len(file) == 0 {
		return nil, fmt.Errorf("%w: no file versions found", common.ErrNotFound)
	}

	var fv meta.FileVersion
	idx := len(file) - 1
	isFound := false

	for idx >= 0 {
		fv = file[idx]
		if fv.Status == meta.StatusReady {
			isFound = true
			break
		}

		idx--
	}

	if !isFound {
		return nil, fmt.Errorf("%w: file version not found", common.ErrNotFound)
	}

	m.logger.Debug().
		Str("bucket", f.Bucket).
		Str("key", f.Key).
		Int("version", fv.Version).
		Int("parts", len(fv.Parts)).
		Msg("get version")

	return &fv, nil
}

func (m *Meta) Close() (err error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil
	}
	m.closed = true

	by, err := json.Marshal(m.state)
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	f, err := os.Create(m.file)
	if err != nil {
		return fmt.Errorf("failed to create file: %w", err)
	}
	defer func() {
		if closeErr := f.Close(); closeErr != nil {
			err = multierror.Append(err, closeErr)
		}
	}()

	n, err := io.Copy(f, bytes.NewReader(by))
	if err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	} else if n != int64(len(by)) {
		return fmt.Errorf("failed to write file: wrote %d of %d bytes", n, len(by))
	}

	return nil
}

func canChangeStatus(prev, next meta.Status) bool {
	return prev == next || prev == meta.StatusLoading
}
