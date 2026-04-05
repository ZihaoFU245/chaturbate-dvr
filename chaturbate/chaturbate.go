package chaturbate

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/avast/retry-go/v4"
	"github.com/grafov/m3u8"
	"github.com/samber/lo"
	"github.com/zihaofu245/chaturbate-dvr/internal"
	"github.com/zihaofu245/chaturbate-dvr/server"
)

const roomDossierMarker = "window.initialRoomDossier"

// Client represents an API client for interacting with Chaturbate.
type Client struct {
	Req *internal.Req
}

// NewClient initializes and returns a new Client instance.
func NewClient() *Client {
	return &Client{
		Req: internal.NewReq(),
	}
}

// GetStream fetches the stream information for a given username.
func (c *Client) GetStream(ctx context.Context, username string) (*Stream, error) {
	return FetchStream(ctx, c.Req, username)
}

// FetchStream retrieves the streaming data from the given username's page.
func FetchStream(ctx context.Context, client *internal.Req, username string) (*Stream, error) {
	body, err := client.Get(ctx, fmt.Sprintf("%s%s", server.Config.Domain, username))
	if err != nil {
		return nil, fmt.Errorf("failed to get page body: %w", err)
	}

	return ParseStream(body)
}

// ParseStream extracts the HLS source URL from the given page body.
func ParseStream(body string) (*Stream, error) {
	sourceData, err := extractRoomDossier(body)
	if err != nil {
		return nil, err
	}

	var room struct {
		RoomStatus string `json:"room_status"`
		HLSSource  string `json:"hls_source"`
	}
	if err := json.Unmarshal([]byte(sourceData), &room); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}
	if room.RoomStatus == "offline" {
		return nil, internal.ErrChannelOffline
	}
	if room.RoomStatus != "" && room.RoomStatus != "public" {
		return nil, internal.ErrPrivateStream
	}
	if room.HLSSource == "" {
		return nil, internal.ErrChannelOffline
	}

	return &Stream{HLSSource: room.HLSSource}, nil
}

func extractRoomDossier(body string) (string, error) {
	idx := strings.Index(body, roomDossierMarker)
	if idx == -1 {
		return "", internal.ErrChannelOffline
	}

	assignment := body[idx+len(roomDossierMarker):]
	eq := strings.IndexByte(assignment, '=')
	if eq == -1 {
		return "", errors.New("room dossier assignment not found")
	}

	value := strings.TrimSpace(assignment[eq+1:])
	if value == "" {
		return "", errors.New("room dossier value is empty")
	}

	switch value[0] {
	case '"', '\'':
		decoded, err := parseQuotedAssignment(value)
		if err != nil {
			return "", fmt.Errorf("parse room dossier string: %w", err)
		}
		return decoded, nil
	case '{':
		decoded, err := parseJSONObject(value)
		if err != nil {
			return "", fmt.Errorf("parse room dossier object: %w", err)
		}
		return decoded, nil
	default:
		return "", fmt.Errorf("unsupported room dossier format: %q", value[0])
	}
}

func parseQuotedAssignment(value string) (string, error) {
	quote := value[0]
	escaped := false
	for i := 1; i < len(value); i++ {
		if escaped {
			escaped = false
			continue
		}
		if value[i] == '\\' {
			escaped = true
			continue
		}
		if value[i] != quote {
			continue
		}

		literal := value[:i+1]
		if quote == '\'' {
			literal = `"` + strings.ReplaceAll(literal[1:len(literal)-1], `"`, `\\"`) + `"`
		}

		decoded, err := strconv.Unquote(literal)
		if err != nil {
			return "", err
		}
		return decoded, nil
	}

	return "", errors.New("unterminated quoted value")
}

func parseJSONObject(value string) (string, error) {
	depth := 0
	inString := false
	escaped := false
	var quote byte

	for i := 0; i < len(value); i++ {
		ch := value[i]
		if inString {
			if escaped {
				escaped = false
				continue
			}
			if ch == '\\' {
				escaped = true
				continue
			}
			if ch == quote {
				inString = false
			}
			continue
		}

		switch ch {
		case '"', '\'':
			inString = true
			quote = ch
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return value[:i+1], nil
			}
		}
	}

	return "", errors.New("unterminated JSON object")
}

// Stream represents an HLS stream source.
type Stream struct {
	HLSSource string
}

// GetPlaylist retrieves the playlist corresponding to the given resolution and framerate.
func (s *Stream) GetPlaylist(ctx context.Context, resolution, framerate int) (*Playlist, error) {
	return FetchPlaylist(ctx, s.HLSSource, resolution, framerate)
}

// FetchPlaylist fetches and decodes the HLS playlist file.
func FetchPlaylist(ctx context.Context, hlsSource string, resolution, framerate int) (*Playlist, error) {
	if hlsSource == "" {
		return nil, errors.New("HLS source is empty")
	}

	resp, err := internal.NewReq().Get(ctx, hlsSource)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch HLS source: %w", err)
	}

	return ParsePlaylist(resp, hlsSource, resolution, framerate)
}

// ParsePlaylist decodes the M3U8 playlist and extracts the variant streams.
func ParsePlaylist(resp, hlsSource string, resolution, framerate int) (*Playlist, error) {
	p, _, err := m3u8.DecodeFrom(strings.NewReader(resp), true)
	if err != nil {
		return nil, fmt.Errorf("failed to decode m3u8 playlist: %w", err)
	}

	masterPlaylist, ok := p.(*m3u8.MasterPlaylist)
	if !ok {
		return nil, errors.New("invalid master playlist format")
	}

	return PickPlaylist(masterPlaylist, hlsSource, resolution, framerate)
}

// Playlist represents an HLS playlist containing variant streams.
type Playlist struct {
	PlaylistURL string
	RootURL     string
	Resolution  int
	Framerate   int
}

// Resolution represents a video resolution and its corresponding framerate.
type Resolution struct {
	Framerate map[int]string // [framerate]url
	Width     int
}

// PickPlaylist selects the best matching variant stream based on resolution and framerate.
func PickPlaylist(masterPlaylist *m3u8.MasterPlaylist, baseURL string, resolution, framerate int) (*Playlist, error) {
	resolutions := map[int]*Resolution{}

	// Extract available resolutions and framerates from the master playlist
	for _, v := range masterPlaylist.Variants {
		parts := strings.Split(v.Resolution, "x")
		if len(parts) != 2 {
			continue
		}
		width, err := strconv.Atoi(parts[1])
		if err != nil {
			return nil, fmt.Errorf("parse resolution: %w", err)
		}
		framerateVal := 30
		if strings.Contains(v.Name, "FPS:60.0") {
			framerateVal = 60
		}
		if _, exists := resolutions[width]; !exists {
			resolutions[width] = &Resolution{Framerate: map[int]string{}, Width: width}
		}
		resolutions[width].Framerate[framerateVal] = v.URI
	}

	// Find exact match for requested resolution
	variant, exists := resolutions[resolution]
	if !exists {
		// Filter resolutions below the requested resolution
		candidates := lo.Filter(lo.Values(resolutions), func(r *Resolution, _ int) bool {
			return r.Width < resolution
		})
		// Pick the highest resolution among the candidates
		variant = lo.MaxBy(candidates, func(a, b *Resolution) bool {
			return a.Width > b.Width
		})
		if variant == nil {
			variant = lo.MinBy(lo.Values(resolutions), func(a, b *Resolution) bool {
				return a.Width < b.Width
			})
		}
	}
	if variant == nil {
		return nil, fmt.Errorf("resolution not found")
	}

	var (
		finalResolution = variant.Width
		finalFramerate  = framerate
	)
	// Select the desired framerate, or fallback to the first available framerate
	playlistURL, exists := variant.Framerate[framerate]
	if !exists {
		for fr, url := range variant.Framerate {
			playlistURL = url
			finalFramerate = fr
			break
		}
	}

	resolvedPlaylistURL, err := resolveURL(baseURL, playlistURL)
	if err != nil {
		return nil, fmt.Errorf("resolve playlist url: %w", err)
	}
	rootURL, err := parentURL(resolvedPlaylistURL)
	if err != nil {
		return nil, fmt.Errorf("resolve segment root url: %w", err)
	}

	return &Playlist{
		PlaylistURL: resolvedPlaylistURL,
		RootURL:     rootURL,
		Resolution:  finalResolution,
		Framerate:   finalFramerate,
	}, nil
}

func resolveURL(baseURL, ref string) (string, error) {
	base, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	resolved, err := base.Parse(ref)
	if err != nil {
		return "", err
	}
	return resolved.String(), nil
}

func parentURL(raw string) (string, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "", err
	}
	parsed.RawQuery = ""
	parsed.Fragment = ""
	path := parsed.Path
	idx := strings.LastIndex(path, "/")
	if idx == -1 {
		parsed.Path = ""
		return parsed.String(), nil
	}
	parsed.Path = path[:idx+1]
	return parsed.String(), nil
}

// WatchHandler is a function type that processes video segments.
type WatchHandler func(b []byte, duration float64) error

// WatchSegments continuously fetches and processes video segments.
func (p *Playlist) WatchSegments(ctx context.Context, handler WatchHandler) error {
	var (
		client      = internal.NewReq()
		lastSeq     uint64
		hasLastSeq  bool
		lastMapSpec string
	)

	for {
		// Fetch the latest playlist
		resp, err := client.Get(ctx, p.PlaylistURL)
		if err != nil {
			return fmt.Errorf("get playlist: %w", err)
		}
		pl, _, err := m3u8.DecodeFrom(strings.NewReader(resp), true)
		if err != nil {
			return fmt.Errorf("decode from: %w", err)
		}
		playlist, ok := pl.(*m3u8.MediaPlaylist)
		if !ok {
			return fmt.Errorf("cast to media playlist")
		}

		// Process new segments
		for i, v := range playlist.Segments {
			if v == nil {
				continue
			}
			seq := v.SeqId
			if seq == 0 {
				seq = playlist.SeqNo + uint64(i)
			}
			if hasLastSeq && seq <= lastSeq {
				continue
			}

			segmentMap := playlist.Map
			if v.Map != nil {
				segmentMap = v.Map
			}
			if segmentMap != nil {
				mapSpec := fmt.Sprintf("%s:%d:%d", segmentMap.URI, segmentMap.Offset, segmentMap.Limit)
				if mapSpec != lastMapSpec {
					mapURL, err := resolveURL(p.RootURL, segmentMap.URI)
					if err != nil {
						return fmt.Errorf("resolve init segment url: %w", err)
					}
					initData, err := client.GetBytes(ctx, mapURL)
					if err != nil {
						return fmt.Errorf("get init segment: %w", err)
					}
					if err := handler(initData, 0); err != nil {
						return fmt.Errorf("handle init segment: %w", err)
					}
					lastMapSpec = mapSpec
				}
			}

			lastSeq = seq
			hasLastSeq = true

			// Fetch segment data with retry mechanism
			pipeline := func() ([]byte, error) {
				segmentURL, err := resolveURL(p.RootURL, v.URI)
				if err != nil {
					return nil, fmt.Errorf("resolve segment url: %w", err)
				}
				return client.GetBytes(ctx, segmentURL)
			}

			resp, err := retry.DoWithData(
				pipeline,
				retry.Context(ctx),
				retry.Attempts(3),
				retry.Delay(600*time.Millisecond),
				retry.DelayType(retry.FixedDelay),
			)
			if err != nil {
				break
			}

			// Process the segment using the provided handler
			if err := handler(resp, v.Duration); err != nil {
				return fmt.Errorf("handler: %w", err)
			}
		}

		<-time.After(1 * time.Second) // time.Duration(playlist.TargetDuration)
	}
}
