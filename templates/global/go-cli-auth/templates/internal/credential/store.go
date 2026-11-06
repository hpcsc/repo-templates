package credential

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/zalando/go-keyring"
)

const (
	storeVariable = "GO_CLI_PROJECT_CREDENTIAL_STORE"
	keyringSource = "the OS credential store"
)

var ErrNotStored = errors.New("no stored {{ .Computed.label }}")

type Store struct {
	file     string
	fileOnly bool
}

func NewStore() (Store, error) {
	var fileOnly bool
	switch value := os.Getenv(storeVariable); value {
	case "", "keyring":
	case "file":
		fileOnly = true
	default:
		return Store{}, fmt.Errorf("%s is %q: set it to keyring or file", storeVariable, value)
	}
	dir, err := configDir()
	if err != nil {
		return Store{}, err
	}
	return Store{file: filepath.Join(dir, "credentials.json"), fileOnly: fileOnly}, nil
}

func (s Store) Get(host string) (Credential, error) {
	if !s.fileOnly {
		if value, err := keyring.Get(name, host); err == nil {
			return Credential{Value: value, Source: keyringSource}, nil
		}
	}
	values, err := s.read()
	if err != nil {
		return Credential{}, err
	}
	value, ok := values[host]
	if !ok {
		return Credential{}, fmt.Errorf("%w for %s", ErrNotStored, host)
	}
	return Credential{Value: value, Source: s.file}, nil
}

func (s Store) Set(host, value string) (string, error) {
	if !s.fileOnly && keyring.Set(name, host, value) == nil {
		_, err := s.removeFromFile(host)
		return keyringSource, err
	}
	values, err := s.read()
	if err != nil {
		return "", err
	}
	values[host] = value
	return s.file, s.write(values)
}

func (s Store) Delete(host string) (bool, error) {
	fromKeyring := !s.fileOnly && keyring.Delete(name, host) == nil
	fromFile, err := s.removeFromFile(host)
	return fromKeyring || fromFile, err
}

func (s Store) removeFromFile(host string) (bool, error) {
	values, err := s.read()
	if err != nil {
		return false, err
	}
	if _, ok := values[host]; !ok {
		return false, nil
	}
	delete(values, host)
	return true, s.write(values)
}

func (s Store) read() (map[string]string, error) {
	text, err := os.ReadFile(s.file)
	if errors.Is(err, fs.ErrNotExist) {
		return map[string]string{}, nil
	}
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	if err := json.Unmarshal(text, &values); err != nil {
		return nil, fmt.Errorf("%s: %w", s.file, err)
	}
	return values, nil
}

func (s Store) write(values map[string]string) error {
	if len(values) == 0 {
		if err := os.Remove(s.file); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return err
		}
		return nil
	}
	text, err := json.MarshalIndent(values, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(s.file)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	temp, err := os.CreateTemp(dir, ".credentials-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.Remove(temp.Name()) }()
	if _, err := temp.Write(text); err != nil {
		_ = temp.Close()
		return err
	}
	if err := temp.Close(); err != nil {
		return err
	}
	return os.Rename(temp.Name(), s.file)
}
