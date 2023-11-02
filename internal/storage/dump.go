package storage

import (
	"context"
	"encoding/json"
	"maps"
	"os"
)

// Dump store just prints in JSON content of request.
type Dump struct{}

func (sd *Dump) Store(_ context.Context, table string, fields map[string]any) (map[string]any, error) {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return maps.Clone(fields), enc.Encode(dumpItem{
		Table:  table,
		Fields: fields,
	})
}

type dumpItem struct {
	Table  string
	Fields map[string]any
}
