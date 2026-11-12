package core

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

type FileDecisions struct {
	Path string
}

func (f FileDecisions) Load() (Decisions, error) {
	data, err := os.ReadFile(f.Path)
	if errors.Is(err, fs.ErrNotExist) {
		return Decisions{}, nil
	}
	if err != nil {
		return nil, err
	}
	decisions := Decisions{}
	if err := json.Unmarshal(data, &decisions); err != nil {
		return nil, err
	}
	return decisions, nil
}

func (f FileDecisions) Save(decisions Decisions) error {
	data, err := json.MarshalIndent(decisions, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(f.Path), ".decisions-*")
	if err != nil {
		return err
	}
	defer os.Remove(tmp.Name())
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmp.Name(), f.Path)
}
