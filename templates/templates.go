package templates

import (
	"embed"
	"html/template"
	"net/http"

	"github.com/pkg/errors"
)

//go:embed *.tmpl
var templates embed.FS

// Render out page to (w, r) with parameters vs.
func Render(w http.ResponseWriter, r *http.Request, page string, vs map[string]interface{}) error {
	var ts *template.Template
	var err error

	if vs == nil {
		return errors.New("passed empty vs")
	}

	if ts, err = template.New("base").ParseFS(templates, "base.tmpl"); err != nil {
		return errors.Wrap(err, "could not parse base template")
	}

	if _, err := ts.New("page").ParseFS(templates, page); err != nil {
		return errors.Wrap(err, "could not parse page template")
	}

	return ts.Execute(w, vs)
}
