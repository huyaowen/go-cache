#!/bin/bash
# Go-Cache Framework 构建脚本
# 自动执行代码生成和编译

set -e

echo "🔧 Building Go-Cache Framework..."
echo ""

# 获取脚本所在目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# 步骤 1: 执行代码生成
echo "📝 Step 1: Running go generate..."
go generate ./...
echo "✅ Code generation complete"
echo ""

# 步骤 2: 编译项目
echo "🏗️  Step 2: Building..."
go build ./...
echo "✅ Build complete"
echo ""

# 步骤 3: 运行测试 (可选)
if [ "$1" == "--test" ] || [ "$1" == "-t" ]; then
    echo "🧪 Step 3: Running tests..."
    go test ./... -v
    echo "✅ Tests complete"
    echo ""
fi

echo "🎉 Build successful!"
