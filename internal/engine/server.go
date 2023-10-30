package engine

import (
	"errors"
	"fmt"
	"html/template"
	"io/fs"
	"net/http"

	"github.com/reddec/web-form/internal/assets"
	"github.com/reddec/web-form/internal/schema"
	"github.com/reddec/web-form/internal/utils"
	"github.com/reddec/web-form/internal/web"

	"github.com/go-chi/chi/v5"
)

var ErrDuplicatedName = errors.New("duplicated form name")

type Config struct {
	Forms           []schema.Form
	Storage         Storage
	WebhooksFactory WebhooksFactory
	AMQPFactory     AMQPFactory
	Listing         bool
	Captcha         []web.Captcha
}

func New(cfg Config, options ...FormOption) (http.Handler, error) {
	views := assets.InsideViews()
	listView := mustParse(views, "list.gohtml")
	viewForm := mustParse(views, "form_base.gohtml", "form.gohtml")
	viewSuccess := mustParse(views, "form_base.gohtml", "success.gohtml")
	viewFail := mustParse(views, "form_base.gohtml", "failed.gohtml")
	viewCode := mustParse(views, "form_base.gohtml", "access.gohtml")
	viewForbidden := mustParse(views, "form_base.gohtml", "forbidden.gohtml")

	mux := chi.NewMux()

	var usedName = utils.NewSet[string]()
	for _, formDef := range cfg.Forms {
		if usedName.Has(formDef.Name) {
			return nil, fmt.Errorf("form %q: %w", formDef.Name, ErrDuplicatedName)
		}
		usedName.Add(formDef.Name)
		mux.Mount("/forms/"+formDef.Name, NewForm(FormConfig{
			Definition:      formDef,
			ViewForm:        viewForm,
			ViewSuccess:     viewSuccess,
			ViewFail:        viewFail,
			ViewCode:        viewCode,
			ViewForbidden:   viewForbidden,
			Storage:         cfg.Storage,
			WebhooksFactory: cfg.WebhooksFactory,
			AMQPFactory:     cfg.AMQPFactory,
			Captcha:         cfg.Captcha,
		}, options...))
	}
	if cfg.Listing {
		mux.Get("/", listViewHandler(cfg.Forms, listView))
	}
	return mux, nil
}

func listViewHandler(forms []schema.Form, listView *template.Template) http.HandlerFunc {
	return func(writer http.ResponseWriter, request *http.Request) {
		req := web.NewRequest(writer, request)

		creds := schema.CredentialsFromContext(request.Context())
		var filteredForms = make([]schema.Form, 0, len(forms))
		for _, f := range forms {
			if f.IsAllowed(creds) {
				filteredForms = append(filteredForms, f)
			}
		}

		req.Set("Definitions", filteredForms)
		req.Set("Context", newRequestContext(req))
		req.Render(http.StatusOK, listView)
	}
}

func mustParse(src fs.FS, base string, overlay ...string) *template.Template {
	v, err := baseParse(src, base, overlay...)
	if err != nil {
		panic(err)
	}
	return v
}

func baseParse(src fs.FS, base string, overlay ...string) (*template.Template, error) {
	var root = template.New("").Funcs(utils.TemplateFuncs())
	var files = append([]string{base}, overlay...)

	for _, file := range files {
		content, err := fs.ReadFile(src, file)
		if err != nil {
			return nil, fmt.Errorf("read %q: %w", file, err)
		}
		sub, err := root.Parse(string(content))
		if err != nil {
			return nil, fmt.Errorf("parse %q: %w", file, err)
		}
		root = sub
	}
	return root, nil
}
