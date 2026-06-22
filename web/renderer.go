package web

import (
	"bytes"
	"fmt"
	"html/template"
	"net/http"

	"github.com/navikt/union-api/pkg/auth"
)

// NavItem describes a single entry in the top navigation bar.
// Add a new NavItem to navItems when a new page is introduced.
type NavItem struct {
	Label string
	Path  string
	Key   string
}

// navItems is the single source of truth for the navigation bar.
// Order here controls the order rendered in the navbar.
var navItems = []NavItem{
	{Label: "Service Accounts", Path: "/serviceaccounts", Key: "serviceaccounts"},
}

// layout is the data envelope passed to every page template.
// It provides the shared chrome (navbar, user) alongside the page-specific Data.
type layout struct {
	User   string
	Active string
	Nav    []NavItem
	Data   any
}

// Renderer holds one parsed template set per page.
// Using a map of sets avoids the {{define "content"}} collision that occurs
// when all page templates are merged into a single *template.Template.
type Renderer struct {
	pages map[string]*template.Template
}

// New parses all page templates at startup (fail fast if a template is broken)
// and returns a Renderer ready to serve requests.
func New() (*Renderer, error) {
	pages := map[string]string{
		"serviceaccounts": "templates/serviceaccounts.html",
	}

	r := &Renderer{pages: make(map[string]*template.Template, len(pages))}

	for key, pagePath := range pages {
		tmpl, err := template.ParseFS(templateFS, "templates/base.html", pagePath)
		if err != nil {
			return nil, fmt.Errorf("web: parsing template %q: %w", key, err)
		}
		r.pages[key] = tmpl
	}

	return r, nil
}

// Render executes the named page template with the given data.
// It writes into a buffer first so that a template error never produces a
// partial response — the caller sees a clean 500 instead.
func (r *Renderer) Render(w http.ResponseWriter, page string, p *auth.Principal, data any) {
	tmpl, ok := r.pages[page]
	if !ok {
		http.Error(w, fmt.Sprintf("unknown page: %q", page), http.StatusInternalServerError)
		return
	}

	vm := layout{
		User:   p.Name,
		Active: page,
		Nav:    navItems,
		Data:   data,
	}

	var buf bytes.Buffer
	if err := tmpl.ExecuteTemplate(&buf, "base", vm); err != nil {
		http.Error(w, "render error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = buf.WriteTo(w)
}
