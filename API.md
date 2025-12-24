# API 文档

## 概述

本文档详细描述了模板引擎的所有 API 接口，包括多主题功能的完整使用方法。

## 核心接口

### Engine 结构体

`Engine` 是模板引擎的核心结构体，负责管理模板加载、渲染和主题切换。

```go
type Engine struct {
    // 私有字段...
}
```

### 创建引擎

#### NewEngine

```go
func NewEngine(templatesDir string, loadFunc LoadTemplateFunc, funcMap FuncMap, opts ...Option) (*Engine, error)
```

创建新的模板引擎实例。

**参数:**
- `templatesDir` (string): 模板目录路径
- `loadFunc` (LoadTemplateFunc): 模板加载函数
- `funcMap` (FuncMap): 自定义模板函数映射
- `opts` (...Option): 可变配置选项

**返回值:**
- `*Engine`: 引擎实例
- `error`: 错误信息

**示例:**
```go
// 基本使用
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, nil)

// 带配置选项
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
    template.EnableMultiTheme(true),
    template.DefaultTheme("default"),
    template.Theme("dark"),
    template.GlobalConstant(map[string]interface{}{
        "siteName": "我的网站",
        "version": "1.0.0",
    }),
)
```

#### NewEngineWithEmbedFS

```go
func NewEngineWithEmbedFS(tmplFS *embed.FS, subDir string, loadFunc LoadEmbedFSTemplateFunc, funcMap FuncMap, opts ...Option) (*Engine, error)
```

使用嵌入式文件系统创建模板引擎。

**参数:**
- `tmplFS` (*embed.FS): 嵌入式文件系统
- `subDir` (string): 模板子目录路径
- `loadFunc` (LoadEmbedFSTemplateFunc): 嵌入式文件系统加载函数
- `funcMap` (FuncMap): 自定义模板函数映射
- `opts` (...Option): 可变配置选项

**示例:**
```go
//go:embed templates/*
var templatesFS embed.FS

engine, err := template.NewEngineWithEmbedFS(&templatesFS, "templates", 
    template.DefaultLoadEmbedFSTemplate, funcMap,
    template.EnableMultiTheme(true),
    template.Theme("default"),
)
```

## 配置选项

### 多主题选项

#### EnableMultiTheme

```go
func EnableMultiTheme(enable bool) Option
```

启用或禁用多主题模式。

**参数:**
- `enable` (bool): 是否启用多主题模式

**示例:**
```go
template.EnableMultiTheme(true)  // 启用多主题
template.EnableMultiTheme(false) // 禁用多主题（默认）
```

#### Theme

```go
func Theme(themeName string) Option
```

设置当前使用的主题。

**参数:**
- `themeName` (string): 主题名称

**示例:**
```go
template.Theme("default")  // 使用默认主题
template.Theme("dark")     // 使用深色主题
```

#### DefaultTheme

```go
func DefaultTheme(themeName string) Option
```

设置默认主题，当指定的主题不存在时会回退到此主题。

**参数:**
- `themeName` (string): 默认主题名称

**示例:**
```go
template.DefaultTheme("default")
```

### 全局数据选项

#### GlobalConstant

```go
func GlobalConstant(constant map[string]interface{}) Option
```

设置全局常量，在所有模板中可通过 `{{ .constant.key }}` 访问。

**参数:**
- `constant` (map[string]interface{}): 常量映射

**示例:**
```go
template.GlobalConstant(map[string]interface{}{
    "siteName": "我的网站",
    "version": "1.0.0",
    "author": "开发者",
})
```

#### GlobalVariable

```go
func GlobalVariable(variable map[string]interface{}) Option
```

设置全局变量，在所有模板中可通过 `{{ .variable.key }}` 访问。

**参数:**
- `variable` (map[string]interface{}): 变量映射

**示例:**
```go
template.GlobalVariable(map[string]interface{}{
    "year": time.Now().Year(),
    "timestamp": time.Now().Unix(),
})
```

## 引擎方法

### 生命周期管理

#### Init

```go
func (e *Engine) Init() error
```

初始化模板引擎，加载模板文件。

**返回值:**
- `error`: 初始化错误

**示例:**
```go
if err := engine.Init(); err != nil {
    log.Fatal("引擎初始化失败:", err)
}
```

#### Close

```go
func (e *Engine) Close()
```

关闭模板引擎，清理资源。

**示例:**
```go
defer engine.Close()
```

#### Watching

```go
func (e *Engine) Watching() error
```

启动文件监听功能，自动重载模板文件。

**返回值:**
- `error`: 监听启动错误

**示例:**
```go
if err := engine.Watching(); err != nil {
    log.Fatal("文件监听启动失败:", err)
}

// 监听错误
go func() {
    for err := range engine.Errors {
        log.Printf("文件监听错误: %v", err)
    }
}()
```

### 渲染方法

#### RenderPage

```go
func (e *Engine) RenderPage(w io.Writer, name string, data interface{}) error
```

渲染页面模板。

**参数:**
- `w` (io.Writer): 输出写入器
- `name` (string): 模板名称
- `data` (interface{}): 模板数据

**返回值:**
- `error`: 渲染错误

**示例:**
```go
data := template.H{
    "title": "文章列表",
    "posts": posts,
}
err := engine.RenderPage(w, "posts/list", data)
```

#### RenderSingle

```go
func (e *Engine) RenderSingle(w io.Writer, name string, data interface{}) error
```

渲染单页模板（不使用布局）。

**参数:**
- `w` (io.Writer): 输出写入器
- `name` (string): 模板名称
- `data` (interface{}): 模板数据

**示例:**
```go
data := template.H{"title": "登录"}
err := engine.RenderSingle(w, "login", data)
```

#### RenderError

```go
func (e *Engine) RenderError(w io.Writer, name string, data interface{}) error
```

渲染错误页面模板。

**参数:**
- `w` (io.Writer): 输出写入器
- `name` (string): 错误模板名称（如 "404", "500"）
- `data` (interface{}): 模板数据

**示例:**
```go
data := template.H{
    "title": "页面未找到",
    "message": "请求的页面不存在",
    "path": r.URL.Path,
}
err := engine.RenderError(w, "404", data)
```

### 多主题方法

#### GetAvailableThemes

```go
func (e *Engine) GetAvailableThemes() []string
```

获取所有可用主题的名称列表。

**返回值:**
- `[]string`: 主题名称列表

**示例:**
```go
themes := engine.GetAvailableThemes()
fmt.Printf("可用主题: %v\n", themes)
// 输出: 可用主题: [default dark colorful]
```

#### GetCurrentTheme

```go
func (e *Engine) GetCurrentTheme() string
```

获取当前激活的主题名称。

**返回值:**
- `string`: 当前主题名称

**示例:**
```go
current := engine.GetCurrentTheme()
fmt.Printf("当前主题: %s\n", current)
// 输出: 当前主题: dark
```

#### SwitchTheme

```go
func (e *Engine) SwitchTheme(themeName string) error
```

切换到指定的主题。

**参数:**
- `themeName` (string): 目标主题名称

**返回值:**
- `error`: 切换错误

**示例:**
```go
// 切换到深色主题
if err := engine.SwitchTheme("dark"); err != nil {
    log.Printf("主题切换失败: %v", err)
    return
}
log.Println("主题切换成功")
```

## 数据类型

### H 类型

```go
type H map[string]interface{}
```

`H` 是 `map[string]interface{}` 的别名，用于简化模板数据的创建。

**示例:**
```go
data := template.H{
    "title": "页面标题",
    "user": User{Name: "张三", Age: 25},
    "items": []string{"item1", "item2", "item3"},
}
```

### FuncMap 类型

```go
type FuncMap map[string]interface{}
```

`FuncMap` 用于定义自定义模板函数。

**示例:**
```go
funcMap := template.FuncMap{
    "formatDate": func(t time.Time) string {
        return t.Format("2006-01-02 15:04:05")
    },
    "upper": strings.ToUpper,
    "add": func(a, b int) int {
        return a + b
    },
}
```

## 主题管理

### ThemeManager 接口

```go
type ThemeManager interface {
    DiscoverThemes() error
    LoadTheme(name string) (*Theme, error)
    GetAvailableThemes() []string
    GetCurrentTheme() string
    ThemeExists(name string) bool
    GetThemeMetadata(name string) (*ThemeMetadata, error)
    SwitchTheme(name string) error
    GetRender() Render
    ReloadCurrentTheme() error
}
```

### Theme 结构体

```go
type Theme struct {
    Name       string        `json:"name"`
    Path       string        `json:"path"`
    IsDefault  bool          `json:"is_default"`
    IsEmbedded bool          `json:"is_embedded"`
    Metadata   ThemeMetadata `json:"metadata"`
}
```

### ThemeMetadata 结构体

```go
type ThemeMetadata struct {
    DisplayName string         `json:"display_name"`
    Description string         `json:"description"`
    Version     string         `json:"version"`
    Author      string         `json:"author"`
    Tags        []string       `json:"tags"`
    Custom      map[string]any `json:"custom"`
}
```

## 错误处理

### ThemeError 结构体

```go
type ThemeError struct {
    Type    ThemeErrorType `json:"type"`
    Theme   string         `json:"theme"`
    Message string         `json:"message"`
    Cause   error          `json:"-"`
}
```

### ThemeErrorType 枚举

```go
type ThemeErrorType int

const (
    ErrThemeNotFound ThemeErrorType = iota
    ErrThemeInvalid
    ErrThemeLoadFailed
    ErrThemeSwitchFailed
    ErrThemeConfigInvalid
)
```

**错误类型说明:**
- `ErrThemeNotFound`: 主题不存在
- `ErrThemeInvalid`: 主题结构无效
- `ErrThemeLoadFailed`: 主题加载失败
- `ErrThemeSwitchFailed`: 主题切换失败
- `ErrThemeConfigInvalid`: 主题配置无效

## 使用模式

### 基本 Web 应用

```go
package main

import (
    "net/http"
    "github.com/nilorg/template"
)

func main() {
    engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, nil)
    if err != nil {
        panic(err)
    }
    defer engine.Close()
    
    engine.Init()
    engine.Watching()
    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        data := template.H{"title": "首页"}
        engine.RenderPage(w, "home", data)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

### 多主题 Web 应用

```go
package main

import (
    "net/http"
    "github.com/nilorg/template"
)

func main() {
    engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, nil,
        template.EnableMultiTheme(true),
        template.DefaultTheme("default"),
        template.Theme("default"),
    )
    if err != nil {
        panic(err)
    }
    defer engine.Close()
    
    engine.Init()
    engine.Watching()
    
    // 主题管理
    http.HandleFunc("/themes", func(w http.ResponseWriter, r *http.Request) {
        data := template.H{
            "availableThemes": engine.GetAvailableThemes(),
            "currentTheme": engine.GetCurrentTheme(),
        }
        engine.RenderPage(w, "themes", data)
    })
    
    // 主题切换
    http.HandleFunc("/switch-theme", func(w http.ResponseWriter, r *http.Request) {
        if r.Method != "POST" {
            http.Error(w, "Method not allowed", 405)
            return
        }
        
        themeName := r.FormValue("theme")
        if err := engine.SwitchTheme(themeName); err != nil {
            http.Error(w, err.Error(), 500)
            return
        }
        
        http.Redirect(w, r, "/themes", 302)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

### 嵌入式文件系统

```go
package main

import (
    "embed"
    "net/http"
    "github.com/nilorg/template"
)

//go:embed templates/*
var templatesFS embed.FS

func main() {
    engine, err := template.NewEngineWithEmbedFS(&templatesFS, "templates",
        template.DefaultLoadEmbedFSTemplate, nil,
        template.EnableMultiTheme(true),
        template.Theme("default"),
    )
    if err != nil {
        panic(err)
    }
    defer engine.Close()
    
    engine.Init()
    
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        data := template.H{"title": "嵌入式应用"}
        engine.RenderPage(w, "home", data)
    })
    
    http.ListenAndServe(":8080", nil)
}
```

## 最佳实践

### 错误处理

```go
// 渲染时的错误处理
func renderPage(w http.ResponseWriter, r *http.Request, engine *template.Engine) {
    data := template.H{"title": "页面"}
    
    if err := engine.RenderPage(w, "page", data); err != nil {
        log.Printf("渲染失败: %v", err)
        
        // 渲染错误页面
        errorData := template.H{
            "title": "服务器错误",
            "message": "页面渲染失败",
        }
        if renderErr := engine.RenderError(w, "500", errorData); renderErr != nil {
            http.Error(w, "Internal Server Error", 500)
        }
    }
}
```

### 主题切换安全性

```go
func switchTheme(engine *template.Engine, themeName string) error {
    // 验证主题名称
    availableThemes := engine.GetAvailableThemes()
    found := false
    for _, theme := range availableThemes {
        if theme == themeName {
            found = true
            break
        }
    }
    
    if !found {
        return fmt.Errorf("主题 '%s' 不存在", themeName)
    }
    
    // 执行切换
    return engine.SwitchTheme(themeName)
}
```

### 性能优化

```go
// 缓存引擎实例
var engineInstance *template.Engine
var engineOnce sync.Once

func getEngine() *template.Engine {
    engineOnce.Do(func() {
        var err error
        engineInstance, err = template.NewEngine("./templates", 
            template.DefaultLoadTemplate, nil,
            template.EnableMultiTheme(true),
        )
        if err != nil {
            panic(err)
        }
        engineInstance.Init()
    })
    return engineInstance
}
```

## 版本兼容性

### v1.x 到 v2.x 迁移

多主题功能在 v2.0 中引入，完全向后兼容 v1.x 版本：

```go
// v1.x 代码无需修改
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, nil)

// v2.x 新功能（可选）
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, nil,
    template.EnableMultiTheme(true), // 新增选项
)
```

### API 稳定性保证

- 所有 v1.x 的 API 在 v2.x 中保持不变
- 新增的多主题 API 使用独立的方法名
- 配置选项采用可选参数模式，不影响现有代码