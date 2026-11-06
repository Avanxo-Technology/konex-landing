// Package site is this site's own code: its routes and handlers.
//
// Everything a handler needs from core comes through the Router: CMS() for
// content, CachedRender() and Render() for pages, State() for application
// data. Core already serves /healthz, /robots.txt, /sitemap.xml, /llms.txt,
// /favicon.ico and POST /cache/purge; registering one of those paths here
// replaces core's.
package site

import (
	"context"
	"errors"
	"net/http"

	"github.com/Avanxo-Technology/gosite-cli/core"
	"github.com/Avanxo-Technology/gosite-cli/core/cms"
)

// Site implements gosite.App. Add Middleware, TemplateData or OnPurge methods
// when the site needs them (see gosite.Middlewarer, TemplateDataer, Purger).
type Site struct{}

func New() *Site { return &Site{} }

// Routes registers the site's pages.
//
// TEMPORARY: "/" serves the under-construction page. When the real home is
// ready, restore the "home" handler below (it reads the Cockpit "home"
// singleton) and delete theme/pages/construccion.html.
func (s *Site) Routes(r gosite.Router) {
	r.GET("/", func(c *gosite.Context) error {
		return r.Render(c, http.StatusOK, "construccion", gosite.Page{
			Title:   "Próximamente | konex-landing",
			SEOData: map[string]any{"noIndex": true},
		})
	})
	r.POST("/lista-espera", joinWaitlist(r))
}

// homeHandler is the real home page, parked until the site goes live. To
// enable it: r.GET("/", homeHandler(r)).
func homeHandler(r gosite.Router) gosite.HandlerFunc {
	return func(c *gosite.Context) error {
		return r.CachedRender(c, "home", func(ctx context.Context) (gosite.Page, error) {
			content, err := r.CMS().SingletonErr(ctx, "home")
			// A "home" singleton nobody has created yet is not a broken site:
			// the page renders its own fallback text until an editor adds it.
			if errors.Is(err, cms.ErrNotFound) {
				content, err = nil, nil
			}
			return gosite.Page{Title: "konex-landing", Content: content}, err
		})
	}
}
