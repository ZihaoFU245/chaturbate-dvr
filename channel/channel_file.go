package channel

import (
	"bytes"
	"errors"
	"fmt"
	"html/template"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

var (
	ffmpegBestVideoEncoder       string
	ffmpegDetectVideoEncoderOnce sync.Once
)

func detectBestFFmpegVideoEncoder() string {
	// Prefer software encoders to avoid needing special hwaccel flags.
	// Order is important: AV1 first, then H.265, then H.264.
	preferred := []string{
		"libsvtav1",
		"libaom-av1",
		"librav1e",
		"libx265",
		"libx264",
	}

	out, err := exec.Command("ffmpeg", "-hide_banner", "-encoders").CombinedOutput()
	if err != nil {
		return ""
	}
	encoders := string(out)

	for _, enc := range preferred {
		// ffmpeg prints one encoder per line; a simple substring match is enough.
		// Typical format: " V..... libx264 ..."
		if strings.Contains(encoders, " "+enc+" ") || strings.Contains(encoders, "\t"+enc+" ") {
			return enc
		}
	}

	return ""
}

func getBestFFmpegVideoEncoder() string {
	ffmpegDetectVideoEncoderOnce.Do(func() {
		ffmpegBestVideoEncoder = detectBestFFmpegVideoEncoder()
	})
	return ffmpegBestVideoEncoder
}

// Pattern holds the date/time and sequence information for the filename pattern
type Pattern struct {
	Username string
	Year     string
	Month    string
	Day      string
	Hour     string
	Minute   string
	Second   string
	Sequence int
}

// NextFile prepares the next file to be created, by cleaning up the last file and generating a new one
func (ch *Channel) NextFile() error {
	if err := ch.Cleanup(); err != nil {
		return err
	}
	filename, err := ch.GenerateFilename()
	if err != nil {
		return err
	}
	if err := ch.CreateNewFile(filename); err != nil {
		return err
	}

	// Increment the sequence number for the next file
	ch.Sequence++
	return nil
}

// Cleanup cleans the file and resets it, called when the stream errors out or before next file was created.
func (ch *Channel) Cleanup() error {
	if ch.File == nil {
		return nil
	}
	filename := ch.File.Name()

	defer func() {
		ch.Filesize = 0
		ch.Duration = 0
	}()

	// Sync the file to ensure data is written to disk
	if err := ch.File.Sync(); err != nil && !errors.Is(err, os.ErrClosed) {
		return fmt.Errorf("sync file: %w", err)
	}
	if err := ch.File.Close(); err != nil && !errors.Is(err, os.ErrClosed) {
		return fmt.Errorf("close file: %w", err)
	}
	ch.File = nil

	// Delete the empty file
	fileInfo, err := os.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat file delete zero file: %w", err)
	}
	if fileInfo != nil && fileInfo.Size() == 0 {
		if err := os.Remove(filename); err != nil {
			return fmt.Errorf("remove zero file: %w", err)
		}
	}

	if ch.Config.ConvertToMP4 && fileInfo != nil && fileInfo.Size() > 0 {
		go ch.convertToMP4(filename)
	}
	return nil
}

// GenerateFilename creates a filename based on the configured pattern and the current timestamp
func (ch *Channel) GenerateFilename() (string, error) {
	var buf bytes.Buffer

	// Parse the filename pattern defined in the channel's config
	tpl, err := template.New("filename").Parse(ch.Config.Pattern)
	if err != nil {
		return "", fmt.Errorf("filename pattern error: %w", err)
	}

	// Get the current time based on the Unix timestamp when the stream was started
	t := time.Unix(ch.StreamedAt, 0)
	pattern := &Pattern{
		Username: ch.Config.Username,
		Sequence: ch.Sequence,
		Year:     t.Format("2006"),
		Month:    t.Format("01"),
		Day:      t.Format("02"),
		Hour:     t.Format("15"),
		Minute:   t.Format("04"),
		Second:   t.Format("05"),
	}

	if err := tpl.Execute(&buf, pattern); err != nil {
		return "", fmt.Errorf("template execution error: %w", err)
	}
	return buf.String(), nil
}

// CreateNewFile creates a new file for the channel using the given filename
func (ch *Channel) CreateNewFile(filename string) error {

	// Ensure the directory exists before creating the file
	if err := os.MkdirAll(filepath.Dir(filename), 0777); err != nil {
		return fmt.Errorf("mkdir all: %w", err)
	}

	// Open the file in append mode, create it if it doesn't exist
	file, err := os.OpenFile(filename+".ts", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0777)
	if err != nil {
		return fmt.Errorf("cannot open file: %s: %w", filename, err)
	}

	ch.File = file
	return nil
}

// ShouldSwitchFile determines whether a new file should be created.
func (ch *Channel) ShouldSwitchFile() bool {
	maxFilesizeBytes := ch.Config.MaxFilesize * 1024 * 1024
	maxDurationSeconds := ch.Config.MaxDuration * 60

	return (ch.Duration >= float64(maxDurationSeconds) && ch.Config.MaxDuration > 0) ||
		(ch.Filesize >= maxFilesizeBytes && ch.Config.MaxFilesize > 0)
}

// convertToMP4 converts a recorded .ts file to .mp4 using ffmpeg.
// Runs in a goroutine to avoid blocking the recording loop.
func (ch *Channel) convertToMP4(tsPath string) {
	mp4Path := strings.TrimSuffix(tsPath, filepath.Ext(tsPath)) + ".mp4"

	ch.Info("converting %s to mp4", filepath.Base(tsPath))

	var stderr bytes.Buffer
	encoder := getBestFFmpegVideoEncoder()
	var cmd *exec.Cmd
	if encoder != "" {
		ch.Info("using ffmpeg video encoder: %s", encoder)
		cmd = exec.Command(
			"ffmpeg",
			"-y",
			"-i", tsPath,
			"-map", "0:v:0",
			"-map", "0:a?",
			"-c:v", encoder,
			"-pix_fmt", "yuv420p",
			"-c:a", "copy",
			"-movflags", "+faststart",
			mp4Path,
		)
	} else {
		// If ffmpeg is present but no preferred encoder exists, keep old behavior.
		ch.Info("no preferred ffmpeg encoder found; falling back to stream copy")
		cmd = exec.Command("ffmpeg", "-y", "-i", tsPath, "-c", "copy", mp4Path)
	}

	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		// If we selected an encoder and it fails (e.g., encoder missing in PATH build),
		// fall back according to AV1 -> H265 -> H264 order.
		trimmed := strings.TrimSpace(stderr.String())
		if encoder != "" {
			ch.Error("convert with %s failed: %s (%s)", encoder, err.Error(), trimmed)
			fallbackOrder := [][]string{{"libsvtav1", "libaom-av1", "librav1e"}, {"libx265"}, {"libx264"}}
			for _, group := range fallbackOrder {
				for _, enc := range group {
					if enc == encoder {
						continue
					}
					var attemptStderr bytes.Buffer
					attempt := exec.Command(
						"ffmpeg",
						"-y",
						"-i", tsPath,
						"-map", "0:v:0",
						"-map", "0:a?",
						"-c:v", enc,
						"-pix_fmt", "yuv420p",
						"-c:a", "copy",
						"-movflags", "+faststart",
						mp4Path,
					)
					attempt.Stderr = &attemptStderr
					runErr := attempt.Run()
					if runErr == nil {
						ch.Info("converted using fallback encoder: %s", enc)
						goto converted
					}
					ch.Error(
						"fallback convert with %s failed: %s (%s)",
						enc,
						runErr.Error(),
						strings.TrimSpace(attemptStderr.String()),
					)
				}
			}
		}
		ch.Error("convert to mp4 failed: %s (%s)", err.Error(), trimmed)
		return
	}

converted:

	if err := os.Remove(tsPath); err != nil {
		ch.Error("remove ts after conversion failed: %s", err.Error())
		return
	}

	ch.Info("converted to mp4: %s", filepath.Base(mp4Path))
}
