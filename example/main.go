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
	}

	var err error
	engine, err = template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
		template.GlobalConstant(map[string]interface{}{
			"siteName": "我的网站",
			"version":  "1.0.0",
		}),
		template.GlobalVariable(map[string]interface{}{
			"year": time.Now().Year(),
		}),
	)
	if err != nil {
		log.Fatalf("创建模板引擎失败: %s\n", err)
	}
	engine.Init()

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

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	// 启动服务器
	go func() {
		log.Println("服务器启动在 http://localhost:8080")
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
			{"id": 3, "title": "Web 开发实践", "summary": "使用 Go 构建 Web 应用", "createdAt": time.Now().Add(-48 * time.Hour)},
		},
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
		"title":     "Go 语言入门",
		"content":   "这是文章的详细内容...",
		"author":    "张三",
		"createdAt": time.Now(),
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
		"title": "用户登录",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := engine.RenderSingle(w, "login", data); err != nil {
		log.Printf("渲染页面失败: %s\n", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
	}
}

// errorHandler 错误页面
func errorHandler(w http.ResponseWriter, r *http.Request, status int) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(status)

	data := template.H{
		"title":   fmt.Sprintf("%d 错误", status),
		"message": http.StatusText(status),
		"path":    r.URL.Path,
	}

	if err := engine.RenderError(w, fmt.Sprintf("%d", status), data); err != nil {
		log.Printf("渲染错误页面失败: %s\n", err)
		http.Error(w, http.StatusText(status), status)
	}
}
