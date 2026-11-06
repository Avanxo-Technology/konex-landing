# konex-landing - project memory

Landing for Konex (rents machinery for construction sites and delivers it door to
door; owners and contractors follow projects, operators and hours from the phone).
A gosite **thin site** (core 0.64.1): Go + Echo + html/template + Cockpit. Plain CSS
(no Tailwind). Storage: S3/MinIO, database: MongoDB. Addons: Forms, Blog,
Analytics, Replica. Read `AGENTS.md` for the thin-site rules.

## Current state: "Próximamente" page + waitlist (temporary)

- `/` serves `theme/pages/construccion.html`, rendered with `r.CachedRender` (it reads
  the Cockpit `webapp` singleton for the share image, so a publish reaches it via purge).
  Fonts (Barlow, Barlow Condensed, JetBrains Mono) load through the layout's
  `{{define "head"}}` block. Styles: the `.construccion` block and the
  "Konex · Próximamente" `.kx-*` block at the end of `static/styles.css`.
- Layout: mobile-first. The page scrolls on mobile and on short laptop screens;
  on a tall desktop it is exactly one screen (`min-height: 100dvh`). The phone
  mockup (`.mock-wrap`) floats (`kx-float`, off with `prefers-reduced-motion`).
  Sideways overflow is clipped on `body:has(.construccion) .container`, NOT on
  `.construccion`: that section is narrower than the window on wide screens and
  clipping there cuts the phone's cast shadow in a hard line.
- The real home handler is parked as `homeHandler(r)` in `site/site.go` (reads the
  Cockpit `home` singleton). It is unused on purpose.
- Why a hard-coded page and not the `home` singleton or a temporary singleton: the
  `home` singleton defines the real page's field schema; filling it with a
  placeholder would force a schema change later. A temporary singleton adds a model
  to clean up. A template needs neither.

### Waitlist (`POST /lista-espera`, `site/waitlist.go`)

- Flow: validate the e-mail -> claim it in Redis (`r.State().SetNX`, key
  `waitlist:email:<lowercased e-mail>`, value = first sign-up time) -> forward
  server to server to the Cockpit **Forms** receiver (`COCKPIT_URL` +
  `/forms/api/submit`, form `lista-espera`). The data lives in Cockpit
  (collection `formSubmissions`; CMS panel > Forms; CSV export). Redis is only the
  duplicate guard and is released if Cockpit rejects or is unreachable.
- Duplicates and honeypot hits (`_hp`) answer "Anotado" without storing anything,
  so a visitor cannot tell whether an address was already there.
- The visitor's IP and User-Agent are forwarded in `X-Forwarded-For` / `User-Agent`.
  Cockpit reads the LAST `X-Forwarded-For` entry (`forms.trustedProxies = 1`). Without
  that, Cockpit sees this container as the client and its throttle (3 s between
  sign-ups and 50 a day per IP+form) is shared by every visitor. A throttle hit is a 429.
- The page upgrades the form to `fetch` (no reload, shows "Anotado. Te escribimos
  apenas abramos." in amber); without JS a plain POST re-renders the page.
- Gotchas: deleting a row in Cockpit does NOT release the Redis key, so that
  address cannot sign up again until `redis-cli del
  konex-landing:app:waitlist:email:<e-mail>`. Whether production Redis persists
  (local has `appendonly yes`) was not checked; if it is flushed, guards are lost
  and a repeated address could be stored twice.
- No e-mail is sent to anyone. The promise on the page is "Un correo al lanzar":
  on launch day export the CSV from Forms and send from your usual tool. SMTP
  (`SMTP_HOST`, ...) is only needed for Forms notifications to the team or for
  sending from Cockpit; it is not configured.
- Replica to prod: never replicate `formSubmissions` or `formSettings` (production
  has its own sign-ups; `formSettings` holds `webhookSecret`). Use `merge`, dry run
  first, select only the models you edited; code-only changes need no Replica at all.

### SEO and share preview

- Title, description, site name, site URL (`https://konexobra.com`), language,
  `robots.txt` and `llms.txt` text are configured in the Cockpit **webapp**
  singleton, not in code. Robots there: `Allow: /` and `Disallow: /lista-espera`
  (that path only receives POSTs; the line is optional tidiness, not security).
  Google reads the page: the old `noIndex: true` was removed on purpose.
- Core emits `og:image` RELATIVE (`/storage/uploads/...`) when assets are local, and
  crawlers (WhatsApp, Facebook, LinkedIn, X) ignore a relative image, so no preview
  shows. Fix in `site/seo.go` (`shareImageURL`), called from the `/` handler: it reads
  the Cockpit webapp singleton (`defaultImage` + `siteUrl`) and builds the absolute URL,
  mirroring core's asset base (`STORAGE_ADAPTER` / `S3_PUBLIC_URL`, else
  `/storage/uploads`). Nothing is hard-coded: changing "Default Image" in Cockpit changes
  the preview. If `siteUrl` is empty and the image is relative it sets nothing (core's
  relative value stays and no preview shows), so keep "Site URL" filled in.
  DELETE `site/seo.go` once core makes the SEO image absolute with the site URL (the blog
  addon already does); that is a gosite gap, report it there.
- After changing a preview, force a re-scrape (Facebook Sharing Debugger >
  "Scrape again"; LinkedIn Post Inspector). WhatsApp caches the first preview.
- The share image was generated from the page itself (1200x630, logo + headline +
  phone over the dirt background, no "próximamente" text so it outlives the launch).

### To go live

1. In `site/site.go`: `r.GET("/", homeHandler(r))`.
2. Delete `theme/pages/construccion.html`, the `.construccion` and `.kx-*` rules in
   `static/styles.css`, and `site/seo.go` (+ its call in `site/site.go`) if core already
   absolutizes the SEO image.
3. Decide about `POST /lista-espera` (keep it for a signup block on the real home, or
   remove it with `site/waitlist.go`); keep the Forms data and plan the launch e-mail.
4. Create the `home` singleton in Cockpit (grant `forms/manage` and
   `replica/manage` in Settings > Roles after first login).
5. Check the SEO of the real page (own title/description/image per page) and add a
   `sitemap.xml` (it is a 404 today) and its `Sitemap:` line in robots.txt.

## Setup notes

- Local: `gosite infra up`, `gosite start konex-landing`. App
  https://konex-landing.test (:8004), CMS https://cms.konex-landing.test (:8005).
- Go is NOT installed on the host. Run it in Docker, e.g.
  `docker run --rm -u $(id -u):$(id -g) -e HOME=/tmp -e GOFLAGS=-buildvcs=false -v "$PWD":/src -w /src golang:1.25 go test ./...`
- `docker-compose.override.yml` sets `GOFLAGS=-buildvcs=false` for the app: the
  source is bind-mounted, git inside the container refuses it ("dubious
  ownership") and air's `go build` fails stamping VCS info.
- Upgrading core: tags are `core/vX.Y.Z` (the CLI and the core can differ; check
  `git ls-remote --tags https://github.com/Avanxo-Technology/gosite-cli.git 'core/*'`).
  The `core/v0.64.0` tag pointed at an old commit; 0.64.1 is the good one.
- Repo: `git@github.com:Avanxo-Technology/konex-landing.git`. Production: Coolify
  on `docker-compose.prod.yml`. Branches: work in a feature branch, PR to
  `development`, then to `main`; `development` is kept equal to `main`.
- Production `.env`: leave `S3_ACL` unset and set `S3_VERIFY=true`; point
  `S3_URL` / `S3_PUBLIC_URL` at the real bucket or CDN (local MinIO uses a
  self-signed cert, hence `S3_VERIFY=false` locally). With `STORAGE_ADAPTER=local`
  (the compose default) asset URLs are always relative to the site.
