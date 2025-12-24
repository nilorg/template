package template

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

// Theme 表示一个主题
type Theme struct {
	Name       string        `json:"name"`        // 主题名称
	Path       string        `json:"path"`        // 主题路径
	IsDefault  bool          `json:"is_default"`  // 是否为默认主题
	IsEmbedded bool          `json:"is_embedded"` // 是否为嵌入式主题
	Metadata   ThemeMetadata `json:"metadata"`    // 主题元数据
}

// ThemeMetadata 主题元数据
type ThemeMetadata struct {
	DisplayName string         `json:"display_name"` // 显示名称
	Description string         `json:"description"`  // 描述
	Version     string         `json:"version"`      // 版本
	Author      string         `json:"author"`       // 作者
	Tags        []string       `json:"tags"`         // 标签
	Custom      map[string]any `json:"custom"`       // 自定义字段
}

// ThemeManager 主题管理器接口
type ThemeManager interface {
	// 发现和加载主题
	DiscoverThemes() error
	LoadTheme(name string) (*Theme, error)

	// 主题查询
	GetAvailableThemes() []string
	GetCurrentTheme() string
	ThemeExists(name string) bool
	GetThemeMetadata(name string) (*ThemeMetadata, error)

	// 主题切换
	SwitchTheme(name string) error

	// 渲染器管理
	GetRender() Render
	ReloadCurrentTheme() error
}

// ThemeDiscovery 主题发现器
type ThemeDiscovery struct {
	baseDir       string
	embedFS       *embed.FS
	subDir        string
	funcMap       FuncMap
	loadFunc      LoadTemplateFunc
	loadEmbedFunc LoadEmbedFSTemplateFunc
}

// DiscoverMode 发现模式
type DiscoverMode int

const (
	ModeLegacy     DiscoverMode = iota // 传统单主题模式
	ModeMultiTheme                     // 多主题模式
	ModeAuto                           // 自动检测模式
)

// ThemeError 主题相关错误
type ThemeError struct {
	Type    ThemeErrorType `json:"type"`
	Theme   string         `json:"theme"`
	Message string         `json:"message"`
	Cause   error          `json:"-"`
}

// Error 实现error接口
func (e *ThemeError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("theme error [%s]: %s (caused by: %v)", e.Theme, e.Message, e.Cause)
	}
	return fmt.Sprintf("theme error [%s]: %s", e.Theme, e.Message)
}

// Unwrap 支持错误链
func (e *ThemeError) Unwrap() error {
	return e.Cause
}

// ThemeErrorType 主题错误类型
type ThemeErrorType int

const (
	ErrThemeNotFound ThemeErrorType = iota
	ErrThemeInvalid
	ErrThemeLoadFailed
	ErrThemeSwitchFailed
	ErrThemeConfigInvalid
)

// String 返回错误类型的字符串表示
func (t ThemeErrorType) String() string {
	switch t {
	case ErrThemeNotFound:
		return "ThemeNotFound"
	case ErrThemeInvalid:
		return "ThemeInvalid"
	case ErrThemeLoadFailed:
		return "ThemeLoadFailed"
	case ErrThemeSwitchFailed:
		return "ThemeSwitchFailed"
	case ErrThemeConfigInvalid:
		return "ThemeConfigInvalid"
	default:
		return "Unknown"
	}
}

// NewThemeDiscovery 创建主题发现器
func NewThemeDiscovery(baseDir string, funcMap FuncMap, loadFunc LoadTemplateFunc) *ThemeDiscovery {
	return &ThemeDiscovery{
		baseDir:  baseDir,
		funcMap:  funcMap,
		loadFunc: loadFunc,
	}
}

// NewThemeDiscoveryWithEmbedFS 创建嵌入式文件系统的主题发现器
func NewThemeDiscoveryWithEmbedFS(embedFS *embed.FS, subDir string, funcMap FuncMap, loadEmbedFunc LoadEmbedFSTemplateFunc) *ThemeDiscovery {
	return &ThemeDiscovery{
		embedFS:       embedFS,
		subDir:        subDir,
		funcMap:       funcMap,
		loadEmbedFunc: loadEmbedFunc,
	}
}

// DetectMode 检测目录模式
func (td *ThemeDiscovery) DetectMode() (DiscoverMode, error) {
	if td.embedFS != nil {
		return td.detectModeEmbedFS()
	}
	return td.detectModeFileSystem()
}

// detectModeFileSystem 检测文件系统模式
func (td *ThemeDiscovery) detectModeFileSystem() (DiscoverMode, error) {
	// 检查是否存在传统结构
	hasLegacyStructure := td.hasLegacyStructure()

	// 检查是否存在主题子目录
	hasThemeSubdirs, err := td.hasThemeSubdirectories()
	if err != nil {
		return ModeLegacy, err
	}

	if hasLegacyStructure && !hasThemeSubdirs {
		return ModeLegacy, nil
	}

	if hasThemeSubdirs {
		return ModeMultiTheme, nil
	}

	// 如果都没有，默认为传统模式
	return ModeLegacy, nil
}

// detectModeEmbedFS 检测嵌入式文件系统模式
func (td *ThemeDiscovery) detectModeEmbedFS() (DiscoverMode, error) {
	// 检查是否存在传统结构
	hasLegacyStructure := td.hasLegacyStructureEmbedFS()

	// 检查是否存在主题子目录
	hasThemeSubdirs, err := td.hasThemeSubdirectoriesEmbedFS()
	if err != nil {
		return ModeLegacy, err
	}

	if hasLegacyStructure && !hasThemeSubdirs {
		return ModeLegacy, nil
	}

	if hasThemeSubdirs {
		return ModeMultiTheme, nil
	}

	// 如果都没有，默认为传统模式
	return ModeLegacy, nil
}

// hasLegacyStructure 检查是否存在传统结构
func (td *ThemeDiscovery) hasLegacyStructure() bool {
	requiredDirs := []string{"layouts", "pages", "singles", "errors"}

	for _, dir := range requiredDirs {
		dirPath := filepath.Join(td.baseDir, dir)
		if !td.dirExists(dirPath) {
			return false
		}
	}
	return true
}

// hasLegacyStructureEmbedFS 检查嵌入式文件系统是否存在传统结构
func (td *ThemeDiscovery) hasLegacyStructureEmbedFS() bool {
	requiredDirs := []string{"layouts", "pages", "singles", "errors"}

	for _, dir := range requiredDirs {
		dirPath := filepath.Join(td.subDir, dir)
		if !td.dirExistsEmbedFS(dirPath) {
			return false
		}
	}
	return true
}

// hasThemeSubdirectories 检查是否存在主题子目录
func (td *ThemeDiscovery) hasThemeSubdirectories() (bool, error) {
	entries, err := os.ReadDir(td.baseDir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			themePath := filepath.Join(td.baseDir, entry.Name())
			if td.isValidThemeDirectory(themePath) {
				return true, nil
			}
		}
	}
	return false, nil
}

// hasThemeSubdirectoriesEmbedFS 检查嵌入式文件系统是否存在主题子目录
func (td *ThemeDiscovery) hasThemeSubdirectoriesEmbedFS() (bool, error) {
	entries, err := fs.ReadDir(td.embedFS, td.subDir)
	if err != nil {
		return false, err
	}

	for _, entry := range entries {
		if entry.IsDir() {
			themePath := filepath.Join(td.subDir, entry.Name())
			if td.isValidThemeDirectoryEmbedFS(themePath) {
				return true, nil
			}
		}
	}
	return false, nil
}

// dirExists 检查目录是否存在
func (td *ThemeDiscovery) dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

// dirExistsEmbedFS 检查嵌入式文件系统中目录是否存在
func (td *ThemeDiscovery) dirExistsEmbedFS(path string) bool {
	_, err := fs.Stat(td.embedFS, path)
	return err == nil
}

// isValidThemeDirectory 检查是否为有效的主题目录
func (td *ThemeDiscovery) isValidThemeDirectory(themePath string) bool {
	return td.ValidateTheme(themePath) == nil
}

// isValidThemeDirectoryEmbedFS 检查嵌入式文件系统中是否为有效的主题目录
func (td *ThemeDiscovery) isValidThemeDirectoryEmbedFS(themePath string) bool {
	return td.ValidateTheme(themePath) == nil
}

// ValidateTheme 验证主题的完整性和有效性
func (td *ThemeDiscovery) ValidateTheme(themePath string) error {
	themeName := filepath.Base(themePath)

	// 1. 验证主题目录结构完整性
	if err := td.validateThemeStructure(themePath); err != nil {
		return &ThemeError{
			Type:    ErrThemeInvalid,
			Theme:   themeName,
			Message: "invalid theme structure",
			Cause:   err,
		}
	}

	// 2. 检查必需的模板文件是否存在
	if err := td.validateRequiredTemplates(themePath); err != nil {
		return &ThemeError{
			Type:    ErrThemeInvalid,
			Theme:   themeName,
			Message: "missing required templates",
			Cause:   err,
		}
	}

	// 3. 验证可选的theme.json配置文件（如果存在）
	if err := td.validateThemeConfig(themePath); err != nil {
		return &ThemeError{
			Type:    ErrThemeConfigInvalid,
			Theme:   themeName,
			Message: "invalid theme configuration",
			Cause:   err,
		}
	}

	return nil
}

// validateThemeStructure 验证主题目录结构
func (td *ThemeDiscovery) validateThemeStructure(themePath string) error {
	requiredDirs := []string{"layouts", "pages", "singles", "errors"}

	for _, dir := range requiredDirs {
		var dirPath string
		var exists bool

		if td.embedFS != nil {
			dirPath = filepath.Join(themePath, dir)
			exists = td.dirExistsEmbedFS(dirPath)
		} else {
			dirPath = filepath.Join(themePath, dir)
			exists = td.dirExists(dirPath)
		}

		if !exists {
			return fmt.Errorf("required directory '%s' not found", dir)
		}
	}

	// 验证partials目录（可选但推荐）
	var partialsPath string
	if td.embedFS != nil {
		partialsPath = filepath.Join(themePath, "partials")
		if !td.dirExistsEmbedFS(partialsPath) {
			// partials目录是可选的，只记录警告
		}
	} else {
		partialsPath = filepath.Join(themePath, "partials")
		if !td.dirExists(partialsPath) {
			// partials目录是可选的，只记录警告
		}
	}

	return nil
}

// validateRequiredTemplates 验证必需的模板文件
func (td *ThemeDiscovery) validateRequiredTemplates(themePath string) error {
	// 检查layouts目录中是否至少有一个布局文件
	layoutsPath := filepath.Join(themePath, "layouts")
	if err := td.validateDirectoryHasTemplates(layoutsPath, "layouts"); err != nil {
		return err
	}

	// 检查pages目录中是否有模板文件（可以在子目录中）
	pagesPath := filepath.Join(themePath, "pages")
	if err := td.validatePagesDirectory(pagesPath); err != nil {
		return err
	}

	// 检查singles目录中是否有模板文件
	singlesPath := filepath.Join(themePath, "singles")
	if err := td.validateDirectoryHasTemplates(singlesPath, "singles"); err != nil {
		return err
	}

	// 检查errors目录中是否有模板文件
	errorsPath := filepath.Join(themePath, "errors")
	if err := td.validateDirectoryHasTemplates(errorsPath, "errors"); err != nil {
		return err
	}

	return nil
}

// validateDirectoryHasTemplates 验证目录中是否包含模板文件
func (td *ThemeDiscovery) validateDirectoryHasTemplates(dirPath, dirName string) error {
	var entries []fs.DirEntry
	var err error

	if td.embedFS != nil {
		entries, err = fs.ReadDir(td.embedFS, dirPath)
	} else {
		entries, err = os.ReadDir(dirPath)
	}

	if err != nil {
		return fmt.Errorf("cannot read %s directory: %w", dirName, err)
	}

	hasTemplates := false

	// 检查直接在目录中的模板文件
	for _, entry := range entries {
		if !entry.IsDir() {
			// 检查是否为模板文件（.tmpl, .html, .gohtml等）
			name := entry.Name()
			if td.isTemplateFile(name) {
				hasTemplates = true
				break
			}
		}
	}

	// 如果直接在目录中没有找到模板，检查子目录（支持分割模板架构）
	if !hasTemplates {
		for _, entry := range entries {
			if entry.IsDir() {
				var subDirPath string
				if td.embedFS != nil {
					subDirPath = filepath.Join(dirPath, entry.Name())
				} else {
					subDirPath = filepath.Join(dirPath, entry.Name())
				}

				// 递归检查子目录中是否有模板文件
				var subEntries []fs.DirEntry
				if td.embedFS != nil {
					subEntries, err = fs.ReadDir(td.embedFS, subDirPath)
				} else {
					subEntries, err = os.ReadDir(subDirPath)
				}

				if err == nil {
					for _, subEntry := range subEntries {
						if !subEntry.IsDir() && td.isTemplateFile(subEntry.Name()) {
							hasTemplates = true
							break
						}
					}
					if hasTemplates {
						break
					}
				}
			}
		}
	}

	if !hasTemplates {
		return fmt.Errorf("%s directory must contain at least one template file (either directly or in subdirectories)", dirName)
	}

	return nil
}

// validatePagesDirectory 验证pages目录（可以包含子目录中的模板）
func (td *ThemeDiscovery) validatePagesDirectory(pagesPath string) error {
	var entries []fs.DirEntry
	var err error

	if td.embedFS != nil {
		entries, err = fs.ReadDir(td.embedFS, pagesPath)
	} else {
		entries, err = os.ReadDir(pagesPath)
	}

	if err != nil {
		return fmt.Errorf("cannot read pages directory: %w", err)
	}

	hasTemplates := false

	// 检查直接在pages目录中的模板文件
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			if td.isTemplateFile(name) {
				hasTemplates = true
				break
			}
		}
	}

	// 如果直接在pages目录中没有找到模板，检查子目录
	if !hasTemplates {
		for _, entry := range entries {
			if entry.IsDir() {
				var subDirPath string
				if td.embedFS != nil {
					subDirPath = filepath.Join(pagesPath, entry.Name())
				} else {
					subDirPath = filepath.Join(pagesPath, entry.Name())
				}

				// 检查子目录中是否有模板文件
				if err := td.validateDirectoryHasTemplates(subDirPath, fmt.Sprintf("pages/%s", entry.Name())); err == nil {
					hasTemplates = true
					break
				}
			}
		}
	}

	if !hasTemplates {
		return fmt.Errorf("pages directory must contain at least one template file (either directly or in subdirectories)")
	}

	return nil
}

// isTemplateFile 检查文件是否为模板文件
func (td *ThemeDiscovery) isTemplateFile(filename string) bool {
	ext := filepath.Ext(filename)
	templateExts := []string{".tmpl", ".html", ".gohtml", ".tpl"}

	for _, validExt := range templateExts {
		if ext == validExt {
			return true
		}
	}
	return false
}

// validateThemeConfig 验证主题配置文件
func (td *ThemeDiscovery) validateThemeConfig(themePath string) error {
	var configPath string
	var data []byte
	var err error

	if td.embedFS != nil {
		configPath = filepath.Join(themePath, "theme.json")
		data, err = fs.ReadFile(td.embedFS, configPath)
	} else {
		configPath = filepath.Join(themePath, "theme.json")
		data, err = os.ReadFile(configPath)
	}

	// 如果theme.json不存在，这是可以接受的
	if err != nil {
		if os.IsNotExist(err) || err == fs.ErrNotExist {
			return nil
		}
		return fmt.Errorf("cannot read theme.json: %w", err)
	}

	// 如果存在theme.json，验证其格式
	var config map[string]any
	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("invalid JSON format in theme.json: %w", err)
	}

	// 验证必需字段（如果存在的话）
	if name, exists := config["name"]; exists {
		if nameStr, ok := name.(string); !ok || nameStr == "" {
			return fmt.Errorf("theme name must be a non-empty string")
		}
	}

	if version, exists := config["version"]; exists {
		if versionStr, ok := version.(string); !ok || versionStr == "" {
			return fmt.Errorf("theme version must be a non-empty string")
		}
	}

	// 验证tags字段（如果存在）
	if tags, exists := config["tags"]; exists {
		if tagsArray, ok := tags.([]any); ok {
			for i, tag := range tagsArray {
				if _, ok := tag.(string); !ok {
					return fmt.Errorf("tag at index %d must be a string", i)
				}
			}
		} else {
			return fmt.Errorf("tags must be an array of strings")
		}
	}

	return nil
}

// DefaultThemeManager 默认主题管理器实现
type DefaultThemeManager struct {
	discovery     *ThemeDiscovery
	themes        map[string]*Theme
	currentTheme  string
	defaultTheme  string
	render        Render
	funcMap       FuncMap
	loadFunc      LoadTemplateFunc
	loadEmbedFunc LoadEmbedFSTemplateFunc
}

// NewDefaultThemeManager 创建默认主题管理器
func NewDefaultThemeManager(baseDir string, funcMap FuncMap, loadFunc LoadTemplateFunc) *DefaultThemeManager {
	discovery := NewThemeDiscovery(baseDir, funcMap, loadFunc)
	return &DefaultThemeManager{
		discovery: discovery,
		themes:    make(map[string]*Theme),
		funcMap:   funcMap,
		loadFunc:  loadFunc,
		render:    NewRender(),
	}
}

// NewDefaultThemeManagerWithEmbedFS 创建嵌入式文件系统的默认主题管理器
func NewDefaultThemeManagerWithEmbedFS(embedFS *embed.FS, subDir string, funcMap FuncMap, loadEmbedFunc LoadEmbedFSTemplateFunc) *DefaultThemeManager {
	discovery := NewThemeDiscoveryWithEmbedFS(embedFS, subDir, funcMap, loadEmbedFunc)
	return &DefaultThemeManager{
		discovery:     discovery,
		themes:        make(map[string]*Theme),
		funcMap:       funcMap,
		loadEmbedFunc: loadEmbedFunc,
		render:        NewRender(),
	}
}

// DiscoverThemes 发现和加载主题
func (tm *DefaultThemeManager) DiscoverThemes() error {
	// 检测模式
	mode, err := tm.discovery.DetectMode()
	if err != nil {
		return &ThemeError{
			Type:    ErrThemeLoadFailed,
			Theme:   "",
			Message: "failed to detect theme mode",
			Cause:   err,
		}
	}

	// 清空现有主题
	tm.themes = make(map[string]*Theme)

	switch mode {
	case ModeLegacy:
		// 传统模式：创建一个默认主题
		return tm.discoverLegacyTheme()
	case ModeMultiTheme:
		// 多主题模式：发现所有主题
		return tm.discoverMultipleThemes()
	default:
		return &ThemeError{
			Type:    ErrThemeLoadFailed,
			Theme:   "",
			Message: fmt.Sprintf("unsupported theme mode: %v", mode),
		}
	}
}

// discoverLegacyTheme 发现传统模式主题
func (tm *DefaultThemeManager) discoverLegacyTheme() error {
	var basePath string
	var isEmbedded bool

	if tm.discovery.embedFS != nil {
		basePath = tm.discovery.subDir
		isEmbedded = true
	} else {
		basePath = tm.discovery.baseDir
		isEmbedded = false
	}

	// 验证传统结构
	if err := tm.discovery.ValidateTheme(basePath); err != nil {
		return &ThemeError{
			Type:    ErrThemeInvalid,
			Theme:   "default",
			Message: "invalid legacy theme structure",
			Cause:   err,
		}
	}

	// 加载元数据
	metadata, err := tm.discovery.LoadThemeMetadata(basePath)
	if err != nil {
		return err
	}

	// 创建默认主题
	theme := &Theme{
		Name:       "default",
		Path:       basePath,
		IsDefault:  true,
		IsEmbedded: isEmbedded,
		Metadata:   *metadata,
	}

	tm.themes["default"] = theme
	tm.currentTheme = "default"
	tm.defaultTheme = "default"

	// 加载默认主题的模板
	if _, err := tm.LoadTheme("default"); err != nil {
		return &ThemeError{
			Type:    ErrThemeLoadFailed,
			Theme:   "default",
			Message: "failed to load legacy theme during discovery",
			Cause:   err,
		}
	}

	return nil
}

// discoverMultipleThemes 发现多个主题
func (tm *DefaultThemeManager) discoverMultipleThemes() error {
	var entries []fs.DirEntry
	var err error
	var basePath string

	if tm.discovery.embedFS != nil {
		basePath = tm.discovery.subDir
		entries, err = fs.ReadDir(tm.discovery.embedFS, basePath)
	} else {
		basePath = tm.discovery.baseDir
		entries, err = os.ReadDir(basePath)
	}

	if err != nil {
		return &ThemeError{
			Type:    ErrThemeLoadFailed,
			Theme:   "",
			Message: "failed to read theme directory",
			Cause:   err,
		}
	}

	foundThemes := 0
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		themeName := entry.Name()
		var themePath string

		if tm.discovery.embedFS != nil {
			themePath = filepath.Join(basePath, themeName)
		} else {
			themePath = filepath.Join(basePath, themeName)
		}

		// 验证主题
		if err := tm.discovery.ValidateTheme(themePath); err != nil {
			// 跳过无效主题，但记录警告
			continue
		}

		// 加载元数据
		metadata, err := tm.discovery.LoadThemeMetadata(themePath)
		if err != nil {
			return err
		}

		// 创建主题对象
		theme := &Theme{
			Name:       themeName,
			Path:       themePath,
			IsDefault:  foundThemes == 0, // 第一个发现的主题作为默认主题
			IsEmbedded: tm.discovery.embedFS != nil,
			Metadata:   *metadata,
		}

		tm.themes[themeName] = theme
		foundThemes++

		// 设置默认主题
		if foundThemes == 1 {
			tm.defaultTheme = themeName
			tm.currentTheme = themeName

			// 加载第一个主题的模板
			if _, err := tm.LoadTheme(themeName); err != nil {
				return &ThemeError{
					Type:    ErrThemeLoadFailed,
					Theme:   themeName,
					Message: "failed to load initial theme during discovery",
					Cause:   err,
				}
			}
		}
	}

	if foundThemes == 0 {
		return &ThemeError{
			Type:    ErrThemeNotFound,
			Theme:   "",
			Message: "no valid themes found in multi-theme directory",
		}
	}

	return nil
}

// LoadTheme 加载指定主题
func (tm *DefaultThemeManager) LoadTheme(name string) (*Theme, error) {
	theme, exists := tm.themes[name]
	if !exists {
		return nil, &ThemeError{
			Type:    ErrThemeNotFound,
			Theme:   name,
			Message: "theme not found",
		}
	}

	// 加载主题的模板
	if err := tm.loadThemeTemplates(theme); err != nil {
		return nil, &ThemeError{
			Type:    ErrThemeLoadFailed,
			Theme:   name,
			Message: "failed to load theme templates",
			Cause:   err,
		}
	}

	return theme, nil
}

// loadThemeTemplates 加载主题模板
func (tm *DefaultThemeManager) loadThemeTemplates(theme *Theme) error {
	// 释放之前的渲染器资源
	tm.clearRender()

	// 创建新的渲染器
	if theme.IsEmbedded {
		if tm.loadEmbedFunc == nil {
			return fmt.Errorf("embedded load function not available")
		}
		tm.render = tm.loadEmbedFunc(tm.discovery.embedFS, theme.Path, tm.funcMap)
	} else {
		if tm.loadFunc == nil {
			return fmt.Errorf("file system load function not available")
		}
		tm.render = tm.loadFunc(theme.Path, tm.funcMap)
	}

	// 验证渲染器是否成功创建
	if tm.render == nil {
		return fmt.Errorf("failed to create render for theme %s", theme.Name)
	}

	// 验证渲染器包含模板
	renderMap := map[string]*template.Template(tm.render)
	if len(renderMap) == 0 {
		return fmt.Errorf("render contains no templates for theme %s", theme.Name)
	}

	return nil
}

// clearRender 清理渲染器资源
func (tm *DefaultThemeManager) clearRender() {
	// 创建新的空渲染器而不是清空现有的
	// 这样可以避免影响已经分配给引擎的渲染器引用
	tm.render = NewRender()
}

// GetRenderStats 获取渲染器统计信息
func (tm *DefaultThemeManager) GetRenderStats() map[string]int {
	stats := make(map[string]int)

	if tm.render == nil {
		return stats
	}

	renderMap := map[string]*template.Template(tm.render)

	// 统计不同类型的模板数量
	layoutCount := 0
	pageCount := 0
	singleCount := 0
	errorCount := 0
	otherCount := 0

	for name := range renderMap {
		switch {
		case strings.Contains(name, ":pages/"):
			pageCount++
		case strings.HasPrefix(name, "singles/"):
			singleCount++
		case strings.HasPrefix(name, "error/"):
			errorCount++
		case strings.Contains(name, "layout"):
			layoutCount++
		default:
			otherCount++
		}
	}

	stats["total"] = len(renderMap)
	stats["layouts"] = layoutCount
	stats["pages"] = pageCount
	stats["singles"] = singleCount
	stats["errors"] = errorCount
	stats["others"] = otherCount

	return stats
}

// ValidateRenderIntegrity 验证渲染器完整性
func (tm *DefaultThemeManager) ValidateRenderIntegrity() error {
	if tm.render == nil {
		return fmt.Errorf("render is nil")
	}

	renderMap := map[string]*template.Template(tm.render)
	if len(renderMap) == 0 {
		return fmt.Errorf("render contains no templates")
	}

	// 验证关键模板类型是否存在
	hasPages := false
	hasSingles := false
	hasErrors := false

	for name, tmpl := range renderMap {
		if tmpl == nil {
			return fmt.Errorf("template %s is nil", name)
		}

		switch {
		case strings.Contains(name, ":pages/"):
			hasPages = true
		case strings.HasPrefix(name, "singles/"):
			hasSingles = true
		case strings.HasPrefix(name, "error/"):
			hasErrors = true
		}
	}

	// 检查是否至少有基本的模板类型
	if !hasPages && !hasSingles && !hasErrors {
		return fmt.Errorf("render contains no recognizable template types")
	}

	return nil
}

// PreloadTheme 预加载主题（不切换当前主题）
func (tm *DefaultThemeManager) PreloadTheme(name string) error {
	theme, exists := tm.themes[name]
	if !exists {
		return &ThemeError{
			Type:    ErrThemeNotFound,
			Theme:   name,
			Message: "theme not found for preloading",
		}
	}

	// 创建临时渲染器进行预加载验证
	var tempRender Render

	if theme.IsEmbedded {
		if tm.loadEmbedFunc == nil {
			return fmt.Errorf("embedded load function not available")
		}
		tempRender = tm.loadEmbedFunc(tm.discovery.embedFS, theme.Path, tm.funcMap)
	} else {
		if tm.loadFunc == nil {
			return fmt.Errorf("file system load function not available")
		}
		tempRender = tm.loadFunc(theme.Path, tm.funcMap)
	}

	// 验证预加载的渲染器
	if tempRender == nil {
		return &ThemeError{
			Type:    ErrThemeLoadFailed,
			Theme:   name,
			Message: "failed to preload theme",
		}
	}

	renderMap := map[string]*template.Template(tempRender)
	if len(renderMap) == 0 {
		return &ThemeError{
			Type:    ErrThemeLoadFailed,
			Theme:   name,
			Message: "preloaded theme contains no templates",
		}
	}

	// 预加载成功，不保存到当前渲染器
	return nil
}

// GetTemplateNames 获取当前渲染器中的所有模板名称
func (tm *DefaultThemeManager) GetTemplateNames() []string {
	if tm.render == nil {
		return []string{}
	}

	renderMap := map[string]*template.Template(tm.render)
	names := make([]string, 0, len(renderMap))

	for name := range renderMap {
		names = append(names, name)
	}

	return names
}

// HasTemplate 检查是否存在指定名称的模板
func (tm *DefaultThemeManager) HasTemplate(templateName string) bool {
	if tm.render == nil {
		return false
	}

	renderMap := map[string]*template.Template(tm.render)
	_, exists := renderMap[templateName]
	return exists
}

// GetMemoryUsage 获取内存使用估算（简单实现）
func (tm *DefaultThemeManager) GetMemoryUsage() map[string]any {
	usage := make(map[string]any)

	usage["themes_count"] = len(tm.themes)
	usage["current_theme"] = tm.currentTheme

	if tm.render != nil {
		renderMap := map[string]*template.Template(tm.render)
		usage["templates_count"] = len(renderMap)
		usage["render_size_estimate"] = len(renderMap) * 1024 // 粗略估算每个模板1KB
	} else {
		usage["templates_count"] = 0
		usage["render_size_estimate"] = 0
	}

	return usage
}

// GetAvailableThemes 获取所有可用主题
func (tm *DefaultThemeManager) GetAvailableThemes() []string {
	themes := make([]string, 0, len(tm.themes))
	for name := range tm.themes {
		themes = append(themes, name)
	}
	return themes
}

// GetCurrentTheme 获取当前主题
func (tm *DefaultThemeManager) GetCurrentTheme() string {
	return tm.currentTheme
}

// ThemeExists 检查主题是否存在
func (tm *DefaultThemeManager) ThemeExists(name string) bool {
	_, exists := tm.themes[name]
	return exists
}

// GetThemeMetadata 获取主题元数据
func (tm *DefaultThemeManager) GetThemeMetadata(name string) (*ThemeMetadata, error) {
	theme, exists := tm.themes[name]
	if !exists {
		return nil, &ThemeError{
			Type:    ErrThemeNotFound,
			Theme:   name,
			Message: "theme not found",
		}
	}

	return &theme.Metadata, nil
}

// SwitchTheme 切换主题
func (tm *DefaultThemeManager) SwitchTheme(name string) error {
	// 检查主题是否存在
	if !tm.ThemeExists(name) {
		return &ThemeError{
			Type:    ErrThemeNotFound,
			Theme:   name,
			Message: "theme not found",
		}
	}

	// 如果已经是当前主题，直接返回
	if tm.currentTheme == name {
		return nil
	}

	// 保存当前状态以便回滚
	previousTheme := tm.currentTheme
	previousRender := tm.render

	// 尝试加载新主题
	_, err := tm.LoadTheme(name)
	if err != nil {
		// 确保状态没有被破坏
		tm.currentTheme = previousTheme
		tm.render = previousRender
		return &ThemeError{
			Type:    ErrThemeSwitchFailed,
			Theme:   name,
			Message: "failed to load new theme during switch",
			Cause:   err,
		}
	}

	// 验证新渲染器是否有效
	if tm.render == nil {
		// 回滚到之前的状态
		tm.currentTheme = previousTheme
		tm.render = previousRender
		return &ThemeError{
			Type:    ErrThemeSwitchFailed,
			Theme:   name,
			Message: "theme switch failed: render is nil after loading",
		}
	}

	// 验证渲染器是否包含必要的模板
	if err := tm.validateRenderTemplates(); err != nil {
		// 回滚到之前的状态
		tm.currentTheme = previousTheme
		tm.render = previousRender
		return &ThemeError{
			Type:    ErrThemeSwitchFailed,
			Theme:   name,
			Message: "theme switch failed: invalid templates in new theme",
			Cause:   err,
		}
	}

	// 成功切换，更新当前主题
	tm.currentTheme = name

	// 更新主题的默认状态（只有当前主题被标记为活跃）
	for themeName, t := range tm.themes {
		// 不改变IsDefault状态，只是切换当前活跃主题
		_ = themeName
		_ = t
	}

	return nil
}

// validateRenderTemplates 验证渲染器中的模板
func (tm *DefaultThemeManager) validateRenderTemplates() error {
	if tm.render == nil {
		return fmt.Errorf("render is nil")
	}

	// 检查渲染器是否为空
	renderMap := map[string]*template.Template(tm.render)
	if len(renderMap) == 0 {
		return fmt.Errorf("render contains no templates")
	}

	// 基本验证：确保至少有一些模板被加载
	hasTemplates := false
	for name, tmpl := range renderMap {
		if tmpl != nil {
			hasTemplates = true
			break
		}
		_ = name // 避免未使用变量警告
	}

	if !hasTemplates {
		return fmt.Errorf("render contains no valid templates")
	}

	return nil
}

// SwitchToDefaultTheme 切换到默认主题
func (tm *DefaultThemeManager) SwitchToDefaultTheme() error {
	if tm.defaultTheme == "" {
		return &ThemeError{
			Type:    ErrThemeNotFound,
			Theme:   "",
			Message: "no default theme configured",
		}
	}

	return tm.SwitchTheme(tm.defaultTheme)
}

// SafeSwitchTheme 安全切换主题（如果失败则保持当前主题）
func (tm *DefaultThemeManager) SafeSwitchTheme(name string) error {
	// 记录切换尝试
	originalTheme := tm.currentTheme

	err := tm.SwitchTheme(name)
	if err != nil {
		// 确保我们仍然在原始主题上
		if tm.currentTheme != originalTheme {
			// 尝试恢复到原始主题
			if restoreErr := tm.SwitchTheme(originalTheme); restoreErr != nil {
				// 如果无法恢复，这是一个严重错误
				return &ThemeError{
					Type:    ErrThemeSwitchFailed,
					Theme:   name,
					Message: fmt.Sprintf("failed to switch to theme and failed to restore original theme '%s'", originalTheme),
					Cause:   err,
				}
			}
		}
		return err
	}

	return nil
}

// GetRender 获取渲染器
func (tm *DefaultThemeManager) GetRender() Render {
	return tm.render
}

// ReloadCurrentTheme 重新加载当前主题
func (tm *DefaultThemeManager) ReloadCurrentTheme() error {
	if tm.currentTheme == "" {
		return &ThemeError{
			Type:    ErrThemeLoadFailed,
			Theme:   "",
			Message: "no current theme to reload",
		}
	}

	_, err := tm.LoadTheme(tm.currentTheme)
	return err
}

// SetDefaultTheme 设置默认主题
func (tm *DefaultThemeManager) SetDefaultTheme(name string) error {
	if !tm.ThemeExists(name) {
		return &ThemeError{
			Type:    ErrThemeNotFound,
			Theme:   name,
			Message: "cannot set non-existent theme as default",
		}
	}

	tm.defaultTheme = name

	// 更新所有主题的默认状态
	for themeName, theme := range tm.themes {
		theme.IsDefault = (themeName == name)
	}

	return nil
}

// GetDefaultTheme 获取默认主题名称
func (tm *DefaultThemeManager) GetDefaultTheme() string {
	return tm.defaultTheme
}

// LoadThemeMetadata 加载主题元数据
func (td *ThemeDiscovery) LoadThemeMetadata(themePath string) (*ThemeMetadata, error) {
	var metadataPath string
	var data []byte
	var err error

	if td.embedFS != nil {
		metadataPath = filepath.Join(themePath, "theme.json")
		data, err = fs.ReadFile(td.embedFS, metadataPath)
	} else {
		metadataPath = filepath.Join(themePath, "theme.json")
		data, err = os.ReadFile(metadataPath)
	}

	if err != nil {
		// 如果没有theme.json文件，返回默认元数据
		return &ThemeMetadata{
			DisplayName: filepath.Base(themePath),
			Description: "Theme without metadata",
			Version:     "1.0.0",
			Author:      "Unknown",
			Tags:        []string{},
			Custom:      make(map[string]any),
		}, nil
	}

	var metadata ThemeMetadata
	if err := json.Unmarshal(data, &metadata); err != nil {
		return nil, &ThemeError{
			Type:    ErrThemeConfigInvalid,
			Theme:   filepath.Base(themePath),
			Message: "invalid theme.json format",
			Cause:   err,
		}
	}

	// 设置默认值
	if metadata.DisplayName == "" {
		metadata.DisplayName = filepath.Base(themePath)
	}
	if metadata.Version == "" {
		metadata.Version = "1.0.0"
	}
	if metadata.Tags == nil {
		metadata.Tags = []string{}
	}
	if metadata.Custom == nil {
		metadata.Custom = make(map[string]any)
	}

	return &metadata, nil
}
