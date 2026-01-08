#!/bin/bash

# AI Code Review - Git Hooks 安装脚本

set -e

echo "🔧 安装 AI Code Review Git Hooks..."

# 检查是否在 Git 仓库中
if [ ! -d ".git" ]; then
    echo "❌ 错误: 当前目录不是 Git 仓库"
    echo "请在 Git 仓库根目录运行此脚本"
    exit 1
fi

# 获取脚本所在目录
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
HOOKS_DIR=".git/hooks"

# 创建 hooks 目录（如果不存在）
mkdir -p "$HOOKS_DIR"

# 安装 hooks
echo "📦 安装 pre-commit hook..."
cp "$SCRIPT_DIR/pre-commit" "$HOOKS_DIR/pre-commit"
chmod +x "$HOOKS_DIR/pre-commit"

echo "📦 安装 commit-msg hook..."
cp "$SCRIPT_DIR/commit-msg" "$HOOKS_DIR/commit-msg"
chmod +x "$HOOKS_DIR/commit-msg"

echo "📦 安装 pre-push hook..."
cp "$SCRIPT_DIR/pre-push" "$HOOKS_DIR/pre-push"
chmod +x "$HOOKS_DIR/pre-push"

echo ""
echo "✅ Git Hooks 安装完成！"
echo ""
echo "已安装的 hooks:"
echo "  - pre-commit:  提交前审查代码"
echo "  - commit-msg:  在提交信息中添加审查摘要"
echo "  - pre-push:    推送前审查所有变更"
echo ""
echo "⚠️  使用前请确保 AI CR 服务已启动:"
echo "  cd ai-cr && go run main.go server"
echo ""
echo "如需卸载，运行: ./hooks/uninstall.sh"
