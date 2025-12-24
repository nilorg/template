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
	// 加载错误页面 - 支持分割模板和传统模板
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

	// 加载错误页面文件夹 - 新的分割模板架构
	var errorDirs []string
	baseErrorPath := filepath.Join(templatesDir, "errors")
	err = filepath.WalkDir(baseErrorPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != baseErrorPath {
			errorDirs = append(errorDirs, path)
		}
		return nil
	})
	if err == nil { // 如果errors目录存在
		// 查找错误布局
		errorLayouts, err := filepath.Glob(filepath.Join(templatesDir, "layouts/error.tmpl"))
		if err != nil {
			panic(err)
		}
		if len(errorLayouts) == 0 {
			// 如果没有专用错误布局，使用单页布局
			errorLayouts, err = filepath.Glob(filepath.Join(templatesDir, "layouts/single.tmpl"))
			if err != nil {
				panic(err)
			}
		}

		for _, errorDir := range errorDirs {
			for _, layout := range errorLayouts {
				errorItems, err := filepath.Glob(filepath.Join(errorDir, "*.tmpl"))
				if err != nil {
					panic(err)
				}
				if len(errorItems) == 0 {
					continue
				}
				files := []string{
					layout,
				}
				files = append(files, partials...)
				files = append(files, errorItems...)
				errorName := errorDir[len(baseErrorPath)+1:]
				tmplName := fmt.Sprintf("%s:error/%s", filepath.Base(layout), errorName)
				r.AddFromFilesFuncs(tmplName, funcMap, files...)
			}
		}
	}

	// 页面文件夹
	var pageDirs []string
	basePagePath := filepath.Join(templatesDir, "pages")
	err = filepath.WalkDir(basePagePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
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
	// 加载单页面 - 支持分割模板和传统模板
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

	// 加载单页面文件夹 - 新的分割模板架构
	var singleDirs []string
	baseSinglePath := filepath.Join(templatesDir, "singles")
	err = filepath.WalkDir(baseSinglePath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && path != baseSinglePath {
			singleDirs = append(singleDirs, path)
		}
		return nil
	})
	if err == nil { // 如果singles目录存在
		// 查找单页布局
		singleLayouts, err := filepath.Glob(filepath.Join(templatesDir, "layouts/single.tmpl"))
		if err != nil {
			panic(err)
		}
		if len(singleLayouts) == 0 {
			// 如果没有专用单页布局，使用默认布局
			singleLayouts, err = filepath.Glob(filepath.Join(templatesDir, "layouts/layout.tmpl"))
			if err != nil {
				panic(err)
			}
		}

		for _, singleDir := range singleDirs {
			for _, layout := range singleLayouts {
				singleItems, err := filepath.Glob(filepath.Join(singleDir, "*.tmpl"))
				if err != nil {
					panic(err)
				}
				if len(singleItems) == 0 {
					continue
				}
				files := []string{
					layout,
				}
				files = append(files, partials...)
				files = append(files, singleItems...)
				singleName := singleDir[len(baseSinglePath)+1:]
				tmplName := fmt.Sprintf("%s:singles/%s", filepath.Base(layout), singleName)
				r.AddFromFilesFuncs(tmplName, funcMap, files...)
			}
		}
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
	// embed.FS 总是使用正斜杠路径，使用 path 包而不是 filepath 包
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
	// 加载错误页面 - 支持分割模板和传统模板
	errors, err := fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "errors/*.tmpl"))
	if err != nil {
		panic(err)
	}
	for _, errPage := range errors {
		tmplName := fmt.Sprintf("error/%s", path.Base(errPage))
		files := []string{
			errPage,
		}
		files = append(files, partials...)
		r.AddFromFSFuncs(tmplName, funcMap, tmplFS, files...)
	}

	// 加载错误页面文件夹 - 新的分割模板架构
	var errorDirs []string
	baseErrorPath := path.Join(tmplFSSUbDir, "errors")
	err = fs.WalkDir(tmplFS, baseErrorPath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != baseErrorPath {
			errorDirs = append(errorDirs, p)
		}
		return nil
	})
	if err == nil { // 如果errors目录存在
		// 查找错误布局
		errorLayouts, err := fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "layouts/error.tmpl"))
		if err != nil {
			panic(err)
		}
		if len(errorLayouts) == 0 {
			// 如果没有专用错误布局，使用单页布局
			errorLayouts, err = fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "layouts/single.tmpl"))
			if err != nil {
				panic(err)
			}
		}

		for _, errorDir := range errorDirs {
			for _, layout := range errorLayouts {
				errorItems, err := fs.Glob(tmplFS, path.Join(errorDir, "*.tmpl"))
				if err != nil {
					panic(err)
				}
				if len(errorItems) == 0 {
					continue
				}
				files := []string{
					layout,
				}
				files = append(files, partials...)
				files = append(files, errorItems...)
				errorName := strings.TrimPrefix(errorDir, baseErrorPath+"/")
				tmplName := fmt.Sprintf("%s:error/%s", path.Base(layout), errorName)
				r.AddFromFSFuncs(tmplName, funcMap, tmplFS, files...)
			}
		}
	}

	// 页面文件夹
	var pageDirs []string
	basePagePath := path.Join(tmplFSSUbDir, "pages")
	err = fs.WalkDir(tmplFS, basePagePath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != basePagePath {
			pageDirs = append(pageDirs, p)
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
			tmplName := fmt.Sprintf("%s:pages/%s", path.Base(layout), pageName)
			r.AddFromFSFuncs(tmplName, funcMap, tmplFS, files...)
		}
	}
	// 加载单页面 - 支持分割模板和传统模板
	singles, err := fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "singles/*.tmpl"))
	if err != nil {
		panic(err)
	}
	for _, singlePage := range singles {
		tmplName := fmt.Sprintf("singles/%s", path.Base(singlePage))
		files := []string{
			singlePage,
		}
		files = append(files, partials...)
		r.AddFromFSFuncs(tmplName, funcMap, tmplFS, files...)
	}

	// 加载单页面文件夹 - 新的分割模板架构
	var singleDirs []string
	baseSinglePath := path.Join(tmplFSSUbDir, "singles")
	err = fs.WalkDir(tmplFS, baseSinglePath, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && p != baseSinglePath {
			singleDirs = append(singleDirs, p)
		}
		return nil
	})
	if err == nil { // 如果singles目录存在
		// 查找单页布局
		singleLayouts, err := fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "layouts/single.tmpl"))
		if err != nil {
			panic(err)
		}
		if len(singleLayouts) == 0 {
			// 如果没有专用单页布局，使用默认布局
			singleLayouts, err = fs.Glob(tmplFS, path.Join(tmplFSSUbDir, "layouts/layout.tmpl"))
			if err != nil {
				panic(err)
			}
		}

		for _, singleDir := range singleDirs {
			for _, layout := range singleLayouts {
				singleItems, err := fs.Glob(tmplFS, path.Join(singleDir, "*.tmpl"))
				if err != nil {
					panic(err)
				}
				if len(singleItems) == 0 {
					continue
				}
				files := []string{
					layout,
				}
				files = append(files, partials...)
				files = append(files, singleItems...)
				singleName := strings.TrimPrefix(singleDir, baseSinglePath+"/")
				tmplName := fmt.Sprintf("%s:singles/%s", path.Base(layout), singleName)
				r.AddFromFSFuncs(tmplName, funcMap, tmplFS, files...)
			}
		}
	}
}
