# Go-Cache Framework Makefile
# 提供便捷的构建、测试、清理命令

.PHONY: help build test clean generate lint fmt all

# 默认目标
all: build

# 帮助信息
help:
	@echo "Go-Cache Framework - Available Commands:"
	@echo ""
	@echo "  make build    - 构建项目 (执行 go generate + go build)"
	@echo "  make test     - 运行测试"
	@echo "  make clean    - 清理生成的文件"
	@echo "  make generate - 仅执行代码生成"
	@echo "  make lint     - 代码检查"
	@echo "  make fmt      - 格式化代码"
	@echo "  make all      - 完整构建 (generate + build + test)"
	@echo ""

# 构建项目
build: generate
	@echo "🏗️  Building..."
	go build ./...
	@echo "✅ Build complete"

# 运行测试
test: generate
	@echo "🧪 Running tests..."
	go test ./... -race -coverprofile=coverage.out
	@echo "✅ Tests complete"
	@echo ""
	@echo "📊 Coverage report:"
	go tool cover -func=coverage.out | tail -1
	@echo ""
	@echo "📄 HTML report: coverage.html"
	go tool cover -html=coverage.out -o coverage.html

# 清理生成的文件
clean:
	@echo "🧹 Cleaning..."
	find . -type d -name ".cache-gen" -exec rm -rf {} + 2>/dev/null || true
	find . -type f -name "coverage.out" -delete 2>/dev/null || true
	find . -type f -name "coverage.html" -delete 2>/dev/null || true
	go clean ./...
	@echo "✅ Clean complete"

# 仅执行代码生成
generate:
	@echo "📝 Generating code..."
	go generate ./...
	@echo "✅ Code generation complete"

# 代码检查
lint:
	@echo "🔍 Running linter..."
	golangci-lint run ./...
	@echo "✅ Lint complete"

# 格式化代码
fmt:
	@echo "📝 Formatting code..."
	go fmt ./...
	@echo "✅ Format complete"

# 完整构建 (generate + build + test)
all: generate build test
	@echo ""
	@echo "🎉 All tasks complete!"
