# router package

Purpose: Web layer built with Gin. It wires authentication, static assets, HTML templates, route handlers, and the SSE updates endpoint used by the UI.

Main files:
- router.go
  - `SetupRouter() *gin.Engine`:
    - Sets Gin to release mode.
    - Loads embedded templates from `router/view` (index and channel_info).
    - Applies basic auth when `server.Config.AdminUsername` and `AdminPassword` are set.
    - Registers static file serving and HTTP routes.
  - `SetupAuth(r)`:
    - Enables basic auth middleware when admin creds are present.
  - `SetupStatic(r)`:
    - Serves files from the embedded `router/view/templates` directory at `/static`.
  - `SetupViews(r)`:
    - Registers routes: `GET /` (index), `GET /updates` (SSE), and POST actions for create/pause/resume/stop/update_config.
  - `LoadHTMLFromEmbedFS(...)`:
    - Loads specific templates from embed FS and registers them with Gin.
- router_handler.go
  - Page model `IndexData` (global config + channels).
  - Handlers:
    - `Index` → renders main page.
    - `Updates` → SSE streaming via `server.Manager.Subscriber`.
    - `CreateChannel` → creates one or multiple comma‑separated usernames.
    - `StopChannel`, `PauseChannel`, `ResumeChannel` → delegate to `server.Manager`.
    - `UpdateConfig` → updates `server.Config.Cookies` & `UserAgent`.
- view/
  - Separate package (`router/view`) that embeds templates and exposes helpers.

Used by:
- `main.start` (when no `--username` is provided) to run the Web UI on the configured port.