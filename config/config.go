package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the application configuration.
type Config struct {
	DefaultDomain string `json:"default_domain"`
	Credentials   []byte
	Token         []byte
}

// Loader defines methods to load configuration, credentials, and token.
type Loader interface {
	LoadConfig() (*Config, error)
	LoadCredentials() ([]byte, error)
	LoadToken() ([]byte, error)
	SaveToken(token []byte) error
}

// FileLoader implements Loader by reading from the filesystem.
type FileLoader struct {
	configDir string
}

// NewFileLoader initializes a FileLoader with the config directory path.
func NewFileLoader() (*FileLoader, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("unable to find user home directory: %w", err)
	}
	configDir := filepath.Join(homeDir, ".calvin")
	return &FileLoader{configDir: configDir}, nil
}

// LoadConfig reads the config.json file.
func (f *FileLoader) LoadConfig() (*Config, error) {
	configPath := filepath.Join(f.configDir, "config.json")
	b, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile(%s): %w", configPath, err)
	}

	var config Config
	if err := json.Unmarshal(b, &config); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	return &config, nil
}

// LoadCredentials reads the credentials.json file.
func (f *FileLoader) LoadCredentials() ([]byte, error) {
	credentialsPath := filepath.Join(f.configDir, "credentials.json")
	bytes, err := os.ReadFile(credentialsPath)
	if err != nil {
		return nil, fmt.Errorf("os.ReadFile(%s): %w", credentialsPath, err)
	}
	return bytes, nil
}

// LoadToken reads the token.json file.
func (f *FileLoader) LoadToken() ([]byte, error) {
	tokenPath := filepath.Join(f.configDir, "token.json")
	bytes, err := os.ReadFile(tokenPath)
	if err != nil {
		return nil, err
	}
	return bytes, nil
}

// SaveToken writes the token.json file.
func (f *FileLoader) SaveToken(token []byte) error {
	tokenPath := filepath.Join(f.configDir, "token.json")
	if err := os.MkdirAll(f.configDir, 0o700); err != nil {
		return fmt.Errorf("unable to create config directory: %w", err)
	}
	if err := os.WriteFile(tokenPath, token, 0o600); err != nil {
		return fmt.Errorf("unable to save token: %w", err)
	}
	return nil
}
