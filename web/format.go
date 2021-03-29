package web

import (
	"bytes"
	"html/template"

	"github.com/yuin/goldmark"
)

var funcMap = template.FuncMap{
	"addUint64": func(x, y uint64) uint64 { return x + y },
	"markdown": func(src string) (template.HTML, error) {
		var buf bytes.Buffer
		// TODO: does goldmark.Convert return error due to the source (not just the io.Writer)?
		// if it does, need to sanitize
		err := goldmark.Convert([]byte(src), &buf)
		html := template.HTML(buf.String()) //nolint:gosec
		return html, err
	},
}
