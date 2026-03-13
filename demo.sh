#!/bin/bash
# Go-Cache 方案 G 演示脚本

set -e

echo "========================================"
echo "  Go-Cache 方案 G 演示"
echo "  零配置，注解后直接使用"
echo "========================================"
echo ""

cd /home/admin/.openclaw/workspace/go-cache-framework/examples/cron-job

echo "📦 步骤 1: 清理旧的生成代码"
rm -rf service/.cache-gen
echo "✅ 清理完成"
echo ""

echo "🔧 步骤 2: 执行代码生成"
go generate ./...
echo ""

echo "📁 步骤 3: 查看生成的文件"
ls -la service/.cache-gen/
echo ""

echo "📄 步骤 4: 查看生成的缓存服务 (前 50 行)"
echo "---"
head -50 service/.cache-gen/product_cached.go
echo "---"
echo ""

echo "🏗️  步骤 5: 编译项目"
go build .
echo "✅ 编译成功"
echo ""

echo "🚀 步骤 6: 运行示例 (3 秒)"
echo "---"
timeout 3 ./cron-job --warmup=false 2>&1 | head -20 || true
echo "---"
echo ""

echo "✅ 演示完成!"
echo ""
echo "📊 关键特性:"
echo "  ✅ 零配置：cached.NewProductService()"
echo "  ✅ 自动缓存：@cacheable 注解自动生效"
echo "  ✅ 类型安全：基于接口的实现"
echo "  ✅ 全局管理：懒加载 Manager"
echo ""
echo "📚 更多信息：查看 QUICKSTART.md"
