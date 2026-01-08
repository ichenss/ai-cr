#!/bin/bash

# AI Code Review - Git Hooks 卸载脚本

set -e

echo "🗑️  卸载 AI Code Review Git Hooks..."

# 检查是否在 Git 仓库中
if [ ! -d ".git" ]; then
    echo "❌ 错误: 当前目录不是 Git 仓库"
    exit 1
fi

HOOKS_DIR=".git/hooks"

# 删除 hooks
if [ -f "$HOOKS_DIR/pre-commit" ]; then
    rm "$HOOKS_DIR/pre-commit"
    echo "✅ 已删除 pre-commit"
fi

if [ -f "$HOOKS_DIR/commit-msg" ]; then
    rm "$HOOKS_DIR/commit-msg"
    echo "✅ 已删除 commit-msg"
fi

if [ -f "$HOOKS_DIR/pre-push" ]; then
    rm "$HOOKS_DIR/pre-push"
    echo "✅ 已删除 pre-push"
fi

echo ""
echo "✅ Git Hooks 卸载完成！"
