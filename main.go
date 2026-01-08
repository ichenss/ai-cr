package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

/* ===================== 配置 ===================== */

const (
	deepseekURL   = "https://api.deepseek.com/v1/chat/completions"
	deepseekModel = "deepseek-chat"
)

// 从环境变量读取 API Key
func getAPIKey() string {
	apiKey := os.Getenv("DEEPSEEK_API_KEY")
	if apiKey == "" {
		log.Fatal("❌ 错误: 未设置 DEEPSEEK_API_KEY 环境变量\n" +
			"请设置: export DEEPSEEK_API_KEY=your-api-key")
	}
	return apiKey
}

/* ===================== 基础类型 ===================== */

type Message struct {
	Role       string     `json:"role"`
	Content    string     `json:"content,omitempty"`
	ToolCalls  []ToolCall `json:"tool_calls,omitempty"`
	ToolCallID string     `json:"tool_call_id,omitempty"`
}

type ToolCall struct {
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Tools    []Tool    `json:"tools,omitempty"`
}

type ChatResponse struct {
	Choices []struct {
		Message      Message `json:"message"`
		FinishReason string  `json:"finish_reason"`
	} `json:"choices"`
}

/* ===================== 工具定义 ===================== */

var tools = []Tool{
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "get_working_directory",
			Description: "获取当前工作目录，用于确定文件路径",
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": map[string]interface{}{},
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "read_file",
			Description: "读取指定文件的内容，支持相对路径和绝对路径",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "文件路径（相对或绝对）",
					},
				},
				"required": []string{"file_path"},
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "read_multiple_files",
			Description: "批量读取多个文件的内容",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_paths": map[string]interface{}{
						"type":        "array",
						"items":       map[string]interface{}{"type": "string"},
						"description": "文件路径列表",
					},
				},
				"required": []string{"file_paths"},
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "list_files",
			Description: "列出目录下的文件",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"directory": map[string]interface{}{
						"type":        "string",
						"description": "目录路径，默认为当前目录",
					},
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "文件匹配模式，如 *.go",
					},
					"recursive": map[string]interface{}{
						"type":        "boolean",
						"description": "是否递归查找子目录",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "search_in_files",
			Description: "在文件中搜索关键字或正则表达式",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"directory": map[string]interface{}{
						"type":        "string",
						"description": "搜索目录",
					},
					"pattern": map[string]interface{}{
						"type":        "string",
						"description": "搜索模式（关键字或正则）",
					},
					"file_extension": map[string]interface{}{
						"type":        "string",
						"description": "文件扩展名，如 .go",
					},
				},
				"required": []string{"directory", "pattern"},
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "get_git_diff",
			Description: "获取 Git 仓库的代码变更",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"target": map[string]interface{}{
						"type":        "string",
						"description": "对比目标，如 HEAD、main、commit hash",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "run_linter",
			Description: "运行代码检查工具（如 golangci-lint、eslint）",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"file_path": map[string]interface{}{
						"type":        "string",
						"description": "要检查的文件路径",
					},
				},
			},
		},
	},
	{
		Type: "function",
		Function: ToolFunction{
			Name:        "analyze_directory",
			Description: "分析目录结构，列出所有代码文件并提供概览",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"directory": map[string]interface{}{
						"type":        "string",
						"description": "要分析的目录路径",
					},
				},
				"required": []string{"directory"},
			},
		},
	},
}

/* ===================== DeepSeek API ===================== */

func callDeepSeek(ctx context.Context, messages []Message, useTools bool) (*ChatResponse, error) {
	req := ChatRequest{
		Model:    deepseekModel,
		Messages: messages,
	}
	if useTools {
		req.Tools = tools
	}

	body, _ := json.Marshal(req)

	httpReq, _ := http.NewRequestWithContext(
		ctx, http.MethodPost, deepseekURL, strings.NewReader(string(body)),
	)
	httpReq.Header.Set("Authorization", "Bearer "+getAPIKey())
	httpReq.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 300 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	var cr ChatResponse
	if err := json.NewDecoder(resp.Body).Decode(&cr); err != nil {
		return nil, err
	}
	return &cr, nil
}

/* ===================== 工具执行 ===================== */

func getStringArg(args map[string]interface{}, key string, defaultVal string) string {
	if args == nil {
		return defaultVal
	}
	val, ok := args[key]
	if !ok {
		return defaultVal
	}
	str, ok := val.(string)
	if !ok {
		return defaultVal
	}
	return str
}

func executeTool(name string, args map[string]interface{}) (string, error) {
	switch name {
	case "get_working_directory":
		wd := getWorkingDirectory()
		return fmt.Sprintf("当前工作目录: %s", wd), nil

	case "read_file":
		filePath := getStringArg(args, "file_path", "")
		if filePath == "" {
			return "", fmt.Errorf("file_path is required")
		}
		return readFile(filePath)

	case "read_multiple_files":
		filePaths, ok := args["file_paths"].([]interface{})
		if !ok {
			return "", fmt.Errorf("file_paths must be an array")
		}
		return readMultipleFiles(filePaths)

	case "list_files":
		directory := getStringArg(args, "directory", ".")
		pattern := getStringArg(args, "pattern", "*")
		recursive := false
		if r, ok := args["recursive"].(bool); ok {
			recursive = r
		}
		return listFiles(directory, pattern, recursive)

	case "search_in_files":
		directory := getStringArg(args, "directory", ".")
		pattern := getStringArg(args, "pattern", "")
		fileExt := getStringArg(args, "file_extension", "")
		return searchInFiles(directory, pattern, fileExt)

	case "get_git_diff":
		target := getStringArg(args, "target", "HEAD")
		return getGitDiff(target)

	case "run_linter":
		filePath := getStringArg(args, "file_path", "")
		return runLinter(filePath)

	case "analyze_directory":
		directory := getStringArg(args, "directory", ".")
		return analyzeDirectory(directory)

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}

/* ===================== 工具实现 ===================== */

func getWorkingDirectory() string {
	wd, _ := os.Getwd()
	return wd
}

func readFile(filePath string) (string, error) {
	// 尝试多个可能的路径
	possiblePaths := []string{
		filePath,
		filepath.Join("..", filePath),    // 上一级目录
		filepath.Join("../..", filePath), // 上两级目录
	}

	var lastErr error
	for _, path := range possiblePaths {
		data, err := os.ReadFile(path)
		if err == nil {
			// 成功读取
			content := string(data)
			if len(content) > 10000 {
				content = content[:10000] + "\n... (文件过长，已截断)"
			}
			return fmt.Sprintf("=== %s ===\n%s", filePath, content), nil
		}
		lastErr = err
	}

	// 所有路径都失败，返回详细错误
	absPath, _ := filepath.Abs(filePath)
	return "", fmt.Errorf("读取文件失败: %s\n尝试的路径: %v\n绝对路径: %s\n错误: %v",
		filePath, possiblePaths, absPath, lastErr)
}

func readMultipleFiles(filePaths []interface{}) (string, error) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("读取 %d 个文件：\n\n", len(filePaths)))

	for i, fp := range filePaths {
		if i >= 10 {
			result.WriteString("\n... (超过10个文件，已截断)")
			break
		}

		filePath, ok := fp.(string)
		if !ok {
			continue
		}

		content, err := readFile(filePath)
		if err != nil {
			result.WriteString(fmt.Sprintf("\n❌ %s: %v\n", filePath, err))
			continue
		}

		result.WriteString(content)
		result.WriteString("\n\n")
	}

	return result.String(), nil
}

func listFiles(directory, pattern string, recursive bool) (string, error) {
	var matches []string
	var err error

	if recursive {
		// 递归查找
		err = filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}
			if !info.IsDir() {
				matched, _ := filepath.Match(pattern, filepath.Base(path))
				if matched {
					matches = append(matches, path)
				}
			}
			return nil
		})
	} else {
		// 非递归
		matches, err = filepath.Glob(filepath.Join(directory, pattern))
	}

	if err != nil {
		return "", fmt.Errorf("列出文件失败: %w", err)
	}

	if len(matches) == 0 {
		return "未找到匹配的文件", nil
	}

	var result strings.Builder
	result.WriteString(fmt.Sprintf("找到 %d 个文件：\n", len(matches)))
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		if !info.IsDir() {
			result.WriteString(fmt.Sprintf("- %s (%d bytes)\n", match, info.Size()))
		}
	}

	return result.String(), nil
}

func searchInFiles(directory, pattern, fileExt string) (string, error) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("在 %s 中搜索 '%s'：\n\n", directory, pattern))

	matchCount := 0
	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// 跳过目录和不匹配的文件
		if info.IsDir() {
			return nil
		}
		if fileExt != "" && filepath.Ext(path) != fileExt {
			return nil
		}

		// 读取文件内容
		data, err := os.ReadFile(path)
		if err != nil {
			return nil
		}

		content := string(data)
		if strings.Contains(content, pattern) {
			matchCount++
			result.WriteString(fmt.Sprintf("📄 %s\n", path))

			// 显示匹配的行
			lines := strings.Split(content, "\n")
			for i, line := range lines {
				if strings.Contains(line, pattern) {
					result.WriteString(fmt.Sprintf("  L%d: %s\n", i+1, line))
				}
			}
			result.WriteString("\n")
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("搜索失败: %w", err)
	}

	if matchCount == 0 {
		return "未找到匹配的内容", nil
	}

	return result.String(), nil
}

func analyzeDirectory(directory string) (string, error) {
	var result strings.Builder
	result.WriteString(fmt.Sprintf("📁 分析目录: %s\n\n", directory))

	// 统计信息
	fileCount := 0
	totalSize := int64(0)
	filesByExt := make(map[string]int)
	var codeFiles []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			fileCount++
			totalSize += info.Size()

			ext := filepath.Ext(path)
			filesByExt[ext]++

			// 收集代码文件
			if isCodeFile(ext) {
				codeFiles = append(codeFiles, path)
			}
		}

		return nil
	})

	if err != nil {
		return "", fmt.Errorf("分析目录失败: %w", err)
	}

	// 输出统计
	result.WriteString(fmt.Sprintf("📊 统计信息：\n"))
	result.WriteString(fmt.Sprintf("- 文件总数: %d\n", fileCount))
	result.WriteString(fmt.Sprintf("- 总大小: %d bytes\n", totalSize))
	result.WriteString(fmt.Sprintf("\n📝 文件类型分布：\n"))
	for ext, count := range filesByExt {
		if ext == "" {
			ext = "(无扩展名)"
		}
		result.WriteString(fmt.Sprintf("- %s: %d 个\n", ext, count))
	}

	// 列出代码文件
	result.WriteString(fmt.Sprintf("\n💻 代码文件列表 (%d 个)：\n", len(codeFiles)))
	for i, file := range codeFiles {
		if i >= 50 {
			result.WriteString("... (超过50个，已截断)\n")
			break
		}
		result.WriteString(fmt.Sprintf("- %s\n", file))
	}

	return result.String(), nil
}

func isCodeFile(ext string) bool {
	codeExts := map[string]bool{
		".go":    true,
		".js":    true,
		".ts":    true,
		".jsx":   true,
		".tsx":   true,
		".py":    true,
		".java":  true,
		".c":     true,
		".cpp":   true,
		".h":     true,
		".rs":    true,
		".php":   true,
		".rb":    true,
		".swift": true,
		".kt":    true,
	}
	return codeExts[ext]
}

func getGitDiff(target string) (string, error) {
	cmd := exec.Command("git", "diff", target)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("获取 git diff 失败: %w", err)
	}

	diff := string(output)
	if diff == "" {
		return "没有代码变更", nil
	}

	// 限制输出长度
	if len(diff) > 20000 {
		diff = diff[:20000] + "\n... (diff 过长，已截断)"
	}

	return diff, nil
}

func runLinter(filePath string) (string, error) {
	if filePath == "" {
		return "", fmt.Errorf("file_path is required")
	}

	// 根据文件扩展名选择 linter
	ext := filepath.Ext(filePath)
	var cmd *exec.Cmd
	var linterName string

	switch ext {
	case ".go":
		// 尝试 golangci-lint，如果没有则用 go vet
		if _, err := exec.LookPath("golangci-lint"); err == nil {
			cmd = exec.Command("golangci-lint", "run", filePath)
			linterName = "golangci-lint"
		} else if _, err := exec.LookPath("go"); err == nil {
			cmd = exec.Command("go", "vet", filePath)
			linterName = "go vet"
		} else {
			return "⚠️ 未安装 Go 相关的 linter 工具\n建议安装: brew install golangci-lint", nil
		}
	case ".js", ".ts", ".jsx", ".tsx":
		if _, err := exec.LookPath("eslint"); err == nil {
			cmd = exec.Command("eslint", filePath)
			linterName = "eslint"
		} else {
			return "⚠️ 未安装 eslint\n建议安装: npm install -g eslint", nil
		}
	case ".py":
		if _, err := exec.LookPath("pylint"); err == nil {
			cmd = exec.Command("pylint", filePath)
			linterName = "pylint"
		} else if _, err := exec.LookPath("flake8"); err == nil {
			cmd = exec.Command("flake8", filePath)
			linterName = "flake8"
		} else {
			return "⚠️ 未安装 Python linter\n建议安装: pip install pylint", nil
		}
	default:
		return fmt.Sprintf("⚠️ 不支持的文件类型: %s\n支持的类型: .go, .js, .ts, .py", ext), nil
	}

	output, err := cmd.CombinedOutput()
	result := string(output)

	if err != nil {
		// linter 发现问题时会返回非 0 退出码
		if result != "" {
			return fmt.Sprintf("🔍 使用 %s 检查结果:\n%s", linterName, result), nil
		}
		return "", fmt.Errorf("运行 %s 失败: %w", linterName, err)
	}

	if result == "" {
		return fmt.Sprintf("✅ %s 检查通过，未发现代码问题", linterName), nil
	}

	return fmt.Sprintf("🔍 使用 %s 检查结果:\n%s", linterName, result), nil
}

/* ===================== Code Review ===================== */

func codeReview(ctx context.Context, request string) (string, error) {
	systemPrompt := `你是一个专业的代码审查专家，擅长发现代码中的问题并提供改进建议。

审查重点：
1. 代码质量：可读性、可维护性、复杂度
2. 潜在 Bug：空指针、边界条件、并发问题
3. 性能问题：算法效率、资源泄漏
4. 安全问题：SQL 注入、XSS、敏感信息泄露
5. 最佳实践：命名规范、错误处理、代码结构

可用工具：
- read_file: 读取单个文件内容
- read_multiple_files: 批量读取多个文件
- list_files: 列出目录文件（支持递归）
- search_in_files: 在文件中搜索关键字
- analyze_directory: 分析目录结构和代码文件
- get_git_diff: 获取代码变更
- run_linter: 运行代码检查工具

工作流程：
1. 使用 analyze_directory 或 list_files 了解目录结构
2. 使用 read_file 或 read_multiple_files 读取具体代码
3. 使用 search_in_files 查找特定模式（如 TODO、FIXME、安全问题）
4. 仔细分析代码，找出问题
5. 给出具体的改进建议和示例代码

注意：
- 对于目录审查，先用 analyze_directory 了解结构，再批量读取关键文件
- 单次最多读取10个文件，避免 token 超限
- 获取代码后，你需要自己分析并给出审查意见`

	messages := []Message{
		{Role: "system", Content: systemPrompt},
		{Role: "user", Content: request},
	}

	// Agent Loop - 最多循环 10 次
	for i := 0; i < 100; i++ {
		resp, err := callDeepSeek(ctx, messages, true)
		if err != nil {
			return "", fmt.Errorf("调用 LLM 失败: %w", err)
		}

		if len(resp.Choices) == 0 {
			return "", fmt.Errorf("LLM 未返回响应")
		}

		choice := resp.Choices[0]
		assistantMsg := choice.Message
		log.Printf("[轮次 %d] finish_reason=%s, tool_calls=%d",
			i+1, choice.FinishReason, len(assistantMsg.ToolCalls))

		// 添加 assistant 消息到历史
		messages = append(messages, assistantMsg)

		// 如果没有 tool_calls，说明 LLM 已经完成分析
		if len(assistantMsg.ToolCalls) == 0 {
			return assistantMsg.Content, nil
		}

		// 执行所有 tool calls
		for _, tc := range assistantMsg.ToolCalls {
			var args map[string]interface{}
			json.Unmarshal([]byte(tc.Function.Arguments), &args)

			log.Printf("执行工具: %s, 参数: %v", tc.Function.Name, args)

			result, err := executeTool(tc.Function.Name, args)
			if err != nil {
				result = fmt.Sprintf("❌ 工具执行失败: %s\n错误详情: %v", tc.Function.Name, err)
				log.Printf("工具执行失败: %s, 错误: %v", tc.Function.Name, err)
			} else {
				log.Printf("工具执行成功: %s", tc.Function.Name)
			}

			// 添加 tool 结果消息
			messages = append(messages, Message{
				Role:       "tool",
				Content:    result,
				ToolCallID: tc.ID,
			})
		}
	}

	return "", fmt.Errorf("达到最大循环次数")
}

/* ===================== Gin Handler ===================== */

func reviewHandlerGin(c *gin.Context) {
	var payload struct {
		Request string `json:"request" binding:"required"`
	}

	if err := c.ShouldBindJSON(&payload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "Invalid request: " + err.Error(),
		})
		return
	}

	log.Printf("收到 Code Review 请求: %s", payload.Request)

	result, err := codeReview(c.Request.Context(), payload.Request)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"review": result,
	})
}

func healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "ok",
	})
}

/* ===================== 原生 HTTP Handler (CLI 模式用) ===================== */

func reviewHandler(w http.ResponseWriter, r *http.Request) {
	var payload struct {
		Request string `json:"request"`
	}

	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, "Invalid request", 400)
		return
	}

	log.Printf("收到 Code Review 请求: %s", payload.Request)

	result, err := codeReview(r.Context(), payload.Request)
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}

	json.NewEncoder(w).Encode(map[string]string{
		"review": result,
	})
}

func cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "*")
		w.Header().Set("Access-Control-Allow-Methods", "*")
		if r.Method == http.MethodOptions {
			return
		}
		next.ServeHTTP(w, r)
	})
}

func health(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("ok"))
}

/* ===================== CLI 模式 ===================== */

func runCLI() {
	if len(os.Args) < 2 {
		fmt.Println("用法:")
		fmt.Println("  ai-cr review <file>           - 审查指定文件")
		fmt.Println("  ai-cr diff                    - 审查 git diff")
		fmt.Println("  ai-cr server                  - 启动 HTTP 服务")
		os.Exit(1)
	}

	command := os.Args[1]
	ctx := context.Background()

	switch command {
	case "review":
		if len(os.Args) < 3 {
			fmt.Println("请指定要审查的文件")
			os.Exit(1)
		}
		filePath := os.Args[2]
		request := fmt.Sprintf("请审查文件: %s", filePath)

		fmt.Println("🔍 开始代码审查...")
		result, err := codeReview(ctx, request)
		if err != nil {
			fmt.Printf("❌ 审查失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n📝 审查结果:")
		fmt.Println(result)

	case "diff":
		request := "请审查当前的 git diff 变更"

		fmt.Println("🔍 开始审查代码变更...")
		result, err := codeReview(ctx, request)
		if err != nil {
			fmt.Printf("❌ 审查失败: %v\n", err)
			os.Exit(1)
		}
		fmt.Println("\n📝 审查结果:")
		fmt.Println(result)

	case "server":
		startServer()

	default:
		fmt.Printf("未知命令: %s\n", command)
		os.Exit(1)
	}
}

func startServer() {
	// 设置 Gin 模式
	gin.SetMode(gin.ReleaseMode)

	r := gin.Default()

	// CORS 中间件
	r.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "*")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	})

	// 路由
	r.GET("/health", healthHandler)
	r.POST("/api/review", reviewHandlerGin)

	log.Println("🚀 AI Code Review 服务启动 :8083")
	log.Println("📌 POST /api/review {\"request\": \"请审查 main.go\"}")

	if err := r.Run(":8083"); err != nil {
		log.Fatalf("服务启动失败: %v", err)
	}
}

/* ===================== main ===================== */

func main() {
	runCLI()
}
