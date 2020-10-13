package template

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// LoadTemplateFunc 加载模板函数类
type LoadTemplateFunc func(templatesDir string, funcMap FuncMap) Render

// Engine 模板引擎
type Engine struct {
	templatesDir     string
	watcher          *fsnotify.Watcher
	Errors           <-chan error
	loadTemplateFunc LoadTemplateFunc
	FuncMap          FuncMap
	HTMLRender       Render
	opts             Options
}

// NewEngine 创建一个gin引擎模板
func NewEngine(templateDir string, tmplFunc LoadTemplateFunc, funcMap FuncMap, opts ...Option) (*Engine, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &Engine{
		templatesDir:     templateDir,
		loadTemplateFunc: tmplFunc,
		watcher:          watcher,
		Errors:           make(<-chan error),
		FuncMap:          funcMap,
		opts:             newOptions(opts...),
	}, nil
}

// Init 初始化
func (en *Engine) Init() {
	en.HTMLRender = en.loadTemplate()
}

// Watching 监听模板文件夹中是否有变动
func (en *Engine) Watching() error {
	en.Errors = en.watcher.Errors

	go func() {
		for {
			event := <-en.watcher.Events
			loadFlag := true
			switch event.Op {
			case fsnotify.Create:
				fileInfo, err := os.Stat(event.Name)
				if err == nil && fileInfo.IsDir() {
					en.watcher.Add(event.Name)
				}
			case fsnotify.Remove, fsnotify.Rename:
				fileInfo, err := os.Stat(event.Name)
				if err == nil && fileInfo.IsDir() {
					en.watcher.Remove(event.Name)
				}
			case fsnotify.Chmod:
				loadFlag = false
			}
			if loadFlag {
				en.HTMLRender = en.loadTemplate()
			}
		}
	}()

	//遍历目录下的所有子目录
	err := filepath.Walk(en.templatesDir, func(path string, info os.FileInfo, err error) error {
		if info.IsDir() {
			err := en.watcher.Add(path)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	return nil
}

// Close 关闭
func (en *Engine) Close() error {
	return en.watcher.Close()
}

// loadTemplate 加载模板
func (en *Engine) loadTemplate() Render {
	return en.loadTemplateFunc(en.templatesDir, en.FuncMap)
}

// PageName 页面
func (en *Engine) PageName(name string) string {
	return fmt.Sprintf("%s:pages/%s", en.opts.Layout, name)
}

// SingleName 单页
func (en *Engine) SingleName(name string) string {
	return fmt.Sprintf("singles/%s.tmpl", name)
}

// ErrorName 单页
func (en *Engine) ErrorName(name string) string {
	return fmt.Sprintf("error/%s.tmpl", name)
}

// RenderPage 渲染页面
func (en *Engine) RenderPage(w io.Writer, name string, data H, opts ...Option) {
	en.render(w, name, "page", data, opts...)
}

// RenderSingle 渲染单页面
func (en *Engine) RenderSingle(w io.Writer, name string, data H, opts ...Option) {
	en.render(w, name, "single", data, opts...)
}

// RenderError 渲染错误页面
func (en *Engine) RenderError(w io.Writer, name string, data H, opts ...Option) {
	en.render(w, name, "error", data, opts...)
}

// render 渲染
func (en *Engine) render(w io.Writer, name, typ string, data H, opts ...Option) {
	opt := en.opts
	for _, o := range opts {
		o(&opt)
	}

	if data == nil {
		data = H{}
	}
	data["constant"] = en.opts.GlobalConstant
	data["variable"] = en.opts.GlobalVariable
	switch typ {
	case "page":
		en.HTMLRender.Execute(en.PageName(name), w, data)
	case "single":
		en.HTMLRender.Execute(en.SingleName(name), w, data)
	case "error":
		en.HTMLRender.Execute(en.ErrorName(name), w, data)
	}
}
