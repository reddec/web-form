package storage

import (
	"context"
	"io"
)

type ClosableStorage interface {
	io.Closer
	Store(ctx context.Context, table string, fields map[string]any) (map[string]any, error)
}

func NopCloser(storage interface {
	Store(ctx context.Context, table string, fields map[string]any) (map[string]any, error)
}) ClosableStorage {
	return &fakeCloser{wrap: storage}
}

type fakeCloser struct {
	wrap interface {
		Store(ctx context.Context, table string, fields map[string]any) (map[string]any, error)
	}
}

func (fc fakeCloser) Close() error {
	return nil
}

func (fc fakeCloser) Store(ctx context.Context, table string, fields map[string]any) (map[string]any, error) {
	return fc.wrap.Store(ctx, table, fields)
}
