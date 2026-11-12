package auth

import (
	"encoding/json"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
)

const generatedPasswordBytes = 18

func LoadOrCreate(path string) (hash PasswordHash, generated string, err error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		generated, hash, err = reset(path)
		return hash, generated, err
	}
	if err != nil {
		return PasswordHash{}, "", err
	}
	if err := json.Unmarshal(data, &hash); err != nil {
		return PasswordHash{}, "", err
	}
	if len(hash.Hash) == 0 || len(hash.Salt) == 0 || hash.Iterations <= 0 {
		return PasswordHash{}, "", errors.New("auth file is incomplete")
	}
	return hash, "", nil
}

func Reset(path string) (string, error) {
	password, _, err := reset(path)
	return password, err
}

func reset(path string) (string, PasswordHash, error) {
	password, err := randomToken(generatedPasswordBytes)
	if err != nil {
		return "", PasswordHash{}, err
	}
	hash, err := HashPassword(password)
	if err != nil {
		return "", PasswordHash{}, err
	}
	if err := writePrivate(path, hash); err != nil {
		return "", PasswordHash{}, err
	}
	return password, hash, nil
}

func writePrivate(path string, v any) error {
	data, err := json.Marshal(v)
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".auth-*")
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
	return os.Rename(tmp.Name(), path)
}
