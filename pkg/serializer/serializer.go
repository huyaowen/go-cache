package serializer

import (
	"bytes"
	"encoding/gob"
	"encoding/json"
	"fmt"
	"sync"

	"github.com/vmihailenco/msgpack/v5"
)

// Serializer 序列化器接口
type Serializer interface {
	Marshal(v interface{}) ([]byte, error)
	Unmarshal(data []byte, v interface{}) error
	Name() string
}

// 序列化器注册表
var (
	serializersMu sync.RWMutex
	serializers   = make(map[string]Serializer)
	defaultSerializer Serializer = &JSONSerializer{}
)

// Register 注册序列化器
func Register(name string, s Serializer) {
	if name == "" || s == nil {
		panic("serializer: name or serializer is nil")
	}
	serializersMu.Lock()
	defer serializersMu.Unlock()
	serializers[name] = s
}

// Get 获取序列化器
func Get(name string) (Serializer, error) {
	if name == "" {
		return defaultSerializer, nil
	}
	serializersMu.RLock()
	defer serializersMu.RUnlock()
	s, ok := serializers[name]
	if !ok {
		return nil, fmt.Errorf("serializer: unknown serializer type: %s", name)
	}
	return s, nil
}

// SetDefault 设置默认序列化器
func SetDefault(name string) error {
	s, err := Get(name)
	if err != nil {
		return err
	}
	serializersMu.Lock()
	defer serializersMu.Unlock()
	defaultSerializer = s
	return nil
}

// GetDefault 获取默认序列化器
func GetDefault() Serializer {
	serializersMu.RLock()
	defer serializersMu.RUnlock()
	return defaultSerializer
}

// ListSerializers 列出所有可用的序列化器
func ListSerializers() []string {
	serializersMu.RLock()
	defer serializersMu.RUnlock()
	names := make([]string, 0, len(serializers))
	for name := range serializers {
		names = append(names, name)
	}
	return names
}

// JSONSerializer JSON 序列化器
type JSONSerializer struct{}

func (s *JSONSerializer) Marshal(v interface{}) ([]byte, error) {
	return json.Marshal(v)
}

func (s *JSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

func (s *JSONSerializer) Name() string {
	return "json"
}

// GobSerializer Gob 序列化器
type GobSerializer struct{}

func (s *GobSerializer) Marshal(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	enc := gob.NewEncoder(&buf)
	err := enc.Encode(v)
	return buf.Bytes(), err
}

func (s *GobSerializer) Unmarshal(data []byte, v interface{}) error {
	buf := bytes.NewBuffer(data)
	dec := gob.NewDecoder(buf)
	return dec.Decode(v)
}

func (s *GobSerializer) Name() string {
	return "gob"
}

// MessagePackSerializer MessagePack 序列化器
type MessagePackSerializer struct{}

func (s *MessagePackSerializer) Marshal(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (s *MessagePackSerializer) Unmarshal(data []byte, v interface{}) error {
	return msgpack.Unmarshal(data, v)
}

func (s *MessagePackSerializer) Name() string {
	return "msgpack"
}

// 初始化时注册所有内置序列化器
func init() {
	Register("json", &JSONSerializer{})
	Register("gob", &GobSerializer{})
	Register("msgpack", &MessagePackSerializer{})
}
