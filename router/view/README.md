# router/view package

Purpose: Embeds and exposes the web UI templates and static assets to the router, and provides helpers for rendering and serving them.

Files:
- `view.go`
  - `//go:embed templates` → embeds all files under `templates/` into an `embed.FS` (`FS`).
  - `var InfoTpl *template.Template` → parsed partial used to render channel info blocks updated over SSE.
  - `init()` → parses `templates/channel_info.html` into `InfoTpl`.
  - `StaticFS() (http.FileSystem, error)` → returns a filesystem rooted at `templates/` for serving static files under `/static`.

Templates and static assets:
- `templates/index.html` → main page layout listing channels and controls.
- `templates/channel_info.html` → partial snippet used for per‑channel info updates via SSE.
- Icon files and scripts (`htmx.min.js`, `sse.min.js`) are also embedded and served from `/static`.

Used by:
- `router.SetupRouter()` to load HTML templates and to mount static files.
- `manager.Publish(EventUpdate, ...)` uses `InfoTpl` to render the updated partial HTML for a channel.