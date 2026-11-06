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

// shareImage is the og:image / twitter:image of the page: the 1200x630 image
// uploaded to Cockpit as the webapp "Default Image".
//
// It is set here, with an absolute URL, because core builds that image from the
// Cockpit asset path and, with local storage, emits it relative
// ("/storage/uploads/..."). Crawlers (WhatsApp, Facebook, LinkedIn, X) ignore a
// relative og:image, so no preview shows. Drop this override once core makes
// the SEO image absolute with the site URL, as the blog addon already does.
// If the image is re-uploaded in Cockpit its path changes: update it here.
const shareImage = "https://konexobra.com/storage/uploads/2026/11/06/konex-og-1200x630_uid_6aee2667c4945.jpg"

// Routes registers the site's pages.
//
// TEMPORARY: "/" serves the under-construction page. When the real home is
// ready, restore the "home" handler below (it reads the Cockpit "home"
// singleton) and delete theme/pages/construccion.html.
func (s *Site) Routes(r gosite.Router) {
	r.GET("/", func(c *gosite.Context) error {
		return r.Render(c, http.StatusOK, "construccion", gosite.Page{
			Title: "Próximamente | konex-landing",
			// Indexable on purpose. Title, description, site name and language come
			// from the Cockpit "webapp" singleton; only the share image is set here
			// (see shareImage).
			SEOData: map[string]any{"image": shareImage},
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
