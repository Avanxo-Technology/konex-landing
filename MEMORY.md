# konex-landing - project memory

Landing for Konex. A gosite **thin site** (core 0.60.0): Go + Echo + html/template
+ Cockpit. Plain CSS (no Tailwind). Storage: S3/MinIO, database: MongoDB.
Addons: Forms, Blog, Analytics, Replica. Read `AGENTS.md` for the thin-site rules.

## Current state: under-construction page (temporary)

- `/` serves `theme/pages/construccion.html` (title "Próximamente | konex-landing",
  `noIndex: true`, rendered with `r.Render`, not cached). Styles: `.construccion`
  block at the end of `static/styles.css`.
- The real home handler is parked as `homeHandler(r)` in `site/site.go` (reads the
  Cockpit `home` singleton). It is unused on purpose.
- Why a hard-coded page and not the `home` singleton or a temporary singleton: the
  `home` singleton defines the real page's field schema; filling it with a
  placeholder would force a schema change later. A temporary singleton adds a model
  to clean up. A template needs neither.

### To go live

1. In `site/site.go`: `r.GET("/", homeHandler(r))`.
2. Delete `theme/pages/construccion.html` and the `.construccion` rules in
   `static/styles.css`.
3. Create the `home` singleton in Cockpit (grant `forms/manage` and
   `replica/manage` in Settings > Roles after first login).

## Setup notes

- Local: `gosite infra up`, `gosite start konex-landing`. App
  https://konex-landing.test (:8004), CMS https://cms.konex-landing.test (:8005).
- Repo: `git@github.com:Avanxo-Technology/konex-landing.git`. Production: Coolify
  on `docker-compose.prod.yml`.
- Production `.env`: leave `S3_ACL` unset and set `S3_VERIFY=true`; point
  `S3_URL` / `S3_PUBLIC_URL` at the real bucket or CDN (local MinIO uses a
  self-signed cert, hence `S3_VERIFY=false` locally).
