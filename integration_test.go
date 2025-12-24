package template

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiThemeIntegration 测试多主题功能的完整集成
func TestMultiThemeIntegration(t *testing.T) {
	// 创建测试目录结构
	testDir := setupMultiThemeTestDir(t)
	defer os.RemoveAll(testDir)

	// 创建多主题引擎 - 使用测试目录作为模板目录
	engine, err := NewEngine(testDir, DefaultLoadTemplate, nil,
		EnableMultiTheme(true),
		DefaultTheme("default"),
		SetTheme("default"),
		GlobalConstant(map[string]any{
			"siteName": "测试网站",
			"version":  "1.0.0",
		}),
	)
	require.NoError(t, err)
	defer engine.Close()

	engine.Init()

	// 测试基本功能
	t.Run("BasicMultiThemeFunctionality", func(t *testing.T) {
		testBasicMultiThemeFunctionality(t, engine)
	})

	// 测试主题切换
	t.Run("ThemeSwitching", func(t *testing.T) {
		testThemeSwitching(t, engine)
	})

	// 测试Web集成
	t.Run("WebIntegration", func(t *testing.T) {
		testWebIntegration(t, engine)
	})

	// 测试向后兼容性
	t.Run("BackwardCompatibility", func(t *testing.T) {
		testBackwardCompatibility(t, testDir)
	})
}

// testBasicMultiThemeFunctionality 测试基本多主题功能
func testBasicMultiThemeFunctionality(t *testing.T, engine *Engine) {
	// 测试获取可用主题
	themes := engine.GetAvailableThemes()
	assert.Contains(t, themes, "default")
	assert.Contains(t, themes, "dark")
	assert.Len(t, themes, 2)

	// 测试获取当前主题
	currentTheme := engine.GetCurrentTheme()
	assert.Equal(t, "default", currentTheme)

	// 测试渲染页面
	var buf bytes.Buffer
	data := H{
		"title":   "测试页面",
		"content": "这是测试内容",
	}

	err := engine.RenderPage(&buf, "test", data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "测试页面")
	assert.Contains(t, output, "这是测试内容")
	assert.Contains(t, output, "default") // 应该包含当前主题信息
}

// testThemeSwitching 测试主题切换功能
func testThemeSwitching(t *testing.T, engine *Engine) {
	// 切换到深色主题
	err := engine.SwitchTheme("dark")
	require.NoError(t, err)

	// 验证当前主题已更改
	assert.Equal(t, "dark", engine.GetCurrentTheme())

	// 测试在新主题下渲染
	var buf bytes.Buffer
	data := H{
		"title":   "深色主题测试",
		"content": "深色主题内容",
	}

	err = engine.RenderPage(&buf, "test", data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "深色主题测试")
	assert.Contains(t, output, "dark")                // 应该包含新主题信息
	assert.Contains(t, output, "background: #1a1a1a") // 深色主题特有样式

	// 切换回默认主题
	err = engine.SwitchTheme("default")
	require.NoError(t, err)
	assert.Equal(t, "default", engine.GetCurrentTheme())

	// 测试切换到不存在的主题
	err = engine.SwitchTheme("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not found")

	// 验证失败后主题没有改变
	assert.Equal(t, "default", engine.GetCurrentTheme())
}

// testWebIntegration 测试Web集成功能
func testWebIntegration(t *testing.T, engine *Engine) {
	// 创建测试服务器
	mux := http.NewServeMux()

	// 主页路由
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		data := H{
			"title":        "首页",
			"currentTheme": engine.GetCurrentTheme(),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := engine.RenderPage(w, "test", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// 主题管理路由
	mux.HandleFunc("/themes", func(w http.ResponseWriter, r *http.Request) {
		data := H{
			"title":           "主题管理",
			"availableThemes": engine.GetAvailableThemes(),
			"currentTheme":    engine.GetCurrentTheme(),
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		if err := engine.RenderPage(w, "themes", data); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})

	// 主题切换路由
	mux.HandleFunc("/switch-theme", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}

		themeName := r.FormValue("theme")
		if themeName == "" {
			http.Error(w, "Theme name is required", http.StatusBadRequest)
			return
		}

		if err := engine.SwitchTheme(themeName); err != nil {
			http.Error(w, fmt.Sprintf("Failed to switch theme: %v", err), http.StatusInternalServerError)
			return
		}

		http.Redirect(w, r, "/themes", http.StatusSeeOther)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	// 测试主页访问
	resp, err := http.Get(server.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "首页")
	assert.Contains(t, bodyStr, "default")

	// 测试主题管理页面
	resp, err = http.Get(server.URL + "/themes")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr = string(body)
	assert.Contains(t, bodyStr, "主题管理")
	assert.Contains(t, bodyStr, "default")
	assert.Contains(t, bodyStr, "dark")

	// 测试主题切换
	form := url.Values{"theme": {"dark"}}
	resp, err = http.PostForm(server.URL+"/switch-theme", form)
	require.NoError(t, err)
	defer resp.Body.Close()

	// PostForm 会自动跟随重定向，所以应该得到最终页面的状态码
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	// 验证主题已切换
	assert.Equal(t, "dark", engine.GetCurrentTheme())

	// 再次访问主页，验证新主题生效
	resp, err = http.Get(server.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr = string(body)
	assert.Contains(t, bodyStr, "dark")
	assert.Contains(t, bodyStr, "background: #1a1a1a")

	// 测试无效主题切换
	form = url.Values{"theme": {"invalid"}}
	resp, err = http.PostForm(server.URL+"/switch-theme", form)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// 验证主题没有改变
	assert.Equal(t, "dark", engine.GetCurrentTheme())
}

// testBackwardCompatibility 测试向后兼容性
func testBackwardCompatibility(t *testing.T, baseDir string) {
	// 创建传统单主题结构
	legacyDir := filepath.Join(baseDir, "legacy")
	err := os.MkdirAll(legacyDir, 0755)
	require.NoError(t, err)

	// 创建传统目录结构
	dirs := []string{"layouts", "pages", "singles", "errors"}
	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(legacyDir, dir), 0755)
		require.NoError(t, err)
	}

	// 创建页面子目录（传统模式也需要）
	err = os.MkdirAll(filepath.Join(legacyDir, "pages", "test"), 0755)
	require.NoError(t, err)

	// 创建传统模板文件
	layoutContent := `<!DOCTYPE html>
<html>
<head><title>{{ .title }}</title></head>
<body>{{ template "content" . }}</body>
</html>`

	pageContent := `{{ define "content" }}
<h1>{{ .title }}</h1>
<p>{{ .content }}</p>
{{ end }}`

	err = os.WriteFile(filepath.Join(legacyDir, "layouts", "layout.tmpl"), []byte(layoutContent), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(legacyDir, "pages", "test", "test.tmpl"), []byte(pageContent), 0644)
	require.NoError(t, err)

	// 测试传统模式（不启用多主题）
	t.Run("LegacyMode", func(t *testing.T) {
		engine, err := NewEngine(legacyDir, DefaultLoadTemplate, nil)
		require.NoError(t, err)
		defer engine.Close()

		engine.Init()

		// 在传统模式下，应该只有一个默认主题
		themes := engine.GetAvailableThemes()
		assert.Len(t, themes, 1)
		assert.Contains(t, themes, "default")

		currentTheme := engine.GetCurrentTheme()
		assert.Equal(t, "default", currentTheme)

		// 测试渲染功能
		var buf bytes.Buffer
		data := H{
			"title":   "传统模式测试",
			"content": "传统模式内容",
		}

		err = engine.RenderPage(&buf, "test", data)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "传统模式测试")
		assert.Contains(t, output, "传统模式内容")
	})

	// 测试自动检测模式
	t.Run("AutoDetectionMode", func(t *testing.T) {
		engine, err := NewEngine(legacyDir, DefaultLoadTemplate, nil,
			EnableMultiTheme(true), // 启用多主题但使用传统结构
		)
		require.NoError(t, err)
		defer engine.Close()

		engine.Init()

		// 应该自动检测为传统模式
		themes := engine.GetAvailableThemes()
		assert.Len(t, themes, 1)
		assert.Contains(t, themes, "default")

		// 功能应该正常工作
		var buf bytes.Buffer
		data := H{
			"title":   "自动检测测试",
			"content": "自动检测内容",
		}

		err = engine.RenderPage(&buf, "test", data)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "自动检测测试")
	})
}

// setupMultiThemeTestDir 创建多主题测试目录结构
func setupMultiThemeTestDir(t *testing.T) string {
	testDir, err := os.MkdirTemp("", "multi_theme_test_*")
	require.NoError(t, err)

	// 创建默认主题
	defaultDir := filepath.Join(testDir, "default")
	createThemeDir(t, defaultDir, "default", "#2c3e50", "#ffffff")

	// 创建深色主题
	darkDir := filepath.Join(testDir, "dark")
	createThemeDir(t, darkDir, "dark", "#bb86fc", "#1a1a1a")

	return testDir
}

// createThemeDir 创建主题目录和文件
func createThemeDir(t *testing.T, themeDir, themeName, primaryColor, backgroundColor string) {
	// 创建目录结构
	dirs := []string{"layouts", "pages", "singles", "errors", "partials"}
	for _, dir := range dirs {
		err := os.MkdirAll(filepath.Join(themeDir, dir), 0755)
		require.NoError(t, err)
	}

	// 创建页面子目录
	pageSubdirs := []string{"test", "themes"}
	for _, subdir := range pageSubdirs {
		err := os.MkdirAll(filepath.Join(themeDir, "pages", subdir), 0755)
		require.NoError(t, err)
	}

	// 创建主题配置文件
	themeConfig := fmt.Sprintf(`{
    "name": "%s",
    "displayName": "%s主题",
    "description": "%s主题描述",
    "version": "1.0.0",
    "author": "测试",
    "tags": ["%s", "test"],
    "custom": {
        "primaryColor": "%s",
        "backgroundColor": "%s"
    }
}`, themeName, themeName, themeName, themeName, primaryColor, backgroundColor)

	err := os.WriteFile(filepath.Join(themeDir, "theme.json"), []byte(themeConfig), 0644)
	require.NoError(t, err)

	// 创建布局模板
	layoutContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>{{ .title }}</title>
    <style>
        body { background: %s; color: %s; font-family: Arial, sans-serif; }
        .theme-info { color: %s; }
    </style>
</head>
<body>
    <div class="theme-info">当前主题: %s</div>
    {{ template "content" . }}
</body>
</html>`, backgroundColor, primaryColor, primaryColor, themeName)

	err = os.WriteFile(filepath.Join(themeDir, "layouts", "layout.tmpl"), []byte(layoutContent), 0644)
	require.NoError(t, err)

	// 创建页面模板 - 放在子目录中
	pageContent := `{{ define "content" }}
<h1>{{ .title }}</h1>
<p>{{ .content }}</p>
{{ end }}`

	err = os.WriteFile(filepath.Join(themeDir, "pages", "test", "test.tmpl"), []byte(pageContent), 0644)
	require.NoError(t, err)

	// 创建主题管理页面模板 - 放在子目录中
	themesContent := `{{ define "content" }}
<h1>{{ .title }}</h1>
<p>可用主题: {{ range .availableThemes }}{{ . }} {{ end }}</p>
<p>当前主题: {{ .currentTheme }}</p>
{{ end }}`

	err = os.WriteFile(filepath.Join(themeDir, "pages", "themes", "themes.tmpl"), []byte(themesContent), 0644)
	require.NoError(t, err)

	// 创建单页模板
	singleContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>{{ .title }}</title>
    <style>body { background: %s; }</style>
</head>
<body>
    <h1>{{ .title }}</h1>
    <p>单页模板 - %s主题</p>
</body>
</html>`, backgroundColor, themeName)

	err = os.WriteFile(filepath.Join(themeDir, "singles", "login.tmpl"), []byte(singleContent), 0644)
	require.NoError(t, err)

	// 创建错误页面模板
	errorContent := fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>{{ .title }}</title>
    <style>body { background: %s; }</style>
</head>
<body>
    <h1>{{ .title }}</h1>
    <p>{{ .message }}</p>
    <p>错误页面 - %s主题</p>
</body>
</html>`, backgroundColor, themeName)

	err = os.WriteFile(filepath.Join(themeDir, "errors", "404.tmpl"), []byte(errorContent), 0644)
	require.NoError(t, err)
}
