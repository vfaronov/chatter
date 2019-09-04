package web

import (
	"html/template"
)

var funcMap = template.FuncMap{
	"addUint64": func(x, y uint64) uint64 { return x + y },
}
