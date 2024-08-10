package main

import (
	"Portfolio/ui"
	"html/template"
	"io/fs"
	"path/filepath"
	"time"
)

var functions = template.FuncMap{
	"humanDate":     humanDate,
	"mdToHTML":      mdToHTML,
	"bytesToString": bytesToString,
}

func humanDate(t time.Time) string {
	return t.Format("02 Jan 2006 at 15:04")
}

func bytesToString(b []byte) string {
	if b != nil {
		return string(b)
	}
	return ""
}

func newTemplateCache() (map[string]*template.Template, error) {

	cache := map[string]*template.Template{}

	pages, err := fs.Glob(ui.Files, "templates/pages/*.tmpl")
	if err != nil {
		return nil, err
	}

	for _, page := range pages {
		name := filepath.Base(page)

		patterns := []string{
			"templates/base.tmpl",
			"templates/partials/*.tmpl",
			page,
		}

		ts, err := template.New(name).Funcs(functions).ParseFS(ui.Files, patterns...)
		if err != nil {
			return nil, err
		}

		cache[name] = ts
	}

	return cache, nil
}
