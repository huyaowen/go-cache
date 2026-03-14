package main

import (
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	// Create performance test directory
	perfDir := "testdata/performance"
	if err := os.MkdirAll(perfDir, 0755); err != nil {
		panic(err)
	}

	// Generate 100 service files with 10 methods each
	for i := 1; i <= 100; i++ {
		filename := filepath.Join(perfDir, fmt.Sprintf("service_%03d.go", i))
		content := generateServiceFile(i)
		if err := os.WriteFile(filename, []byte(content), 0644); err != nil {
			panic(err)
		}
	}

	fmt.Printf("Generated 100 service files with 1000 total methods in %s\n", perfDir)
}

func generateServiceFile(index int) string {
	content := fmt.Sprintf(`package performance

// Service%03d represents service number %d
type Service%03d struct{}

`, index, index, index)

	// Add 10 methods per service
	for j := 1; j <= 10; j++ {
		cacheName := fmt.Sprintf("cache_%03d", index)
		method := fmt.Sprintf(`// @cacheable(cache="%s", key="#id", ttl="30m")
func (s *Service%03d) GetMethod%d(id int64) (string, error) {
	return "result", nil
}

`, cacheName, index, j)
		content += method
	}

	return content
}
