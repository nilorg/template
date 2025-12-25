package template

import (
	"embed"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

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
	done                chan struct{}
	loadTemplateFunc    LoadTemplateFunc
	loadTemplateEmbedFS LoadEmbedFSTemplateFunc
	FuncMap             FuncMap
	HTMLRender          Render
	opts                Options

	// 主题管理相关字段
	themeManager   ThemeManager // 主题管理器
	currentTheme   string       // 当前激活的主题名称
	multiThemeMode bool         // 是否启用多主题模式
}

// NewEngine 创建一个gin引擎模板
func NewEngine(templateDir string, tmplFunc LoadTemplateFunc, funcMap FuncMap, opts ...Option) (*Engine, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	options := newOptions(opts...)

	engine := &Engine{
		templatesDir:     templateDir,
		loadTemplateFunc: tmplFunc,
		watcher:          watcher,
		done:             make(chan struct{}),
		FuncMap:          funcMap,
		opts:             options,
		// 初始化主题相关字段
		currentTheme:   options.Theme,
		multiThemeMode: options.MultiThemeMode,
	}

	return engine, nil
}

func NewEngineWithEmbedFS(tmplFS *embed.FS, tmplFSSUbDir string, tmplFunc LoadEmbedFSTemplateFunc, funcMap FuncMap, opts ...Option) (*Engine, error) {
	options := newOptions(opts...)

	engine := &Engine{
		tmplFS:              tmplFS,
		tmplFSSUbDir:        tmplFSSUbDir,
		loadTemplateEmbedFS: tmplFunc,
		FuncMap:             funcMap,
		opts:                options,
		// 初始化主题相关字段
		currentTheme:   options.Theme,
		multiThemeMode: options.MultiThemeMode,
	}

	return engine, nil
}

// Init 初始化
func (en *Engine) Init() {
	// 初始化主题管理器
	if err := en.initThemeManager(); err != nil {
		// 如果主题管理器初始化失败，回退到传统模式
		en.HTMLRender = en.loadTemplate()
		return
	}

	// 尝试使用主题管理器加载模板
	if en.themeManager != nil {
		render := en.themeManager.GetRender()
		if render != nil {
			en.HTMLRender = render
			return
		}
	}

	// 如果主题管理器不可用，使用传统方式加载
	en.HTMLRender = en.loadTemplate()
}

// initThemeManager 初始化主题管理器
func (en *Engine) initThemeManager() error {
	// 创建主题管理器
	var themeManager ThemeManager

	if en.tmplFS != nil {
		// 嵌入式文件系统模式
		themeManager = NewDefaultThemeManagerWithEmbedFS(
			en.tmplFS,
			en.tmplFSSUbDir,
			en.FuncMap,
			en.loadTemplateEmbedFS,
		)
	} else {
		// 文件系统模式
		themeManager = NewDefaultThemeManager(
			en.templatesDir,
			en.FuncMap,
			en.loadTemplateFunc,
		)
	}

	// 发现主题
	if err := themeManager.DiscoverThemes(); err != nil {
		// 主题发现失败，但不应该阻止引擎初始化
		return err
	}

	// 设置主题管理器
	en.themeManager = themeManager

	// 确定要使用的主题
	targetTheme := en.determineTargetTheme()

	// 如果指定了特定主题，尝试切换到该主题
	if targetTheme != "" {
		if err := themeManager.SwitchTheme(targetTheme); err != nil {
			// 切换失败，但不阻止初始化
			return err
		}
	} else {
		// 如果没有指定主题，加载当前主题
		currentTheme := themeManager.GetCurrentTheme()
		if currentTheme != "" {
			if _, err := themeManager.LoadTheme(currentTheme); err != nil {
				return err
			}
		}
	}

	// 更新当前主题状态
	en.currentTheme = themeManager.GetCurrentTheme()

	return nil
}

// determineTargetTheme 确定目标主题
func (en *Engine) determineTargetTheme() string {
	// 1. 优先使用选项中指定的主题
	if en.opts.Theme != "" {
		return en.opts.Theme
	}

	// 2. 使用引擎字段中的当前主题
	if en.currentTheme != "" {
		return en.currentTheme
	}

	// 3. 使用选项中的默认主题
	if en.opts.DefaultTheme != "" {
		return en.opts.DefaultTheme
	}

	// 4. 让主题管理器自动选择
	return ""
}

// Watching 监听模板文件夹中是否有变动
func (en *Engine) Watching() error {
	// 检查基本条件
	if en.watcher == nil {
		return fmt.Errorf("watcher is nil")
	}

	// 对于嵌入式文件系统，不支持文件监听
	if en.tmplFS != nil {
		return fmt.Errorf("file watching not supported for embedded filesystem")
	}

	if en.templatesDir == "" {
		return fmt.Errorf("template directory is empty")
	}

	en.Errors = en.watcher.Errors

	// 启动文件监听协程
	go func() {
		for {
			select {
			case <-en.done:
				return
			case event, ok := <-en.watcher.Events:
				if !ok {
					return
				}
				en.handleFileEvent(event)
			}
		}
	}()

	// 设置监听目录
	return en.setupWatching()
}

// handleFileEvent 处理文件系统事件
func (en *Engine) handleFileEvent(event fsnotify.Event) {
	shouldReload := false

	switch event.Op {
	case fsnotify.Create:
		// 处理新创建的文件或目录
		if fileInfo, err := os.Stat(event.Name); err == nil {
			if fileInfo.IsDir() {
				// 新目录：添加到监听器
				en.watcher.Add(event.Name)
			} else {
				// 新文件：检查是否需要重载
				shouldReload = en.shouldReloadForFile(event.Name)
			}
		}

	case fsnotify.Write:
		// 文件内容修改：检查是否需要重载
		shouldReload = en.shouldReloadForFile(event.Name)

	case fsnotify.Remove, fsnotify.Rename:
		// 文件或目录删除/重命名
		if fileInfo, err := os.Stat(event.Name); err == nil && fileInfo.IsDir() {
			// 目录删除：从监听器移除
			en.watcher.Remove(event.Name)
		}
		// 对于文件删除，也需要重载模板
		shouldReload = en.shouldReloadForFile(event.Name)

	case fsnotify.Chmod:
		// 权限变更通常不需要重载模板
		shouldReload = false
	}

	// 如果需要重载，执行重载操作
	if shouldReload {
		en.reloadTemplates()
	}
}

// setupWatching 设置文件监听
func (en *Engine) setupWatching() error {
	// 确定要监听的目录
	watchDir := en.getWatchDirectory()

	// 验证监听目录
	if err := en.validateWatchDirectory(watchDir); err != nil {
		return fmt.Errorf("cannot setup watching: %w", err)
	}

	// 遍历目录下的所有子目录并添加到监听器
	err := filepath.Walk(watchDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if err := en.watcher.Add(path); err != nil {
				return fmt.Errorf("failed to add watch path %s: %w", path, err)
			}
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("failed to setup watching for directory %s: %w", watchDir, err)
	}

	return nil
}

// getWatchDirectory 获取要监听的目录
func (en *Engine) getWatchDirectory() string {
	// 如果有主题管理器且当前主题不为空，监听当前主题目录
	if en.themeManager != nil && en.currentTheme != "" {
		if theme, err := en.themeManager.LoadTheme(en.currentTheme); err == nil {
			// 只有非嵌入式主题才需要文件监听
			if !theme.IsEmbedded && theme.Path != "" {
				// 验证主题路径是否存在
				if info, err := os.Stat(theme.Path); err == nil && info.IsDir() {
					return theme.Path
				}
			}
		}
	}

	// 否则监听基础模板目录
	return en.templatesDir
}

// getWatchDirectories 获取所有需要监听的目录（用于多主题模式）
func (en *Engine) getWatchDirectories() []string {
	var directories []string

	// 如果是多主题模式，可能需要监听多个目录
	if en.IsMultiThemeMode() && en.themeManager != nil {
		// 当前只监听活跃主题的目录
		watchDir := en.getWatchDirectory()
		if watchDir != "" {
			directories = append(directories, watchDir)
		}
	} else {
		// 传统模式，监听基础目录
		if en.templatesDir != "" {
			directories = append(directories, en.templatesDir)
		}
	}

	return directories
}

// validateWatchDirectory 验证监听目录是否有效
func (en *Engine) validateWatchDirectory(dir string) error {
	if dir == "" {
		return fmt.Errorf("watch directory is empty")
	}

	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("watch directory does not exist: %s", dir)
	}

	if !info.IsDir() {
		return fmt.Errorf("watch path is not a directory: %s", dir)
	}

	return nil
}

// reloadTemplates 重新加载模板
func (en *Engine) reloadTemplates() {
	// 如果有主题管理器，使用主题管理器重新加载
	if en.themeManager != nil {
		if err := en.reloadCurrentThemeTemplates(); err != nil {
			// 如果主题重载失败，尝试回退到传统方式
			en.HTMLRender = en.loadTemplate()
		}
		return
	}

	// 否则使用传统方式重新加载
	en.HTMLRender = en.loadTemplate()
}

// reloadCurrentThemeTemplates 重新加载当前主题的模板
func (en *Engine) reloadCurrentThemeTemplates() error {
	if en.themeManager == nil {
		return fmt.Errorf("theme manager not available")
	}

	// 重新加载当前主题
	if err := en.themeManager.ReloadCurrentTheme(); err != nil {
		return fmt.Errorf("failed to reload current theme: %w", err)
	}

	// 更新渲染器
	if render := en.themeManager.GetRender(); render != nil {
		en.HTMLRender = render
		return nil
	}

	return fmt.Errorf("failed to get render after theme reload")
}

// isFileInCurrentTheme 检查文件是否属于当前主题
func (en *Engine) isFileInCurrentTheme(filePath string) bool {
	if en.themeManager == nil || en.currentTheme == "" {
		// 传统模式下，所有模板目录下的文件都属于当前"主题"
		return strings.HasPrefix(filePath, en.templatesDir)
	}

	// 获取当前主题路径
	currentThemePath := en.getWatchDirectory()
	if currentThemePath == "" {
		return false
	}

	// 检查文件是否在当前主题目录下
	absThemePath, err := filepath.Abs(currentThemePath)
	if err != nil {
		return false
	}

	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		return false
	}

	return strings.HasPrefix(absFilePath, absThemePath)
}

// shouldReloadForFile 判断文件变化是否应该触发重载
func (en *Engine) shouldReloadForFile(filePath string) bool {
	// 检查文件是否属于当前主题
	if !en.isFileInCurrentTheme(filePath) {
		return false
	}

	// 检查文件扩展名是否为模板文件
	return en.isTemplateFile(filePath)
}

// isTemplateFile 检查文件是否为模板文件
func (en *Engine) isTemplateFile(filename string) bool {
	ext := filepath.Ext(filename)
	templateExts := []string{".tmpl", ".html", ".gohtml", ".tpl"}

	for _, validExt := range templateExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// Close 关闭
func (en *Engine) Close() error {
	if en.done != nil {
		close(en.done)
	}
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
	// 单页(singles)不使用布局，但在多主题模式下需要区分不同主题的单页
	// 单页的核心设计理念：自包含的完整HTML页面，不依赖布局
	return fmt.Sprintf("singles/%s.%s", name, opts.Suffix)
}

// ErrorName 单页
func (en *Engine) ErrorName(name string) string {
	return en.ErrorNameWithOptions(name, en.opts)
}

// ErrorNameWithOptions 错误页面（带选项）
func (en *Engine) ErrorNameWithOptions(name string, opts Options) string {
	// 首先尝试新的分割模板格式（使用布局）
	if en.multiThemeMode {
		// 对于错误页面，首先尝试error.tmpl布局
		splitTemplateName := fmt.Sprintf("error.tmpl:error/%s", name)
		if en.HTMLRender.HasTemplate(splitTemplateName) {
			return splitTemplateName
		}
		// 如果没有专用错误布局，尝试单页布局
		splitTemplateName = fmt.Sprintf("single.tmpl:error/%s", name)
		if en.HTMLRender.HasTemplate(splitTemplateName) {
			return splitTemplateName
		}
	}
	// 回退到传统格式
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

// 主题相关的公共API方法

// GetAvailableThemes 获取所有可用主题列表
func (en *Engine) GetAvailableThemes() []string {
	if en.themeManager == nil {
		// 传统模式下返回默认主题
		return []string{"default"}
	}
	return en.themeManager.GetAvailableThemes()
}

// GetCurrentTheme 获取当前激活的主题名称
func (en *Engine) GetCurrentTheme() string {
	if en.themeManager == nil {
		// 传统模式下返回默认主题
		return "default"
	}
	return en.themeManager.GetCurrentTheme()
}

// SwitchTheme 切换到指定主题
func (en *Engine) SwitchTheme(themeName string) error {
	if en.themeManager == nil {
		return &ThemeError{
			Type:    ErrThemeSwitchFailed,
			Theme:   themeName,
			Message: "theme manager not available (legacy mode)",
		}
	}

	// 执行主题切换
	if err := en.themeManager.SwitchTheme(themeName); err != nil {
		return err
	}

	// 更新引擎状态
	en.currentTheme = en.themeManager.GetCurrentTheme()

	// 更新渲染器
	if render := en.themeManager.GetRender(); render != nil {
		en.HTMLRender = render
	}

	// 如果有文件监听器且不是嵌入式文件系统，更新监听目录
	if en.watcher != nil && en.tmplFS == nil {
		// 重新设置文件监听器以监听新主题目录
		if err := en.updateWatcherForTheme(); err != nil {
			// 监听器更新失败不应该阻止主题切换
			// 只记录错误但不返回失败
			// 可以考虑添加日志记录
		}
	}

	return nil
}

// updateWatcherForTheme 为主题切换更新监听器
func (en *Engine) updateWatcherForTheme() error {
	if en.watcher == nil {
		return nil
	}

	// 对于嵌入式文件系统，不需要更新监听器
	if en.tmplFS != nil {
		return nil
	}

	// 获取新的监听目录
	newWatchDir := en.getWatchDirectory()

	// 验证新的监听目录
	if err := en.validateWatchDirectory(newWatchDir); err != nil {
		return fmt.Errorf("invalid watch directory after theme switch: %w", err)
	}

	// 重新创建监听器并设置新的监听路径
	return en.updateWatcher()
}

// ThemeExists 检查指定主题是否存在
func (en *Engine) ThemeExists(themeName string) bool {
	if en.themeManager == nil {
		// 传统模式下只有默认主题存在
		return themeName == "default"
	}
	return en.themeManager.ThemeExists(themeName)
}

// GetThemeMetadata 获取指定主题的元数据
func (en *Engine) GetThemeMetadata(themeName string) (*ThemeMetadata, error) {
	if en.themeManager == nil {
		if themeName == "default" {
			// 返回默认的传统模式元数据
			return &ThemeMetadata{
				DisplayName: "Default Theme",
				Description: "Legacy mode default theme",
				Version:     "1.0.0",
				Author:      "System",
				Tags:        []string{"legacy", "default"},
				Custom:      make(map[string]any),
			}, nil
		}
		return nil, &ThemeError{
			Type:    ErrThemeNotFound,
			Theme:   themeName,
			Message: "theme not found in legacy mode",
		}
	}
	return en.themeManager.GetThemeMetadata(themeName)
}

// IsMultiThemeMode 检查是否启用了多主题模式
func (en *Engine) IsMultiThemeMode() bool {
	return en.multiThemeMode && en.themeManager != nil
}

// ReloadCurrentTheme 重新加载当前主题
func (en *Engine) ReloadCurrentTheme() error {
	if en.themeManager == nil {
		// 传统模式下重新加载模板
		en.HTMLRender = en.loadTemplate()
		return nil
	}

	return en.reloadCurrentThemeTemplates()
}

// GetWatchedDirectories 获取当前正在监听的目录信息
func (en *Engine) GetWatchedDirectories() []string {
	if en.watcher == nil {
		return []string{}
	}

	// 由于fsnotify不提供获取监听路径的方法，
	// 我们返回当前应该监听的目录
	return en.getWatchDirectories()
}

// RestartWatching 重启文件监听（用于调试和故障恢复）
func (en *Engine) RestartWatching() error {
	if en.watcher == nil {
		return fmt.Errorf("watcher not initialized")
	}

	// 对于嵌入式文件系统，不支持文件监听
	if en.tmplFS != nil {
		return fmt.Errorf("file watching not supported for embedded filesystem")
	}

	// 清理现有监听器
	if err := en.clearWatcher(); err != nil {
		return fmt.Errorf("failed to clear watcher: %w", err)
	}

	// 重新启动监听
	return en.Watching()
}

// GetThemeManager 获取主题管理器（用于高级操作）
func (en *Engine) GetThemeManager() ThemeManager {
	return en.themeManager
}

// updateWatcher 更新文件监听器
func (en *Engine) updateWatcher() error {
	if en.watcher == nil {
		return nil
	}

	// 对于嵌入式文件系统，不需要更新监听器
	if en.tmplFS != nil {
		return nil
	}

	// 移除所有现有的监听路径
	// 注意：fsnotify.Watcher 没有直接的方法来列出所有监听的路径
	// 所以我们需要重新创建监听器

	// 关闭现有监听器
	if err := en.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close existing watcher: %w", err)
	}

	// 创建新的监听器
	newWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create new watcher: %w", err)
	}

	en.watcher = newWatcher
	en.Errors = en.watcher.Errors

	// 重新设置监听
	return en.setupWatching()
}

// clearWatcher 清理监听器的所有路径
func (en *Engine) clearWatcher() error {
	if en.watcher == nil {
		return nil
	}

	// 由于fsnotify没有提供列出所有监听路径的方法，
	// 我们通过重新创建监听器来清理所有路径
	if err := en.watcher.Close(); err != nil {
		return fmt.Errorf("failed to close watcher: %w", err)
	}

	// 创建新的空监听器
	newWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return fmt.Errorf("failed to create new watcher: %w", err)
	}

	en.watcher = newWatcher
	en.Errors = en.watcher.Errors

	return nil
}

// isWatchingActive 检查文件监听是否处于活跃状态
func (en *Engine) isWatchingActive() bool {
	return en.watcher != nil && en.done != nil
}
