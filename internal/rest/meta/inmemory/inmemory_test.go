package inmemory

import (
	"context"
	"fmt"
	"testing"

	"github.com/rs/zerolog"

	"github.com/stretchr/testify/assert"
	"github.com/theoptz/basic-s3/internal/rest/meta"
)

func TestMeta_GetVersion(t *testing.T) {
	const (
		bucket = "bucket"
		key    = "key"
	)

	type args struct {
		ctx context.Context
		f   *meta.File
	}
	tests := []struct {
		name    string
		storage *Meta
		args    args
		want    *meta.FileVersion
		wantErr assert.ErrorAssertionFunc
	}{
		{
			name: "expired context",
			storage: func() *Meta {
				return &Meta{}
			}(),
			args: args{
				ctx: func() context.Context {
					ctx, cancel := context.WithCancel(context.Background())
					cancel()
					return ctx
				}(),
				f: &meta.File{
					Bucket: bucket,
					Key:    key,
				},
			},
			wantErr: assert.Error,
		},
		{
			name: "no file provided",
			storage: func() *Meta {
				return &Meta{}
			}(),
			args: args{
				ctx: context.Background(),
				f:   nil,
			},
			wantErr: assert.Error,
		},
		{
			name: "file not found",
			storage: func() *Meta {
				res := &Meta{
					state:  make(map[string][]meta.FileVersion),
					logger: zerolog.Nop(),
				}

				return res
			}(),
			args: args{
				ctx: context.Background(),
				f: &meta.File{
					Bucket: bucket,
					Key:    key,
				},
			},
			wantErr: assert.Error,
		},
		{
			name: "no file versions found",
			storage: func() *Meta {
				state := make(map[string][]meta.FileVersion)
				state[meta.File{Bucket: bucket, Key: key}.String()] = []meta.FileVersion{}

				res := &Meta{
					state:  state,
					logger: zerolog.Nop(),
				}

				return res
			}(),
			args: args{
				ctx: context.Background(),
				f: &meta.File{
					Bucket: bucket,
					Key:    key,
				},
			},
			wantErr: assert.Error,
		},
		{
			name: "file versions not found",
			storage: func() *Meta {
				state := make(map[string][]meta.FileVersion)
				state[meta.File{Bucket: bucket, Key: key}.String()] = []meta.FileVersion{
					{
						Version:     0,
						ContentType: "text/plain",
						Status:      meta.StatusLoading,
						Parts:       nil,
					},
					{
						Version:     1,
						ContentType: "text/plain",
						Status:      meta.StatusError,
						Parts:       nil,
					},
				}

				res := &Meta{
					state:  state,
					logger: zerolog.Nop(),
				}

				return res
			}(),
			args: args{
				ctx: context.Background(),
				f: &meta.File{
					Bucket: bucket,
					Key:    key,
				},
			},
			wantErr: assert.Error,
		},
		{
			name: "success",
			storage: func() *Meta {
				state := make(map[string][]meta.FileVersion)
				state[meta.File{Bucket: bucket, Key: key}.String()] = []meta.FileVersion{
					{
						Version:     0,
						ContentType: "text/plain",
						Status:      meta.StatusLoading,
						Parts:       nil,
					},
					{
						Version:     1,
						ContentType: "text/plain",
						Status:      meta.StatusError,
						Parts:       nil,
					},
					{
						Version:     2,
						ContentType: "text/plain",
						Status:      meta.StatusReady,
						Parts:       nil,
					},
				}

				res := &Meta{
					state:  state,
					logger: zerolog.Nop(),
				}

				return res
			}(),
			args: args{
				ctx: context.Background(),
				f: &meta.File{
					Bucket: bucket,
					Key:    key,
				},
			},
			want: &meta.FileVersion{
				Version:     2,
				ContentType: "text/plain",
				Status:      meta.StatusReady,
				Parts:       nil,
			},
			wantErr: assert.NoError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := tt.storage.GetVersion(tt.args.ctx, tt.args.f)
			if !tt.wantErr(t, err, fmt.Sprintf("GetVersion(%v, %v)", tt.args.ctx, tt.args.f)) {
				return
			}
			assert.Equalf(t, tt.want, got, "GetVersion(%v, %v)", tt.args.ctx, tt.args.f)
		})
	}
}
