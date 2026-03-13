package serializer

import (
	"testing"
	"time"
)

type TestStruct struct {
	ID   int
	Name string
	Data []byte
	Time time.Time
}

func TestJSONSerializer(t *testing.T) {
	s := &JSONSerializer{}
	
	original := TestStruct{
		ID:   123,
		Name: "test",
		Data: []byte("hello"),
		Time: time.Now(),
	}
	
	data, err := s.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	var result TestStruct
	if err := s.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	
	if result.ID != original.ID {
		t.Errorf("ID mismatch: %d != %d", result.ID, original.ID)
	}
	if result.Name != original.Name {
		t.Errorf("Name mismatch: %s != %s", result.Name, original.Name)
	}
}

func TestGobSerializer(t *testing.T) {
	s := &GobSerializer{}
	
	original := TestStruct{
		ID:   456,
		Name: "gob-test",
		Data: []byte("world"),
		Time: time.Now(),
	}
	
	data, err := s.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	var result TestStruct
	if err := s.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	
	if result.ID != original.ID {
		t.Errorf("ID mismatch: %d != %d", result.ID, original.ID)
	}
	if result.Name != original.Name {
		t.Errorf("Name mismatch: %s != %s", result.Name, original.Name)
	}
}

func TestMessagePackSerializer(t *testing.T) {
	s := &MessagePackSerializer{}
	
	original := TestStruct{
		ID:   789,
		Name: "msgpack-test",
		Data: []byte("messagepack"),
		Time: time.Now(),
	}
	
	data, err := s.Marshal(original)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}
	
	var result TestStruct
	if err := s.Unmarshal(data, &result); err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}
	
	if result.ID != original.ID {
		t.Errorf("ID mismatch: %d != %d", result.ID, original.ID)
	}
	if result.Name != original.Name {
		t.Errorf("Name mismatch: %s != %s", result.Name, original.Name)
	}
}

func TestRegisterAndGet(t *testing.T) {
	// 测试获取已注册的序列化器
	s, err := Get("json")
	if err != nil {
		t.Fatalf("Get json failed: %v", err)
	}
	if s.Name() != "json" {
		t.Errorf("Expected json, got %s", s.Name())
	}
	
	s, err = Get("gob")
	if err != nil {
		t.Fatalf("Get gob failed: %v", err)
	}
	if s.Name() != "gob" {
		t.Errorf("Expected gob, got %s", s.Name())
	}
	
	s, err = Get("msgpack")
	if err != nil {
		t.Fatalf("Get msgpack failed: %v", err)
	}
	if s.Name() != "msgpack" {
		t.Errorf("Expected msgpack, got %s", s.Name())
	}
	
	// 测试获取不存在的序列化器
	_, err = Get("nonexistent")
	if err == nil {
		t.Error("Expected error for nonexistent serializer")
	}
}

func TestDefaultSerializer(t *testing.T) {
	// 默认应该是 JSON
	s := GetDefault()
	if s.Name() != "json" {
		t.Errorf("Expected default to be json, got %s", s.Name())
	}
	
	// 更改默认
	err := SetDefault("gob")
	if err != nil {
		t.Fatalf("SetDefault failed: %v", err)
	}
	
	s = GetDefault()
	if s.Name() != "gob" {
		t.Errorf("Expected default to be gob, got %s", s.Name())
	}
	
	// 恢复默认
	SetDefault("json")
}

func TestListSerializers(t *testing.T) {
	names := ListSerializers()
	if len(names) < 3 {
		t.Errorf("Expected at least 3 serializers, got %d", len(names))
	}
	
	expected := map[string]bool{
		"json":    false,
		"gob":     false,
		"msgpack": false,
	}
	
	for _, name := range names {
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}
	
	for name, found := range expected {
		if !found {
			t.Errorf("Expected serializer %s not found", name)
		}
	}
}
