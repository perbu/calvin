package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Setup: Create a temporary config directory and config file.
	tempDir := t.TempDir()
	configContent := `{"default_domain": "example.com"}`
	configPath := filepath.Join(tempDir, "config.json")
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	loader := &FileLoader{configDir: tempDir}
	config, err := loader.LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	if config.DefaultDomain != "example.com" {
		t.Errorf("Expected DefaultDomain to be 'example.com', got '%s'", config.DefaultDomain)
	}
}
