package utils

import (
	"bytes"
	"html/template"
	"time"

	"github.com/Masterminds/sprig/v3"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/renderer/html"
)

func TemplateFuncs() template.FuncMap {
	// TODO: maybe cache
	md := goldmark.New(
		goldmark.WithExtensions(extension.GFM),
		goldmark.WithRendererOptions(
			html.WithHardWraps(),
			html.WithXHTML(),
		),
	)

	funcs := sprig.HtmlFuncMap()
	funcs["markdown"] = func(value string) (template.HTML, error) {
		var buffer bytes.Buffer
		err := md.Convert([]byte(value), &buffer)
		return template.HTML(buffer.String()), err //nolint:gosec
	}
	funcs["html"] = func(value string) template.HTML {
		return template.HTML(value) //nolint:gosec
	}
	funcs["timezone"] = func() string {
		return time.Local.String()
	}
	return funcs
}
