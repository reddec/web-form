package storage

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/oklog/ulid/v2"
)

func NewFileStore(rootDir string) *FileStore {
	return &FileStore{directory: rootDir}
}

// FileStore stores each submission as single file in JSON with ULID + .json as name under directory, equal to table name.
// It DOES NOT escape table name AT ALL.
// Result set contains all source fields plus ID (string), equal to filename without extension.
type FileStore struct {
	directory string
}

func (fs *FileStore) Store(_ context.Context, table string, fields map[string]any) (map[string]any, error) {
	id, err := ulid.New(ulid.Now(), rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("generate ULID: %w", err)
	}
	dir := filepath.Join(fs.directory, table)
	if err := os.MkdirAll(dir, 0750); err != nil {
		return nil, fmt.Errorf("create base dir %q: %w", dir, err)
	}
	sid := id.String()
	p := filepath.Join(dir, sid+".json")

	var data = make(map[string]any, len(fields)+1)
	for k, v := range fields {
		data[k] = v
	}
	data["ID"] = sid

	document, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("serialize document: %w", err)
	}

	return data, atomicWrite(p, document)
}

func atomicWrite(file string, content []byte) error {
	d := filepath.Dir(file)
	n := filepath.Base(file)
	f, err := os.CreateTemp(d, n+".tmp.*")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	defer os.Remove(f.Name())
	defer f.Close()

	if _, err := f.Write(content); err != nil {
		return fmt.Errorf("write temp file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	return os.Rename(f.Name(), file)
}
