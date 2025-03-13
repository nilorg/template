package template

import (
	"errors"
	"fmt"
	"html/template"
	"io"
	"io/fs"
	"path/filepath"
)

// H is a shortcut for map[string]interface{}
type H map[string]interface{}

// FuncMap ...
type FuncMap template.FuncMap

// NewFuncMap instance
func NewFuncMap() FuncMap {
	return make(FuncMap)
}

// Render type
type Render map[string]*template.Template

// NewRender instance
func NewRender() Render {
	return make(Render)
}

// Add new template
func (r Render) Add(name string, tmpl *template.Template) error {
	if tmpl == nil {
		return errors.New("template can not be nil")
	}
	if len(name) == 0 {
		return errors.New("template name cannot be empty")
	}
	if _, ok := r[name]; ok {
		return fmt.Errorf("template %s already exists", name)
	}
	r[name] = tmpl
	return nil
}

// AddFromFilesFuncs supply add template from file callback func
func (r Render) AddFromFilesFuncs(name string, funcMap FuncMap, files ...string) *template.Template {
	tname := filepath.Base(files[0])
	tmpl := template.Must(template.New(tname).Funcs(template.FuncMap(funcMap)).ParseFiles(files...))
	r.Add(name, tmpl)
	return tmpl
}

// AddFromFSFuncs supply add template from fs callback func
func (r Render) AddFromFSFuncs(name string, funcMap FuncMap, fs fs.FS, files ...string) *template.Template {
	tname := filepath.Base(files[0])
	tmpl := template.Must(template.New(tname).Funcs(template.FuncMap(funcMap)).ParseFS(fs, files...))
	r.Add(name, tmpl)
	return tmpl
}

// Execute 执行
func (r Render) Execute(name string, wr io.Writer, data interface{}) error {
	t, ok := r[name]
	if !ok {
		return fmt.Errorf("template %s not exists", name)
	}
	return t.Execute(wr, data)
}
