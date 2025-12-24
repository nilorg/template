# 迁移指南

本指南帮助您将现有的单主题项目迁移到支持多主题的新版本。

## 版本兼容性

### 完全向后兼容

新版本的多主题功能完全向后兼容，您的现有代码无需任何修改即可继续工作：

```go
// 现有代码保持不变
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap)
if err != nil {
    panic(err)
}
engine.Init()
```

系统会自动检测您的模板目录结构：
- 如果是传统结构（layouts、pages 等直接在 templates 下），自动使用传统模式
- 如果包含主题子目录，自动启用多主题模式

## 迁移策略

### 策略 1: 保持现有结构（推荐）

**适用场景**: 暂时不需要多主题功能，但希望升级到新版本

**操作步骤**: 无需任何操作，直接升级即可

```bash
# 现有目录结构保持不变
./templates
├── layouts/
│   └── layout.tmpl
├── pages/
│   └── home.tmpl
├── singles/
│   └── login.tmpl
├── errors/
│   └── 404.tmpl
└── partials/
    └── header.tmpl
```

**代码变更**: 无需修改任何代码

### 策略 2: 渐进式迁移

**适用场景**: 希望逐步引入多主题功能

**第一步**: 将现有模板移动到默认主题目录

```bash
# 创建默认主题目录
mkdir templates/default

# 移动现有模板
mv templates/layouts templates/default/
mv templates/pages templates/default/
mv templates/singles templates/default/
mv templates/errors templates/default/
mv templates/partials templates/default/
```

**第二步**: 启用多主题模式

```go
// 更新代码以启用多主题
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
    template.EnableMultiTheme(true),        // 启用多主题
    template.DefaultTheme("default"),       // 设置默认主题
    template.Theme("default"),              // 使用默认主题
)
```

**第三步**: 添加主题配置文件（可选）

```bash
# 创建主题配置文件
cat > templates/default/theme.json << EOF
{
    "name": "default",
    "displayName": "默认主题",
    "description": "从单主题迁移的默认主题",
    "version": "1.0.0",
    "author": "您的名字",
    "tags": ["default", "migrated"]
}
EOF
```

### 策略 3: 完全迁移到多主题

**适用场景**: 立即使用多主题功能

**操作步骤**:

1. **重构目录结构**:
```bash
# 备份现有模板
cp -r templates templates_backup

# 创建多主题结构
mkdir -p templates/{default,dark}

# 移动现有模板到默认主题
mv templates_backup/layouts templates/default/
mv templates_backup/pages templates/default/
mv templates_backup/singles templates/default/
mv templates_backup/errors templates/default/
mv templates_backup/partials templates/default/

# 复制到其他主题（后续自定义）
cp -r templates/default/* templates/dark/
```

2. **更新应用代码**:
```go
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
    template.EnableMultiTheme(true),
    template.DefaultTheme("default"),
    template.Theme("default"),
    template.GlobalConstant(map[string]interface{}{
        "siteName": "我的网站",
        "version": "2.0.0",
    }),
)

// 添加主题管理路由
http.HandleFunc("/themes", themesHandler)
http.HandleFunc("/switch-theme", switchThemeHandler)
```

3. **创建主题管理功能**:
```go
func themesHandler(w http.ResponseWriter, r *http.Request) {
    data := template.H{
        "title": "主题管理",
        "availableThemes": engine.GetAvailableThemes(),
        "currentTheme": engine.GetCurrentTheme(),
    }
    engine.RenderPage(w, "themes", data)
}

func switchThemeHandler(w http.ResponseWriter, r *http.Request) {
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
}
```

## 常见迁移场景

### 场景 1: 企业内部系统

**需求**: 支持不同部门使用不同主题

**迁移方案**:
```bash
templates/
├── corporate/          # 企业主题
├── finance/           # 财务部门主题
├── hr/               # 人事部门主题
└── default/          # 默认主题
```

**实现代码**:
```go
// 根据用户部门设置主题
func setThemeByDepartment(engine *template.Engine, user User) {
    themeMap := map[string]string{
        "finance": "finance",
        "hr": "hr",
        "default": "corporate",
    }
    
    theme := themeMap[user.Department]
    if theme == "" {
        theme = "default"
    }
    
    engine.SwitchTheme(theme)
}
```

### 场景 2: 多租户 SaaS 应用

**需求**: 每个租户可以自定义主题

**迁移方案**:
```bash
templates/
├── tenant_001/        # 租户1主题
├── tenant_002/        # 租户2主题
├── saas_default/      # SaaS默认主题
└── admin/            # 管理后台主题
```

**实现代码**:
```go
// 中间件：根据租户设置主题
func tenantThemeMiddleware(engine *template.Engine) func(http.Handler) http.Handler {
    return func(next http.Handler) http.Handler {
        return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
            tenantID := getTenantID(r) // 从请求中获取租户ID
            themeName := fmt.Sprintf("tenant_%s", tenantID)
            
            // 检查租户主题是否存在
            availableThemes := engine.GetAvailableThemes()
            themeExists := false
            for _, theme := range availableThemes {
                if theme == themeName {
                    themeExists = true
                    break
                }
            }
            
            if themeExists {
                engine.SwitchTheme(themeName)
            } else {
                engine.SwitchTheme("saas_default")
            }
            
            next.ServeHTTP(w, r)
        })
    }
}
```

### 场景 3: 电商网站

**需求**: 支持节日主题、品牌主题等

**迁移方案**:
```bash
templates/
├── default/           # 默认主题
├── christmas/         # 圣诞节主题
├── valentine/         # 情人节主题
├── brand_nike/        # 耐克品牌主题
└── brand_adidas/      # 阿迪达斯品牌主题
```

**实现代码**:
```go
// 自动主题切换
func autoSwitchTheme(engine *template.Engine) {
    now := time.Now()
    
    // 节日主题
    if isChristmasTime(now) {
        engine.SwitchTheme("christmas")
        return
    }
    
    if isValentineTime(now) {
        engine.SwitchTheme("valentine")
        return
    }
    
    // 默认主题
    engine.SwitchTheme("default")
}

func isChristmasTime(t time.Time) bool {
    return t.Month() == 12 && t.Day() >= 20 && t.Day() <= 26
}
```

## 嵌入式文件系统迁移

### 从文件系统迁移到嵌入式

**原有代码**:
```go
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap)
```

**迁移后代码**:
```go
//go:embed templates/*
var templatesFS embed.FS

engine, err := template.NewEngineWithEmbedFS(&templatesFS, "templates",
    template.DefaultLoadEmbedFSTemplate, funcMap,
    template.EnableMultiTheme(true),
    template.Theme("default"),
)
```

### 构建脚本更新

**Makefile 示例**:
```makefile
# 构建嵌入式版本
build-embed:
	go build -tags embed -o app-embed ./cmd/app

# 构建文件系统版本
build-fs:
	go build -o app-fs ./cmd/app

# 开发模式（使用文件系统）
dev:
	go run ./cmd/app

# 生产模式（使用嵌入式）
prod: build-embed
	./app-embed
```

## 测试迁移

### 单元测试更新

**原有测试**:
```go
func TestRender(t *testing.T) {
    engine, err := template.NewEngine("./testdata/templates", 
        template.DefaultLoadTemplate, nil)
    require.NoError(t, err)
    
    // 测试渲染...
}
```

**迁移后测试**:
```go
func TestRenderMultiTheme(t *testing.T) {
    engine, err := template.NewEngine("./testdata/templates", 
        template.DefaultLoadTemplate, nil,
        template.EnableMultiTheme(true),
        template.Theme("default"),
    )
    require.NoError(t, err)
    
    // 测试多主题功能
    themes := engine.GetAvailableThemes()
    assert.Contains(t, themes, "default")
    
    // 测试主题切换
    err = engine.SwitchTheme("dark")
    assert.NoError(t, err)
    assert.Equal(t, "dark", engine.GetCurrentTheme())
}
```

### 集成测试

```go
func TestThemeSwitchingIntegration(t *testing.T) {
    // 设置测试环境
    engine := setupTestEngine(t)
    server := httptest.NewServer(setupRoutes(engine))
    defer server.Close()
    
    // 测试默认主题
    resp, err := http.Get(server.URL + "/")
    require.NoError(t, err)
    body, _ := io.ReadAll(resp.Body)
    assert.Contains(t, string(body), "default")
    
    // 测试主题切换
    form := url.Values{"theme": {"dark"}}
    resp, err = http.PostForm(server.URL + "/switch-theme", form)
    require.NoError(t, err)
    
    // 验证主题已切换
    resp, err = http.Get(server.URL + "/")
    require.NoError(t, err)
    body, _ = io.ReadAll(resp.Body)
    assert.Contains(t, string(body), "dark")
}
```

## 性能考虑

### 内存使用优化

**问题**: 多主题可能增加内存使用

**解决方案**:
```go
// 只在需要时加载主题
engine, err := template.NewEngine("./templates", template.DefaultLoadTemplate, funcMap,
    template.EnableMultiTheme(true),
    template.Theme("default"), // 只加载默认主题
)

// 按需切换主题
func switchThemeOnDemand(engine *template.Engine, themeName string) error {
    current := engine.GetCurrentTheme()
    if current == themeName {
        return nil // 已经是目标主题，无需切换
    }
    
    return engine.SwitchTheme(themeName)
}
```

### 缓存策略

```go
// 主题切换缓存
type ThemeCache struct {
    mu     sync.RWMutex
    themes map[string]*CachedTheme
}

type CachedTheme struct {
    LastUsed time.Time
    Engine   *template.Engine
}

func (tc *ThemeCache) GetOrCreate(themeName string) (*template.Engine, error) {
    tc.mu.RLock()
    cached, exists := tc.themes[themeName]
    tc.mu.RUnlock()
    
    if exists && time.Since(cached.LastUsed) < 5*time.Minute {
        cached.LastUsed = time.Now()
        return cached.Engine, nil
    }
    
    // 创建新的主题引擎...
}
```

## 故障排除

### 常见问题

1. **主题切换后页面显示异常**
   ```go
   // 检查主题完整性
   func validateTheme(engine *template.Engine, themeName string) error {
       themes := engine.GetAvailableThemes()
       for _, theme := range themes {
           if theme == themeName {
               return nil
           }
       }
       return fmt.Errorf("主题 %s 不存在", themeName)
   }
   ```

2. **嵌入式文件系统路径问题**
   ```go
   // 确保路径正确
   //go:embed templates/*
   var templatesFS embed.FS
   
   // 检查嵌入的文件
   func checkEmbeddedFiles() {
       entries, err := fs.ReadDir(templatesFS, "templates")
       if err != nil {
           log.Fatal("无法读取嵌入的模板目录:", err)
       }
       
       for _, entry := range entries {
           log.Printf("发现文件/目录: %s", entry.Name())
       }
   }
   ```

3. **模板语法错误**
   ```go
   // 添加模板验证
   func validateTemplates(engine *template.Engine) error {
       themes := engine.GetAvailableThemes()
       for _, theme := range themes {
           if err := engine.SwitchTheme(theme); err != nil {
               return fmt.Errorf("主题 %s 验证失败: %v", theme, err)
           }
       }
       return nil
   }
   ```

### 调试技巧

```go
// 启用调试模式
func enableDebugMode(engine *template.Engine) {
    // 记录主题切换
    originalSwitch := engine.SwitchTheme
    engine.SwitchTheme = func(themeName string) error {
        log.Printf("切换主题: %s -> %s", engine.GetCurrentTheme(), themeName)
        err := originalSwitch(themeName)
        if err != nil {
            log.Printf("主题切换失败: %v", err)
        } else {
            log.Printf("主题切换成功")
        }
        return err
    }
}
```

## 回滚计划

如果迁移过程中遇到问题，可以按以下步骤回滚：

1. **代码回滚**:
   ```bash
   git checkout HEAD~1  # 回到上一个版本
   ```

2. **目录结构回滚**:
   ```bash
   # 恢复原始目录结构
   rm -rf templates
   mv templates_backup templates
   ```

3. **依赖版本回滚**:
   ```bash
   go mod edit -require=github.com/nilorg/template@v1.x.x
   go mod tidy
   ```

## 迁移检查清单

- [ ] 备份现有模板文件
- [ ] 测试现有功能在新版本中的兼容性
- [ ] 决定迁移策略（保持现状/渐进式/完全迁移）
- [ ] 更新代码以启用多主题（如需要）
- [ ] 创建主题配置文件
- [ ] 添加主题管理功能（如需要）
- [ ] 更新单元测试和集成测试
- [ ] 性能测试和优化
- [ ] 文档更新
- [ ] 部署和监控

## 获得帮助

如果在迁移过程中遇到问题，可以：

1. 查看 [API 文档](API.md)
2. 参考 [示例项目](example-multi-theme/)
3. 提交 [GitHub Issue](https://github.com/nilorg/template/issues)
4. 查看 [常见问题解答](FAQ.md)