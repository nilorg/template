package template

import (
	"embed"
	"fmt"
	"io/fs"
	"path"
	"path/filepath"
	"strings"
)

func loadTemplate(r *Render, templatesDir string, funcMap FuncMap) {
	// 加载局部页面
	partials, err := filepath.Glob(filepath.Join(templatesDir, "partials/*.tmpl"))
	if err != nil {
		panic(err)
	}
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
		files := []string{
			errPage,
		}
		files = append(files, partials...)
		r.AddFromFilesFuncs(tmplName, funcMap, files...)
	}

	// 页面文件夹
	var pageDirs []string
	basePagePath := filepath.Join(templatesDir, "pages")
	err = filepath.WalkDir(basePagePath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && path != basePagePath {
			pageDirs = append(pageDirs, path)
		}
		return nil
	})
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
		files := []string{
			singlePage,
		}
		files = append(files, partials...)
		r.AddFromFilesFuncs(tmplName, funcMap, files...)
	}
}

// DefaultLoadTemplate ...
func DefaultLoadTemplate(templatesDir string, funcMap FuncMap) Render {
	r := NewRender()
	loadTemplate(&r, templatesDir, funcMap)
	return r
}

func DefaultLoadTemplateWithEmbedFS(tmplFS *embed.FS, tmplFSSUbDir string, funcMap FuncMap) Render {
	r := NewRender()
	loadTemplateWithEmbedFS(&r, tmplFS, tmplFSSUbDir, funcMap)
	return r
}

func loadTemplateWithEmbedFS(r *Render, tmplFS *embed.FS, tmplFSSUbDir string, funcMap FuncMap) {
	// embed.FS 总是使用正斜杠路径，使用 path.Join 而不是 filepath.Join
	// 加载局部页面
	partials, err := fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "partials/*.tmpl"))
	if err != nil {
		panic(err)
	}
	// 加载布局
	layouts, err := fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "layouts/*.tmpl"))
	if err != nil {
		panic(err)
	}
	// 加载错误页面
	errors, err := fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "errors/*.tmpl"))
	if err != nil {
		panic(err)
	}
	for _, errPage := range errors {
		tmplName := fmt.Sprintf("error/%s", filepath.Base(errPage))
		files := []string{
			errPage,
		}
		files = append(files, partials...)
		r.AddFromFSFuncs(tmplName, funcMap, tmplFS, files...)
	}

	// 页面文件夹
	var pageDirs []string
	basePagePath := path.Join(tmplFSSUbDir, "pages")
	err = fs.WalkDir(tmplFS, basePagePath, func(path string, d fs.DirEntry, err error) error {
		if d.IsDir() && path != basePagePath {
			pageDirs = append(pageDirs, path)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
	for _, pageDir := range pageDirs {
		for _, layout := range layouts {
			pageItems, err := fs.Glob(tmplFS, path.Join(pageDir, "*.tmpl"))
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
			pageName := strings.TrimPrefix(pageDir, basePagePath+"/")
			tmplName := fmt.Sprintf("%s:pages/%s", filepath.Base(layout), pageName)
			r.AddFromFSFuncs(tmplName, funcMap, tmplFS, files...)
		}
	}
	// 加载单页面
	singles, err := fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "singles/*.tmpl"))
	if err != nil {
		panic(err)
	}
	for _, singlePage := range singles {
		tmplName := fmt.Sprintf("singles/%s", filepath.Base(singlePage))
		files := []string{
			singlePage,
		}
		files = append(files, partials...)
		r.AddFromFSFuncs(tmplName, funcMap, tmplFS, files...)
	}
}
