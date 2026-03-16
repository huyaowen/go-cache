package scan

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// DetectModulePath 检测 go.mod 获取模块路径
func DetectModulePath() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			content, err := os.ReadFile(goModPath)
			if err != nil {
				return "", err
			}

			lines := strings.Split(string(content), "\n")
			for _, line := range lines {
				line = strings.TrimSpace(line)
				if strings.HasPrefix(line, "module ") {
					return strings.TrimPrefix(line, "module "), nil
				}
			}

			return "", fmt.Errorf("module directive not found in go.mod")
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return "", fmt.Errorf("go.mod not found")
}

// GetImportPath 根据模块路径和文件路径获取导入路径
func GetImportPath(modulePath, filePath string) string {
	dir := filepath.Dir(filePath)

	goModDir := findGoModDir()
	if goModDir == "" {
		return filepath.Base(dir)
	}

	rel, err := filepath.Rel(goModDir, dir)
	if err != nil {
		return filepath.Base(dir)
	}

	rel = filepath.ToSlash(rel)
	if rel == "." {
		return modulePath
	}

	return modulePath + "/" + rel
}

// findGoModDir 查找 go.mod 所在目录
func findGoModDir() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}

	for {
		goModPath := filepath.Join(dir, "go.mod")
		if _, err := os.Stat(goModPath); err == nil {
			return dir
		}

		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}
