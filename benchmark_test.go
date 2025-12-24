package template

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

// BenchmarkLegacyModePerformance 基准测试传统模式性能
func BenchmarkLegacyModePerformance(b *testing.B) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "benchmark_legacy_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建传统结构
	if err := createLegacyStructure(tempDir); err != nil {
		b.Fatalf("Failed to create legacy structure: %v", err)
	}

	funcMap := NewFuncMap()

	b.Run("EngineCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap)
			if err != nil {
				b.Fatalf("Failed to create engine: %v", err)
			}
			engine.Init()
			engine.Close()
		}
	})

	// 创建一个引擎用于后续测试
	engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap)
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	engine.Init()
	defer engine.Close()

	b.Run("PageRendering", func(b *testing.B) {
		data := H{
			"title":   "Benchmark Test",
			"content": "This is benchmark content",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := engine.RenderPage(&buf, "sample", data)
			if err != nil {
				b.Fatalf("Failed to render page: %v", err)
			}
		}
	})

	b.Run("SingleRendering", func(b *testing.B) {
		data := H{
			"title": "Benchmark Single",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := engine.RenderSingle(&buf, "sample", data)
			if err != nil {
				b.Fatalf("Failed to render single: %v", err)
			}
		}
	})

	b.Run("ErrorRendering", func(b *testing.B) {
		data := H{
			"title":   "404 Error",
			"message": "Page not found",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := engine.RenderError(&buf, "sample", data)
			if err != nil {
				b.Fatalf("Failed to render error: %v", err)
			}
		}
	})

	b.Run("ThemeAPIQueries", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = engine.GetAvailableThemes()
			_ = engine.GetCurrentTheme()
			_ = engine.ThemeExists("default")
		}
	})
}

// BenchmarkMultiThemeModePerformance 基准测试多主题模式性能
func BenchmarkMultiThemeModePerformance(b *testing.B) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "benchmark_multi_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建多个主题
	themes := []string{"default", "dark", "colorful"}
	for _, themeName := range themes {
		themeDir := filepath.Join(tempDir, themeName)
		if err := createThemeStructure(themeDir); err != nil {
			b.Fatalf("Failed to create theme structure for %s: %v", themeName, err)
		}
	}

	funcMap := NewFuncMap()

	b.Run("EngineCreation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
			if err != nil {
				b.Fatalf("Failed to create engine: %v", err)
			}
			engine.Init()
			engine.Close()
		}
	})

	// 创建一个引擎用于后续测试
	engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	engine.Init()
	defer engine.Close()

	b.Run("PageRendering", func(b *testing.B) {
		data := H{
			"title":   "Benchmark Test",
			"content": "This is benchmark content",
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := engine.RenderPage(&buf, "sample", data)
			if err != nil {
				b.Fatalf("Failed to render page: %v", err)
			}
		}
	})

	b.Run("ThemeSwitching", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			targetTheme := themes[i%len(themes)]
			err := engine.SwitchTheme(targetTheme)
			if err != nil {
				b.Fatalf("Failed to switch to theme %s: %v", targetTheme, err)
			}
		}
	})

	b.Run("ThemeAPIQueries", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = engine.GetAvailableThemes()
			_ = engine.GetCurrentTheme()
			_ = engine.ThemeExists("default")
			_, _ = engine.GetThemeMetadata("default")
		}
	})

	b.Run("ThemeDiscovery", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			manager := NewDefaultThemeManager(tempDir, funcMap, DefaultLoadTemplate)
			err := manager.DiscoverThemes()
			if err != nil {
				b.Fatalf("Failed to discover themes: %v", err)
			}
		}
	})

	b.Run("ThemeValidation", func(b *testing.B) {
		discovery := NewThemeDiscovery(tempDir, funcMap, DefaultLoadTemplate)
		themeDir := filepath.Join(tempDir, "default")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			err := discovery.ValidateTheme(themeDir)
			if err != nil {
				b.Fatalf("Failed to validate theme: %v", err)
			}
		}
	})
}

// BenchmarkPerformanceComparison 性能对比基准测试
func BenchmarkPerformanceComparison(b *testing.B) {
	// 创建两个独立的临时目录
	legacyDir, err := os.MkdirTemp("", "benchmark_legacy_comp_*")
	if err != nil {
		b.Fatalf("Failed to create legacy dir: %v", err)
	}
	defer os.RemoveAll(legacyDir)

	multiDir, err := os.MkdirTemp("", "benchmark_multi_comp_*")
	if err != nil {
		b.Fatalf("Failed to create multi dir: %v", err)
	}
	defer os.RemoveAll(multiDir)

	// 创建相同的模板结构
	if err := createLegacyStructure(legacyDir); err != nil {
		b.Fatalf("Failed to create legacy structure: %v", err)
	}

	// 在多主题目录中创建一个默认主题，内容与传统模式相同
	defaultThemeDir := filepath.Join(multiDir, "default")
	if err := createThemeStructure(defaultThemeDir); err != nil {
		b.Fatalf("Failed to create default theme structure: %v", err)
	}

	funcMap := NewFuncMap()

	// 创建两个引擎
	legacyEngine, err := NewEngine(legacyDir, DefaultLoadTemplate, funcMap)
	if err != nil {
		b.Fatalf("Failed to create legacy engine: %v", err)
	}
	legacyEngine.Init()
	defer legacyEngine.Close()

	multiEngine, err := NewEngine(multiDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
	if err != nil {
		b.Fatalf("Failed to create multi-theme engine: %v", err)
	}
	multiEngine.Init()
	defer multiEngine.Close()

	data := H{
		"title":   "Performance Comparison",
		"content": "Testing performance between legacy and multi-theme modes",
	}

	b.Run("LegacyMode_PageRendering", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := legacyEngine.RenderPage(&buf, "sample", data)
			if err != nil {
				b.Fatalf("Failed to render page in legacy mode: %v", err)
			}
		}
	})

	b.Run("MultiThemeMode_PageRendering", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var buf bytes.Buffer
			err := multiEngine.RenderPage(&buf, "sample", data)
			if err != nil {
				b.Fatalf("Failed to render page in multi-theme mode: %v", err)
			}
		}
	})

	b.Run("LegacyMode_ThemeQueries", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = legacyEngine.GetAvailableThemes()
			_ = legacyEngine.GetCurrentTheme()
			_ = legacyEngine.ThemeExists("default")
		}
	})

	b.Run("MultiThemeMode_ThemeQueries", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			_ = multiEngine.GetAvailableThemes()
			_ = multiEngine.GetCurrentTheme()
			_ = multiEngine.ThemeExists("default")
		}
	})
}

// BenchmarkMemoryUsage 内存使用基准测试
func BenchmarkMemoryUsage(b *testing.B) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "benchmark_memory_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建多个不同大小的主题
	smallThemes := []string{"small1", "small2"}
	largeThemes := []string{"large1", "large2"}

	// 创建小主题（少量模板）
	for _, themeName := range smallThemes {
		themeDir := filepath.Join(tempDir, themeName)
		if err := createThemeStructureWithTemplates(themeDir, 3); err != nil {
			b.Fatalf("Failed to create small theme %s: %v", themeName, err)
		}
	}

	// 创建大主题（大量模板）
	for _, themeName := range largeThemes {
		themeDir := filepath.Join(tempDir, themeName)
		if err := createThemeStructureWithTemplates(themeDir, 20); err != nil {
			b.Fatalf("Failed to create large theme %s: %v", themeName, err)
		}
	}

	funcMap := NewFuncMap()

	b.Run("SmallTheme_MemoryUsage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap,
				EnableMultiTheme(true),
				SetTheme("small1"))
			if err != nil {
				b.Fatalf("Failed to create engine: %v", err)
			}
			engine.Init()

			// 执行一些操作来加载模板
			var buf bytes.Buffer
			data := H{"title": "Memory Test"}
			_ = engine.RenderPage(&buf, "sample", data)

			engine.Close()
		}
	})

	b.Run("LargeTheme_MemoryUsage", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap,
				EnableMultiTheme(true),
				SetTheme("large1"))
			if err != nil {
				b.Fatalf("Failed to create engine: %v", err)
			}
			engine.Init()

			// 执行一些操作来加载模板
			var buf bytes.Buffer
			data := H{"title": "Memory Test"}
			_ = engine.RenderPage(&buf, "sample", data)

			engine.Close()
		}
	})

	b.Run("ThemeSwitching_MemoryUsage", func(b *testing.B) {
		engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
		if err != nil {
			b.Fatalf("Failed to create engine: %v", err)
		}
		engine.Init()
		defer engine.Close()

		themes := append(smallThemes, largeThemes...)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			targetTheme := themes[i%len(themes)]
			err := engine.SwitchTheme(targetTheme)
			if err != nil {
				b.Fatalf("Failed to switch to theme %s: %v", targetTheme, err)
			}

			// 执行渲染操作
			var buf bytes.Buffer
			data := H{"title": "Switch Test"}
			_ = engine.RenderPage(&buf, "sample", data)
		}
	})
}

// BenchmarkConcurrentAccess 并发访问基准测试
func BenchmarkConcurrentAccess(b *testing.B) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "benchmark_concurrent_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建多个主题
	themes := []string{"theme1", "theme2", "theme3"}
	for _, themeName := range themes {
		themeDir := filepath.Join(tempDir, themeName)
		if err := createThemeStructure(themeDir); err != nil {
			b.Fatalf("Failed to create theme structure for %s: %v", themeName, err)
		}
	}

	funcMap := NewFuncMap()
	engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	engine.Init()
	defer engine.Close()

	data := H{
		"title":   "Concurrent Test",
		"content": "Testing concurrent access",
	}

	b.Run("ConcurrentRendering", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				var buf bytes.Buffer
				err := engine.RenderPage(&buf, "sample", data)
				if err != nil {
					b.Fatalf("Failed to render page: %v", err)
				}
			}
		})
	})

	b.Run("ConcurrentThemeQueries", func(b *testing.B) {
		b.ResetTimer()
		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				_ = engine.GetAvailableThemes()
				_ = engine.GetCurrentTheme()
				_ = engine.ThemeExists("theme1")
			}
		})
	})
}

// BenchmarkFileSystemOperations 文件系统操作基准测试
func BenchmarkFileSystemOperations(b *testing.B) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "benchmark_fs_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建多个主题
	themes := []string{"fs1", "fs2", "fs3", "fs4", "fs5"}
	for _, themeName := range themes {
		themeDir := filepath.Join(tempDir, themeName)
		if err := createThemeStructure(themeDir); err != nil {
			b.Fatalf("Failed to create theme structure for %s: %v", themeName, err)
		}
	}

	funcMap := NewFuncMap()

	b.Run("ThemeDiscovery", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			discovery := NewThemeDiscovery(tempDir, funcMap, DefaultLoadTemplate)
			_, err := discovery.DetectMode()
			if err != nil {
				b.Fatalf("Failed to detect mode: %v", err)
			}
		}
	})

	b.Run("ThemeValidation", func(b *testing.B) {
		discovery := NewThemeDiscovery(tempDir, funcMap, DefaultLoadTemplate)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, themeName := range themes {
				themeDir := filepath.Join(tempDir, themeName)
				err := discovery.ValidateTheme(themeDir)
				if err != nil {
					b.Fatalf("Failed to validate theme %s: %v", themeName, err)
				}
			}
		}
	})

	b.Run("MetadataLoading", func(b *testing.B) {
		discovery := NewThemeDiscovery(tempDir, funcMap, DefaultLoadTemplate)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			for _, themeName := range themes {
				themeDir := filepath.Join(tempDir, themeName)
				_, err := discovery.LoadThemeMetadata(themeDir)
				if err != nil {
					b.Fatalf("Failed to load metadata for theme %s: %v", themeName, err)
				}
			}
		}
	})
}

// Helper functions are already defined in theme_test.go, so we don't redeclare them here

// BenchmarkRealWorldScenarios 真实世界场景基准测试
func BenchmarkRealWorldScenarios(b *testing.B) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "benchmark_realworld_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// 创建类似真实应用的主题结构
	themes := []string{"corporate", "blog", "ecommerce"}
	for _, themeName := range themes {
		themeDir := filepath.Join(tempDir, themeName)
		if err := createRealisticThemeStructure(themeDir, themeName); err != nil {
			b.Fatalf("Failed to create realistic theme structure for %s: %v", themeName, err)
		}
	}

	funcMap := NewFuncMap()
	engine, err := NewEngine(tempDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
	if err != nil {
		b.Fatalf("Failed to create engine: %v", err)
	}
	engine.Init()
	defer engine.Close()

	b.Run("WebsiteSimulation", func(b *testing.B) {
		// 模拟网站访问模式：主要是页面渲染，偶尔主题切换
		pageData := H{
			"title":    "Real World Page",
			"content":  "This simulates a real website page with content",
			"user":     "John Doe",
			"products": []string{"Product A", "Product B", "Product C"},
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 90% 页面渲染
			if i%10 < 9 {
				var buf bytes.Buffer
				err := engine.RenderPage(&buf, "sample", pageData)
				if err != nil {
					b.Fatalf("Failed to render page: %v", err)
				}
			} else {
				// 10% 主题切换
				targetTheme := themes[i%len(themes)]
				_ = engine.SwitchTheme(targetTheme)
			}
		}
	})

	b.Run("AdminPanelSimulation", func(b *testing.B) {
		// 模拟管理面板：频繁的主题查询和切换
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			// 查询主题信息
			_ = engine.GetAvailableThemes()
			_ = engine.GetCurrentTheme()

			// 偶尔切换主题
			if i%5 == 0 {
				targetTheme := themes[i%len(themes)]
				_ = engine.SwitchTheme(targetTheme)
			}

			// 获取主题元数据
			if i%3 == 0 {
				currentTheme := engine.GetCurrentTheme()
				_, _ = engine.GetThemeMetadata(currentTheme)
			}
		}
	})
}

// createRealisticThemeStructure 创建更真实的主题结构
func createRealisticThemeStructure(themeDir, themeName string) error {
	// 创建基本目录结构
	dirs := []string{"layouts", "pages", "singles", "errors", "partials"}
	for _, dir := range dirs {
		dirPath := filepath.Join(themeDir, dir)
		if err := os.MkdirAll(dirPath, 0755); err != nil {
			return err
		}
	}

	// 创建主题配置文件
	themeConfig := fmt.Sprintf(`{
    "name": "%s",
    "displayName": "%s Theme",
    "description": "A %s theme for the application",
    "version": "1.0.0",
    "author": "Theme Developer",
    "tags": ["%s", "responsive", "modern"]
}`, themeName, themeName, themeName, themeName)

	configPath := filepath.Join(themeDir, "theme.json")
	if err := os.WriteFile(configPath, []byte(themeConfig), 0644); err != nil {
		return err
	}

	// 创建更多样化的模板文件
	templates := map[string][]string{
		"layouts":  {"main.tmpl", "admin.tmpl", "minimal.tmpl"},
		"pages":    {"home.tmpl", "about.tmpl", "contact.tmpl", "products.tmpl", "blog.tmpl"},
		"singles":  {"login.tmpl", "register.tmpl", "profile.tmpl", "settings.tmpl"},
		"errors":   {"404.tmpl", "500.tmpl", "403.tmpl"},
		"partials": {"header.tmpl", "footer.tmpl", "sidebar.tmpl", "nav.tmpl"},
	}

	for dir, files := range templates {
		dirPath := filepath.Join(themeDir, dir)
		for _, fileName := range files {
			filePath := filepath.Join(dirPath, fileName)
			content := fmt.Sprintf(`<!-- %s %s template -->
<div class="%s-%s">
    <h1>{{ .title }}</h1>
    <p>{{ .content }}</p>
</div>`, themeName, fileName, themeName, dir)
			if err := os.WriteFile(filePath, []byte(content), 0644); err != nil {
				return err
			}
		}
	}

	return nil
}

// BenchmarkStartupTime 启动时间基准测试
func BenchmarkStartupTime(b *testing.B) {
	// 创建临时目录
	tempDir, err := os.MkdirTemp("", "benchmark_startup_*")
	if err != nil {
		b.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	funcMap := NewFuncMap()

	b.Run("LegacyMode_Startup", func(b *testing.B) {
		// 创建传统结构
		legacyDir := filepath.Join(tempDir, "legacy")
		if err := createLegacyStructure(legacyDir); err != nil {
			b.Fatalf("Failed to create legacy structure: %v", err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()

			engine, err := NewEngine(legacyDir, DefaultLoadTemplate, funcMap)
			if err != nil {
				b.Fatalf("Failed to create engine: %v", err)
			}

			engine.Init()

			// 记录启动时间
			elapsed := time.Since(start)
			b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/startup")

			engine.Close()
		}
	})

	b.Run("MultiThemeMode_Startup", func(b *testing.B) {
		// 创建多主题结构
		multiDir := filepath.Join(tempDir, "multi")
		themes := []string{"theme1", "theme2", "theme3"}
		for _, themeName := range themes {
			themeDir := filepath.Join(multiDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				b.Fatalf("Failed to create theme structure for %s: %v", themeName, err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()

			engine, err := NewEngine(multiDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
			if err != nil {
				b.Fatalf("Failed to create engine: %v", err)
			}

			engine.Init()

			// 记录启动时间
			elapsed := time.Since(start)
			b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/startup")

			engine.Close()
		}
	})

	b.Run("MultiThemeMode_WithManyThemes_Startup", func(b *testing.B) {
		// 创建大量主题的多主题结构
		manyThemesDir := filepath.Join(tempDir, "many")
		for i := 0; i < 10; i++ {
			themeName := fmt.Sprintf("theme%d", i)
			themeDir := filepath.Join(manyThemesDir, themeName)
			if err := createThemeStructure(themeDir); err != nil {
				b.Fatalf("Failed to create theme structure for %s: %v", themeName, err)
			}
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			start := time.Now()

			engine, err := NewEngine(manyThemesDir, DefaultLoadTemplate, funcMap, EnableMultiTheme(true))
			if err != nil {
				b.Fatalf("Failed to create engine: %v", err)
			}

			engine.Init()

			// 记录启动时间
			elapsed := time.Since(start)
			b.ReportMetric(float64(elapsed.Nanoseconds()), "ns/startup")

			engine.Close()
		}
	})
}
