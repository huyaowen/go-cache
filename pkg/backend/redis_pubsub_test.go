package backend

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestCacheInvalidator_Broadcast(t *testing.T) {
	t.Skip("Skipping test that requires Redis")

	config := DefaultCacheInvalidatorConfig()
	config.Addr = "localhost:6379"
	config.Channel = "test:invalidation:" + time.Now().Format("150405")

	invalidator, err := NewCacheInvalidator(config)
	if err != nil {
		t.Fatalf("Failed to create invalidator: %v", err)
	}
	defer invalidator.Close()

	err = invalidator.Broadcast("test-key-123")
	if err != nil {
		t.Errorf("Broadcast failed: %v", err)
	}
}

func TestCacheInvalidator_BroadcastWithCache(t *testing.T) {
	t.Skip("Skipping test that requires Redis")

	config := DefaultCacheInvalidatorConfig()
	config.Addr = "localhost:6379"
	config.Channel = "test:invalidation:" + time.Now().Format("150405")

	invalidator, err := NewCacheInvalidator(config)
	if err != nil {
		t.Fatalf("Failed to create invalidator: %v", err)
	}
	defer invalidator.Close()

	err = invalidator.BroadcastWithCache("users", "user-123")
	if err != nil {
		t.Errorf("BroadcastWithCache failed: %v", err)
	}
}

func TestCacheInvalidator_Subscribe(t *testing.T) {
	t.Skip("Skipping test that requires Redis")

	config := DefaultCacheInvalidatorConfig()
	config.Addr = "localhost:6379"
	config.Channel = "test:invalidation:" + time.Now().Format("150405")

	invalidator, err := NewCacheInvalidator(config)
	if err != nil {
		t.Fatalf("Failed to create invalidator: %v", err)
	}
	defer invalidator.Close()

	var wg sync.WaitGroup
	received := make([]string, 0)
	mu := sync.Mutex{}

	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		invalidator.SubscribeWithContext(ctx, func(key string) {
			mu.Lock()
			defer mu.Unlock()
			received = append(received, key)
		})
	}()

	time.Sleep(100 * time.Millisecond)

	err = invalidator.Broadcast("test-key-1")
	if err != nil {
		t.Errorf("Broadcast failed: %v", err)
	}

	time.Sleep(100 * time.Millisecond)

	err = invalidator.Broadcast("test-key-2")
	if err != nil {
		t.Errorf("Broadcast failed: %v", err)
	}

	time.Sleep(500 * time.Millisecond)

	mu.Lock()
	if len(received) != 2 {
		t.Errorf("Expected 2 messages, got %d: %v", len(received), received)
	}
	mu.Unlock()

	invalidator.Close()
	wg.Wait()
}

func TestInvalidationMessage_Marshal(t *testing.T) {
	msg := &InvalidationMessage{
		CacheName: "users",
		Key:       "user-123",
		Timestamp: 1234567890,
	}

	data, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := "users:user-123"
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestInvalidationMessage_MarshalSimple(t *testing.T) {
	msg := &InvalidationMessage{
		Key:       "simple-key",
		Timestamp: 1234567890,
	}

	data, err := msg.Marshal()
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	expected := "simple-key"
	if string(data) != expected {
		t.Errorf("Expected %s, got %s", expected, string(data))
	}
}

func TestUnmarshalInvalidationMessage(t *testing.T) {
	tests := []struct {
		name      string
		data      string
		wantKey   string
		wantCache string
	}{
		{
			name:    "simple key",
			data:    "user-123",
			wantKey: "user-123",
		},
		{
			name:      "cache and key",
			data:      "users:user-123",
			wantKey:   "user-123",
			wantCache: "users",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg, err := UnmarshalInvalidationMessage([]byte(tt.data))
			if err != nil {
				t.Fatalf("Unmarshal failed: %v", err)
			}

			if msg.Key != tt.wantKey {
				t.Errorf("Key mismatch: got %s, want %s", msg.Key, tt.wantKey)
			}

			if tt.wantCache != "" && msg.CacheName != tt.wantCache {
				t.Errorf("CacheName mismatch: got %s, want %s", msg.CacheName, tt.wantCache)
			}
		})
	}
}

func TestCacheInvalidator_Close(t *testing.T) {
	t.Skip("Skipping test that requires Redis")

	config := DefaultCacheInvalidatorConfig()
	config.Addr = "localhost:6379"

	invalidator, err := NewCacheInvalidator(config)
	if err != nil {
		t.Fatalf("Failed to create invalidator: %v", err)
	}

	err = invalidator.Close()
	if err != nil {
		t.Errorf("First close failed: %v", err)
	}

	err = invalidator.Close()
	if err != nil {
		t.Errorf("Second close should be no-op: %v", err)
	}

	if !invalidator.IsClosed() {
		t.Error("Invalidator should be closed")
	}
}

func TestCacheInvalidator_BroadcastAfterClose(t *testing.T) {
	t.Skip("Skipping test that requires Redis")

	config := DefaultCacheInvalidatorConfig()
	config.Addr = "localhost:6379"

	invalidator, err := NewCacheInvalidator(config)
	if err != nil {
		t.Fatalf("Failed to create invalidator: %v", err)
	}
	invalidator.Close()

	err = invalidator.Broadcast("test-key")
	if err == nil {
		t.Error("Expected error when broadcasting after close")
	}
}
