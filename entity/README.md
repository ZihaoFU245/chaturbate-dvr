# entity package

Purpose: Shared domain types used across the application: configuration, channel metadata for the UI, and event identifiers.

Types:
- `type Event = string` and constants:
  - `EventUpdate` (channel info/state update)
  - `EventLog` (log lines update)
- `type ChannelConfig`:
  - Per-channel settings: `IsPaused`, `Username`, `Framerate`, `Resolution`, `Pattern`, `MaxDuration` (minutes), `MaxFilesize` (MB), `CreatedAt` (unix seconds).
  - `Sanitize()`: strips invalid characters and trims spaces from `Username`.
- `type ChannelInfo`:
  - View model used by templates: online/paused booleans, display strings for `Duration`/`Filesize`, current filename, timestamps, and a pointer to global `Config` for nested templates.
- `type Config`:
  - Global app configuration populated from CLI flags: version, username, admin credentials for web UI, recording quality options, file split thresholds, web port, polling interval, cookies, user agent, and base domain.

Used by:
- `config` (produces `*Config`).
- `channel` (consumes `ChannelConfig`; exports `ChannelInfo`).
- `manager`/`server`/`router` (pass `ChannelInfo` and `Event`s).