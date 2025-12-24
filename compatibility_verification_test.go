package template

import (
	"os"
	"path/filepath"
	"testing"
)

// TestFinalCompatibilityVerification 最终兼容性验证测试
func TestFinalCompatibilityVerification(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "final_compatibility_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	funcMap := NewFuncMap()

	t.Run("传统API完全兼容性", func(t *testing.T) {
		// 创建传统结构
		if err := createLegacyStructure(tempDir); err != nil {
			t.Fatalf("Failed to create legacy structure: %v", err)
		}

		// 使用完全传统的API创建引擎
		engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap)
		if err != nil {
			t.Fatalf("Failed to create engine with legacy API: %v", err)
		}

		engine.Init()
		defer engine.Close()

		// 验证所有传统API方法都正常工作
		if engine.HTMLRender == nil {
			t.Errorf("HTMLRender should not be nil")
		}

		// 测试页面名称生成
		pageName := engine.PageName("index")
		expectedPageName := "layout.tmpl:pages/index"
		if pageName != expectedPageName {
			t.Errorf("Expected page name '%s', got '%s'", expectedPageName, pageName)
		}

		// 测试单页名称生成
		singleName := engine.SingleName("about")
		expectedSingleName := "singles/about.tmpl"
		if singleName != expectedSingleName {
			t.Errorf("Expected single name '%s', got '%s'", expectedSingleName, singleName)
		}

		// 测试错误页面名称生成
		errorName := engine.ErrorName("404")
		expectedErrorName := "error/404.tmpl"
		if errorName != expectedErrorName {
			t.Errorf("Expected error name '%s', got '%s'", expectedErrorName, errorName)
		}

		// 测试主题API在传统模式下的行为
		themes := engine.GetAvailableThemes()
		if len(themes) != 1 || themes[0] != "default" {
			t.Errorf("Expected ['default'], got %v", themes)
		}

		currentTheme := engine.GetCurrentTheme()
		if currentTheme != "default" {
			t.Errorf("Expected current theme 'default', got '%s'", currentTheme)
		}

		if !engine.ThemeExists("default") {
			t.Errorf("Default theme should exist")
		}

		if engine.ThemeExists("non-existent") {
			t.Errorf("Non-existent theme should not exist")
		}

		// 测试主题切换在传统模式下应该失败
		err = engine.SwitchTheme("another-theme")
		if err == nil {
			t.Errorf("Expected error when switching theme in legacy mode")
		}

		// 验证多主题模式标志
		if engine.IsMultiThemeMode() {
			t.Errorf("Should not be in multi-theme mode")
		}

		// 测试重新加载功能
		err = engine.ReloadCurrentTheme()
		if err != nil {
			t.Errorf("Failed to reload current theme: %v", err)
		}
	})

	t.Run("传统选项完全兼容性", func(t *testing.T) {
		// 测试所有传统选项的组合
		options := []Option{
			StatusCode(201),
			Layout("custom.tmpl"),
			Suffix("html"),
			GlobalConstant(map[string]interface{}{
				"siteName": "Test Site",
				"version":  "1.0.0",
			}),
			GlobalVariable(map[string]interface{}{
				"year": 2023,
				"user": "test",
			}),
		}

		optEngine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, options...)
		if err != nil {
			t.Fatalf("Failed to create engine with legacy options: %v", err)
		}

		optEngine.Init()
		defer optEngine.Close()

		// 验证选项正确应用
		if optEngine.opts.StatusCode != 201 {
			t.Errorf("Expected StatusCode 201, got %d", optEngine.opts.StatusCode)
		}

		if optEngine.opts.Layout != "custom.tmpl" {
			t.Errorf("Expected Layout 'custom.tmpl', got '%s'", optEngine.opts.Layout)
		}

		if optEngine.opts.Suffix != "html" {
			t.Errorf("Expected Suffix 'html', got '%s'", optEngine.opts.Suffix)
		}

		// 验证全局常量
		if siteName, ok := optEngine.opts.GlobalConstant["siteName"]; !ok || siteName != "Test Site" {
			t.Errorf("Expected siteName 'Test Site', got %v", siteName)
		}

		// 验证全局变量
		if year, ok := optEngine.opts.GlobalVariable["year"]; !ok || year != 2023 {
			t.Errorf("Expected year 2023, got %v", year)
		}

		// 验证API方法使用了正确的选项
		pageName := optEngine.PageName("test")
		expectedPageName := "custom.tmpl:pages/test"
		if pageName != expectedPageName {
			t.Errorf("Expected page name '%s', got '%s'", expectedPageName, pageName)
		}

		singleName := optEngine.SingleName("test")
		expectedSingleName := "singles/test.html"
		if singleName != expectedSingleName {
			t.Errorf("Expected single name '%s', got '%s'", expectedSingleName, singleName)
		}
	})

	t.Run("多主题模式向后兼容性", func(t *testing.T) {
		// 创建多主题目录
		multiDir := filepath.Join(tempDir, "multi")
		themes := []string{"default", "dark"}
		for _, themeName := range themes {
			themeDir := filepath.Join(multiDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				t.Fatalf("Failed to create theme structure for %s: %v", themeName, err)
			}
		}

		// 使用新的多主题API但保持向后兼容
		multiEngine, err := NewEngine(multiDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
		if err != nil {
			t.Fatalf("Failed to create multi-theme engine: %v", err)
		}

		multiEngine.Init()
		defer multiEngine.Close()

		// 验证所有传统API仍然工作
		if multiEngine.HTMLRender == nil {
			t.Errorf("HTMLRender should not be nil in multi-theme mode")
		}

		// 传统的名称生成方法应该仍然工作
		pageName := multiEngine.PageName("index")
		if pageName == "" {
			t.Errorf("PageName should return non-empty string")
		}

		singleName := multiEngine.SingleName("about")
		if singleName == "" {
			t.Errorf("SingleName should return non-empty string")
		}

		errorName := multiEngine.ErrorName("404")
		if errorName == "" {
			t.Errorf("ErrorName should return non-empty string")
		}

		// 新的主题API应该工作
		availableThemes := multiEngine.GetAvailableThemes()
		if len(availableThemes) < 1 {
			t.Errorf("Should have at least one theme")
		}

		currentTheme := multiEngine.GetCurrentTheme()
		if currentTheme == "" {
			t.Errorf("Current theme should not be empty")
		}

		// 验证多主题模式标志
		if !multiEngine.IsMultiThemeMode() {
			t.Errorf("Should be in multi-theme mode")
		}

		// 测试主题切换功能
		for _, themeName := range availableThemes {
			if themeName != currentTheme {
				err := multiEngine.SwitchTheme(themeName)
				if err != nil {
					t.Errorf("Failed to switch to theme %s: %v", themeName, err)
				}

				newCurrentTheme := multiEngine.GetCurrentTheme()
				if newCurrentTheme != themeName {
					t.Errorf("Expected current theme %s, got %s", themeName, newCurrentTheme)
				}
				break
			}
		}
	})

	t.Run("嵌入式文件系统API兼容性", func(t *testing.T) {
		// 测试嵌入式文件系统API的存在性和基本功能
		embedEngine, err := NewEngineWithEmbedFS(nil, "templates", DefaultLoadTemplateWithEmbedFS, funcMap)
		if err != nil {
			t.Errorf("NewEngineWithEmbedFS should not fail: %v", err)
		}

		if embedEngine == nil {
			t.Errorf("Engine should not be nil")
		}

		// 验证嵌入式相关字段
		if embedEngine.loadTemplateEmbedFS == nil {
			t.Errorf("loadTemplateEmbedFS should be set")
		}

		// 测试带选项的嵌入式引擎创建
		options := []Option{
			SetTheme("test-theme"),
			EnableMultiTheme(true),
			DefaultTheme("default"),
		}

		embeddedEngine, err := NewEngineWithEmbedFS(nil, "custom", DefaultLoadTemplateWithEmbedFS, funcMap, options...)
		if err != nil {
			t.Errorf("NewEngineWithEmbedFS with options should not fail: %v", err)
		}

		// 验证选项正确应用
		if embeddedEngine.opts.Theme != "test-theme" {
			t.Errorf("Expected Theme 'test-theme', got '%s'", embeddedEngine.opts.Theme)
		}

		if !embeddedEngine.opts.MultiThemeMode {
			t.Errorf("Expected MultiThemeMode to be true")
		}

		if embeddedEngine.tmplFSSUbDir != "custom" {
			t.Errorf("Expected tmplFSSUbDir 'custom', got '%s'", embeddedEngine.tmplFSSUbDir)
		}
	})
}

// TestExistingExampleCompatibility 测试现有示例的兼容性
func TestExistingExampleCompatibility(t *testing.T) {
	// 验证现有example目录的结构仍然有效
	exampleDir := "example"

	// 检查example目录是否存在
	if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
		t.Skip("Example directory not found, skipping compatibility test")
	}

	// 检查templates目录结构
	templatesDir := filepath.Join(exampleDir, "templates")
	requiredDirs := []string{"layouts", "pages", "singles", "errors"}

	for _, dir := range requiredDirs {
		dirPath := filepath.Join(templatesDir, dir)
		if _, err := os.Stat(dirPath); os.IsNotExist(err) {
			t.Errorf("Required directory %s not found in example", dir)
		}
	}

	// 验证main.go文件存在
	mainFile := filepath.Join(exampleDir, "main.go")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Errorf("main.go not found in example directory")
	}

	t.Logf("Example directory structure is compatible")
}

// TestMultiThemeExampleCompatibility 测试多主题示例的兼容性
func TestMultiThemeExampleCompatibility(t *testing.T) {
	// 验证多主题示例目录的结构
	exampleDir := "example-multi-theme"

	// 检查example-multi-theme目录是否存在
	if _, err := os.Stat(exampleDir); os.IsNotExist(err) {
		t.Skip("Multi-theme example directory not found, skipping compatibility test")
	}

	// 检查templates目录结构
	templatesDir := filepath.Join(exampleDir, "templates")
	expectedThemes := []string{"default", "dark", "colorful"}

	for _, theme := range expectedThemes {
		themeDir := filepath.Join(templatesDir, theme)
		if _, err := os.Stat(themeDir); os.IsNotExist(err) {
			t.Errorf("Theme directory %s not found in multi-theme example", theme)
			continue
		}

		// 检查每个主题的必需目录
		requiredDirs := []string{"layouts", "pages", "singles", "errors"}
		for _, dir := range requiredDirs {
			dirPath := filepath.Join(themeDir, dir)
			if _, err := os.Stat(dirPath); os.IsNotExist(err) {
				t.Errorf("Required directory %s not found in theme %s", dir, theme)
			}
		}

		// 检查theme.json文件
		themeConfigFile := filepath.Join(themeDir, "theme.json")
		if _, err := os.Stat(themeConfigFile); os.IsNotExist(err) {
			t.Errorf("theme.json not found in theme %s", theme)
		}
	}

	// 验证main.go文件存在
	mainFile := filepath.Join(exampleDir, "main.go")
	if _, err := os.Stat(mainFile); os.IsNotExist(err) {
		t.Errorf("main.go not found in multi-theme example directory")
	}

	t.Logf("Multi-theme example directory structure is compatible")
}

// TestAPIConsistency 测试API一致性
func TestAPIConsistency(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "api_consistency_*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建传统结构
	if err := createLegacyStructure(tempDir); err != nil {
		t.Fatalf("Failed to create legacy structure: %v", err)
	}

	funcMap := NewFuncMap()

	// 创建两个引擎：一个传统模式，一个多主题模式（但只有一个主题）
	legacyEngine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap)
	if err != nil {
		t.Fatalf("Failed to create legacy engine: %v", err)
	}
	legacyEngine.Init()
	defer legacyEngine.Close()

	// 创建单主题的多主题引擎
	multiDir := filepath.Join(tempDir, "multi")
	defaultThemeDir := filepath.Join(multiDir, "default")
	if err := createThemeStructure(defaultThemeDir); err != nil {
		t.Fatalf("Failed to create default theme structure: %v", err)
	}

	multiEngine, err := NewEngine(multiDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
	if err != nil {
		t.Fatalf("Failed to create multi-theme engine: %v", err)
	}
	multiEngine.Init()
	defer multiEngine.Close()

	// 测试API一致性
	testCases := []struct {
		name     string
		testFunc func(engine *Engine) interface{}
	}{
		{
			name: "PageName",
			testFunc: func(engine *Engine) interface{} {
				return engine.PageName("test")
			},
		},
		{
			name: "SingleName",
			testFunc: func(engine *Engine) interface{} {
				return engine.SingleName("test")
			},
		},
		{
			name: "ErrorName",
			testFunc: func(engine *Engine) interface{} {
				return engine.ErrorName("404")
			},
		},
		{
			name: "GetCurrentTheme",
			testFunc: func(engine *Engine) interface{} {
				return engine.GetCurrentTheme()
			},
		},
		{
			name: "ThemeExists_default",
			testFunc: func(engine *Engine) interface{} {
				return engine.ThemeExists("default")
			},
		},
		{
			name: "ThemeExists_nonexistent",
			testFunc: func(engine *Engine) interface{} {
				return engine.ThemeExists("nonexistent")
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			legacyResult := tc.testFunc(legacyEngine)
			multiResult := tc.testFunc(multiEngine)

			// 对于某些API，结果应该完全一致
			switch tc.name {
			case "PageName", "SingleName", "ErrorName":
				if legacyResult != multiResult {
					t.Errorf("API %s: legacy result %v != multi-theme result %v", tc.name, legacyResult, multiResult)
				}
			case "GetCurrentTheme":
				// 两者都应该返回"default"
				if legacyResult != "default" || multiResult != "default" {
					t.Errorf("API %s: expected 'default' for both, got legacy=%v, multi=%v", tc.name, legacyResult, multiResult)
				}
			case "ThemeExists_default":
				// 两者都应该返回true
				if legacyResult != true || multiResult != true {
					t.Errorf("API %s: expected true for both, got legacy=%v, multi=%v", tc.name, legacyResult, multiResult)
				}
			case "ThemeExists_nonexistent":
				// 两者都应该返回false
				if legacyResult != false || multiResult != false {
					t.Errorf("API %s: expected false for both, got legacy=%v, multi=%v", tc.name, legacyResult, multiResult)
				}
			}
		})
	}
}
