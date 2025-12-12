# channel package

Purpose: Encapsulates the lifecycle of a single recording "channel" (a Chaturbate username). It manages monitoring, recording, file rotation, state, and publishing updates/logs to the web UI via SSE.

Key types and files:
- channel.go
  - `type Channel`: in-memory state for one channel (pause/online status, timing/size counters, current file handle, etc.).
  - `New(conf *entity.ChannelConfig) *Channel`: constructor; starts a background publisher.
  - `Publisher()`: listens for logs/updates and publishes via `server.Manager.Publish`.
  - `WithCancel(ctx)`: creates/stores a `CancelFunc` to stop/pause monitoring.
  - `Info`, `Error`: logging helpers that also stream logs to UI.
  - `ExportInfo() *entity.ChannelInfo`: converts runtime state to UI-friendly info.
  - `Pause()`, `Stop()`, `Resume(startSeq int)`: control lifecycle; `Resume` starts monitoring after an optional stagger delay.
  - `UpdateOnlineStatus(isOnline bool)`: sets online flag and publishes.
- channel_record.go
  - `Monitor()`: long-running loop with retry that discovers a stream, selects a playlist, and consumes segments.
  - `Update()`: sends an update signal for SSE broadcast.
  - `RecordStream(ctx, client)`: resolves playlist for configured resolution/FPS; calls `WatchSegments`.
  - `HandleSegment(b, duration)`: appends TS bytes to current file, updates counters, rotates file if max limits reached.
- channel_file.go
  - `Pattern` struct: fields used by filename template.
  - `NextFile()`, `Cleanup()`: close/remove zero-sized file and prepare next file for rotation.
  - `GenerateFilename()`: applies `ChannelConfig.Pattern` using `text/template`-like syntax (actually `html/template`).
  - `CreateNewFile(name)`: ensures directory exists and opens `name.ts` for append.
  - `ShouldSwitchFile()`: checks `MaxDuration`/`MaxFilesize` thresholds.

Important interactions:
- Uses `chaturbate` package to fetch stream and playlist and to watch HLS segments.
- Uses `internal` helpers for formatting, HTTP, and parsing.
- Publishes events via `server.Manager` and reads global `server.Config`.
- Uses `entity` for shared types (`ChannelConfig`, `ChannelInfo`).

Runtime flow (per channel):
1) `Resume` → `Monitor` loop → `RecordStream` resolves playlist.
2) `WatchSegments` pulls `.ts` parts and calls `HandleSegment` for each.
3) `HandleSegment` writes to file, updates state, triggers UI updates, and rotates file when needed.
4) On errors/offline/private/Cloudflare: retry with fixed delay; pause/stop cancel the context.