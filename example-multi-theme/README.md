# 多主题示例应用

这是一个展示多主题功能的示例应用，演示了如何使用模板引擎的多主题支持功能。

## 功能特性

- 🎨 **多主题支持**: 支持在运行时动态切换主题
- 🔄 **热重载**: 支持主题文件的热重载功能
- 📱 **响应式设计**: 所有主题都支持响应式布局
- 🎯 **主题管理**: 提供可视化的主题管理界面
- 📁 **分离式模板**: 支持将页面模板拆分为多个文件（header.tmpl、content.tmpl、script.tmpl）
- 🧪 **完整测试**: 包含功能测试、性能测试、并发测试

## 可用主题

### 1. 默认主题 (default)
- **设计理念**: 简洁清爽，适合日常使用
- **配色方案**: 蓝色系，专业稳重
- **适用场景**: 商务网站、文档站点

### 2. 深色主题 (dark)
- **设计理念**: 深色背景，护眼设计
- **配色方案**: 深灰色背景 + 紫色强调色
- **适用场景**: 夜间浏览、开发者工具

### 3. 彩色主题 (colorful)
- **设计理念**: 活泼有趣，充满活力
- **配色方案**: 彩虹渐变色
- **适用场景**: 创意网站、儿童应用

## 目录结构

```
example-multi-theme/
├── main.go                    # 主程序文件
├── main_test.go              # 完整的测试套件
├── README.md                  # 说明文档
└── templates/                 # 模板目录
    ├── default/              # 默认主题
    │   ├── theme.json        # 主题配置文件
    │   ├── layouts/          # 布局模板
    │   ├── pages/            # 页面模板
    │   │   └── posts/
    │   │       └── list/     # 分离式模板示例
    │   │           ├── header.tmpl   # 页面头部和样式
    │   │           ├── content.tmpl  # 主要内容
    │   │           └── script.tmpl   # JavaScript代码
    │   ├── singles/          # 单页模板
    │   ├── errors/           # 错误页面模板
    │   └── partials/         # 部分模板
    ├── dark/                 # 深色主题
    │   └── ...               # 相同的目录结构
    └── colorful/             # 彩色主题
        └── ...               # 相同的目录结构
```

## 运行示例

1. **启动应用**:
   ```bash
   cd example-multi-theme
   go run main.go
   ```

2. **访问应用**:
   - 主页: http://localhost:8080
   - 文章列表: http://localhost:8080/posts
   - 主题管理: http://localhost:8080/themes
   - 登录页面: http://localhost:8080/login

3. **切换主题**:
   - 访问 `/themes` 页面
   - 点击"切换到此主题"按钮
   - 主题会立即生效

## 主题配置文件

每个主题都包含一个 `theme.json` 配置文件，用于描述主题的元数据：

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

## API 使用示例

### 创建多主题引擎

```go
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
    template.GlobalConstant(map[string]interface{}{
        "siteName": "多主题示例网站",
        "version":  "2.0.0",
    }),
    // 启用多主题模式
    template.EnableMultiTheme(true),
    // 设置默认主题
    template.DefaultTheme("default"),
    // 指定初始主题
    template.Theme("default"),
)
```

### 主题管理 API

```go
// 获取可用主题列表
themes := engine.GetAvailableThemes()

// 获取当前主题
currentTheme := engine.GetCurrentTheme()

// 切换主题
err := engine.SwitchTheme("dark")
```

## 开发指南

### 创建新主题

1. 在 `templates/` 目录下创建新的主题目录
2. 复制现有主题的目录结构
3. 修改模板文件的样式和内容
4. 创建 `theme.json` 配置文件
5. 重启应用或使用热重载功能

### 主题目录结构要求

每个主题必须包含以下目录：
- `layouts/` - 布局模板
- `pages/` - 页面模板
- `singles/` - 单页模板
- `errors/` - 错误页面模板
- `partials/` - 部分模板（可选）

### 模板变量

所有模板都可以访问以下变量：
- `{{ .currentTheme }}` - 当前主题名称
- `{{ .constant.siteName }}` - 网站名称
- `{{ .constant.version }}` - 版本号
- `{{ .variable.year }}` - 当前年份

## 注意事项

1. **向后兼容**: 多主题功能完全向后兼容，现有的单主题项目无需修改即可使用
2. **性能优化**: 系统只加载当前激活主题的模板，避免内存浪费
3. **错误处理**: 主题切换失败时会自动回滚到之前的主题
4. **文件监听**: 支持主题文件的热重载，开发时修改模板文件会自动生效

## 扩展功能

- 支持主题继承和覆盖
- 支持主题插件系统
- 支持用户自定义主题
- 支持主题商店和在线下载