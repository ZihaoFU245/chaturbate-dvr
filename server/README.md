# server package

Purpose: Hosts application‑wide singletons: the global runtime configuration and the manager interface used by other packages at runtime.

Files:
- `config.go`
  - `var Config *entity.Config` → global pointer set in `main.start` via `config.New`.
- `manager.go`
  - `var Manager IManager` → global implementation set in `main.start` via `manager.New()`.
  - `type IManager` interface summarizing operations exposed to the web layer and channels:
    - Channel lifecycle: `CreateChannel`, `StopChannel`, `PauseChannel`, `ResumeChannel`.
    - Query: `ChannelInfo()`.
    - Pub/Sub: `Publish(name string, ch *entity.ChannelInfo)`, `Subscriber(w, r)`.
    - Persistence: `LoadConfig()`, `SaveConfig()`.

Used by:
- Almost all packages import `server` to read `Config` or to call `Manager` methods (e.g., `router` handlers, `channel` Publisher, `internal` request headers).