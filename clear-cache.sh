#!/bin/bash

# 清理所有项目的 AI CR 缓存

echo "🧹 清理 AI Code Review 缓存..."

# 查找所有 .git/ai-cr-cache 目录
find . -type d -path "*/.git/ai-cr-cache" 2>/dev/null | while read -r cache_dir; do
    project_dir=$(dirname $(dirname "$cache_dir"))
    echo "  清理: $project_dir"
    rm -rf "$cache_dir"
done

echo "✅ 缓存清理完成"
echo ""
echo "💡 提示: 下次 commit 时会重新生成缓存"
