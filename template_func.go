package template

import (
	"fmt"
	"path/filepath"

	"github.com/nilorg/sdk/path"
)

func loadTemplate(r *Render, templatesDir string, funcMap FuncMap) {
	// 加载布局
	layouts, err := filepath.Glob(filepath.Join(templatesDir, "layouts/*.tmpl"))
	if err != nil {
		panic(err)
	}
	// 加载错误页面
	errors, err := filepath.Glob(filepath.Join(templatesDir, "errors/*.tmpl"))
	if err != nil {
		panic(err)
	}
	for _, errPage := range errors {
		tmplName := fmt.Sprintf("error/%s", filepath.Base(errPage))
		r.AddFromFilesFuncs(tmplName, funcMap, errPage)
	}
	// 加载局部页面
	partials, err := filepath.Glob(filepath.Join(templatesDir, "partials/*.tmpl"))
	if err != nil {
		panic(err)
	}

	// 页面文件夹
	var pageDirs []string
	basePagePath := filepath.Join(templatesDir, "pages")
	err = path.Dirs(basePagePath, &pageDirs)
	if err != nil {
		panic(err)
	}
	for _, pageDir := range pageDirs {
		for _, layout := range layouts {
			pageItems, err := filepath.Glob(filepath.Join(pageDir, "*.tmpl"))
			if err != nil {
				panic(err)
			}
			if len(pageItems) == 0 {
				continue
			}
			files := []string{
				layout,
			}
			files = append(files, partials...)
			files = append(files, pageItems...)
			pageName := pageDir[len(basePagePath)+1:]
			tmplName := fmt.Sprintf("%s:pages/%s", filepath.Base(layout), pageName)
			r.AddFromFilesFuncs(tmplName, funcMap, files...)
		}
	}
	// 加载单页面
	singles, err := filepath.Glob(filepath.Join(templatesDir, "singles/*.tmpl"))
	if err != nil {
		panic(err)
	}
	for _, singlePage := range singles {
		tmplName := fmt.Sprintf("singles/%s", filepath.Base(singlePage))
		r.AddFromFilesFuncs(tmplName, funcMap, singlePage)
	}
}

// DefaultLoadTemplate ...
func DefaultLoadTemplate(templatesDir string, funcMap FuncMap) Render {
	r := NewRender()
	loadTemplate(&r, templatesDir, funcMap)
	return r
}
