# chaturbate package

Purpose: Low-level integration with Chaturbate pages and HLS playlists. It discovers a room's HLS source URL from the HTML, selects a suitable variant playlist by resolution/FPS, and continuously fetches `.ts` media segments.

Main types and functions:
- `type Client`:
  - Wraps `internal.Req` for HTTP.
  - `NewClient()` → constructor.
  - `GetStream(ctx, username)` → returns a `*Stream` for the given channel username (via `FetchStream`).
- `FetchStream(ctx, req, username)`:
  - Requests the channel page at `server.Config.Domain + username`.
  - Verifies `playlist.m3u8` exists; otherwise returns `internal.ErrChannelOffline`.
  - Calls `ParseStream`.
- `ParseStream(body)`:
  - Extracts `window.initialRoomDossier` from HTML via regex.
  - Decodes JSON and reads `hls_source` → returns `&Stream{HLSSource}`.
- `type Stream`:
  - Holds `HLSSource` (master playlist URL).
  - `GetPlaylist(ctx, resolution, framerate)` → `FetchPlaylist`.
- `FetchPlaylist(ctx, hlsSource, resolution, framerate)`:
  - GETs the master playlist, then `ParsePlaylist`.
- `ParsePlaylist(resp, hlsSource, resolution, framerate)`:
  - Decodes using `github.com/grafov/m3u8` as a master playlist.
  - Calls `PickPlaylist` to select a variant.
- `type Playlist` { PlaylistURL, RootURL, Resolution, Framerate }:
  - URL fields are derived from the HLSSource base + variant URI.
- `PickPlaylist(master, baseURL, resolution, framerate)`:
  - Builds a map of available widths → framerates → URIs.
  - Prefers an exact resolution; otherwise falls back to the highest below the target.
  - Prefers requested FPS; otherwise picks first available.
- `type WatchHandler func(b []byte, duration float64) error`:
  - Callback type invoked for each media segment.
- `(*Playlist) WatchSegments(ctx, handler)`:
  - Polls the media playlist, tracks `lastSeq` via `internal.SegmentSeq`.
  - Fetches new `.ts` segments with retry (`avast/retry-go`).
  - Calls `handler` with segment bytes and duration.

Key dependencies:
- `internal.Req` for HTTP with headers/cookies/user agent from `server.Config`.
- `github.com/grafov/m3u8` for playlist parsing.
- `github.com/avast/retry-go/v4` for robust segment fetching.

Used by:
- The `channel` package (`RecordStream` → `GetPlaylist` → `WatchSegments`).