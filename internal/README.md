# internal package

Purpose: Shared utilities and low-level helpers used across the application.

Files and responsibilities:
- internal.go
  - `FormatDuration(seconds float64) string` → `h:mm:ss` formatting (empty string for 0).
  - `FormatFilesize(bytes int) string` → human-readable sizes (KB/MB/GB; empty for 0).
  - `SegmentSeq(filename string) int` → extracts trailing `_NNN.ts` sequence number; returns `-1` if not found.
- internal_err.go
  - Common sentinel errors used for control flow: `ErrChannelExists`, `ErrChannelNotFound`, `ErrCloudflareBlocked`, `ErrAgeVerification`, `ErrChannelOffline`, `ErrPrivateStream`, `ErrPaused`, `ErrStopped`.
- internal_req.go
  - `type Req` and HTTP utilities.
  - `NewReq()` creates an `http.Client` with a cloned transport and disabled TLS verification (to be tolerant of sources).
  - `CreateTransport()` clones `http.DefaultTransport` (honors env proxies) and sets `InsecureSkipVerify`.
  - `(*Req) Get(ctx, url) (string, error)` and `GetBytes(ctx, url) ([]byte, error)` with checks for Cloudflare/Age verification content and 403 → `ErrPrivateStream`.
  - `CreateRequest(ctx, url) (*http.Request, context.CancelFunc, error)`: adds timeout, calls `SetRequestHeaders`.
  - `SetRequestHeaders(req)`: sets `X-Requested-With`, optional `User-Agent` and cookies based on `server.Config`.
  - `ParseCookies("k=v; k2=v2") map[string]string`.

Used by:
- `chaturbate` (HTTP/playlist fetching, segment downloads).
- `channel` (formatting, segment sequence parsing for playlist handling).
- Other packages indirectly through error values.