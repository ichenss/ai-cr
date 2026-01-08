# AI Code Review

基于 DeepSeek 的智能代码审查工具，支持 Git Hook 集成。

## 快速开始

### 1. 克隆项目

```bash
git clone <your-repo-url>
cd <your-repo>
```

### 2. 配置 API Key

```bash
# 进入 ai-cr 目录
cd ai-cr

# 复制环境变量模板
cp .env.example .env

# 编辑 .env 文件，填入你的 DeepSeek API Key
vim .env
```

`.env` 文件内容：
```bash
DEEPSEEK_API_KEY=sk-your-actual-api-key-here
```

**获取 API Key：**
1. 访问 [DeepSeek Platform](https://platform.deepseek.com/api_keys)
2. 注册/登录账号
3. 创建 API Key
4. 复制到 `.env` 文件

### 3. 启动 AI CR 服务

```bash
# 安装依赖
go mod tidy

# 方式1：使用 .env 文件（推荐）
# 安装 godotenv
go get github.com/joho/godotenv

# 启动服务
go run main.go server

# 方式2：直接设置环境变量
export DEEPSEEK_API_KEY=sk-your-api-key
go run main.go server
```

你会看到：
```
🚀 AI Code Review 服务启动 :8083
📌 POST /api/review {"request": "请审查 main.go"}
```

**保持这个终端运行！**

### 3. 安装 Git Hooks（在你的项目中）

打开新终端，进入你要审查的项目：

```bash
# 例如：进入 safe-user-center 项目
cd safe-user-center

# 安装 Git Hooks
bash ../ai-cr/hooks/install.sh
```

安装成功后会显示：
```
✅ Git Hooks 安装完成！

已安装的 hooks:
  - pre-commit:  提交前审查代码（不阻拦）
  - commit-msg:  在提交信息中添加审查摘要
  - pre-push:    推送前严格审查（会阻拦）
```

### 4. 开始使用

现在你的 Git 操作会自动触发 AI 审查：

```bash
# 修改代码
vim controller/login.go

# 提交（不会阻拦，后台审查）
git add .
git commit -m "fix: 修复登录问题"

# 推送（严格审查，发现严重问题会阻拦）
git push origin main
```

## 系统要求

### 必需
- Go 1.21+
- Git
- jq（JSON 解析工具）
- DeepSeek API Key（[获取地址](https://platform.deepseek.com/api_keys)）

### 安装 jq

```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# CentOS/RHEL
sudo yum install jq
```

## 配置

### API Key 配置

**方式一：环境变量文件（推荐）**

```bash
cd ai-cr
cp .env.example .env
vim .env  # 填入你的 API Key
```

**方式二：直接设置环境变量**

```bash
# 临时设置（当前终端有效）
export DEEPSEEK_API_KEY=sk-your-api-key

# 永久设置（添加到 ~/.zshrc 或 ~/.bashrc）
echo 'export DEEPSEEK_API_KEY=sk-your-api-key' >> ~/.zshrc
source ~/.zshrc
```

**方式三：启动时指定**

```bash
DEEPSEEK_API_KEY=sk-your-api-key go run main.go server
```

## 使用方式

### 方式一：Git Hook（推荐）

安装后自动工作，无需额外操作。

**commit 时：**
- ✅ 不阻拦提交
- ✅ 后台审查代码
- ✅ 发现严重问题会警告
- ✅ 缓存审查结果

**push 时：**
- ❌ 发现严重问题会阻止推送
- ✅ 使用缓存避免重复审查
- ✅ 需要输入 `FORCE_PUSH` 才能强制推送

### 方式二：CLI 命令

```bash
cd ai-cr

# 审查单个文件
go run main.go review ../safe-user-center/controller/login.go

# 审查 git diff
cd ../safe-user-center
git diff > /tmp/changes.diff
cd ../ai-cr
go run main.go diff

# 启动 HTTP 服务
go run main.go server
```

### 方式三：HTTP API

```bash
# 审查文件
curl -X POST http://localhost:8083/api/review \
  -H "Content-Type: application/json" \
  -d '{
    "request": "请审查 safe-user-center/controller/login.go"
  }'

# 审查 git diff
curl -X POST http://localhost:8083/api/review \
  -H "Content-Type: application/json" \
  -d '{
    "request": "请审查当前的 git diff"
  }'
```

## 配置

### 修改 API Key

不要直接修改代码！使用环境变量：

```bash
# 编辑 .env 文件
vim ai-cr/.env

# 或设置环境变量
export DEEPSEEK_API_KEY=your-new-api-key
```

### 自定义审查规则

编辑 Git Hook 文件：

```bash
# 编辑 pre-push hook
vim .git/hooks/pre-push

# 修改审查请求
-d "{\"request\": \"请严格审查以下文件，重点关注：\n1. 你的自定义规则...\"}"
```

### 跳过审查

```bash
# 跳过 pre-commit
git commit --no-verify -m "message"

# 跳过 pre-push
git push --no-verify

# 或在 push 时输入 FORCE_PUSH
```

## 团队使用

### 方案一：每个人本地运行

**优点：** 简单，无需额外部署  
**缺点：** 每个人都要启动服务

1. 每个开发者克隆项目
2. 启动 AI CR 服务：`cd ai-cr && go run main.go server`
3. 在自己的项目中安装 hooks

### 方案二：共享服务器

**优点：** 统一管理，无需本地启动  
**缺点：** 需要部署服务器

1. 在团队服务器上部署 AI CR 服务
2. 修改 hooks 中的 URL：
   ```bash
   # 编辑 hooks/pre-commit 和 hooks/pre-push
   curl -s http://your-server:8083/health
   ```
3. 团队成员只需安装 hooks

### 方案三：Docker 部署

```bash
# 构建镜像
cd ai-cr
docker build -t ai-cr:latest .

# 运行容器
docker run -d -p 8083:8083 ai-cr:latest

# 团队成员修改 hooks URL 为服务器地址
```

## 常见问题

### 1. Hook 不执行

```bash
# 检查权限
ls -la .git/hooks/
# 应该显示 -rwxr-xr-x

# 添加执行权限
chmod +x .git/hooks/pre-commit
chmod +x .git/hooks/pre-push
```

### 2. 服务连接失败

```bash
# 检查服务是否运行
curl http://localhost:8083/health

# 如果失败，启动服务
cd ai-cr && go run main.go server
```

### 3. jq 命令未找到

```bash
# macOS
brew install jq

# Linux
sudo apt-get install jq
```

### 4. API Key 未设置

```bash
# 错误信息
❌ 错误: 未设置 DEEPSEEK_API_KEY 环境变量
请设置: export DEEPSEEK_API_KEY=your-api-key

# 解决方案
export DEEPSEEK_API_KEY=sk-your-api-key
go run main.go server
```

### 5. 审查太慢

- 检查网络连接（需要访问 DeepSeek API）
- 考虑使用缓存（commit 时已审查，push 时使用缓存）
- 减少审查的文件数量

### 5. 想临时禁用审查

```bash
# 方法1：跳过 hook
git commit --no-verify
git push --no-verify

# 方法2：卸载 hooks
cd your-project
bash ../ai-cr/hooks/uninstall.sh

# 方法3：停止服务
# 停止 ai-cr 服务，hooks 会自动跳过
```

## 卸载

```bash
cd your-project
bash ../ai-cr/hooks/uninstall.sh
```

## 支持的语言

- Go (.go)
- JavaScript/TypeScript (.js, .ts, .jsx, .tsx)
- Python (.py)
- Java (.java)
- C/C++ (.c, .cpp, .h)
- Rust (.rs)
- PHP (.php)
- Ruby (.rb)
- Swift (.swift)
- Kotlin (.kt)

## 示例输出

### commit 时（不阻拦）

```
🤖 AI Code Review 检查中...
📝 发现 1 个代码文件变更
controller/login.go

🔄 后台审查中...
✅ 提交继续（AI 审查在后台进行）
💡 提示: push 时会进行严格审查

[main abc1234] fix: 修复登录问题
 1 file changed, 10 insertions(+), 5 deletions(-)

⚠️⚠️⚠️  发现严重问题！ ⚠️⚠️⚠️
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
## 代码审查报告
1. **安全问题**: 错误信息直接返回给用户
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
💡 提示: 代码已提交，但 push 时会再次审查
```

### push 时（严格审查）

```
🚀 AI Code Review - 推送前严格检查...
📝 发现未推送的提交:
abc1234 fix: 修复登录问题

💾 使用缓存的审查结果（已在 commit 时审查过）

📋 AI 审查结果:
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
## 代码审查报告

### 发现的问题

1. **安全问题** (严重)
   - 第177行：错误信息直接返回给用户，可能泄露敏感信息
   - 建议：使用统一的错误码

2. **错误处理**
   - 第234行：err 未检查就继续执行
   - 建议：添加 if err != nil 检查
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

❌ 发现严重问题，不允许推送！

请修复以上问题后再推送。

如果确认要强制推送，请输入: FORCE_PUSH
> _
```

## 贡献

欢迎提交 Issue 和 Pull Request！

## License

MIT
