package credential

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

type config struct {
	CredentialCommand string `yaml:"credentialCommand"`
}

func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.yaml"), nil
}

func configDir() (string, error) {
	if dir := os.Getenv("XDG_CONFIG_HOME"); filepath.IsAbs(dir) {
		return filepath.Join(dir, name), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", name), nil
}

func credentialCommand(path string) (string, error) {
	text, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	var c config
	if err := yaml.Unmarshal(text, &c); err != nil {
		return "", fmt.Errorf("%s: %w", path, err)
	}
	return c.CredentialCommand, nil
}
