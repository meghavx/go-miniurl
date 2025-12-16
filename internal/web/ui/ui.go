package ui

import (
	"html/template"
	"net/http"
)

var formTmpl = template.Must(
	template.ParseFiles("static/partials/url-form.html"),
)

func RenderForm(w http.ResponseWriter, r *http.Request) {
	data := struct {
		Label       string
		Placeholder string
		Endpoint    string
	}{
		Label:       r.URL.Query().Get("label"),
		Placeholder: r.URL.Query().Get("placeholder"),
		Endpoint:    r.URL.Query().Get("endpoint"),
	}

	// Sensible defaults
	if data.Label == "" {
		data.Label = "Enter URL"
	}
	if data.Placeholder == "" {
		data.Placeholder = "https://example.com"
	}
	if data.Endpoint == "" {
		data.Endpoint = "/shorten-url"
	}

	w.Header().Set("Content-Type", "text/html")
	_ = formTmpl.Execute(w, data)
}
