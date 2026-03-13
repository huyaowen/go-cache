package config

import (
	"os"
	"testing"
	"time"
)

func TestLoadFromString(t *testing.T) {
	yamlConfig := `
caches:
  users:
    backend: redis
    addr: localhost:6379
    default_ttl: 30m
    max_ttl: 24h
  products:
    backend: memory
    max_size: 10000
    default_ttl: 1h
`

	cfg, err := LoadFromString(yamlConfig)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Caches) != 2 {
		t.Errorf("Expected 2 caches, got %d", len(cfg.Caches))
	}

	// Test users cache
	users, ok := cfg.Caches["users"]
	if !ok {
		t.Fatal("users cache not found")
	}
	if users.Backend != "redis" {
		t.Errorf("Expected users backend to be redis, got %s", users.Backend)
	}
	if users.Addr != "localhost:6379" {
		t.Errorf("Expected users addr to be localhost:6379, got %s", users.Addr)
	}
	if users.DefaultTTL != 30*time.Minute {
		t.Errorf("Expected users default_ttl to be 30m, got %v", users.DefaultTTL)
	}
	if users.MaxTTL != 24*time.Hour {
		t.Errorf("Expected users max_ttl to be 24h, got %v", users.MaxTTL)
	}

	// Test products cache
	products, ok := cfg.Caches["products"]
	if !ok {
		t.Fatal("products cache not found")
	}
	if products.Backend != "memory" {
		t.Errorf("Expected products backend to be memory, got %s", products.Backend)
	}
	if products.MaxSize != 10000 {
		t.Errorf("Expected products max_size to be 10000, got %d", products.MaxSize)
	}
	if products.DefaultTTL != 1*time.Hour {
		t.Errorf("Expected products default_ttl to be 1h, got %v", products.DefaultTTL)
	}
}

func TestLoadFromStringDefaults(t *testing.T) {
	yamlConfig := `
caches:
  default:
`

	cfg, err := LoadFromString(yamlConfig)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	cache, ok := cfg.Caches["default"]
	if !ok {
		t.Fatal("default cache not found")
	}

	if cache.Backend != "memory" {
		t.Errorf("Expected default backend to be memory, got %s", cache.Backend)
	}
	if cache.DefaultTTL != 30*time.Minute {
		t.Errorf("Expected default default_ttl to be 30m, got %v", cache.DefaultTTL)
	}
	if cache.MaxTTL != 24*time.Hour {
		t.Errorf("Expected default max_ttl to be 24h, got %v", cache.MaxTTL)
	}
	if cache.MaxSize != 10000 {
		t.Errorf("Expected default max_size to be 10000, got %d", cache.MaxSize)
	}
}

func TestLoad(t *testing.T) {
	// Create temp file
	tmpfile, err := os.CreateTemp("", "cache-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpfile.Name())

	yamlConfig := `
caches:
  test:
    backend: memory
    max_size: 5000
`

	if _, err := tmpfile.Write([]byte(yamlConfig)); err != nil {
		t.Fatal(err)
	}
	if err := tmpfile.Close(); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(tmpfile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	if len(cfg.Caches) != 1 {
		t.Errorf("Expected 1 cache, got %d", len(cfg.Caches))
	}

	cache, ok := cfg.Caches["test"]
	if !ok {
		t.Fatal("test cache not found")
	}
	if cache.MaxSize != 5000 {
		t.Errorf("Expected max_size to be 5000, got %d", cache.MaxSize)
	}
}

func TestLoadNonExistent(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("Expected error for non-existent file")
	}
}
