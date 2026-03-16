package scan

import (
	"crypto/md5"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const stateFileName = ".gocache_state.json"

// LoadState 加载上次扫描状态
func LoadState() *State {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return &State{Files: make(map[string]FileState)}
	}

	statePath := filepath.Join(homeDir, ".cache", "gocache", stateFileName)
	data, err := os.ReadFile(statePath)
	if err != nil {
		return &State{Files: make(map[string]FileState)}
	}

	var state State
	if err := json.Unmarshal(data, &state); err != nil {
		return &State{Files: make(map[string]FileState)}
	}

	if state.Files == nil {
		state.Files = make(map[string]FileState)
	}

	return &state
}

// SaveState 保存扫描状态
func SaveState(state *State) error {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	cacheDir := filepath.Join(homeDir, ".cache", "gocache")
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return err
	}

	statePath := filepath.Join(cacheDir, stateFileName)
	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(statePath, data, 0644)
}

// IsFileModified 检查文件是否被修改
func IsFileModified(path string, info os.FileInfo, state *State) bool {
	fileState, exists := state.Files[path]
	if !exists {
		return true
	}

	if info.ModTime().After(fileState.ModTime) {
		return true
	}

	currentHash := fileHash(path)
	if currentHash != fileState.Hash {
		return true
	}

	return false
}

// UpdateFileState 更新文件状态
func UpdateFileState(state *State, path string, info os.FileInfo) {
	hash := fileHash(path)
	state.Files[path] = FileState{
		ModTime: info.ModTime(),
		Hash:    hash,
	}
}

// fileHash 计算文件 Hash
func fileHash(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}

	hash := md5.Sum(data)
	return fmt.Sprintf("%x", hash)
}

// CleanOldState 清理过期状态
func CleanOldState(state *State, maxAge time.Duration) {
	cutoff := Now().Add(-maxAge)
	for path, fileState := range state.Files {
		if fileState.ModTime.Before(cutoff) {
			delete(state.Files, path)
		}
	}
}
