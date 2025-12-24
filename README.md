# Template Engine

一个功能强大的 Go 模板引擎，支持多主题、热重载、嵌入式文件系统等特性。

## 功能特性

- 🎨 **多主题支持**: 支持多个主题并可在运行时动态切换
- 🔄 **热重载**: 开发时自动监听文件变化并重新加载模板
- 📦 **嵌入式文件系统**: 支持将模板打包到二进制文件中
- 🔧 **灵活配置**: 丰富的配置选项和自定义函数支持
- 🔙 **向后兼容**: 完全兼容现有的单主题项目
- 📁 **分离式模板**: 支持将页面模板拆分为多个文件（header.tmpl、content.tmpl、script.tmpl等）

## 快速开始

### 基本使用（单主题模式）

```go
package main

import (
    "net/http"
    "github.com/nilorg/template"
)

func main() {
    // 创建模板引擎
    engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, nil)
    if err != nil {
        panic(err)
    }
    engine.Init()

    // 渲染页面
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        data := template.H{"title": "Hello World"}
        engine.RenderPage(w, "home", data)
    })

    http.ListenAndServe(":8080", nil)
}
```

### 多主题模式

```go
package main

import (
    "net/http"
    "github.com/nilorg/template"
)

func main() {
    // 创建支持多主题的模板引擎
    engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, nil,
        template.EnableMultiTheme(true),        // 启用多主题模式
        template.DefaultTheme("default"),       // 设置默认主题
    )
    if err != nil {
        panic(err)
    }
    engine.Init()

    // 设置初始主题
    err = engine.SwitchTheme("default")
    if err != nil {
        panic(err)
    }

    // 主题管理
    http.HandleFunc("/themes", func(w http.ResponseWriter, r *http.Request) {
        themes := engine.GetAvailableThemes()
        current := engine.GetCurrentTheme()
        // 渲染主题管理页面...
    })

    // 主题切换
    http.HandleFunc("/switch-theme", func(w http.ResponseWriter, r *http.Request) {
        themeName := r.FormValue("theme")
        err := engine.SwitchTheme(themeName)
        // 处理切换结果...
    })

    http.ListenAndServe(":8080", nil)
}
```

## 目录结构

### 单主题模式（传统结构）
```bash
./templates
├── errors/
│   └── 404.tmpl
├── layouts/
│   ├── layout.tmpl
│   └── pjax_layout.tmpl
├── pages/
│   └── posts/
│       ├── list/
│       │   └── posts.tmpl
│       └── detail/
│           └── posts_detail.tmpl
├── singles/
│   └── login.tmpl
└── partials/
    └── header.tmpl
```

### 多主题模式
```bash
./templates
├── default/                    # 默认主题
│   ├── theme.json             # 主题配置文件
│   ├── layouts/
│   ├── pages/
│   ├── singles/
│   ├── errors/
│   └── partials/
├── dark/                      # 深色主题
│   ├── theme.json
│   ├── layouts/
│   ├── pages/
│   ├── singles/
│   ├── errors/
│   └── partials/
└── colorful/                  # 彩色主题
    ├── theme.json
    ├── layouts/
    ├── pages/
    ├── singles/
    ├── errors/
    └── partials/
```

### 分离式模板结构（推荐）
```bash
./templates/default/pages/posts/list/
├── header.tmpl                # 页面头部和样式
├── content.tmpl              # 主要内容
└── script.tmpl               # JavaScript代码

# 每个文件使用不同的 define 块：
# header.tmpl:  {{ define "header" }}...{{ end }}
# content.tmpl: {{ define "content" }}...{{ end }}
# script.tmpl:  {{ define "script" }}...{{ end }}
```

## 分离式模板详解

### 概念
分离式模板允许您将一个页面的不同部分拆分到多个文件中，每个文件负责特定的功能：

- **header.tmpl**: 页面标题、CSS样式、meta标签等
- **content.tmpl**: 页面主要内容
- **script.tmpl**: JavaScript代码和交互逻辑

### 优势
1. **代码组织**: 将样式、内容、脚本分离，便于维护
2. **团队协作**: 不同开发者可以同时编辑不同部分
3. **复用性**: 可以在不同页面间复用特定部分
4. **主题一致性**: 每个主题可以有独特的样式和交互

### 示例

**header.tmpl**:
```html
{{ define "header" }}
<title>{{ .title }} - {{ .constant.siteName }}</title>
<style>
    body { font-family: Arial, sans-serif; }
    .container { max-width: 1200px; margin: 0 auto; }
</style>
{{ end }}
```

**content.tmpl**:
```html
{{ define "content" }}
<h1>{{ .title }}</h1>
<div class="posts-list">
    {{ range .posts }}
    <article>
        <h2>{{ .title }}</h2>
        <p>{{ .summary }}</p>
    </article>
    {{ end }}
</div>
{{ end }}
```

**script.tmpl**:
```html
{{ define "script" }}
<script>
document.addEventListener('DOMContentLoaded', function() {
    console.log('页面已加载完成');
});
</script>
{{ end }}
```

## API 文档

### 创建引擎

#### NewEngine
```go
func NewEngine(templatesDir string, loadFunc LoadTemplateFunc, funcMap FuncMap, opts ...Option) (*Engine, error)
```

创建新的模板引擎实例。

**参数:**
- `templatesDir`: 模板目录路径
- `loadFunc`: 模板加载函数
- `funcMap`: 自定义模板函数映射
- `opts`: 配置选项

**示例:**
```go
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
    template.EnableMultiTheme(true),
    template.DefaultTheme("default"),
)
```

#### NewEngineWithEmbedFS
```go
func NewEngineWithEmbedFS(tmplFS *embed.FS, subDir string, loadFunc LoadEmbedFSTemplateFunc, funcMap FuncMap, opts ...Option) (*Engine, error)
```

使用嵌入式文件系统创建模板引擎。

**参数:**
- `tmplFS`: 嵌入式文件系统
- `subDir`: 模板子目录
- `loadFunc`: 嵌入式文件系统加载函数
- `funcMap`: 自定义模板函数映射
- `opts`: 配置选项

### 配置选项

#### 多主题相关选项

```go
// 启用多主题模式
func EnableMultiTheme(enable bool) Option

// 设置默认主题
func DefaultTheme(themeName string) Option
```

#### 其他选项

```go
// 设置全局常量
func GlobalConstant(constant map[string]interface{}) Option

// 设置全局变量
func GlobalVariable(variable map[string]interface{}) Option
```

### 引擎方法

#### 基本方法

```go
// 初始化引擎
func (e *Engine) Init()

// 启动文件监听
func (e *Engine) Watching() error

// 关闭引擎
func (e *Engine) Close()
```

#### 渲染方法

```go
// 渲染页面模板
func (e *Engine) RenderPage(w io.Writer, name string, data interface{}) error

// 渲染单页模板
func (e *Engine) RenderSingle(w io.Writer, name string, data interface{}) error

// 渲染错误页面
func (e *Engine) RenderError(w io.Writer, name string, data interface{}) error
```

#### 多主题方法

```go
// 获取可用主题列表
func (e *Engine) GetAvailableThemes() []string

// 获取当前主题名称
func (e *Engine) GetCurrentTheme() string

// 切换主题
func (e *Engine) SwitchTheme(themeName string) error
```

## 主题配置

### theme.json 配置文件

每个主题可以包含一个 `theme.json` 配置文件来描述主题信息：

```json
{
    "name": "default",
    "displayName": "默认主题",
    "description": "简洁的默认主题，适合日常使用",
    "version": "1.0.0",
    "author": "开发团队",
    "tags": ["default", "clean", "simple"],
    "custom": {
        "primaryColor": "#2c3e50",
        "accentColor": "#3498db",
        "backgroundColor": "#ffffff"
    }
}
```

### 主题结构要求

每个主题目录必须包含以下子目录：
- `layouts/` - 布局模板（必需）
- `pages/` - 页面模板（必需）
- `singles/` - 单页模板（必需）
- `errors/` - 错误页面模板（必需）
- `partials/` - 部分模板（可选）

## 示例项目

### 基本示例
查看 `example/` 目录了解基本的单主题使用方法。

### 多主题示例
查看 `example-multi-theme/` 目录了解完整的多主题功能演示，包括：
- 三个不同风格的主题（默认、深色、彩色）
- 主题管理界面
- 运行时主题切换
- 主题配置文件使用
- 分离式模板结构演示

运行多主题示例：
```bash
cd example-multi-theme
go run main.go
```

访问 http://localhost:8080/themes 体验主题切换功能。

## 向后兼容性保证

### 完全兼容现有项目

多主题功能采用**渐进式增强**设计，确保现有项目无需任何修改即可继续使用：

#### 1. API 完全兼容
```go
// 现有代码无需修改，继续正常工作
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap)
if err != nil {
    panic(err)
}
engine.Init()

// 所有现有方法保持不变
engine.RenderPage(w, "home", data)
engine.RenderSingle(w, "login", data)
engine.RenderError(w, "404", data)
```

#### 2. 目录结构兼容
```bash
# 现有的传统结构继续工作
./templates
├── layouts/
├── pages/
├── singles/
├── errors/
└── partials/
```

#### 3. 配置选项兼容
```go
// 现有的所有配置选项继续工作
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
    template.GlobalConstant(map[string]interface{}{
        "siteName": "My Site",
    }),
    template.GlobalVariable(map[string]interface{}{
        "year": time.Now().Year(),
    }),
    // 新的多主题选项是可选的
    template.EnableMultiTheme(true),  // 可选
    template.DefaultTheme("default"), // 可选
)
```

#### 4. 自动模式检测
系统会自动检测目录结构：
- **传统模式**: 当 `templates/` 直接包含 `layouts/`、`pages/` 等目录时
- **多主题模式**: 当 `templates/` 包含主题子目录时
- **混合模式**: 支持传统结构与主题目录共存

#### 5. 性能保证
- 传统模式下性能与原版本完全相同
- 多主题模式仅在需要时加载额外功能
- 内存使用保持在合理范围内

### 升级路径

#### 零修改升级
```go
// 步骤1: 更新依赖（无代码修改）
go get github.com/nilorg/template@latest

// 步骤2: 现有代码继续工作
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap)
// 完全相同的行为，零修改
```

#### 渐进式启用多主题
```go
// 步骤1: 启用多主题模式（保持现有模板）
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
    template.EnableMultiTheme(true),
)

// 步骤2: 逐步迁移模板到主题目录
// 现有模板会被自动识别为默认主题
```

## 迁移指南

### 从单主题迁移到多主题

1. **保持现有结构**（可选）
   ```bash
   # 现有项目无需修改，自动识别为传统模式
   ./templates
   ├── layouts/
   ├── pages/
   └── ...
   ```

2. **创建多主题结构**
   ```bash
   # 将现有模板移动到默认主题目录
   mkdir templates/default
   mv templates/layouts templates/default/
   mv templates/pages templates/default/
   # ... 移动其他目录
   ```

3. **更新代码**
   ```go
   // 添加多主题选项
   engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
       template.EnableMultiTheme(true),  // 新增
       template.DefaultTheme("default"), // 新增
   )
   ```

### 迁移到分离式模板

1. **识别可拆分的部分**
   - CSS样式 → header.tmpl
   - 主要内容 → content.tmpl  
   - JavaScript → script.tmpl

2. **拆分现有模板**
   ```bash
   # 原文件: posts.tmpl
   # 拆分为:
   ├── header.tmpl
   ├── content.tmpl
   └── script.tmpl
   ```

3. **更新布局模板**
   ```html
   <!-- 在 layout.tmpl 中添加 script 块 -->
   {{ template "script" . }}
   </body>
   ```

### API 兼容性详细说明

#### 核心方法保持不变
```go
// 渲染方法签名完全相同
func (e *Engine) RenderPage(w io.Writer, name string, data interface{}) error
func (e *Engine) RenderSingle(w io.Writer, name string, data interface{}) error  
func (e *Engine) RenderError(w io.Writer, name string, data interface{}) error

// 引擎管理方法保持不变
func (e *Engine) Init()
func (e *Engine) Close()
func (e *Engine) Watching() error
```

#### 配置选项向后兼容
```go
// 所有现有选项继续工作
template.GlobalConstant(map[string]interface{}{})  // ✅ 兼容
template.GlobalVariable(map[string]interface{}{})  // ✅ 兼容

// 新增选项是可选的
template.EnableMultiTheme(true)    // 新增，可选
template.DefaultTheme("default")   // 新增，可选
```

#### 模板语法完全相同
```html
<!-- 现有模板语法无需修改 -->
{{ .title }}
{{ range .items }}
{{ template "partial" . }}
{{ end }}
```

#### 错误处理兼容
- 所有现有错误类型保持不变
- 新增的主题相关错误不影响现有逻辑
- 错误消息格式保持一致

#### 性能特征保证
- 传统模式下的性能与原版本相同
- 内存使用模式保持一致
- 模板加载时间不受影响

## 最佳实践

### 主题设计

1. **保持一致的结构**: 所有主题应该有相同的模板文件结构
2. **使用主题配置**: 利用 `theme.json` 描述主题特性
3. **响应式设计**: 确保主题在不同设备上都能正常显示
4. **性能考虑**: 避免在主题中使用过多的外部资源

### 分离式模板设计

1. **职责分离**: 
   - header.tmpl: 只包含样式和meta信息
   - content.tmpl: 只包含页面内容结构
   - script.tmpl: 只包含JavaScript逻辑

2. **命名规范**:
   - 使用描述性的文件名
   - 保持各主题间文件名一致
   - 使用标准的define块名称

3. **依赖管理**:
   - 避免模板间的强依赖
   - 使用全局变量传递共享数据
   - 保持模板的独立性

### 开发建议

1. **使用热重载**: 开发时启用 `Watching()` 功能
2. **错误处理**: 妥善处理主题切换失败的情况
3. **测试覆盖**: 为每个主题编写测试用例
4. **文档维护**: 为自定义主题编写使用文档

## 故障排除

### 常见问题

1. **主题切换失败**
   - 检查主题目录结构是否完整
   - 确认所有必需的模板文件都存在
   - 查看错误日志获取详细信息

2. **模板加载错误**
   - 验证模板语法是否正确
   - 检查文件路径和权限
   - 确认嵌入式文件系统路径正确

3. **分离式模板问题**
   - 确保所有define块都有对应的模板文件
   - 检查布局模板是否包含所有必需的template调用
   - 验证模板函数是否正确定义

4. **性能问题**
   - 避免频繁切换主题
   - 使用适当的缓存策略
   - 监控内存使用情况

### 调试技巧

```go
// 启用详细日志
engine.SetDebugMode(true)

// 检查主题状态
themes := engine.GetAvailableThemes()
current := engine.GetCurrentTheme()
log.Printf("Available themes: %v, Current: %s", themes, current)

// 检查模板加载
log.Printf("Template loaded: %v", engine.IsTemplateLoaded("posts/list"))
```

## 贡献指南

欢迎提交 Issue 和 Pull Request！

## 许可证

MIT License
