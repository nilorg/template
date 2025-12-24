package template

import "net/http"

// Options 可选参数列表
type Options struct {
	StatusCode     int
	Layout         string
	GlobalVariable map[string]any
	GlobalConstant map[string]any
	Suffix         string
	// 主题相关字段
	Theme          string // 指定的主题名称
	DefaultTheme   string // 默认主题名称
	MultiThemeMode bool   // 是否启用多主题模式
}

// newOptions 创建可选参数
func newOptions(opts ...Option) Options {
	opt := Options{
		StatusCode:     http.StatusOK,
		Layout:         "layout.tmpl",
		GlobalVariable: map[string]any{},
		GlobalConstant: map[string]any{},
		Suffix:         "tmpl",
		// 主题相关默认值
		Theme:          "",    // 空字符串表示未指定主题
		DefaultTheme:   "",    // 空字符串表示使用自动检测的默认主题
		MultiThemeMode: false, // 默认关闭多主题模式，使用自动检测
	}
	for _, o := range opts {
		o(&opt)
	}
	return opt
}

// Option 为可选参数赋值的函数
type Option func(*Options)

// StatusCode ...
func StatusCode(statusCode int) Option {
	return func(o *Options) {
		o.StatusCode = statusCode
	}
}

// Layout ...
func Layout(layout string) Option {
	return func(o *Options) {
		o.Layout = layout
	}
}

// GlobalVariable ...
func GlobalVariable(variable map[string]any) Option {
	return func(o *Options) {
		o.GlobalVariable = variable
	}
}

// GlobalConstant ...
func GlobalConstant(constant map[string]any) Option {
	return func(o *Options) {
		o.GlobalConstant = constant
	}
}

func Suffix(suffix string) Option {
	return func(o *Options) {
		o.Suffix = suffix
	}
}

// SetTheme 设置指定的主题名称
func SetTheme(themeName string) Option {
	return func(o *Options) {
		o.Theme = themeName
	}
}

// DefaultTheme 设置默认主题名称
func DefaultTheme(themeName string) Option {
	return func(o *Options) {
		o.DefaultTheme = themeName
	}
}

// EnableMultiTheme 启用或禁用多主题模式
func EnableMultiTheme(enable bool) Option {
	return func(o *Options) {
		o.MultiThemeMode = enable
	}
}
