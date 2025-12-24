package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nilorg/template"
)

var engine *template.Engine

func main() {
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

	var err error
	// 创建支持多主题的模板引擎
	engine, err = template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
		template.GlobalConstant(map[string]interface{}{
			"siteName": "多主题示例网站",
			"version":  "2.0.0",
		}),
		template.GlobalVariable(map[string]interface{}{
			"year": time.Now().Year(),
		}),
		// 启用多主题模式并设置默认主题
		template.EnableMultiTheme(true),
		template.DefaultTheme("default"),
	)
	if err != nil {
		log.Fatalf("创建模板引擎失败: %s\n", err)
	}
	engine.Init()

	// 设置初始主题为默认主题
	err = engine.SwitchTheme("default")
	if err != nil {
		log.Fatalf("设置默认主题失败: %s\n", err)
	}

	err = engine.Watching()
	if err != nil {
		log.Fatalf("启动模板监听失败: %s\n", err)
	}

	// 监听模板加载错误
	go func() {
		for err := range engine.Errors {
			log.Printf("模板监听错误: %s\n", err)
		}
	}()

	// 设置路由
	mux := http.NewServeMux()
	mux.HandleFunc("/", homeHandler)
	mux.HandleFunc("/posts", postsListHandler)
	mux.HandleFunc("/posts/detail", postsDetailHandler)
	mux.HandleFunc("/login", loginHandler)

	// 新增主题管理路由
	mux.HandleFunc("/themes", themesHandler)
	mux.HandleFunc("/switch-theme", switchThemeHandler)

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// 启动服务器
	go func() {
		log.Println("多主题示例服务器启动在 http://localhost:8080")
		log.Println("访问 /themes 查看可用主题并切换")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("服务器启动失败: %s\n", err)
		}
	}()

	// 等待退出信号
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("正在关闭服务器...")

	// 优雅关闭
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("服务器关闭失败: %s\n", err)
	}

	engine.Close()
	log.Println("服务器已关闭")
}

// homeHandler 首页 - 重定向到文章列表
func homeHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		errorHandler(w, r, http.StatusNotFound)
		return
	}
	http.Redirect(w, r, "/posts", http.StatusFound)
}

// postsListHandler 文章列表页
func postsListHandler(w http.ResponseWriter, r *http.Request) {
	data := template.H{
		"title": "文章列表",
		"posts": []map[string]interface{}{
			{"id": 1, "title": "Go 语言入门", "summary": "学习 Go 语言的基础知识", "createdAt": time.Now()},
			{"id": 2, "title": "Go 模板引擎", "summary": "使用 Go 标准库的模板引擎", "createdAt": time.Now().Add(-24 * time.Hour)},
			{"id": 3, "title": "多主题系统设计", "summary": "如何设计一个灵活的多主题系统", "createdAt": time.Now().Add(-48 * time.Hour)},
		},
		"currentTheme": engine.GetCurrentTheme(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := engine.RenderPage(w, "posts/list", data); err != nil {
		log.Printf("渲染页面失败: %s\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// postsDetailHandler 文章详情页
func postsDetailHandler(w http.ResponseWriter, r *http.Request) {
	data := template.H{
		"title":        "Go 语言入门",
		"content":      "这是文章的详细内容...\n\n当前使用的主题展示了不同的样式设计。",
		"author":       "张三",
		"createdAt":    time.Now(),
		"currentTheme": engine.GetCurrentTheme(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := engine.RenderPage(w, "posts/detail", data); err != nil {
		log.Printf("渲染页面失败: %s\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// loginHandler 登录页（单页面）
func loginHandler(w http.ResponseWriter, r *http.Request) {
	data := template.H{
		"title":        "用户登录",
		"currentTheme": engine.GetCurrentTheme(),
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := engine.RenderSingle(w, "login", data); err != nil {
		log.Printf("渲染页面失败: %s\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// themesHandler 主题管理页面
func themesHandler(w http.ResponseWriter, r *http.Request) {
	availableThemes := engine.GetAvailableThemes()
	currentTheme := engine.GetCurrentTheme()

	data := template.H{
		"title":           "主题管理",
		"availableThemes": availableThemes,
		"currentTheme":    currentTheme,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := engine.RenderPage(w, "themes", data); err != nil {
		log.Printf("渲染主题页面失败: %s\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// switchThemeHandler 主题切换处理器
func switchThemeHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	themeName := r.FormValue("theme")
	if themeName == "" {
		http.Error(w, "Theme name is required", http.StatusBadRequest)
		return
	}

	// 切换主题
	if err := engine.SwitchTheme(themeName); err != nil {
		log.Printf("主题切换失败: %s\n", err)
		http.Error(w, fmt.Sprintf("Failed to switch theme: %v", err), http.StatusInternalServerError)
		return
	}

	log.Printf("主题已切换到: %s\n", themeName)

	// 重定向回主题管理页面
	http.Redirect(w, r, "/themes", http.StatusSeeOther)
}

// errorHandler 错误页面
func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	data := template.H{
		"title":        fmt.Sprintf("%d 错误", status),
		"message":      http.StatusText(status),
		"path":         r.URL.Path,
		"currentTheme": engine.GetCurrentTheme(),
	}

	if err := engine.RenderError(w, fmt.Sprintf("%d", status), data); err != nil {
		log.Printf("渲染错误页面失败: %s\n", err)
		http.Error(w, http.StatusText(status), status)
	}
}
