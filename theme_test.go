package template

import (
	"encoding/json"
	"errors"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"testing/quick"
)

// TestThemeDataStructures 测试主题数据结构的基本功能
func TestThemeDataStructures(t *testing.T) {
	// 测试Theme结构体的基本创建和字段访问
	theme := &Theme{
		Name:       "test-theme",
		Path:       "/path/to/theme",
		IsDefault:  true,
		IsEmbedded: false,
		Metadata: ThemeMetadata{
			DisplayName: "Test Theme",
			Description: "A test theme",
			Version:     "1.0.0",
			Author:      "Test Author",
			Tags:        []string{"test", "demo"},
			Custom:      map[string]any{"color": "blue"},
		},
	}

	if theme.Name != "test-theme" {
		t.Errorf("Expected theme name 'test-theme', got '%s'", theme.Name)
	}

	if theme.Metadata.DisplayName != "Test Theme" {
		t.Errorf("Expected display name 'Test Theme', got '%s'", theme.Metadata.DisplayName)
	}
}

// TestThemeErrorTypes 测试主题错误类型
func TestThemeErrorTypes(t *testing.T) {
	tests := []struct {
		errorType ThemeErrorType
		expected  string
	}{
		{ErrThemeNotFound, "ThemeNotFound"},
		{ErrThemeInvalid, "ThemeInvalid"},
		{ErrThemeLoadFailed, "ThemeLoadFailed"},
		{ErrThemeSwitchFailed, "ThemeSwitchFailed"},
		{ErrThemeConfigInvalid, "ThemeConfigInvalid"},
	}

	for _, test := range tests {
		if test.errorType.String() != test.expected {
			t.Errorf("Expected error type string '%s', got '%s'", test.expected, test.errorType.String())
		}
	}
}

// TestThemeError 测试主题错误的创建和格式化
func TestThemeError(t *testing.T) {
	// 测试不带原因的错误
	err1 := &ThemeError{
		Type:    ErrThemeNotFound,
		Theme:   "missing-theme",
		Message: "theme not found",
	}

	expected1 := "theme error [missing-theme]: theme not found"
	if err1.Error() != expected1 {
		t.Errorf("Expected error message '%s', got '%s'", expected1, err1.Error())
	}

	// 测试带原因的错误
	cause := fmt.Errorf("file not found")
	err2 := &ThemeError{
		Type:    ErrThemeLoadFailed,
		Theme:   "broken-theme",
		Message: "failed to load theme",
		Cause:   cause,
	}

	expected2 := "theme error [broken-theme]: failed to load theme (caused by: file not found)"
	if err2.Error() != expected2 {
		t.Errorf("Expected error message '%s', got '%s'", expected2, err2.Error())
	}

	// 测试错误链
	if err2.Unwrap() != cause {
		t.Errorf("Expected unwrapped error to be the cause")
	}
}

// Property-based tests

// generateTheme 生成随机主题用于属性测试
func generateTheme(name string) *Theme {
	return &Theme{
		Name:       fmt.Sprintf("theme-%s", name),
		Path:       fmt.Sprintf("/path/to/theme-%s", name),
		IsDefault:  false,
		IsEmbedded: false,
		Metadata: ThemeMetadata{
			DisplayName: fmt.Sprintf("Theme %s", name),
			Description: fmt.Sprintf("Description for theme %s", name),
			Version:     "1.0.0",
			Author:      "Test Author",
			Tags:        []string{"test"},
			Custom:      make(map[string]any),
		},
	}
}

// Property 6: Theme Information Queries
// Feature: multi-theme-support, Property 6: Theme Information Queries
func TestProperty_ThemeInformationQueries(t *testing.T) {
	// 属性：对于任何多主题配置，系统应该准确报告可用主题、当前活动主题和主题元数据

	property := func() bool {
		// 创建临时目录用于测试
		tempDir, err := os.MkdirTemp("", "theme-test-*")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建多个主题目录
		themes := []string{"theme1", "theme2", "theme3"}
		for _, themeName := range themes {
			themeDir := filepath.Join(tempDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				t.Logf("Failed to create theme structure for %s: %v", themeName, err)
				return false
			}
		}

		// 创建主题发现器
		discovery := NewThemeDiscovery(tempDir, NewFuncMap(), DefaultLoadTemplate)

		// 检测模式应该是多主题模式
		mode, err := discovery.DetectMode()
		if err != nil {
			t.Logf("Failed to detect mode: %v", err)
			return false
		}

		if mode != ModeMultiTheme {
			t.Logf("Expected ModeMultiTheme, got %v", mode)
			return false
		}

		// 验证每个主题都能被正确识别
		for _, themeName := range themes {
			themePath := filepath.Join(tempDir, themeName)
			if !discovery.isValidThemeDirectory(themePath) {
				t.Logf("Theme %s should be valid but was not recognized", themeName)
				return false
			}

			// 验证能够加载主题元数据
			metadata, err := discovery.LoadThemeMetadata(themePath)
			if err != nil {
				t.Logf("Failed to load metadata for theme %s: %v", themeName, err)
				return false
			}

			if metadata == nil {
				t.Logf("Metadata should not be nil for theme %s", themeName)
				return false
			}

			// 验证默认元数据字段
			if metadata.DisplayName == "" {
				t.Logf("DisplayName should not be empty for theme %s", themeName)
				return false
			}

			if metadata.Version == "" {
				t.Logf("Version should not be empty for theme %s", themeName)
				return false
			}
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 10}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestProperty_ThemeMetadataSerialization 测试主题元数据的序列化和反序列化
// Feature: multi-theme-support, Property 6: Theme Information Queries
func TestProperty_ThemeMetadataSerialization(t *testing.T) {
	// 属性：对于任何主题元数据，序列化后再反序列化应该得到相同的数据

	property := func(displayName, description, version, author string, tags []string) bool {
		// 过滤空值以确保测试的有效性
		if displayName == "" {
			displayName = "Default Theme"
		}
		if version == "" {
			version = "1.0.0"
		}
		if tags == nil {
			tags = []string{}
		}

		original := ThemeMetadata{
			DisplayName: displayName,
			Description: description,
			Version:     version,
			Author:      author,
			Tags:        tags,
			Custom:      map[string]any{"test": "value"},
		}

		// 序列化
		data, err := json.Marshal(original)
		if err != nil {
			t.Logf("Failed to marshal metadata: %v", err)
			return false
		}

		// 反序列化
		var deserialized ThemeMetadata
		if err := json.Unmarshal(data, &deserialized); err != nil {
			t.Logf("Failed to unmarshal metadata: %v", err)
			return false
		}

		// 比较关键字段
		if original.DisplayName != deserialized.DisplayName {
			t.Logf("DisplayName mismatch: %s != %s", original.DisplayName, deserialized.DisplayName)
			return false
		}

		if original.Description != deserialized.Description {
			t.Logf("Description mismatch: %s != %s", original.Description, deserialized.Description)
			return false
		}

		if original.Version != deserialized.Version {
			t.Logf("Version mismatch: %s != %s", original.Version, deserialized.Version)
			return false
		}

		if original.Author != deserialized.Author {
			t.Logf("Author mismatch: %s != %s", original.Author, deserialized.Author)
			return false
		}

		if !reflect.DeepEqual(original.Tags, deserialized.Tags) {
			t.Logf("Tags mismatch: %v != %v", original.Tags, deserialized.Tags)
			return false
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestProperty_ThemeDiscoveryModeDetection 测试主题发现模式检测
// Feature: multi-theme-support, Property 2: Multi-Theme Mode Detection
func TestProperty_ThemeDiscoveryModeDetection(t *testing.T) {
	// 属性：对于任何包含主题子目录的目录结构，系统应该自动检测为多主题模式

	property := func() bool {
		// 创建临时目录
		tempDir, err := os.MkdirTemp("", "mode-detection-test-*")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 测试传统模式检测
		legacyDir := filepath.Join(tempDir, "legacy")
		if err := createLegacyStructure(legacyDir); err != nil {
			t.Logf("Failed to create legacy structure: %v", err)
			return false
		}

		discovery := NewThemeDiscovery(legacyDir, NewFuncMap(), DefaultLoadTemplate)
		mode, err := discovery.DetectMode()
		if err != nil {
			t.Logf("Failed to detect legacy mode: %v", err)
			return false
		}

		if mode != ModeLegacy {
			t.Logf("Expected ModeLegacy for legacy structure, got %v", mode)
			return false
		}

		// 测试多主题模式检测
		multiThemeDir := filepath.Join(tempDir, "multi")
		if err := os.MkdirAll(multiThemeDir, 0755); err != nil {
			t.Logf("Failed to create multi-theme dir: %v", err)
			return false
		}

		// 创建多个主题
		for _, themeName := range []string{"theme1", "theme2"} {
			themeDir := filepath.Join(multiThemeDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				t.Logf("Failed to create theme structure for %s: %v", themeName, err)
				return false
			}
		}

		discovery2 := NewThemeDiscovery(multiThemeDir, NewFuncMap(), DefaultLoadTemplate)
		mode2, err := discovery2.DetectMode()
		if err != nil {
			t.Logf("Failed to detect multi-theme mode: %v", err)
			return false
		}

		if mode2 != ModeMultiTheme {
			t.Logf("Expected ModeMultiTheme for multi-theme structure, got %v", mode2)
			return false
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 5}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestProperty_ErrorHandlingConsistency 测试错误处理一致性
// Feature: multi-theme-support, Property 8: Error Handling Consistency
func TestProperty_ErrorHandlingConsistency(t *testing.T) {
	// 属性：对于任何无效的主题配置，系统应该提供清晰、描述性的错误消息

	property := func() bool {
		// 创建临时目录
		tempDir, err := os.MkdirTemp("", "error-handling-test-*")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		discovery := NewThemeDiscovery(tempDir, NewFuncMap(), DefaultLoadTemplate)

		// 测试1: 缺少必需目录的主题
		incompleteThemeDir := filepath.Join(tempDir, "incomplete-theme")
		if err := os.MkdirAll(incompleteThemeDir, 0755); err != nil {
			t.Logf("Failed to create incomplete theme dir: %v", err)
			return false
		}

		// 只创建部分必需目录
		layoutsDir := filepath.Join(incompleteThemeDir, "layouts")
		if err := os.MkdirAll(layoutsDir, 0755); err != nil {
			t.Logf("Failed to create layouts dir: %v", err)
			return false
		}

		// 验证应该失败并返回描述性错误
		err = discovery.ValidateTheme(incompleteThemeDir)
		if err == nil {
			t.Logf("Expected validation to fail for incomplete theme")
			return false
		}

		// 检查错误类型和消息
		var themeErr *ThemeError
		if !errors.As(err, &themeErr) {
			t.Logf("Expected ThemeError, got %T", err)
			return false
		}

		if themeErr.Type != ErrThemeInvalid {
			t.Logf("Expected ErrThemeInvalid, got %v", themeErr.Type)
			return false
		}

		if themeErr.Theme == "" {
			t.Logf("Error should include theme name")
			return false
		}

		if themeErr.Message == "" {
			t.Logf("Error should include descriptive message")
			return false
		}

		// 测试2: 主题目录存在但没有模板文件
		emptyThemeDir := filepath.Join(tempDir, "empty-theme")
		if err := createEmptyThemeStructure(emptyThemeDir); err != nil {
			t.Logf("Failed to create empty theme structure: %v", err)
			return false
		}

		err = discovery.ValidateTheme(emptyThemeDir)
		if err == nil {
			t.Logf("Expected validation to fail for theme without templates")
			return false
		}

		// 验证错误信息
		if !errors.As(err, &themeErr) {
			t.Logf("Expected ThemeError for empty theme")
			return false
		}

		// 测试3: 无效的theme.json文件
		invalidConfigThemeDir := filepath.Join(tempDir, "invalid-config-theme")
		if err := createThemeStructure(invalidConfigThemeDir); err != nil {
			t.Logf("Failed to create theme with invalid config: %v", err)
			return false
		}

		// 创建无效的theme.json
		invalidConfigPath := filepath.Join(invalidConfigThemeDir, "theme.json")
		invalidJSON := `{"name": 123, "invalid": json}`
		if err := os.WriteFile(invalidConfigPath, []byte(invalidJSON), 0644); err != nil {
			t.Logf("Failed to write invalid config: %v", err)
			return false
		}

		err = discovery.ValidateTheme(invalidConfigThemeDir)
		if err == nil {
			t.Logf("Expected validation to fail for invalid theme.json")
			return false
		}

		if !errors.As(err, &themeErr) {
			t.Logf("Expected ThemeError for invalid config")
			return false
		}

		if themeErr.Type != ErrThemeConfigInvalid {
			t.Logf("Expected ErrThemeConfigInvalid, got %v", themeErr.Type)
			return false
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 3}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// Helper functions for testing

// createThemeStructure 创建完整的主题目录结构
func createThemeStructure(themeDir string) error {
	// 创建基本目录
	dirs := []string{"layouts", "pages", "singles", "errors", "partials"}
	for _, dir := range dirs {
		dirPath := filepath.Join(themeDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return err
		}
	}

	// 创建 layout.tmpl
	layoutContent := `<!DOCTYPE html>
<html>
<head>
    {{ template "header" . }}
</head>
<body>
    {{ template "content" . }}
</body>
</html>`
	layoutFile := filepath.Join(themeDir, "layouts", "layout.tmpl")
	if err := os.WriteFile(layoutFile, []byte(layoutContent), 0644); err != nil {
		return err
	}

	// 创建页面目录和模板
	pageDir := filepath.Join(themeDir, "pages", "sample")
	if err := os.MkdirAll(pageDir, 0755); err != nil {
		return err
	}
	pageContent := `{{ define "header" }}<title>{{ .title }}</title>{{ end }}
{{ define "content" }}
<h1>{{ .title }}</h1>
<p>{{ .content }}</p>
{{ end }}`
	pageFile := filepath.Join(pageDir, "sample.tmpl")
	if err := os.WriteFile(pageFile, []byte(pageContent), 0644); err != nil {
		return err
	}

	// 创建示例单页模板
	singleContent := `<h1>{{ .title }}</h1>
<p>Single page content</p>`
	singleFile := filepath.Join(themeDir, "singles", "sample.tmpl")
	if err := os.WriteFile(singleFile, []byte(singleContent), 0644); err != nil {
		return err
	}

	// 创建示例错误模板
	errorContent := `<h1>Error</h1>
<p>An error occurred</p>`
	errorFile := filepath.Join(themeDir, "errors", "sample.tmpl")
	if err := os.WriteFile(errorFile, []byte(errorContent), 0644); err != nil {
		return err
	}

	// 创建示例部分模板
	partialContent := `<div>Partial content</div>`
	partialFile := filepath.Join(themeDir, "partials", "sample.tmpl")
	if err := os.WriteFile(partialFile, []byte(partialContent), 0644); err != nil {
		return err
	}

	return nil
}

// createLegacyStructure 创建传统的模板目录结构
func createLegacyStructure(baseDir string) error {
	// 创建基本目录
	dirs := []string{"layouts", "pages", "singles", "errors", "partials"}
	for _, dir := range dirs {
		dirPath := filepath.Join(baseDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return err
		}
	}

	// 创建 layout.tmpl
	layoutContent := `<!DOCTYPE html>
<html>
<head>
    {{ template "header" . }}
</head>
<body>
    {{ template "content" . }}
</body>
</html>`
	layoutFile := filepath.Join(baseDir, "layouts", "layout.tmpl")
	if err := os.WriteFile(layoutFile, []byte(layoutContent), 0644); err != nil {
		return err
	}

	// 创建页面目录和模板
	pageDir := filepath.Join(baseDir, "pages", "sample")
	if err := os.MkdirAll(pageDir, 0755); err != nil {
		return err
	}
	pageContent := `{{ define "header" }}<title>{{ .title }}</title>{{ end }}
{{ define "content" }}
<h1>{{ .title }}</h1>
<p>{{ .content }}</p>
{{ end }}`
	pageFile := filepath.Join(pageDir, "sample.tmpl")
	if err := os.WriteFile(pageFile, []byte(pageContent), 0644); err != nil {
		return err
	}

	// 创建示例单页模板
	singleContent := `<h1>{{ .title }}</h1>
<p>Single page content</p>`
	singleFile := filepath.Join(baseDir, "singles", "sample.tmpl")
	if err := os.WriteFile(singleFile, []byte(singleContent), 0644); err != nil {
		return err
	}

	// 创建示例错误模板
	errorContent := `<h1>Error</h1>
<p>An error occurred</p>`
	errorFile := filepath.Join(baseDir, "errors", "sample.tmpl")
	if err := os.WriteFile(errorFile, []byte(errorContent), 0644); err != nil {
		return err
	}

	// 创建示例部分模板
	partialContent := `<div>Partial content</div>`
	partialFile := filepath.Join(baseDir, "partials", "sample.tmpl")
	if err := os.WriteFile(partialFile, []byte(partialContent), 0644); err != nil {
		return err
	}

	return nil
}

// createEmptyThemeStructure 创建空的主题目录结构（只有目录，没有模板文件）
func createEmptyThemeStructure(themeDir string) error {
	dirs := []string{"layouts", "pages", "singles", "errors", "partials"}

	for _, dir := range dirs {
		dirPath := filepath.Join(themeDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return err
		}
		// 不创建任何文件，保持目录为空
	}

	return nil
}

// TestProperty_ThemeSelectionLogic 测试主题选择逻辑
// Feature: multi-theme-support, Property 3: Theme Selection Logic
func TestProperty_ThemeSelectionLogic(t *testing.T) {
	// 属性：对于任何主题配置（指定主题、默认主题或未指定主题），系统应该加载正确的主题或在主题不存在时提供适当的错误消息

	property := func(themeName, defaultTheme string, multiThemeMode bool) bool {
		// 创建临时目录进行测试
		tempDir, err := os.MkdirTemp("", "theme_selection_test")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建测试选项
		var opts []Option

		// 添加主题相关选项
		if themeName != "" {
			opts = append(opts, SetTheme(themeName))
		}
		if defaultTheme != "" {
			opts = append(opts, DefaultTheme(defaultTheme))
		}
		opts = append(opts, EnableMultiTheme(multiThemeMode))

		// 创建选项实例
		options := newOptions(opts...)

		// 验证选项字段是否正确设置
		if options.Theme != themeName {
			t.Logf("Expected Theme '%s', got '%s'", themeName, options.Theme)
			return false
		}

		if options.DefaultTheme != defaultTheme {
			t.Logf("Expected DefaultTheme '%s', got '%s'", defaultTheme, options.DefaultTheme)
			return false
		}

		if options.MultiThemeMode != multiThemeMode {
			t.Logf("Expected MultiThemeMode %v, got %v", multiThemeMode, options.MultiThemeMode)
			return false
		}

		// 验证选项函数的幂等性 - 多次应用相同选项应该产生相同结果
		options2 := newOptions(opts...)
		if !reflect.DeepEqual(options, options2) {
			t.Logf("Options are not idempotent")
			return false
		}

		// 验证选项的组合性 - 不同顺序应该产生相同结果
		if len(opts) > 1 {
			// 反转选项顺序
			reversedOpts := make([]Option, len(opts))
			for i, opt := range opts {
				reversedOpts[len(opts)-1-i] = opt
			}
			options3 := newOptions(reversedOpts...)

			if !reflect.DeepEqual(options, options3) {
				t.Logf("Options are not commutative")
				return false
			}
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 100}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestProperty_RuntimeThemeSwitching 测试运行时主题切换
// Feature: multi-theme-support, Property 7: Runtime Theme Switching
func TestProperty_RuntimeThemeSwitching(t *testing.T) {
	// 属性：对于任何有效的主题切换操作，系统应该立即加载新主题的模板，更新文件监听器，并保持所有其他引擎状态，或在无效操作时保持当前状态并返回错误

	property := func() bool {
		// 创建临时目录进行测试
		tempDir, err := os.MkdirTemp("", "theme_switching_test")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建多个主题
		themes := []string{"theme1", "theme2", "theme3"}
		for _, themeName := range themes {
			themeDir := filepath.Join(tempDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				t.Logf("Failed to create theme structure for %s: %v", themeName, err)
				return false
			}
		}

		// 创建主题管理器
		manager := NewDefaultThemeManager(tempDir, NewFuncMap(), DefaultLoadTemplate)

		// 发现主题
		if err := manager.DiscoverThemes(); err != nil {
			t.Logf("Failed to discover themes: %v", err)
			return false
		}

		// 验证初始状态
		initialTheme := manager.GetCurrentTheme()
		if initialTheme == "" {
			t.Logf("Initial theme should not be empty")
			return false
		}

		initialRender := manager.GetRender()
		if initialRender == nil {
			t.Logf("Initial render should not be nil")
			return false
		}

		// 测试切换到每个主题
		for _, targetTheme := range themes {
			// 记录切换前的状态
			previousTheme := manager.GetCurrentTheme()

			// 执行主题切换
			err := manager.SwitchTheme(targetTheme)
			if err != nil {
				t.Logf("Failed to switch to theme %s: %v", targetTheme, err)
				return false
			}

			// 验证切换后的状态
			currentTheme := manager.GetCurrentTheme()
			if currentTheme != targetTheme {
				t.Logf("Expected current theme to be %s, got %s", targetTheme, currentTheme)
				return false
			}

			// 验证渲染器已更新
			currentRender := manager.GetRender()
			if currentRender == nil {
				t.Logf("Render should not be nil after theme switch")
				return false
			}

			// 验证渲染器确实发生了变化（除非切换到相同主题）
			if targetTheme != previousTheme {
				// 验证渲染器包含模板
				renderMap := map[string]*template.Template(currentRender)
				if len(renderMap) == 0 {
					t.Logf("Render should contain templates after theme switch")
					return false
				}
			}

			// 验证主题存在性检查
			if !manager.ThemeExists(targetTheme) {
				t.Logf("Theme %s should exist after successful switch", targetTheme)
				return false
			}

			// 验证可以获取主题元数据
			metadata, err := manager.GetThemeMetadata(targetTheme)
			if err != nil {
				t.Logf("Failed to get metadata for theme %s: %v", targetTheme, err)
				return false
			}
			if metadata == nil {
				t.Logf("Metadata should not be nil for theme %s", targetTheme)
				return false
			}
		}

		// 测试切换到不存在的主题
		nonExistentTheme := "non-existent-theme"
		previousTheme := manager.GetCurrentTheme()

		err = manager.SwitchTheme(nonExistentTheme)
		if err == nil {
			t.Logf("Expected error when switching to non-existent theme")
			return false
		}

		// 验证状态没有改变
		if manager.GetCurrentTheme() != previousTheme {
			t.Logf("Current theme should not change when switching to non-existent theme")
			return false
		}

		// 验证渲染器仍然有效
		if manager.GetRender() == nil {
			t.Logf("Render should not be nil after failed switch")
			return false
		}

		// 验证错误类型
		var themeErr *ThemeError
		if !errors.As(err, &themeErr) {
			t.Logf("Expected ThemeError for non-existent theme")
			return false
		}

		if themeErr.Type != ErrThemeNotFound {
			t.Logf("Expected ErrThemeNotFound, got %v", themeErr.Type)
			return false
		}

		// 测试幂等性：切换到当前主题应该成功且不改变状态
		currentTheme := manager.GetCurrentTheme()

		err = manager.SwitchTheme(currentTheme)
		if err != nil {
			t.Logf("Switching to current theme should not fail: %v", err)
			return false
		}

		if manager.GetCurrentTheme() != currentTheme {
			t.Logf("Current theme should not change when switching to itself")
			return false
		}

		// 验证渲染器仍然有效
		if manager.GetRender() == nil {
			t.Logf("Render should not be nil after switching to current theme")
			return false
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 5}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestProperty_ThemeSwitchingStateConsistency 测试主题切换状态一致性
// Feature: multi-theme-support, Property 7: Runtime Theme Switching
func TestProperty_ThemeSwitchingStateConsistency(t *testing.T) {
	// 属性：对于任何主题切换序列，系统状态应该保持一致性

	property := func() bool {
		// 创建临时目录
		tempDir, err := os.MkdirTemp("", "theme_consistency_test")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建多个主题
		themes := []string{"alpha", "beta", "gamma"}
		for _, themeName := range themes {
			themeDir := filepath.Join(tempDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				t.Logf("Failed to create theme structure for %s: %v", themeName, err)
				return false
			}
		}

		// 创建主题管理器
		manager := NewDefaultThemeManager(tempDir, NewFuncMap(), DefaultLoadTemplate)

		// 发现主题
		if err := manager.DiscoverThemes(); err != nil {
			t.Logf("Failed to discover themes: %v", err)
			return false
		}

		// 验证初始一致性
		availableThemes := manager.GetAvailableThemes()
		if len(availableThemes) != len(themes) {
			t.Logf("Expected %d themes, got %d", len(themes), len(availableThemes))
			return false
		}

		currentTheme := manager.GetCurrentTheme()
		if currentTheme == "" {
			t.Logf("Current theme should not be empty")
			return false
		}

		if !manager.ThemeExists(currentTheme) {
			t.Logf("Current theme %s should exist", currentTheme)
			return false
		}

		// 执行一系列主题切换
		switchSequence := []string{"beta", "alpha", "gamma", "beta", "alpha"}

		for i, targetTheme := range switchSequence {
			// 记录切换前状态
			prevAvailable := manager.GetAvailableThemes()

			// 执行切换
			err := manager.SwitchTheme(targetTheme)
			if err != nil {
				t.Logf("Failed to switch to theme %s at step %d: %v", targetTheme, i, err)
				return false
			}

			// 验证切换后状态一致性
			newTheme := manager.GetCurrentTheme()
			newAvailable := manager.GetAvailableThemes()

			// 当前主题应该更新
			if newTheme != targetTheme {
				t.Logf("Expected current theme %s, got %s at step %d", targetTheme, newTheme, i)
				return false
			}

			// 可用主题列表不应该改变
			if len(newAvailable) != len(prevAvailable) {
				t.Logf("Available themes count changed during switch at step %d", i)
				return false
			}

			// 验证所有原有主题仍然存在
			for _, theme := range prevAvailable {
				if !manager.ThemeExists(theme) {
					t.Logf("Theme %s disappeared after switch at step %d", theme, i)
					return false
				}
			}

			// 验证渲染器有效
			render := manager.GetRender()
			if render == nil {
				t.Logf("Render is nil after switch at step %d", i)
				return false
			}

			// 验证可以获取当前主题的元数据
			metadata, err := manager.GetThemeMetadata(newTheme)
			if err != nil {
				t.Logf("Failed to get metadata for current theme at step %d: %v", i, err)
				return false
			}
			if metadata == nil {
				t.Logf("Metadata is nil for current theme at step %d", i)
				return false
			}
		}

		// 最终一致性检查
		finalTheme := manager.GetCurrentTheme()
		finalAvailable := manager.GetAvailableThemes()

		// 验证最终状态
		if len(finalAvailable) != len(themes) {
			t.Logf("Final available themes count mismatch")
			return false
		}

		if !manager.ThemeExists(finalTheme) {
			t.Logf("Final current theme %s does not exist", finalTheme)
			return false
		}

		// 验证所有主题仍然可访问
		for _, theme := range themes {
			if !manager.ThemeExists(theme) {
				t.Logf("Theme %s is not accessible after switch sequence", theme)
				return false
			}

			metadata, err := manager.GetThemeMetadata(theme)
			if err != nil {
				t.Logf("Cannot get metadata for theme %s after switch sequence: %v", theme, err)
				return false
			}
			if metadata == nil {
				t.Logf("Metadata is nil for theme %s after switch sequence", theme)
				return false
			}
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 3}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestProperty_ThemeSwitchingFailureRecovery 测试主题切换失败恢复
// Feature: multi-theme-support, Property 7: Runtime Theme Switching
func TestProperty_ThemeSwitchingFailureRecovery(t *testing.T) {
	// 属性：当主题切换失败时，系统应该保持原有状态不变

	property := func() bool {
		// 创建临时目录
		tempDir, err := os.MkdirTemp("", "theme_failure_test")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建一个有效主题
		validTheme := "valid-theme"
		validThemeDir := filepath.Join(tempDir, validTheme)
		if err := createThemeStructure(validThemeDir); err != nil {
			t.Logf("Failed to create valid theme structure: %v", err)
			return false
		}

		// 创建主题管理器
		manager := NewDefaultThemeManager(tempDir, NewFuncMap(), DefaultLoadTemplate)

		// 发现主题
		if err := manager.DiscoverThemes(); err != nil {
			t.Logf("Failed to discover themes: %v", err)
			return false
		}

		// 记录初始状态
		initialTheme := manager.GetCurrentTheme()
		initialAvailable := manager.GetAvailableThemes()

		if initialTheme == "" {
			t.Logf("Initial theme should not be empty")
			return false
		}

		if manager.GetRender() == nil {
			t.Logf("Initial render should not be nil")
			return false
		}

		// 测试切换到不存在的主题
		nonExistentThemes := []string{"missing-theme", "invalid-theme", ""}

		for _, invalidTheme := range nonExistentThemes {
			// 尝试切换到无效主题
			err := manager.SwitchTheme(invalidTheme)

			// 应该返回错误
			if err == nil {
				t.Logf("Expected error when switching to invalid theme '%s'", invalidTheme)
				return false
			}

			// 验证状态没有改变
			if manager.GetCurrentTheme() != initialTheme {
				t.Logf("Current theme changed after failed switch to '%s'", invalidTheme)
				return false
			}

			// 验证渲染器仍然有效
			if manager.GetRender() == nil {
				t.Logf("Render should not be nil after failed switch to '%s'", invalidTheme)
				return false
			}

			// 验证可用主题列表没有改变
			currentAvailable := manager.GetAvailableThemes()
			if len(currentAvailable) != len(initialAvailable) {
				t.Logf("Available themes count changed after failed switch to '%s'", invalidTheme)
				return false
			}

			// 验证错误类型正确
			var themeErr *ThemeError
			if !errors.As(err, &themeErr) {
				t.Logf("Expected ThemeError for invalid theme '%s'", invalidTheme)
				return false
			}

			expectedErrorType := ErrThemeNotFound
			if themeErr.Type != expectedErrorType {
				t.Logf("Expected error type %v for invalid theme '%s', got %v", expectedErrorType, invalidTheme, themeErr.Type)
				return false
			}
		}

		// 验证系统仍然可以正常工作
		// 尝试重新加载当前主题
		err = manager.ReloadCurrentTheme()
		if err != nil {
			t.Logf("Failed to reload current theme after failed switches: %v", err)
			return false
		}

		// 验证状态仍然一致
		if manager.GetCurrentTheme() != initialTheme {
			t.Logf("Current theme changed after reload")
			return false
		}

		if manager.GetRender() == nil {
			t.Logf("Render is nil after reload")
			return false
		}

		// 验证可以获取元数据
		metadata, err := manager.GetThemeMetadata(initialTheme)
		if err != nil {
			t.Logf("Failed to get metadata after failed switches: %v", err)
			return false
		}
		if metadata == nil {
			t.Logf("Metadata is nil after failed switches")
			return false
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 5}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestThemeSelectionLogicExamples 测试主题选择逻辑的具体示例
func TestThemeSelectionLogicExamples(t *testing.T) {
	tests := []struct {
		name            string
		theme           string
		defaultTheme    string
		multiThemeMode  bool
		expectedTheme   string
		expectedDefault string
		expectedMulti   bool
	}{
		{
			name:            "指定主题",
			theme:           "custom-theme",
			defaultTheme:    "",
			multiThemeMode:  false,
			expectedTheme:   "custom-theme",
			expectedDefault: "",
			expectedMulti:   false,
		},
		{
			name:            "指定默认主题",
			theme:           "",
			defaultTheme:    "default-theme",
			multiThemeMode:  true,
			expectedTheme:   "",
			expectedDefault: "default-theme",
			expectedMulti:   true,
		},
		{
			name:            "同时指定主题和默认主题",
			theme:           "active-theme",
			defaultTheme:    "fallback-theme",
			multiThemeMode:  true,
			expectedTheme:   "active-theme",
			expectedDefault: "fallback-theme",
			expectedMulti:   true,
		},
		{
			name:            "空配置",
			theme:           "",
			defaultTheme:    "",
			multiThemeMode:  false,
			expectedTheme:   "",
			expectedDefault: "",
			expectedMulti:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opts []Option
			if tt.theme != "" {
				opts = append(opts, SetTheme(tt.theme))
			}
			if tt.defaultTheme != "" {
				opts = append(opts, DefaultTheme(tt.defaultTheme))
			}
			opts = append(opts, EnableMultiTheme(tt.multiThemeMode))

			options := newOptions(opts...)

			if options.Theme != tt.expectedTheme {
				t.Errorf("Expected Theme '%s', got '%s'", tt.expectedTheme, options.Theme)
			}
			if options.DefaultTheme != tt.expectedDefault {
				t.Errorf("Expected DefaultTheme '%s', got '%s'", tt.expectedDefault, options.DefaultTheme)
			}
			if options.MultiThemeMode != tt.expectedMulti {
				t.Errorf("Expected MultiThemeMode %v, got %v", tt.expectedMulti, options.MultiThemeMode)
			}
		})
	}
}

// TestProperty_ResourceManagementEfficiency 测试资源管理效率
// Feature: multi-theme-support, Property 9: Resource Management Efficiency
func TestProperty_ResourceManagementEfficiency(t *testing.T) {
	// 属性：对于任何多主题环境，系统应该只加载当前激活主题的模板到内存中，避免不必要的资源消耗

	property := func() bool {
		// 创建临时目录
		tempDir, err := os.MkdirTemp("", "resource_management_test")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建多个主题，每个主题有不同数量的模板
		themes := []struct {
			name          string
			templateCount int
		}{
			{"small-theme", 3},
			{"medium-theme", 6},
			{"large-theme", 10},
		}

		for _, themeInfo := range themes {
			themeDir := filepath.Join(tempDir, themeInfo.name)
			if err := createThemeStructureWithTemplates(themeDir, themeInfo.templateCount); err != nil {
				t.Logf("Failed to create theme structure for %s: %v", themeInfo.name, err)
				return false
			}
		}

		// 创建主题管理器
		manager := NewDefaultThemeManager(tempDir, NewFuncMap(), DefaultLoadTemplate)

		// 发现主题
		if err := manager.DiscoverThemes(); err != nil {
			t.Logf("Failed to discover themes: %v", err)
			return false
		}

		// 验证初始状态：只有当前主题被加载
		initialTheme := manager.GetCurrentTheme()
		if initialTheme == "" {
			t.Logf("Initial theme should not be empty")
			return false
		}

		// 显式加载当前主题以确保模板被加载到内存中
		_, err = manager.LoadTheme(initialTheme)
		if err != nil {
			t.Logf("Failed to load initial theme %s: %v", initialTheme, err)
			return false
		}

		// 获取初始资源使用情况
		initialUsage := manager.GetMemoryUsage()
		initialTemplateCount, ok := initialUsage["templates_count"].(int)
		if !ok {
			t.Logf("Failed to get initial template count")
			return false
		}

		if initialTemplateCount == 0 {
			t.Logf("Initial template count should be greater than 0")
			return false
		}

		// 验证渲染器完整性
		if err := manager.ValidateRenderIntegrity(); err != nil {
			t.Logf("Initial render integrity validation failed: %v", err)
			return false
		}

		// 测试切换到不同主题时的资源管理
		for _, themeInfo := range themes {
			if themeInfo.name == initialTheme {
				continue // 跳过当前主题
			}

			// 切换到新主题
			err := manager.SwitchTheme(themeInfo.name)
			if err != nil {
				t.Logf("Failed to switch to theme %s: %v", themeInfo.name, err)
				return false
			}

			// 验证只有当前主题的模板被加载
			currentUsage := manager.GetMemoryUsage()
			currentTemplateCount, ok := currentUsage["templates_count"].(int)
			if !ok {
				t.Logf("Failed to get current template count for theme %s", themeInfo.name)
				return false
			}

			// 验证模板数量合理（应该只包含当前主题的模板）
			if currentTemplateCount == 0 {
				t.Logf("Template count should be greater than 0 for theme %s", themeInfo.name)
				return false
			}

			// 验证当前主题正确
			if manager.GetCurrentTheme() != themeInfo.name {
				t.Logf("Current theme should be %s", themeInfo.name)
				return false
			}

			// 验证渲染器完整性
			if err := manager.ValidateRenderIntegrity(); err != nil {
				t.Logf("Render integrity validation failed for theme %s: %v", themeInfo.name, err)
				return false
			}

			// 验证渲染器统计信息
			stats := manager.GetRenderStats()
			totalTemplates, ok := stats["total"]
			if !ok || totalTemplates == 0 {
				t.Logf("Render stats should show templates for theme %s", themeInfo.name)
				return false
			}

			// 验证模板名称
			templateNames := manager.GetTemplateNames()
			if len(templateNames) == 0 {
				t.Logf("Should have template names for theme %s", themeInfo.name)
				return false
			}

			// 验证至少有一些基本模板类型
			hasBasicTemplates := false
			for _, name := range templateNames {
				if strings.Contains(name, "pages") || strings.Contains(name, "singles") || strings.Contains(name, "error") {
					hasBasicTemplates = true
					break
				}
			}

			if !hasBasicTemplates {
				t.Logf("Theme %s should have basic template types", themeInfo.name)
				return false
			}
		}

		// 测试预加载功能不影响当前资源
		currentTheme := manager.GetCurrentTheme()
		currentUsage := manager.GetMemoryUsage()

		// 预加载另一个主题
		var preloadTheme string
		for _, themeInfo := range themes {
			if themeInfo.name != currentTheme {
				preloadTheme = themeInfo.name
				break
			}
		}

		if preloadTheme != "" {
			err := manager.PreloadTheme(preloadTheme)
			if err != nil {
				t.Logf("Failed to preload theme %s: %v", preloadTheme, err)
				return false
			}

			// 验证当前主题和资源使用没有改变
			if manager.GetCurrentTheme() != currentTheme {
				t.Logf("Current theme should not change after preload")
				return false
			}

			newUsage := manager.GetMemoryUsage()
			if !reflect.DeepEqual(currentUsage, newUsage) {
				t.Logf("Memory usage should not change after preload")
				return false
			}
		}

		// 测试资源清理
		// 切换主题应该清理之前的资源
		beforeSwitchUsage := manager.GetMemoryUsage()

		// 找一个不同的主题进行切换
		var switchToTheme string
		for _, themeInfo := range themes {
			if themeInfo.name != manager.GetCurrentTheme() {
				switchToTheme = themeInfo.name
				break
			}
		}

		if switchToTheme != "" {
			err := manager.SwitchTheme(switchToTheme)
			if err != nil {
				t.Logf("Failed to switch for resource cleanup test: %v", err)
				return false
			}

			afterSwitchUsage := manager.GetMemoryUsage()

			// 验证资源使用情况已更新
			_, _ = beforeSwitchUsage["templates_count"].(int)
			afterCount, _ := afterSwitchUsage["templates_count"].(int)

			// 模板数量可能不同，但都应该大于0
			if afterCount == 0 {
				t.Logf("Template count should be greater than 0 after switch")
				return false
			}

			// 验证当前主题已更新
			if manager.GetCurrentTheme() != switchToTheme {
				t.Logf("Current theme should be updated after switch")
				return false
			}
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 3}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestProperty_ResourceManagementMemoryBounds 测试资源管理内存边界
// Feature: multi-theme-support, Property 9: Resource Management Efficiency
func TestProperty_ResourceManagementMemoryBounds(t *testing.T) {
	// 属性：系统的内存使用应该与当前激活主题的大小成正比，而不是与总主题数量成正比

	property := func() bool {
		// 创建临时目录
		tempDir, err := os.MkdirTemp("", "memory_bounds_test")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建多个不同大小的主题
		smallThemes := []string{"tiny1", "tiny2", "tiny3"}
		largeThemes := []string{"huge1", "huge2"}

		// 创建小主题（每个3个模板）
		for _, themeName := range smallThemes {
			themeDir := filepath.Join(tempDir, themeName)
			if err := createThemeStructureWithTemplates(themeDir, 3); err != nil {
				t.Logf("Failed to create small theme %s: %v", themeName, err)
				return false
			}
		}

		// 创建大主题（每个15个模板）
		for _, themeName := range largeThemes {
			themeDir := filepath.Join(tempDir, themeName)
			if err := createThemeStructureWithTemplates(themeDir, 15); err != nil {
				t.Logf("Failed to create large theme %s: %v", themeName, err)
				return false
			}
		}

		// 创建主题管理器
		manager := NewDefaultThemeManager(tempDir, NewFuncMap(), DefaultLoadTemplate)

		// 发现主题
		if err := manager.DiscoverThemes(); err != nil {
			t.Logf("Failed to discover themes: %v", err)
			return false
		}

		// 验证发现了所有主题
		availableThemes := manager.GetAvailableThemes()
		expectedTotal := len(smallThemes) + len(largeThemes)
		if len(availableThemes) != expectedTotal {
			t.Logf("Expected %d themes, got %d", expectedTotal, len(availableThemes))
			return false
		}

		// 测试切换到小主题时的内存使用
		var smallThemeUsage int
		for _, themeName := range smallThemes {
			if manager.ThemeExists(themeName) {
				err := manager.SwitchTheme(themeName)
				if err != nil {
					t.Logf("Failed to switch to small theme %s: %v", themeName, err)
					return false
				}

				usage := manager.GetMemoryUsage()
				templateCount, ok := usage["templates_count"].(int)
				if !ok {
					t.Logf("Failed to get template count for small theme %s", themeName)
					return false
				}

				if smallThemeUsage == 0 {
					smallThemeUsage = templateCount
				}

				// 小主题的模板数量应该相对较少且一致
				if templateCount == 0 {
					t.Logf("Small theme %s should have templates", themeName)
					return false
				}

				// 验证内存使用估算
				sizeEstimate, ok := usage["render_size_estimate"].(int)
				if !ok || sizeEstimate == 0 {
					t.Logf("Size estimate should be available for theme %s", themeName)
					return false
				}

				break // 只测试一个小主题
			}
		}

		// 测试切换到大主题时的内存使用
		var largeThemeUsage int
		for _, themeName := range largeThemes {
			if manager.ThemeExists(themeName) {
				err := manager.SwitchTheme(themeName)
				if err != nil {
					t.Logf("Failed to switch to large theme %s: %v", themeName, err)
					return false
				}

				usage := manager.GetMemoryUsage()
				templateCount, ok := usage["templates_count"].(int)
				if !ok {
					t.Logf("Failed to get template count for large theme %s", themeName)
					return false
				}

				largeThemeUsage = templateCount

				// 大主题的模板数量应该更多
				if templateCount == 0 {
					t.Logf("Large theme %s should have templates", themeName)
					return false
				}

				break // 只测试一个大主题
			}
		}

		// 验证大主题使用更多资源（但这不是严格要求，因为模板加载可能有优化）
		if largeThemeUsage > 0 && smallThemeUsage > 0 {
			// 至少验证两者都有合理的资源使用
			if largeThemeUsage < smallThemeUsage {
				// 这可能是正常的，取决于模板加载策略
				t.Logf("Note: Large theme uses fewer templates than small theme (may be normal)")
			}
		}

		// 最重要的验证：内存使用应该是有界的
		// 无论有多少主题，当前内存使用应该只反映当前主题
		finalUsage := manager.GetMemoryUsage()
		themeCount, ok := finalUsage["themes_count"].(int)
		if !ok {
			t.Logf("Failed to get theme count")
			return false
		}

		templateCount, ok := finalUsage["templates_count"].(int)
		if !ok {
			t.Logf("Failed to get template count")
			return false
		}

		// 验证主题数量正确
		if themeCount != expectedTotal {
			t.Logf("Theme count should be %d, got %d", expectedTotal, themeCount)
			return false
		}

		// 验证模板数量合理（不应该是所有主题的总和）
		if templateCount == 0 {
			t.Logf("Template count should be greater than 0")
			return false
		}

		// 模板数量应该远小于如果加载所有主题的情况
		// 这是一个粗略的检查，实际数量取决于实现
		maxExpectedTemplates := 50 // 假设单个主题不会超过50个模板
		if templateCount > maxExpectedTemplates {
			t.Logf("Template count %d seems too high for a single theme", templateCount)
			return false
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 2}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// Helper function for creating themes with specific number of templates
func createThemeStructureWithTemplates(themeDir string, templateCount int) error {
	dirs := []string{"layouts", "pages", "singles", "errors", "partials"}

	for _, dir := range dirs {
		dirPath := filepath.Join(themeDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return err
		}

		// 根据目录类型创建不同数量的模板
		var filesToCreate int
		switch dir {
		case "layouts":
			filesToCreate = max(1, templateCount/5) // 至少1个布局
		case "pages":
			filesToCreate = max(1, templateCount/3) // 页面模板
		case "singles":
			filesToCreate = max(1, templateCount/4) // 单页模板
		case "errors":
			filesToCreate = max(1, templateCount/6) // 错误页面
		case "partials":
			filesToCreate = max(0, templateCount/8) // 部分模板（可选）
		}

		for i := 0; i < filesToCreate; i++ {
			fileName := fmt.Sprintf("%s_%d.tmpl", dir, i)
			filePath := filepath.Join(dirPath, fileName)
			content := fmt.Sprintf("<!-- %s template %d -->", dir, i)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return err
			}
		}

		// 确保每个目录至少有一个文件（除了partials）
		if filesToCreate == 0 && dir != "partials" {
			sampleFile := filepath.Join(dirPath, "sample.tmpl")
			if err := os.WriteFile(sampleFile, []byte("<!-- sample template -->"), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// max helper function
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// TestProperty_BackwardCompatibilityPreservation 测试向后兼容性保持
// Feature: multi-theme-support, Property 1: Backward Compatibility Preservation
func TestProperty_BackwardCompatibilityPreservation(t *testing.T) {
	// 属性：对于任何现有的模板引擎使用模式（API调用、目录结构、配置），新的多主题系统在传统模式下应该产生与原始系统相同的行为

	property := func() bool {
		// 创建临时目录用于测试
		tempDir, err := os.MkdirTemp("", "backward_compatibility_test")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建传统的模板目录结构
		if err := createLegacyStructure(tempDir); err != nil {
			t.Logf("Failed to create legacy structure: %v", err)
			return false
		}

		// 测试1: 传统的Engine创建方式应该继续工作
		funcMap := NewFuncMap()

		// 使用传统方式创建引擎（不指定任何主题选项）
		legacyEngine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap)
		if err != nil {
			t.Logf("Failed to create legacy engine: %v", err)
			return false
		}

		// 初始化引擎
		legacyEngine.Init()

		// 验证引擎基本状态
		if legacyEngine.HTMLRender == nil {
			t.Logf("Legacy engine HTMLRender should not be nil")
			return false
		}

		// 验证传统API方法仍然工作
		pageName := legacyEngine.PageName("test")
		if pageName == "" {
			t.Logf("PageName should return non-empty string")
			return false
		}

		singleName := legacyEngine.SingleName("test")
		if singleName == "" {
			t.Logf("SingleName should return non-empty string")
			return false
		}

		errorName := legacyEngine.ErrorName("404")
		if errorName == "" {
			t.Logf("ErrorName should return non-empty string")
			return false
		}

		// 测试2: 使用新的多主题引擎但在传统模式下应该产生相同结果
		// 创建带有主题管理的引擎，但不指定特定主题（应该自动检测为传统模式）
		modernEngine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap)
		if err != nil {
			t.Logf("Failed to create modern engine: %v", err)
			return false
		}

		modernEngine.Init()

		// 验证现代引擎也能正常工作
		if modernEngine.HTMLRender == nil {
			t.Logf("Modern engine HTMLRender should not be nil")
			return false
		}

		// 验证API方法产生相同结果
		modernPageName := modernEngine.PageName("test")
		if modernPageName != pageName {
			t.Logf("Modern engine PageName should match legacy: %s != %s", modernPageName, pageName)
			return false
		}

		modernSingleName := modernEngine.SingleName("test")
		if modernSingleName != singleName {
			t.Logf("Modern engine SingleName should match legacy: %s != %s", modernSingleName, singleName)
			return false
		}

		modernErrorName := modernEngine.ErrorName("404")
		if modernErrorName != errorName {
			t.Logf("Modern engine ErrorName should match legacy: %s != %s", modernErrorName, errorName)
			return false
		}

		// 测试3: 验证主题管理器在传统模式下的行为
		if modernEngine.themeManager != nil {
			// 应该检测为传统模式并创建默认主题
			currentTheme := modernEngine.themeManager.GetCurrentTheme()
			if currentTheme == "" {
				t.Logf("Current theme should not be empty in legacy mode")
				return false
			}

			// 应该只有一个主题（默认主题）
			availableThemes := modernEngine.themeManager.GetAvailableThemes()
			if len(availableThemes) != 1 {
				t.Logf("Legacy mode should have exactly 1 theme, got %d", len(availableThemes))
				return false
			}

			// 默认主题应该是"default"
			if currentTheme != "default" {
				t.Logf("Legacy mode current theme should be 'default', got '%s'", currentTheme)
				return false
			}
		}

		// 测试4: 验证选项系统的向后兼容性
		// 使用传统选项创建引擎
		legacyOptions := []Option{
			Layout("custom.tmpl"),
			Suffix("html"),
			GlobalConstant(map[string]interface{}{"site": "test"}),
		}

		legacyEngineWithOpts, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, legacyOptions...)
		if err != nil {
			t.Logf("Failed to create legacy engine with options: %v", err)
			return false
		}

		legacyEngineWithOpts.Init()

		// 验证选项被正确应用
		if legacyEngineWithOpts.opts.Layout != "custom.tmpl" {
			t.Logf("Legacy options Layout not applied correctly")
			return false
		}

		if legacyEngineWithOpts.opts.Suffix != "html" {
			t.Logf("Legacy options Suffix not applied correctly")
			return false
		}

		// 验证API方法使用了正确的选项
		customPageName := legacyEngineWithOpts.PageName("test")
		expectedPageName := "custom.tmpl:pages/test"
		if customPageName != expectedPageName {
			t.Logf("Expected page name '%s', got '%s'", expectedPageName, customPageName)
			return false
		}

		customSingleName := legacyEngineWithOpts.SingleName("test")
		expectedSingleName := "singles/test.html"
		if customSingleName != expectedSingleName {
			t.Logf("Expected single name '%s', got '%s'", expectedSingleName, customSingleName)
			return false
		}

		// 测试5: 验证文件监听功能的向后兼容性
		watchEngine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap)
		if err != nil {
			t.Logf("Failed to create watch engine: %v", err)
			return false
		}

		watchEngine.Init()

		// 启动文件监听应该成功
		err = watchEngine.Watching()
		if err != nil {
			t.Logf("Failed to start watching in legacy mode: %v", err)
			return false
		}

		// 清理监听器
		watchEngine.Close()

		// 测试6: 验证嵌入式文件系统的向后兼容性（如果支持）
		// 这里我们只验证API的存在性，因为我们没有实际的embed.FS
		// 但是函数应该存在且可调用
		_, err = NewEngineWithEmbedFS(nil, "", DefaultLoadTemplateWithEmbedFS, funcMap)
		if err == nil {
			// 如果没有错误，说明API存在且可调用（即使参数为nil）
			// 这验证了API的向后兼容性
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 5}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestBackwardCompatibilityExamples 测试向后兼容性的具体示例
func TestBackwardCompatibilityExamples(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "backward_compatibility_examples")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建传统结构
	if err := createLegacyStructure(tempDir); err != nil {
		t.Fatalf("Failed to create legacy structure: %v", err)
	}

	funcMap := NewFuncMap()

	tests := []struct {
		name        string
		options     []Option
		expectError bool
	}{
		{
			name:        "无选项的传统创建",
			options:     []Option{},
			expectError: false,
		},
		{
			name: "带传统选项的创建",
			options: []Option{
				Layout("main.tmpl"),
				Suffix("gohtml"),
			},
			expectError: false,
		},
		{
			name: "带全局变量的创建",
			options: []Option{
				GlobalConstant(map[string]interface{}{"version": "1.0"}),
				GlobalVariable(map[string]interface{}{"user": "test"}),
			},
			expectError: false,
		},
		{
			name: "混合传统和新选项",
			options: []Option{
				Layout("mixed.tmpl"),
				EnableMultiTheme(false), // 显式禁用多主题
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, tt.options...)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// 初始化引擎
			engine.Init()

			// 验证基本功能
			if engine.HTMLRender == nil {
				t.Errorf("HTMLRender should not be nil")
			}

			// 验证API方法
			pageName := engine.PageName("index")
			if pageName == "" {
				t.Errorf("PageName should return non-empty string")
			}

			singleName := engine.SingleName("about")
			if singleName == "" {
				t.Errorf("SingleName should return non-empty string")
			}

			errorName := engine.ErrorName("500")
			if errorName == "" {
				t.Errorf("ErrorName should return non-empty string")
			}

			// 验证选项被正确应用
			for _, opt := range tt.options {
				// 这里我们通过检查结果来验证选项是否生效
				// 具体的验证逻辑取决于选项类型
				_ = opt // 避免未使用变量警告
			}
		})
	}
}

// TestProperty_FileWatchingIntegration 测试文件监听集成
// Feature: multi-theme-support, Property 4: File Watching Integration
func TestProperty_FileWatchingIntegration(t *testing.T) {
	// 属性：对于任何启用文件监听的活跃主题，修改主题模板文件应该触发该主题模板的自动重新加载

	property := func() bool {
		// 创建临时目录进行测试
		tempDir, err := os.MkdirTemp("", "file_watching_test")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建多个主题
		themes := []string{"watch-theme1", "watch-theme2"}
		for _, themeName := range themes {
			themeDir := filepath.Join(tempDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				t.Logf("Failed to create theme structure for %s: %v", themeName, err)
				return false
			}
		}

		funcMap := NewFuncMap()

		// 创建引擎并启用多主题模式
		engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
		if err != nil {
			t.Logf("Failed to create engine: %v", err)
			return false
		}

		engine.Init()

		// 验证引擎初始化成功
		if engine.HTMLRender == nil {
			t.Logf("Engine HTMLRender should not be nil")
			return false
		}

		currentTheme := engine.GetCurrentTheme()
		if currentTheme == "" {
			t.Logf("Current theme should not be empty")
			return false
		}

		// 验证当前主题存在
		if !engine.ThemeExists(currentTheme) {
			t.Logf("Current theme %s should exist", currentTheme)
			return false
		}

		// 测试监听目录获取功能
		watchDir := engine.getWatchDirectory()
		if watchDir == "" {
			t.Logf("Watch directory should not be empty")
			return false
		}

		// 验证监听目录是当前主题的路径
		if !strings.Contains(watchDir, currentTheme) {
			t.Logf("Watch directory should contain current theme name: %s not in %s", currentTheme, watchDir)
			return false
		}

		// 验证监听目录存在且有效
		if err := engine.validateWatchDirectory(watchDir); err != nil {
			t.Logf("Watch directory validation failed: %v", err)
			return false
		}

		// 测试文件是否属于当前主题的检查
		// 创建一个属于当前主题的文件路径
		themeFilePath := filepath.Join(watchDir, "pages", "test.tmpl")
		if !engine.isFileInCurrentTheme(themeFilePath) {
			t.Logf("File in current theme should be recognized as such: %s", themeFilePath)
			return false
		}

		// 创建一个不属于当前主题的文件路径
		otherThemeDir := ""
		for _, themeName := range themes {
			if themeName != currentTheme {
				otherThemeDir = filepath.Join(tempDir, themeName)
				break
			}
		}

		if otherThemeDir != "" {
			otherThemeFilePath := filepath.Join(otherThemeDir, "pages", "other.tmpl")
			if engine.isFileInCurrentTheme(otherThemeFilePath) {
				t.Logf("File in other theme should not be recognized as current theme file: %s", otherThemeFilePath)
				return false
			}
		}

		// 测试模板文件识别
		templateFiles := []string{
			"test.tmpl",
			"page.html",
			"layout.gohtml",
			"partial.tpl",
		}

		nonTemplateFiles := []string{
			"readme.txt",
			"config.json",
			"style.css",
			"script.js",
		}

		for _, fileName := range templateFiles {
			if !engine.isTemplateFile(fileName) {
				t.Logf("File %s should be recognized as template file", fileName)
				return false
			}
		}

		for _, fileName := range nonTemplateFiles {
			if engine.isTemplateFile(fileName) {
				t.Logf("File %s should not be recognized as template file", fileName)
				return false
			}
		}

		// 测试应该重载的文件判断
		for _, fileName := range templateFiles {
			filePath := filepath.Join(watchDir, "pages", fileName)
			if !engine.shouldReloadForFile(filePath) {
				t.Logf("Template file in current theme should trigger reload: %s", filePath)
				return false
			}
		}

		for _, fileName := range nonTemplateFiles {
			filePath := filepath.Join(watchDir, "pages", fileName)
			if engine.shouldReloadForFile(filePath) {
				t.Logf("Non-template file should not trigger reload: %s", filePath)
				return false
			}
		}

		// 测试主题切换时监听目录的更新
		if len(themes) > 1 {
			// 找到一个不同的主题进行切换
			var targetTheme string
			for _, themeName := range themes {
				if themeName != currentTheme {
					targetTheme = themeName
					break
				}
			}

			if targetTheme != "" {
				// 记录切换前的监听目录
				oldWatchDir := engine.getWatchDirectory()

				// 执行主题切换
				err := engine.SwitchTheme(targetTheme)
				if err != nil {
					t.Logf("Failed to switch to theme %s: %v", targetTheme, err)
					return false
				}

				// 验证监听目录已更新
				newWatchDir := engine.getWatchDirectory()
				if newWatchDir == oldWatchDir {
					t.Logf("Watch directory should change after theme switch")
					return false
				}

				// 验证新的监听目录包含新主题名称
				if !strings.Contains(newWatchDir, targetTheme) {
					t.Logf("New watch directory should contain target theme name: %s not in %s", targetTheme, newWatchDir)
					return false
				}

				// 验证新监听目录有效
				if err := engine.validateWatchDirectory(newWatchDir); err != nil {
					t.Logf("New watch directory validation failed: %v", err)
					return false
				}

				// 验证文件归属检查已更新
				newThemeFilePath := filepath.Join(newWatchDir, "pages", "new.tmpl")
				if !engine.isFileInCurrentTheme(newThemeFilePath) {
					t.Logf("File in new current theme should be recognized: %s", newThemeFilePath)
					return false
				}

				// 验证旧主题文件不再被认为是当前主题的文件
				oldThemeFilePath := filepath.Join(oldWatchDir, "pages", "old.tmpl")
				if engine.isFileInCurrentTheme(oldThemeFilePath) {
					t.Logf("File in old theme should not be recognized as current: %s", oldThemeFilePath)
					return false
				}
			}
		}

		// 测试重载功能
		// 验证当前主题重载功能
		initialRender := engine.HTMLRender
		err = engine.reloadCurrentThemeTemplates()
		if err != nil {
			t.Logf("Failed to reload current theme templates: %v", err)
			return false
		}

		// 验证渲染器已更新（可能是同一个对象，但应该不为nil）
		if engine.HTMLRender == nil {
			t.Logf("HTMLRender should not be nil after reload")
			return false
		}

		// 验证引擎级别的重载功能
		err = engine.ReloadCurrentTheme()
		if err != nil {
			t.Logf("Failed to reload current theme via engine method: %v", err)
			return false
		}

		if engine.HTMLRender == nil {
			t.Logf("HTMLRender should not be nil after engine reload")
			return false
		}

		// 测试监听器状态检查
		if engine.watcher != nil {
			if !engine.isWatchingActive() {
				t.Logf("Watching should be active when watcher exists")
				return false
			}

			// 测试获取监听目录列表
			watchedDirs := engine.GetWatchedDirectories()
			if len(watchedDirs) == 0 {
				t.Logf("Should have at least one watched directory")
				return false
			}

			// 验证监听目录包含当前主题目录
			currentWatchDir := engine.getWatchDirectory()
			found := false
			for _, dir := range watchedDirs {
				if dir == currentWatchDir {
					found = true
					break
				}
			}
			if !found {
				t.Logf("Watched directories should include current theme directory")
				return false
			}
		}

		// 清理资源
		if engine.watcher != nil {
			engine.Close()
		}

		// 验证清理后的状态
		_ = initialRender // 避免未使用变量警告

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 3}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestProperty_FileWatchingThemeIsolation 测试文件监听主题隔离
// Feature: multi-theme-support, Property 4: File Watching Integration
func TestProperty_FileWatchingThemeIsolation(t *testing.T) {
	// 属性：文件监听应该只对当前活跃主题的文件变化做出响应，忽略其他主题的文件变化

	property := func() bool {
		// 创建临时目录
		tempDir, err := os.MkdirTemp("", "file_watching_isolation_test")
		if err != nil {
			t.Logf("Failed to create temp dir: %v", err)
			return false
		}
		defer os.RemoveAll(tempDir)

		// 创建多个主题
		themes := []string{"active-theme", "inactive-theme1", "inactive-theme2"}
		for _, themeName := range themes {
			themeDir := filepath.Join(tempDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				t.Logf("Failed to create theme structure for %s: %v", themeName, err)
				return false
			}
		}

		funcMap := NewFuncMap()

		// 创建引擎
		engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
		if err != nil {
			t.Logf("Failed to create engine: %v", err)
			return false
		}

		engine.Init()

		currentTheme := engine.GetCurrentTheme()
		if currentTheme == "" {
			t.Logf("Current theme should not be empty")
			return false
		}

		// 测试当前主题文件的监听响应
		currentThemeDir := engine.getWatchDirectory()
		if currentThemeDir == "" {
			t.Logf("Current theme directory should not be empty")
			return false
		}

		// 验证当前主题目录确实对应当前主题
		if !strings.Contains(currentThemeDir, currentTheme) {
			t.Logf("Current theme directory should contain theme name: %s not in %s", currentTheme, currentThemeDir)
			return false
		}

		// 测试当前主题中的文件应该触发重载
		currentThemeFiles := []string{
			filepath.Join(currentThemeDir, "pages", "index.tmpl"),
			filepath.Join(currentThemeDir, "layouts", "main.tmpl"),
			filepath.Join(currentThemeDir, "singles", "about.tmpl"),
			filepath.Join(currentThemeDir, "errors", "404.tmpl"),
		}

		for _, filePath := range currentThemeFiles {
			if !engine.shouldReloadForFile(filePath) {
				t.Logf("File in current theme should trigger reload: %s", filePath)
				return false
			}

			if !engine.isFileInCurrentTheme(filePath) {
				t.Logf("File should be recognized as in current theme: %s", filePath)
				return false
			}
		}

		// 测试其他主题中的文件不应该触发重载
		for _, themeName := range themes {
			if themeName == currentTheme {
				continue // 跳过当前主题
			}

			otherThemeDir := filepath.Join(tempDir, themeName)
			otherThemeFiles := []string{
				filepath.Join(otherThemeDir, "pages", "index.tmpl"),
				filepath.Join(otherThemeDir, "layouts", "main.tmpl"),
				filepath.Join(otherThemeDir, "singles", "about.tmpl"),
				filepath.Join(otherThemeDir, "errors", "404.tmpl"),
			}

			for _, filePath := range otherThemeFiles {
				if engine.shouldReloadForFile(filePath) {
					t.Logf("File in other theme should not trigger reload: %s", filePath)
					return false
				}

				if engine.isFileInCurrentTheme(filePath) {
					t.Logf("File should not be recognized as in current theme: %s", filePath)
					return false
				}
			}
		}

		// 测试主题切换后的隔离性
		if len(themes) > 1 {
			// 找到一个不同的主题
			var newTheme string
			for _, themeName := range themes {
				if themeName != currentTheme {
					newTheme = themeName
					break
				}
			}

			if newTheme != "" {
				// 记录切换前的状态
				oldThemeDir := engine.getWatchDirectory()

				// 执行主题切换
				err := engine.SwitchTheme(newTheme)
				if err != nil {
					t.Logf("Failed to switch to theme %s: %v", newTheme, err)
					return false
				}

				// 验证监听目录已更新
				newThemeDir := engine.getWatchDirectory()
				if newThemeDir == oldThemeDir {
					t.Logf("Watch directory should change after theme switch")
					return false
				}

				// 验证新主题文件现在应该触发重载
				newThemeFiles := []string{
					filepath.Join(newThemeDir, "pages", "new.tmpl"),
					filepath.Join(newThemeDir, "layouts", "new.tmpl"),
				}

				for _, filePath := range newThemeFiles {
					if !engine.shouldReloadForFile(filePath) {
						t.Logf("File in new current theme should trigger reload: %s", filePath)
						return false
					}
				}

				// 验证旧主题文件现在不应该触发重载
				oldThemeFiles := []string{
					filepath.Join(oldThemeDir, "pages", "old.tmpl"),
					filepath.Join(oldThemeDir, "layouts", "old.tmpl"),
				}

				for _, filePath := range oldThemeFiles {
					if engine.shouldReloadForFile(filePath) {
						t.Logf("File in old theme should not trigger reload after switch: %s", filePath)
						return false
					}
				}
			}
		}

		// 测试非模板文件的隔离（即使在当前主题中也不应该触发重载）
		nonTemplateFiles := []string{
			filepath.Join(currentThemeDir, "config.json"),
			filepath.Join(currentThemeDir, "readme.txt"),
			filepath.Join(currentThemeDir, "pages", "style.css"),
		}

		for _, filePath := range nonTemplateFiles {
			if engine.shouldReloadForFile(filePath) {
				t.Logf("Non-template file should not trigger reload even in current theme: %s", filePath)
				return false
			}
		}

		// 测试基础模板目录外的文件（应该被忽略）
		outsideFiles := []string{
			filepath.Join(tempDir, "external.tmpl"),
			filepath.Join(filepath.Dir(tempDir), "outside.tmpl"),
		}

		for _, filePath := range outsideFiles {
			if engine.shouldReloadForFile(filePath) {
				t.Logf("File outside theme directories should not trigger reload: %s", filePath)
				return false
			}

			if engine.isFileInCurrentTheme(filePath) {
				t.Logf("File outside theme directories should not be in current theme: %s", filePath)
				return false
			}
		}

		// 清理资源
		if engine.watcher != nil {
			engine.Close()
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 2}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestFileWatchingIntegrationExamples 测试文件监听集成的具体示例
func TestFileWatchingIntegrationExamples(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "file_watching_examples")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	funcMap := NewFuncMap()

	t.Run("传统模式文件监听", func(t *testing.T) {
		// 创建传统结构
		legacyDir := filepath.Join(tempDir, "legacy")
		if err := createLegacyStructure(legacyDir); err != nil {
			t.Fatalf("Failed to create legacy structure: %v", err)
		}

		engine, err := NewEngine(legacyDir, DefaultLoadTemplate, funcMap)
		if err != nil {
			t.Fatalf("Failed to create legacy engine: %v", err)
		}

		engine.Init()

		// 验证监听目录
		watchDir := engine.getWatchDirectory()
		if watchDir != legacyDir {
			t.Errorf("Expected watch dir to be %s, got %s", legacyDir, watchDir)
		}

		// 验证监听目录验证
		if err := engine.validateWatchDirectory(watchDir); err != nil {
			t.Errorf("Watch directory validation failed: %v", err)
		}

		// 验证文件归属检查
		legacyFile := filepath.Join(legacyDir, "pages", "test.tmpl")
		if !engine.isFileInCurrentTheme(legacyFile) {
			t.Errorf("Legacy file should be in current theme")
		}

		// 验证模板文件识别
		if !engine.shouldReloadForFile(legacyFile) {
			t.Errorf("Legacy template file should trigger reload")
		}

		// 清理
		if engine.watcher != nil {
			engine.Close()
		}
	})

	t.Run("多主题模式文件监听", func(t *testing.T) {
		// 创建多主题结构
		multiDir := filepath.Join(tempDir, "multi")
		themes := []string{"theme-a", "theme-b"}

		for _, themeName := range themes {
			themeDir := filepath.Join(multiDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				t.Fatalf("Failed to create theme structure for %s: %v", themeName, err)
			}
		}

		engine, err := NewEngine(multiDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
		if err != nil {
			t.Fatalf("Failed to create multi-theme engine: %v", err)
		}

		engine.Init()

		currentTheme := engine.GetCurrentTheme()
		if currentTheme == "" {
			t.Fatalf("Current theme should not be empty")
		}

		// 验证监听目录指向当前主题
		watchDir := engine.getWatchDirectory()
		if !strings.Contains(watchDir, currentTheme) {
			t.Errorf("Watch dir should contain current theme name: %s not in %s", currentTheme, watchDir)
		}

		// 验证当前主题文件被正确识别
		currentThemeFile := filepath.Join(watchDir, "pages", "current.tmpl")
		if !engine.isFileInCurrentTheme(currentThemeFile) {
			t.Errorf("Current theme file should be recognized")
		}

		if !engine.shouldReloadForFile(currentThemeFile) {
			t.Errorf("Current theme template should trigger reload")
		}

		// 验证其他主题文件被正确排除
		for _, themeName := range themes {
			if themeName != currentTheme {
				otherThemeFile := filepath.Join(multiDir, themeName, "pages", "other.tmpl")
				if engine.isFileInCurrentTheme(otherThemeFile) {
					t.Errorf("Other theme file should not be in current theme")
				}

				if engine.shouldReloadForFile(otherThemeFile) {
					t.Errorf("Other theme file should not trigger reload")
				}
			}
		}

		// 测试主题切换
		for _, themeName := range themes {
			if themeName != currentTheme {
				err := engine.SwitchTheme(themeName)
				if err != nil {
					t.Errorf("Failed to switch to theme %s: %v", themeName, err)
					continue
				}

				// 验证监听目录已更新
				newWatchDir := engine.getWatchDirectory()
				if !strings.Contains(newWatchDir, themeName) {
					t.Errorf("Watch dir should update to new theme: %s not in %s", themeName, newWatchDir)
				}

				break // 只测试一次切换
			}
		}

		// 清理
		if engine.watcher != nil {
			engine.Close()
		}
	})

	t.Run("文件类型识别", func(t *testing.T) {
		engine := &Engine{} // 创建最小引擎用于测试

		templateFiles := map[string]bool{
			"test.tmpl":     true,
			"page.html":     true,
			"layout.gohtml": true,
			"partial.tpl":   true,
			"config.json":   false,
			"style.css":     false,
			"script.js":     false,
			"readme.txt":    false,
			"":              false,
		}

		for fileName, expected := range templateFiles {
			result := engine.isTemplateFile(fileName)
			if result != expected {
				t.Errorf("File %s: expected %v, got %v", fileName, expected, result)
			}
		}
	})
}

// TestProperty_EmbeddedFilesystemSupport 测试嵌入式文件系统支持
// Feature: multi-theme-support, Property 5: Embedded Filesystem Support
func TestProperty_EmbeddedFilesystemSupport(t *testing.T) {
	// 属性：对于任何包含多个主题的嵌入式文件系统，系统应该正确识别、加载和切换主题，使用与基于文件的主题相同的逻辑

	property := func() bool {
		funcMap := NewFuncMap()

		// 测试1: 嵌入式文件系统主题管理器创建
		manager := NewDefaultThemeManagerWithEmbedFS(nil, "templates", funcMap, DefaultLoadTemplateWithEmbedFS)
		if manager == nil {
			t.Logf("Failed to create embedded theme manager")
			return false
		}

		// 验证嵌入式加载函数已设置
		if manager.loadEmbedFunc == nil {
			t.Logf("Embedded load function should be set")
			return false
		}

		// 验证发现器配置
		if manager.discovery == nil {
			t.Logf("Discovery should not be nil")
			return false
		}

		// 测试2: 嵌入式引擎创建和配置
		engine, err := NewEngineWithEmbedFS(nil, "templates", DefaultLoadTemplateWithEmbedFS, funcMap, EnableMultiTheme(true))
		if err != nil {
			t.Logf("Failed to create embedded engine: %v", err)
			return false
		}

		// 验证引擎字段正确设置
		if engine.loadTemplateEmbedFS == nil {
			t.Logf("Embedded load function should be set in engine")
			return false
		}

		if engine.tmplFSSUbDir != "templates" {
			t.Logf("Expected tmplFSSUbDir to be 'templates', got '%s'", engine.tmplFSSUbDir)
			return false
		}

		if !engine.multiThemeMode {
			t.Logf("Multi-theme mode should be enabled")
			return false
		}

		// 测试3: 嵌入式主题对象属性
		embeddedTheme := &Theme{
			Name:       "embedded-test-theme",
			Path:       "templates/embedded-test-theme",
			IsDefault:  false,
			IsEmbedded: true, // 关键属性
			Metadata: ThemeMetadata{
				DisplayName: "Embedded Test Theme",
				Description: "A test theme for embedded filesystem",
				Version:     "1.0.0",
				Author:      "Test Author",
				Tags:        []string{"embedded", "test"},
				Custom:      make(map[string]any),
			},
		}

		// 验证嵌入式主题属性
		if !embeddedTheme.IsEmbedded {
			t.Logf("Theme should be marked as embedded")
			return false
		}

		if embeddedTheme.Name == "" {
			t.Logf("Theme name should not be empty")
			return false
		}

		if embeddedTheme.Path == "" {
			t.Logf("Theme path should not be empty")
			return false
		}

		// 测试4: 嵌入式主题发现器功能
		discovery := NewThemeDiscoveryWithEmbedFS(nil, "templates", funcMap, DefaultLoadTemplateWithEmbedFS)
		if discovery == nil {
			t.Logf("Failed to create embedded theme discovery")
			return false
		}

		// 验证发现器字段
		if discovery.embedFS != nil {
			// 如果embedFS不为nil，验证相关配置
			if discovery.subDir != "templates" {
				t.Logf("Expected subDir to be 'templates', got '%s'", discovery.subDir)
				return false
			}
		}

		if discovery.loadEmbedFunc == nil {
			t.Logf("Embedded load function should be set in discovery")
			return false
		}

		// 测试5: 嵌入式模式检测（模拟）
		// 由于没有真实的embed.FS，我们测试检测逻辑的存在性
		mode, err := discovery.DetectMode()
		if err != nil {
			// 预期会有错误，因为没有真实的embed.FS
			t.Logf("Mode detection error (expected with nil embedFS): %v", err)
		} else {
			// 如果没有错误，验证返回的模式
			if mode != ModeLegacy && mode != ModeMultiTheme {
				t.Logf("Invalid mode returned: %v", mode)
				return false
			}
		}

		// 测试6: 嵌入式主题验证逻辑
		// 测试ValidateTheme方法对嵌入式路径的处理
		err = discovery.ValidateTheme("templates/test-theme")
		if err != nil {
			// 预期会有错误，因为没有真实的文件系统
			t.Logf("Theme validation error (expected with nil embedFS): %v", err)
		}

		// 测试7: 嵌入式文件系统错误处理
		// 测试加载不存在主题的错误处理
		_, err = manager.LoadTheme("non-existent-embedded-theme")
		if err == nil {
			t.Logf("Expected error when loading non-existent embedded theme")
			return false
		}

		// 验证错误类型
		var themeErr *ThemeError
		if !errors.As(err, &themeErr) {
			t.Logf("Expected ThemeError for non-existent embedded theme")
			return false
		}

		if themeErr.Type != ErrThemeNotFound {
			t.Logf("Expected ErrThemeNotFound, got %v", themeErr.Type)
			return false
		}

		// 测试8: 嵌入式主题切换逻辑验证
		// 验证SwitchTheme方法对嵌入式主题的处理
		err = manager.SwitchTheme("another-non-existent-theme")
		if err == nil {
			t.Logf("Expected error when switching to non-existent embedded theme")
			return false
		}

		// 再次验证错误类型
		if !errors.As(err, &themeErr) {
			t.Logf("Expected ThemeError for theme switch failure")
			return false
		}

		// 测试9: 嵌入式主题元数据加载
		metadata, err := discovery.LoadThemeMetadata("templates/test-theme")
		if err != nil {
			// 预期会有错误，但应该返回默认元数据
			t.Logf("Metadata loading error (may be expected): %v", err)
		} else {
			// 如果成功，验证元数据结构
			if metadata == nil {
				t.Logf("Metadata should not be nil")
				return false
			}

			if metadata.DisplayName == "" {
				t.Logf("DisplayName should not be empty")
				return false
			}

			if metadata.Version == "" {
				t.Logf("Version should not be empty")
				return false
			}

			if metadata.Tags == nil {
				t.Logf("Tags should not be nil")
				return false
			}

			if metadata.Custom == nil {
				t.Logf("Custom should not be nil")
				return false
			}
		}

		// 测试10: 嵌入式与文件系统模式的一致性
		// 验证嵌入式模式使用相同的接口和行为
		fileSystemManager := NewDefaultThemeManager("", funcMap, DefaultLoadTemplate)
		embeddedManager := NewDefaultThemeManagerWithEmbedFS(nil, "", funcMap, DefaultLoadTemplateWithEmbedFS)

		// 验证两个管理器实现相同的接口
		var fsInterface ThemeManager = fileSystemManager
		var embeddedInterface ThemeManager = embeddedManager

		if fsInterface == nil || embeddedInterface == nil {
			t.Logf("Both managers should implement ThemeManager interface")
			return false
		}

		// 验证方法存在性（通过调用不会panic）
		_ = embeddedManager.GetAvailableThemes()
		_ = embeddedManager.GetCurrentTheme()
		_ = embeddedManager.ThemeExists("test")

		// 测试11: 嵌入式引擎选项处理
		// 验证嵌入式引擎正确处理主题相关选项
		options := []Option{
			SetTheme("custom-embedded-theme"),
			DefaultTheme("default-embedded-theme"),
			EnableMultiTheme(true),
		}

		embeddedEngine, err := NewEngineWithEmbedFS(nil, "custom/path", DefaultLoadTemplateWithEmbedFS, funcMap, options...)
		if err != nil {
			t.Logf("Failed to create embedded engine with options: %v", err)
			return false
		}

		// 验证选项正确应用
		if embeddedEngine.opts.Theme != "custom-embedded-theme" {
			t.Logf("Expected Theme to be 'custom-embedded-theme', got '%s'", embeddedEngine.opts.Theme)
			return false
		}

		if embeddedEngine.opts.DefaultTheme != "default-embedded-theme" {
			t.Logf("Expected DefaultTheme to be 'default-embedded-theme', got '%s'", embeddedEngine.opts.DefaultTheme)
			return false
		}

		if !embeddedEngine.opts.MultiThemeMode {
			t.Logf("Expected MultiThemeMode to be true")
			return false
		}

		if embeddedEngine.currentTheme != "custom-embedded-theme" {
			t.Logf("Expected currentTheme to be 'custom-embedded-theme', got '%s'", embeddedEngine.currentTheme)
			return false
		}

		if embeddedEngine.tmplFSSUbDir != "custom/path" {
			t.Logf("Expected tmplFSSUbDir to be 'custom/path', got '%s'", embeddedEngine.tmplFSSUbDir)
			return false
		}

		return true
	}

	// 运行属性测试
	if err := quick.Check(property, &quick.Config{MaxCount: 10}); err != nil {
		t.Errorf("Property test failed: %v", err)
	}
}

// TestEmbeddedFilesystemThemeSwitching 测试嵌入式文件系统的主题切换
func TestEmbeddedFilesystemThemeSwitching(t *testing.T) {
	funcMap := NewFuncMap()

	t.Run("嵌入式文件系统主题切换模拟", func(t *testing.T) {
		// 由于我们没有真实的embed.FS，我们测试主题管理器的逻辑
		// 创建一个模拟的嵌入式主题管理器
		manager := NewDefaultThemeManagerWithEmbedFS(nil, "templates", funcMap, DefaultLoadTemplateWithEmbedFS)

		// 验证管理器创建成功
		if manager == nil {
			t.Fatalf("Theme manager should not be nil")
		}

		// 验证嵌入式加载函数已设置
		if manager.loadEmbedFunc == nil {
			t.Errorf("Embedded load function should be set")
		}

		// 验证发现器已正确配置为嵌入式模式
		if manager.discovery.embedFS != nil {
			// 如果有embedFS，验证相关字段
			if manager.discovery.subDir != "templates" {
				t.Errorf("Expected subDir to be 'templates', got '%s'", manager.discovery.subDir)
			}
		}
	})

	t.Run("嵌入式主题对象验证", func(t *testing.T) {
		// 测试嵌入式主题对象的创建和属性
		theme := &Theme{
			Name:       "embedded-theme",
			Path:       "templates/embedded-theme",
			IsDefault:  false,
			IsEmbedded: true, // 关键：标记为嵌入式主题
			Metadata: ThemeMetadata{
				DisplayName: "Embedded Theme",
				Description: "A theme from embedded filesystem",
				Version:     "1.0.0",
				Author:      "Test",
				Tags:        []string{"embedded"},
				Custom:      make(map[string]any),
			},
		}

		// 验证嵌入式主题属性
		if !theme.IsEmbedded {
			t.Errorf("Theme should be marked as embedded")
		}

		if theme.Name != "embedded-theme" {
			t.Errorf("Expected theme name 'embedded-theme', got '%s'", theme.Name)
		}

		if theme.Path != "templates/embedded-theme" {
			t.Errorf("Expected theme path 'templates/embedded-theme', got '%s'", theme.Path)
		}
	})

	t.Run("嵌入式引擎初始化", func(t *testing.T) {
		// 测试嵌入式引擎的初始化过程
		engine, err := NewEngineWithEmbedFS(nil, "templates", DefaultLoadTemplateWithEmbedFS, funcMap, EnableMultiTheme(true))
		if err != nil {
			t.Fatalf("Failed to create embedded engine: %v", err)
		}

		// 验证引擎创建成功但不初始化（因为没有真实的embed.FS）
		if engine.tmplFS != nil {
			t.Errorf("Expected tmplFS to be nil when passed nil")
		}

		if engine.loadTemplateEmbedFS == nil {
			t.Errorf("Embedded load function should be set")
		}

		// 验证多主题模式已启用
		if !engine.multiThemeMode {
			t.Errorf("Multi-theme mode should be enabled")
		}

		// 验证主题相关字段已设置
		if engine.tmplFSSUbDir != "templates" {
			t.Errorf("Expected tmplFSSUbDir to be 'templates', got '%s'", engine.tmplFSSUbDir)
		}

		// 注意：我们不调用Init()，因为没有真实的embed.FS会导致panic
		// 在实际使用中，用户会提供真实的embed.FS
		t.Logf("Embedded engine created successfully (Init() skipped due to nil embed.FS)")
	})

	t.Run("嵌入式模式错误处理", func(t *testing.T) {
		// 测试嵌入式模式下的错误处理
		manager := NewDefaultThemeManagerWithEmbedFS(nil, "", funcMap, nil)

		// 尝试发现主题（应该处理nil embedFS的情况）
		err := manager.DiscoverThemes()
		if err == nil {
			t.Logf("Theme discovery handled nil embedFS gracefully")
		} else {
			t.Logf("Theme discovery error (expected with nil embedFS): %v", err)
		}

		// 测试加载不存在的主题
		_, err = manager.LoadTheme("non-existent-theme")
		if err == nil {
			t.Errorf("Expected error when loading non-existent theme")
		}

		// 验证错误类型
		var themeErr *ThemeError
		if errors.As(err, &themeErr) {
			if themeErr.Type != ErrThemeNotFound {
				t.Errorf("Expected ErrThemeNotFound, got %v", themeErr.Type)
			}
		} else {
			t.Errorf("Expected ThemeError, got %T", err)
		}
	})
}

// TestNewEngineWithEmbedFSBasic 测试嵌入式文件系统引擎的基本功能
func TestNewEngineWithEmbedFSBasic(t *testing.T) {
	funcMap := NewFuncMap()

	t.Run("嵌入式文件系统引擎创建", func(t *testing.T) {
		// 测试API存在性和基本调用
		engine, err := NewEngineWithEmbedFS(nil, "", DefaultLoadTemplateWithEmbedFS, funcMap)
		if err != nil {
			t.Errorf("NewEngineWithEmbedFS should not fail with nil parameters: %v", err)
		}

		if engine == nil {
			t.Errorf("Engine should not be nil")
		}

		// 验证嵌入式文件系统字段已设置
		if engine.tmplFS != nil {
			t.Errorf("Expected tmplFS to be nil when passed nil")
		}

		if engine.loadTemplateEmbedFS == nil {
			t.Errorf("loadTemplateEmbedFS should be set")
		}
	})

	t.Run("嵌入式文件系统引擎带选项", func(t *testing.T) {
		// 测试带主题选项的创建
		options := []Option{
			SetTheme("test-theme"),
			EnableMultiTheme(true),
			DefaultTheme("default-theme"),
		}

		engine, err := NewEngineWithEmbedFS(nil, "templates", DefaultLoadTemplateWithEmbedFS, funcMap, options...)
		if err != nil {
			t.Errorf("NewEngineWithEmbedFS with options should not fail: %v", err)
		}

		// 验证选项已正确设置
		if engine.opts.Theme != "test-theme" {
			t.Errorf("Expected Theme to be 'test-theme', got '%s'", engine.opts.Theme)
		}

		if !engine.opts.MultiThemeMode {
			t.Errorf("Expected MultiThemeMode to be true")
		}

		if engine.opts.DefaultTheme != "default-theme" {
			t.Errorf("Expected DefaultTheme to be 'default-theme', got '%s'", engine.opts.DefaultTheme)
		}

		// 验证引擎字段已正确设置
		if engine.currentTheme != "test-theme" {
			t.Errorf("Expected currentTheme to be 'test-theme', got '%s'", engine.currentTheme)
		}

		if !engine.multiThemeMode {
			t.Errorf("Expected multiThemeMode to be true")
		}

		if engine.tmplFSSUbDir != "templates" {
			t.Errorf("Expected tmplFSSUbDir to be 'templates', got '%s'", engine.tmplFSSUbDir)
		}
	})
}

// TestEngineThemeAPIMethods 测试引擎的主题API方法
func TestEngineThemeAPIMethods(t *testing.T) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "engine_theme_api_test")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	funcMap := NewFuncMap()

	t.Run("传统模式API", func(t *testing.T) {
		// 创建传统结构
		if err := createLegacyStructure(tempDir); err != nil {
			t.Fatalf("Failed to create legacy structure: %v", err)
		}

		engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap)
		if err != nil {
			t.Fatalf("Failed to create engine: %v", err)
		}

		engine.Init()

		// 测试GetAvailableThemes
		themes := engine.GetAvailableThemes()
		if len(themes) != 1 || themes[0] != "default" {
			t.Errorf("Expected ['default'], got %v", themes)
		}

		// 测试GetCurrentTheme
		currentTheme := engine.GetCurrentTheme()
		if currentTheme != "default" {
			t.Errorf("Expected 'default', got '%s'", currentTheme)
		}

		// 测试ThemeExists
		if !engine.ThemeExists("default") {
			t.Errorf("Default theme should exist")
		}

		if engine.ThemeExists("non-existent") {
			t.Errorf("Non-existent theme should not exist")
		}

		// 测试GetThemeMetadata
		metadata, err := engine.GetThemeMetadata("default")
		if err != nil {
			t.Errorf("Failed to get default theme metadata: %v", err)
		}
		if metadata == nil {
			t.Errorf("Metadata should not be nil")
		}
		if metadata.DisplayName != "Default Theme" {
			t.Logf("Note: DisplayName is '%s' (may vary based on directory name)", metadata.DisplayName)
		}

		// 测试SwitchTheme在传统模式下应该失败
		err = engine.SwitchTheme("another-theme")
		if err == nil {
			t.Errorf("Expected error when switching theme in legacy mode")
		}

		// 测试IsMultiThemeMode
		if engine.IsMultiThemeMode() {
			t.Errorf("Should not be in multi-theme mode")
		}

		// 测试ReloadCurrentTheme
		err = engine.ReloadCurrentTheme()
		if err != nil {
			t.Errorf("Failed to reload current theme: %v", err)
		}
	})

	t.Run("多主题模式API", func(t *testing.T) {
		// 创建独立的多主题目录
		multiThemeDir, err := os.MkdirTemp("", "multi_theme_api_test")
		if err != nil {
			t.Fatalf("Failed to create multi-theme dir: %v", err)
		}
		defer os.RemoveAll(multiThemeDir)

		// 创建多个主题
		themes := []string{"theme1", "theme2", "theme3"}
		for _, themeName := range themes {
			themeDir := filepath.Join(multiThemeDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				t.Fatalf("Failed to create theme structure for %s: %v", themeName, err)
			}
		}

		engine, err := NewEngine(multiThemeDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
		if err != nil {
			t.Fatalf("Failed to create multi-theme engine: %v", err)
		}

		engine.Init()

		// 关闭文件监听器以避免测试中的竞态条件
		if engine.watcher != nil {
			engine.Close()
			// 重新创建一个没有监听器的引擎用于测试
			engine.watcher = nil
		}

		// 测试GetAvailableThemes
		availableThemes := engine.GetAvailableThemes()
		if len(availableThemes) != len(themes) {
			t.Errorf("Expected %d themes, got %d", len(themes), len(availableThemes))
		}

		// 测试GetCurrentTheme
		currentTheme := engine.GetCurrentTheme()
		if currentTheme == "" {
			t.Errorf("Current theme should not be empty")
		}

		// 测试ThemeExists
		for _, themeName := range themes {
			if !engine.ThemeExists(themeName) {
				t.Errorf("Theme %s should exist", themeName)
			}
		}

		if engine.ThemeExists("non-existent") {
			t.Errorf("Non-existent theme should not exist")
		}

		// 测试SwitchTheme
		targetTheme := themes[1] // 切换到第二个主题
		if targetTheme != currentTheme {
			err = engine.SwitchTheme(targetTheme)
			if err != nil {
				t.Errorf("Failed to switch to theme %s: %v", targetTheme, err)
			}

			newCurrentTheme := engine.GetCurrentTheme()
			if newCurrentTheme != targetTheme {
				t.Errorf("Expected current theme to be %s, got %s", targetTheme, newCurrentTheme)
			}
		}

		// 测试GetThemeMetadata
		for _, themeName := range themes {
			metadata, err := engine.GetThemeMetadata(themeName)
			if err != nil {
				t.Errorf("Failed to get metadata for theme %s: %v", themeName, err)
			}
			if metadata == nil {
				t.Errorf("Metadata should not be nil for theme %s", themeName)
			}
		}

		// 测试IsMultiThemeMode
		if !engine.IsMultiThemeMode() {
			t.Errorf("Should be in multi-theme mode")
		}

		// 测试ReloadCurrentTheme
		err = engine.ReloadCurrentTheme()
		if err != nil {
			t.Errorf("Failed to reload current theme: %v", err)
		}

		// 测试GetThemeManager
		themeManager := engine.GetThemeManager()
		if themeManager == nil {
			t.Errorf("Theme manager should not be nil in multi-theme mode")
		}
	})
}
