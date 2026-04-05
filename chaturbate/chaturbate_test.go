package chaturbate

import (
	"errors"
	"testing"

	"github.com/zihaofu245/chaturbate-dvr/internal"
)

func TestParseStreamPublicRoom(t *testing.T) {
	body := `<script>window.initialRoomDossier = "{\u0022room_status\u0022:\u0022public\u0022,\u0022hls_source\u0022:\u0022https://edge.example/playlist.m3u8\u0022}";</script>`

	stream, err := ParseStream(body)
	if err != nil {
		t.Fatalf("ParseStream() error = %v", err)
	}
	if stream.HLSSource != "https://edge.example/playlist.m3u8" {
		t.Fatalf("ParseStream() HLS source = %q", stream.HLSSource)
	}
}

func TestParseStreamOfflineRoom(t *testing.T) {
	body := `<script>window.initialRoomDossier = "{\u0022room_status\u0022:\u0022offline\u0022,\u0022hls_source\u0022:\u0022\u0022}";</script>`

	_, err := ParseStream(body)
	if !errors.Is(err, internal.ErrChannelOffline) {
		t.Fatalf("ParseStream() error = %v, want ErrChannelOffline", err)
	}
}

func TestParseStreamPrivateRoom(t *testing.T) {
	body := `<script>window.initialRoomDossier = "{\u0022room_status\u0022:\u0022private\u0022,\u0022hls_source\u0022:\u0022\u0022}";</script>`

	_, err := ParseStream(body)
	if !errors.Is(err, internal.ErrPrivateStream) {
		t.Fatalf("ParseStream() error = %v, want ErrPrivateStream", err)
	}
}
