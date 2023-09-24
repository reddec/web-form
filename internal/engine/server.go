package engine

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"log/slog"
	"net/http"
	"strconv"

	"sql-web-form/internal/assets"
	"sql-web-form/internal/schema"
	"sql-web-form/internal/utils"

	"github.com/go-chi/chi/v5"
)

var ErrDuplicatedName = errors.New("duplicated form name")

type Config struct {
	Forms      []schema.Form
	Storage    Storage
	Dispatcher WebhooksDispatcher
	Listing    bool
}

func New(cfg Config, options ...FormOption) (http.Handler, error) {
	templates, err := template.New("").Funcs(utils.TemplateFuncs()).ParseFS(assets.Views, "views/*.gohtml")
	if err != nil {
		return nil, fmt.Errorf("parse templates: %w", err)
	}
	listView := templates.Lookup("list.gohtml")
	if err != nil {
		return nil, fmt.Errorf("get static dir: %w", err)
	}

	renderer := &Renderer{
		funcs: utils.TemplateFuncs(),
	}

	mux := chi.NewMux()

	var usedName = utils.NewSet[string]()
	for _, formDef := range cfg.Forms {
		if usedName.Has(formDef.Name) {
			return nil, fmt.Errorf("form %q: %w", formDef.Name, ErrDuplicatedName)
		}
		usedName.Add(formDef.Name)
		mux.Mount("/forms/"+formDef.Name, NewForm(FormConfig{
			Definition:         formDef,
			Renderer:           renderer,
			ViewForm:           templates.Lookup("form.gohtml"),
			ViewResult:         templates.Lookup("result.gohtml"),
			Storage:            cfg.Storage,
			WebhooksDispatcher: cfg.Dispatcher,
		}, options...))
	}
	if cfg.Listing {
		mux.Get("/", listViewHandler(cfg.Forms, renderer, listView))
	}
	return mux, nil
}

func listViewHandler(forms []schema.Form, render *Renderer, listView *template.Template) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		vc := &serverViewContext{
			View:        render.View(request),
			Definitions: forms,
		}

		var buffer bytes.Buffer
		err := listView.Execute(&buffer, vc)
		if err != nil {
			slog.Error("failed render list view", "error", err)
			writer.WriteHeader(http.StatusInternalServerError)
			return
		}

		writer.Header().Set("Content-Type", "text/html")
		writer.Header().Set("Content-Length", strconv.Itoa(buffer.Len()))
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write(buffer.Bytes())
	}
}

type serverViewContext struct {
	*View
	Definitions []schema.Form
}
