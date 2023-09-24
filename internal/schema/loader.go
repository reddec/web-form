package schema

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

func FormsFromStream(reader io.Reader) ([]Form, error) {
	dec := yaml.NewDecoder(reader)
	var forms []Form
	for {
		form := Default()
		err := dec.Decode(&form)
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, err
		}
		forms = append(forms, form)
	}

	return forms, nil
}

func FormsFromFile(fs fs.FS, file string) ([]Form, error) {
	f, err := fs.Open(file)
	if err != nil {
		return nil, fmt.Errorf("open: %w", err)
	}
	defer f.Close()

	forms, err := FormsFromStream(f)
	if err != nil {
		return nil, fmt.Errorf("read %q: %w", file, err)
	}

	name := file[:len(file)-len(filepath.Ext(file))]
	for i := range forms {
		if forms[i].Name == "" {
			forms[i].Name = name
		}
	}

	return forms, nil
}

func FormsFromFS(src fs.FS) ([]Form, error) {
	var forms []Form
	err := fs.WalkDir(src, ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		ext := strings.ToLower(filepath.Ext(path))
		if d.IsDir() || !(ext == ".yaml" || ext == ".yml" || ext == ".json") {
			return nil
		}

		list, err := FormsFromFile(src, path)
		if err != nil {
			return fmt.Errorf("read forms from FS: %w", err)
		}
		forms = append(forms, list...)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("read forms: %w", err)
	}
	return forms, nil
}
