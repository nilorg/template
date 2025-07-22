package template

import "net/http"

// Options 可选参数列表
type Options struct {
	StatusCode     int
	Layout         string
	GlobalVariable map[string]interface{}
	GlobalConstant map[string]interface{}
	Suffix         string
}

// newOptions 创建可选参数
func newOptions(opts ...Option) Options {
	opt := Options{
		StatusCode:     http.StatusOK,
		Layout:         "layout.tmpl",
		GlobalVariable: map[string]interface{}{},
		GlobalConstant: map[string]interface{}{},
		Suffix:         "tmpl",
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
func GlobalVariable(variable map[string]interface{}) Option {
	return func(o *Options) {
		o.GlobalVariable = variable
	}
}

// GlobalConstant ...
func GlobalConstant(constant map[string]interface{}) Option {
	return func(o *Options) {
		o.GlobalConstant = constant
	}
}

func Suffix(suffix string) Option {
	return func(o *Options) {
		o.Suffix = suffix
	}
}
