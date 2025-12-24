package main

import (
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/nilorg/template"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiThemeExampleApplication 测试多主题示例应用的完整功能
func TestMultiThemeExampleApplication(t *testing.T) {
	// 创建测试引擎
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"mod": func(a, b int) int {
			return a % b
		},
	}

	testEngine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
		template.GlobalConstant(map[string]interface{}{
			"siteName": "多主题示例网站",
			"version":  "2.0.0",
		}),
		template.GlobalVariable(map[string]interface{}{
			"year": time.Now().Year(),
		}),
		template.EnableMultiTheme(true),
		template.DefaultTheme("default"),
	)
	require.NoError(t, err)
	defer testEngine.Close()

	testEngine.Init()

	// 设置当前主题为默认主题
	err = testEngine.SwitchTheme("default")
	require.NoError(t, err)

	// 设置全局引擎变量供处理器使用
	engine = testEngine

	// 创建测试服务器
	mux := setupTestRoutes()
	server := httptest.NewServer(mux)
	defer server.Close()

	// 运行测试套件
	t.Run("HomePageRedirect", func(t *testing.T) {
		testHomePageRedirect(t, server)
	})

	t.Run("PostsListPage", func(t *testing.T) {
		testPostsListPage(t, server)
	})

	t.Run("PostsDetailPage", func(t *testing.T) {
		testPostsDetailPage(t, server)
	})

	t.Run("LoginPage", func(t *testing.T) {
		testLoginPage(t, server)
	})

	t.Run("ThemesManagementPage", func(t *testing.T) {
		testThemesManagementPage(t, server)
	})

	t.Run("ThemeSwitching", func(t *testing.T) {
		testThemeSwitching(t, server)
	})

	t.Run("ErrorHandling", func(t *testing.T) {
		testErrorHandling(t, server)
	})

	t.Run("BackwardCompatibility", func(t *testing.T) {
		testBackwardCompatibility(t)
	})
}

// setupTestRoutes 设置测试路由
func setupTestRoutes() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/posts", postsListHandler)
	mux.HandleFunc("/posts/detail", postsDetailHandler)
	mux.HandleFunc("/login", loginHandler)
	mux.HandleFunc("/themes", themesHandler)
	mux.HandleFunc("/switch-theme", switchThemeHandler)
	return mux
}

// testHomePageRedirect 测试首页重定向
func testHomePageRedirect(t *testing.T, server *httptest.Server) {
	// 测试根路径重定向
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不跟随重定向
		},
	}

	resp, err := client.Get(server.URL + "/")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusFound, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Equal(t, "/posts", location)

	// 测试无效路径返回404
	resp, err = client.Get(server.URL + "/invalid-path")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)
}

// testPostsListPage 测试文章列表页面
func testPostsListPage(t *testing.T, server *httptest.Server) {
	resp, err := http.Get(server.URL + "/posts")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)
	assert.Equal(t, "text/html; charset=utf-8", resp.Header.Get("Content-Type"))

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)

	// 验证页面内容
	assert.Contains(t, bodyStr, "文章列表")
	assert.Contains(t, bodyStr, "Go 语言入门")
	assert.Contains(t, bodyStr, "Go 模板引擎")
	assert.Contains(t, bodyStr, "多主题系统设计")

	// 验证主题信息
	assert.Contains(t, bodyStr, "default") // 当前主题应该是default

	// 验证导航链接
	assert.Contains(t, bodyStr, `href="/themes"`)
	assert.Contains(t, bodyStr, "主题管理")
}

// testPostsDetailPage 测试文章详情页面
func testPostsDetailPage(t *testing.T, server *httptest.Server) {
	resp, err := http.Get(server.URL + "/posts/detail?id=1")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)

	// 验证页面内容
	assert.Contains(t, bodyStr, "Go 语言入门")
	assert.Contains(t, bodyStr, "这是文章的详细内容")
	assert.Contains(t, bodyStr, "张三")
	assert.Contains(t, bodyStr, "当前使用的主题展示了不同的样式设计")

	// 验证主题信息
	assert.Contains(t, bodyStr, "default")
}

// testLoginPage 测试登录页面
func testLoginPage(t *testing.T, server *httptest.Server) {
	resp, err := http.Get(server.URL + "/login")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)

	// 验证页面内容
	assert.Contains(t, bodyStr, "用户登录")
	assert.Contains(t, bodyStr, `type="text"`)
	assert.Contains(t, bodyStr, `type="password"`)
	assert.Contains(t, bodyStr, "请输入用户名")
	assert.Contains(t, bodyStr, "请输入密码")

	// 验证主题信息
	assert.Contains(t, bodyStr, "default")

	// 验证返回链接
	assert.Contains(t, bodyStr, `href="/"`)
	assert.Contains(t, bodyStr, `href="/themes"`)
}

// testThemesManagementPage 测试主题管理页面
func testThemesManagementPage(t *testing.T, server *httptest.Server) {
	resp, err := http.Get(server.URL + "/themes")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)

	// 验证页面内容
	assert.Contains(t, bodyStr, "主题管理")
	assert.Contains(t, bodyStr, "当前主题: default")

	// 验证可用主题
	assert.Contains(t, bodyStr, "default")
	assert.Contains(t, bodyStr, "dark")
	assert.Contains(t, bodyStr, "colorful")

	// 验证主题描述
	assert.Contains(t, bodyStr, "简洁的默认主题")
	assert.Contains(t, bodyStr, "深色主题，适合夜间浏览")
	assert.Contains(t, bodyStr, "彩色主题，活泼有趣的设计")

	// 验证切换按钮
	assert.Contains(t, bodyStr, "切换到此主题")
	assert.Contains(t, bodyStr, `method="POST"`)
	assert.Contains(t, bodyStr, `action="/switch-theme"`)

	// 验证当前主题标识
	assert.Contains(t, bodyStr, "当前使用")
}

// testThemeSwitching 测试主题切换功能
func testThemeSwitching(t *testing.T, server *httptest.Server) {
	// 确保初始主题是default
	assert.Equal(t, "default", engine.GetCurrentTheme())

	// 创建不跟随重定向的客户端
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不跟随重定向
		},
	}

	// 测试切换到深色主题
	form := url.Values{"theme": {"dark"}}
	resp, err := client.PostForm(server.URL+"/switch-theme", form)
	require.NoError(t, err)
	defer resp.Body.Close()

	// 应该重定向到主题管理页面
	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	location := resp.Header.Get("Location")
	assert.Equal(t, "/themes", location)

	// 验证主题已切换
	assert.Equal(t, "dark", engine.GetCurrentTheme())

	// 访问主题管理页面验证切换结果
	resp, err = http.Get(server.URL + "/themes")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "当前主题: dark")
	assert.Contains(t, bodyStr, "深色主题")

	// 测试切换到彩色主题
	form = url.Values{"theme": {"colorful"}}
	resp, err = client.PostForm(server.URL+"/switch-theme", form)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Equal(t, "colorful", engine.GetCurrentTheme())

	// 验证彩色主题的特殊样式
	resp, err = http.Get(server.URL + "/posts")
	require.NoError(t, err)
	defer resp.Body.Close()

	body, err = io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr = string(body)
	assert.Contains(t, bodyStr, "colorful")
	assert.Contains(t, bodyStr, "linear-gradient") // 彩色主题特有的渐变样式

	// 测试切换到无效主题
	form = url.Values{"theme": {"invalid"}}
	resp, err = client.PostForm(server.URL+"/switch-theme", form)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusInternalServerError, resp.StatusCode)

	// 验证主题没有改变
	assert.Equal(t, "colorful", engine.GetCurrentTheme())

	// 测试空主题名
	form = url.Values{}
	resp, err = client.PostForm(server.URL+"/switch-theme", form)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	// 测试GET请求到切换接口
	resp, err = http.Get(server.URL + "/switch-theme")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusMethodNotAllowed, resp.StatusCode)

	// 恢复到默认主题
	form = url.Values{"theme": {"default"}}
	resp, err = client.PostForm(server.URL+"/switch-theme", form)
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusSeeOther, resp.StatusCode)
	assert.Equal(t, "default", engine.GetCurrentTheme())
}

// testErrorHandling 测试错误处理
func testErrorHandling(t *testing.T, server *httptest.Server) {
	// 测试404错误页面
	resp, err := http.Get(server.URL + "/nonexistent")
	require.NoError(t, err)
	defer resp.Body.Close()

	assert.Equal(t, http.StatusNotFound, resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	require.NoError(t, err)

	bodyStr := string(body)
	assert.Contains(t, bodyStr, "404")
	assert.Contains(t, bodyStr, "Not Found")
	assert.Contains(t, bodyStr, "/nonexistent")
	assert.Contains(t, bodyStr, "default") // 当前主题信息

	// 验证错误页面的导航链接
	assert.Contains(t, bodyStr, `href="/"`)
	assert.Contains(t, bodyStr, `href="/themes"`)
}

// testBackwardCompatibility 测试向后兼容性
func testBackwardCompatibility(t *testing.T) {
	// 创建传统模式引擎（不启用多主题）
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"mod": func(a, b int) int {
			return a % b
		},
	}

	// 测试使用传统API创建引擎
	legacyEngine, err := template.NewEngine("./templates/default", template.DefaultLoadTemplate, funcMap,
		template.GlobalConstant(map[string]interface{}{
			"siteName": "传统模式网站",
			"version":  "1.0.0",
		}),
	)
	require.NoError(t, err)
	defer legacyEngine.Close()

	legacyEngine.Init()

	// 在传统模式下，应该只有一个默认主题
	themes := legacyEngine.GetAvailableThemes()
	assert.Len(t, themes, 1)
	assert.Contains(t, themes, "default")

	currentTheme := legacyEngine.GetCurrentTheme()
	assert.Equal(t, "default", currentTheme)

	// 测试传统渲染功能
	data := template.H{
		"title": "传统模式测试",
		"posts": []map[string]interface{}{
			{"id": 1, "title": "测试文章", "summary": "测试摘要", "createdAt": time.Now()},
		},
	}

	// 测试页面渲染
	var buf strings.Builder
	err = legacyEngine.RenderPage(&buf, "posts/list", data)
	require.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "传统模式测试")
	assert.Contains(t, output, "测试文章")

	// 测试单页渲染
	buf.Reset()
	loginData := template.H{"title": "登录页面"}
	err = legacyEngine.RenderSingle(&buf, "login", loginData)
	require.NoError(t, err)

	output = buf.String()
	assert.Contains(t, output, "登录页面")

	// 测试错误页面渲染
	buf.Reset()
	errorData := template.H{
		"title":   "404 错误",
		"message": "页面未找到",
		"path":    "/test",
	}
	err = legacyEngine.RenderError(&buf, "404", errorData)
	require.NoError(t, err)

	output = buf.String()
	assert.Contains(t, output, "404 错误")
	assert.Contains(t, output, "页面未找到")
}

// TestMultiThemeExamplePerformance 测试多主题示例应用性能
func TestMultiThemeExamplePerformance(t *testing.T) {
	// 创建测试引擎
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"mod": func(a, b int) int {
			return a % b
		},
	}

	testEngine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
		template.EnableMultiTheme(true),
		template.DefaultTheme("default"),
	)
	require.NoError(t, err)
	defer testEngine.Close()

	testEngine.Init()

	// 设置当前主题为默认主题
	err = testEngine.SwitchTheme("default")
	require.NoError(t, err)

	engine = testEngine

	// 创建测试服务器
	mux := setupTestRoutes()
	server := httptest.NewServer(mux)
	defer server.Close()

	// 性能测试：页面渲染
	t.Run("PageRenderingPerformance", func(t *testing.T) {
		start := time.Now()

		for i := 0; i < 100; i++ {
			resp, err := http.Get(server.URL + "/posts")
			require.NoError(t, err)
			resp.Body.Close()
		}

		duration := time.Since(start)
		t.Logf("100次页面请求耗时: %v", duration)
		assert.Less(t, duration, 5*time.Second)
	})

	// 性能测试：主题切换
	t.Run("ThemeSwitchingPerformance", func(t *testing.T) {
		themes := []string{"default", "dark", "colorful"}

		start := time.Now()

		for i := 0; i < 30; i++ {
			theme := themes[i%len(themes)]
			form := url.Values{"theme": {theme}}

			resp, err := http.PostForm(server.URL+"/switch-theme", form)
			require.NoError(t, err)
			resp.Body.Close()
		}

		duration := time.Since(start)
		t.Logf("30次主题切换耗时: %v", duration)
		assert.Less(t, duration, 3*time.Second)
	})
}

// TestMultiThemeExampleConcurrency 测试多主题示例应用并发安全
func TestMultiThemeExampleConcurrency(t *testing.T) {
	// 创建测试引擎
	funcMap := template.FuncMap{
		"formatDate": func(t time.Time) string {
			return t.Format("2006-01-02 15:04:05")
		},
		"list": func(items ...interface{}) []interface{} {
			return items
		},
		"mod": func(a, b int) int {
			return a % b
		},
	}

	testEngine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
		template.EnableMultiTheme(true),
		template.DefaultTheme("default"),
	)
	require.NoError(t, err)
	defer testEngine.Close()

	testEngine.Init()

	// 设置当前主题为默认主题
	err = testEngine.SwitchTheme("default")
	require.NoError(t, err)

	engine = testEngine

	// 创建测试服务器
	mux := setupTestRoutes()
	server := httptest.NewServer(mux)
	defer server.Close()

	// 并发测试：同时访问不同页面
	t.Run("ConcurrentPageAccess", func(t *testing.T) {
		const numGoroutines = 10
		const numRequests = 20

		done := make(chan bool, numGoroutines)

		urls := []string{"/posts", "/login", "/themes"}

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				for j := 0; j < numRequests; j++ {
					url := urls[j%len(urls)]
					resp, err := http.Get(server.URL + url)
					if err != nil {
						t.Errorf("Goroutine %d: 请求失败: %v", id, err)
						return
					}

					if resp.StatusCode != http.StatusOK {
						t.Errorf("Goroutine %d: 状态码错误: %d", id, resp.StatusCode)
					}

					resp.Body.Close()
				}
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}
	})

	// 并发测试：同时进行主题切换
	t.Run("ConcurrentThemeSwitching", func(t *testing.T) {
		const numGoroutines = 5
		const numSwitches = 10

		done := make(chan bool, numGoroutines)
		themes := []string{"default", "dark", "colorful"}

		for i := 0; i < numGoroutines; i++ {
			go func(id int) {
				defer func() { done <- true }()

				// 创建不跟随重定向的客户端
				client := &http.Client{
					CheckRedirect: func(req *http.Request, via []*http.Request) error {
						return http.ErrUseLastResponse // 不跟随重定向
					},
				}

				for j := 0; j < numSwitches; j++ {
					theme := themes[j%len(themes)]
					form := url.Values{"theme": {theme}}

					resp, err := client.PostForm(server.URL+"/switch-theme", form)
					if err != nil {
						t.Errorf("Goroutine %d: 主题切换请求失败: %v", id, err)
						return
					}

					if resp.StatusCode != http.StatusSeeOther {
						t.Errorf("Goroutine %d: 主题切换状态码错误: %d", id, resp.StatusCode)
					}

					resp.Body.Close()
				}
			}(i)
		}

		// 等待所有goroutine完成
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		// 验证最终状态一致性
		finalTheme := engine.GetCurrentTheme()
		assert.Contains(t, themes, finalTheme)
	})
}
