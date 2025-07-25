package template

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/fsnotify/fsnotify"
)

// LoadTemplateFunc 加载模板函数类
type LoadTemplateFunc func(templatesDir string, funcMap FuncMap) Render

// LoadEmbedFSTemplateFunc 加载模板函数类
type LoadEmbedFSTemplateFunc func(tmplFS *embed.FS, tmplFSSUbDir string, funcMap FuncMap) Render

// Engine 模板引擎
type Engine struct {
	templatesDir        string
	tmplFS              *embed.FS
	tmplFSSUbDir        string
	watcher             *fsnotify.Watcher
	Errors              <-chan error
	loadTemplateFunc    LoadTemplateFunc
	loadTemplateEmbedFS LoadEmbedFSTemplateFunc
	FuncMap             FuncMap
	HTMLRender          Render
	opts                Options
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

func NewEngineWithEmbedFS(tmplFS *embed.FS, tmplFSSUbDir string, tmplFunc LoadEmbedFSTemplateFunc, funcMap FuncMap, opts ...Option) (*Engine, error) {
	return &Engine{
		tmplFS:              tmplFS,
		tmplFSSUbDir:        tmplFSSUbDir,
		loadTemplateEmbedFS: tmplFunc,
		Errors:              make(<-chan error),
		FuncMap:             funcMap,
		opts:                newOptions(opts...),
	}, nil
}

// Init 初始化
func (en *Engine) Init() {
	en.HTMLRender = en.loadTemplate()
}

// Watching 监听模板文件夹中是否有变动
func (en *Engine) Watching() error {
	if en.templatesDir == "" {
		return fmt.Errorf("template directory is empty")
	}
	if en.watcher == nil {
		return fmt.Errorf("watcher is nil")
	}
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
	if en.watcher == nil {
		return nil
	}
	return en.watcher.Close()
}

// loadTemplate 加载模板
func (en *Engine) loadTemplate() Render {
	if en.templatesDir != "" {
		return en.loadTemplateFunc(en.templatesDir, en.FuncMap)
	} else {
		return en.loadTemplateEmbedFS(en.tmplFS, en.tmplFSSUbDir, en.FuncMap)
	}
}

// PageName 页面
func (en *Engine) PageName(name string) string {
	return en.PageNameWithOptions(name, en.opts)
}

// PageNameWithOptions 页面（带选项）
func (en *Engine) PageNameWithOptions(name string, opts Options) string {
	return fmt.Sprintf("%s:pages/%s", opts.Layout, name)
}

// SingleName 单页
func (en *Engine) SingleName(name string) string {
	return en.SingleNameWithOptions(name, en.opts)
}

// SingleNameWithOptions 单页（带选项）
func (en *Engine) SingleNameWithOptions(name string, opts Options) string {
	return fmt.Sprintf("singles/%s.%s", name, opts.Suffix)
}

// ErrorName 单页
func (en *Engine) ErrorName(name string) string {
	return en.ErrorNameWithOptions(name, en.opts)
}

// ErrorNameWithOptions 错误页面（带选项）
func (en *Engine) ErrorNameWithOptions(name string, opts Options) string {
	return fmt.Sprintf("error/%s.%s", name, opts.Suffix)
}

// RenderPage 渲染页面
func (en *Engine) RenderPage(w io.Writer, name string, data H, opts ...Option) error {
	return en.render(w, name, "page", data, opts...)
}

// RenderSingle 渲染单页面
func (en *Engine) RenderSingle(w io.Writer, name string, data H, opts ...Option) error {
	return en.render(w, name, "single", data, opts...)
}

// RenderError 渲染错误页面
func (en *Engine) RenderError(w io.Writer, name string, data H, opts ...Option) error {
	return en.render(w, name, "error", data, opts...)
}

// render 渲染
func (en *Engine) render(w io.Writer, name, typ string, data H, opts ...Option) error {
	opt := en.opts
	for _, o := range opts {
		o(&opt)
	}

	if data == nil {
		data = H{}
	}
	data["constant"] = opt.GlobalConstant
	data["variable"] = opt.GlobalVariable
	switch typ {
	case "page":
		return en.HTMLRender.Execute(en.PageNameWithOptions(name, opt), w, data)
	case "single":
		return en.HTMLRender.Execute(en.SingleNameWithOptions(name, opt), w, data)
	case "error":
		return en.HTMLRender.Execute(en.ErrorNameWithOptions(name, opt), w, data)
	default:
		return fmt.Errorf("unknown render type: %s", typ)
	}
}
