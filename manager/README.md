# manager package

Purpose: Orchestrates channel lifecycles and provides a Server‑Sent Events (SSE) hub for the web UI. It owns the in‑memory set of channels, persists/restores channel configs, and publishes UI updates.

Key type:
- `type Manager struct`:
  - Fields: `Channels` (sync.Map username → *channel.Channel), `SSE` (*sse.Server).

Construction:
- `New() (*Manager, error)`:
  - Creates an SSE server with stream `updates`.

Channel management:
- `CreateChannel(conf *entity.ChannelConfig, shouldSave bool) error`:
  - Sanitizes username, prevents duplicates, stores channel and starts monitoring.
  - Optionally persists to `./conf/channels.json`.
- `StopChannel(username)`, `PauseChannel(username)`, `ResumeChannel(username)`:
  - Control per‑channel state and persist configuration.
- `ChannelInfo() []*entity.ChannelInfo`:
  - Collects `ExportInfo()` for all channels and sorts: online first, then by username.

Persistence:
- `SaveConfig()` writes the slice of `ChannelConfig` to `./conf/channels.json`.
- `LoadConfig()` reads persisted channels, creates `channel.Channel` instances, and resumes (staggered) unless paused.

SSE / Web integration:
- `Publish(evt entity.Event, info *entity.ChannelInfo)`:
  - For `EventUpdate`: renders partial HTML via `router/view.InfoTpl` and publishes to `updates` stream with event `<username>-info`.
  - For `EventLog`: publishes newline‑joined logs with event `<username>-log`.
- `Subscriber(w, r)`:
  - Gin handler helper: `m.SSE.ServeHTTP(w, r)`.

Used by:
- `main` (instantiation and global `server.Manager`), `router` (invoke actions, subscribe for updates), and `channel` (publishing via `server.Manager`).