package main

import (
	"flag"
	"fmt"
	"image"
	"io"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	resources "github.com/olivierh59500/match-it/assets"
	audiox "github.com/olivierh59500/match-it/internal/audio"
)

const (
	videoFPS            = 60
	audioSampleRate     = 48000
	audioChannels       = 2
	audioBytesPerSample = 2
)

type options struct {
	output       string
	ffmpeg       string
	crf          int
	audioBitrate string
	audioGain    float64
}

func main() {
	var opts options
	flag.StringVar(&opts.output, "output", "media/matchit-desktop-presentation.webm", "output WebM path")
	flag.StringVar(&opts.ffmpeg, "ffmpeg", "ffmpeg", "ffmpeg executable")
	flag.IntVar(&opts.crf, "crf", 34, "VP9 constant quality (lower is higher quality)")
	flag.StringVar(&opts.audioBitrate, "audio-bitrate", "64k", "Opus bitrate")
	flag.Float64Var(&opts.audioGain, "audio-gain", 8, "soundtrack gain in dB")
	flag.Parse()

	if err := run(opts); err != nil {
		log.Fatal(err)
	}
}

func run(opts options) error {
	if opts.crf < 0 || opts.crf > 63 {
		return fmt.Errorf("CRF must be between 0 and 63")
	}
	if opts.audioGain < -30 || opts.audioGain > 20 {
		return fmt.Errorf("audio gain must be between -30 and 20 dB")
	}
	assets, err := loadDemoAssets()
	if err != nil {
		return err
	}
	plan, err := buildDemoPlan()
	if err != nil {
		return err
	}
	frames := totalDemoFrames(len(plan.moves))
	duration := time.Duration(float64(frames) / videoFPS * float64(time.Second))
	fmt.Printf("level=%d pairs=%d frames=%d duration=%s\n", plan.level, len(plan.moves), frames, duration.Round(time.Millisecond))

	audioPath, err := generateAudio(frames, splashFrames)
	if err != nil {
		return err
	}
	defer func() {
		if err := os.Remove(audioPath); err != nil && !os.IsNotExist(err) {
			log.Printf("remove temporary audio: %v", err)
		}
	}()

	if err := os.MkdirAll(filepath.Dir(opts.output), 0o755); err != nil {
		return err
	}
	encoder, err := newFFmpegSink(opts, audioPath)
	if err != nil {
		return err
	}
	start := time.Now()
	if err := generateDemo(assets, plan, encoder); err != nil {
		encoder.Abort()
		return err
	}
	if err := encoder.Close(); err != nil {
		return err
	}
	info, err := os.Stat(opts.output)
	if err != nil {
		return err
	}
	fmt.Printf("wrote %s (%s) in %s\n", opts.output, humanBytes(info.Size()), time.Since(start).Round(time.Millisecond))
	return nil
}

func generateAudio(totalFrames, silenceFrames int) (string, error) {
	f, err := os.CreateTemp("", "matchit-video-*.pcm")
	if err != nil {
		return "", err
	}
	path := f.Name()
	fail := func(err error) (string, error) {
		_ = f.Close()
		_ = os.Remove(path)
		return "", err
	}

	bytesPerFrame := audioSampleRate / videoFPS * audioChannels * audioBytesPerSample
	if err := writeSilence(f, int64(silenceFrames*bytesPerFrame)); err != nil {
		return fail(err)
	}
	music, err := resources.Files.ReadFile("music/Chambers of Shaolin - Trapped in China.ym")
	if err != nil {
		return fail(err)
	}
	player, err := audiox.NewYMPlayer(music, audioSampleRate, true)
	if err != nil {
		return fail(err)
	}
	remaining := int64((totalFrames - silenceFrames) * bytesPerFrame)
	buffer := make([]byte, 64*1024)
	for remaining > 0 {
		chunk := int64(len(buffer))
		if chunk > remaining {
			chunk = remaining
		}
		n, readErr := player.Read(buffer[:chunk])
		if n > 0 {
			if err := writeAll(f, buffer[:n]); err != nil {
				_ = player.Close()
				return fail(err)
			}
			remaining -= int64(n)
		}
		if readErr != nil {
			_ = player.Close()
			return fail(readErr)
		}
		if n == 0 {
			_ = player.Close()
			return fail(io.ErrNoProgress)
		}
	}
	if err := player.Close(); err != nil {
		return fail(err)
	}
	if err := f.Close(); err != nil {
		_ = os.Remove(path)
		return "", err
	}
	return path, nil
}

func writeSilence(writer io.Writer, bytes int64) error {
	zeroes := make([]byte, 64*1024)
	for bytes > 0 {
		chunk := int64(len(zeroes))
		if chunk > bytes {
			chunk = bytes
		}
		if err := writeAll(writer, zeroes[:chunk]); err != nil {
			return err
		}
		bytes -= chunk
	}
	return nil
}

type ffmpegSink struct {
	command *exec.Cmd
	stdin   io.WriteCloser
}

func newFFmpegSink(opts options, audioPath string) (*ffmpegSink, error) {
	arguments := []string{
		"-hide_banner",
		"-loglevel", "warning",
		"-y",
		"-f", "rawvideo",
		"-pixel_format", "rgba",
		"-video_size", fmt.Sprintf("%dx%d", videoWidth, videoHeight),
		"-framerate", fmt.Sprint(videoFPS),
		"-i", "pipe:0",
		"-f", "s16le",
		"-ar", fmt.Sprint(audioSampleRate),
		"-ac", fmt.Sprint(audioChannels),
		"-i", audioPath,
		"-map", "0:v:0",
		"-map", "1:a:0",
		"-af", fmt.Sprintf("volume=%.2fdB", opts.audioGain),
		"-c:v", "libvpx-vp9",
		"-crf", fmt.Sprint(opts.crf),
		"-b:v", "0",
		"-deadline", "good",
		"-cpu-used", "3",
		"-row-mt", "1",
		"-g", "120",
		"-pix_fmt", "yuv420p",
		"-c:a", "libopus",
		"-b:a", opts.audioBitrate,
		"-vbr", "on",
		"-application", "audio",
		"-shortest",
		"-metadata", "title=Match'it - Desktop Gameplay",
		opts.output,
	}
	command := exec.Command(opts.ffmpeg, arguments...)
	command.Stdout = os.Stdout
	command.Stderr = os.Stderr
	stdin, err := command.StdinPipe()
	if err != nil {
		return nil, err
	}
	if err := command.Start(); err != nil {
		_ = stdin.Close()
		return nil, err
	}
	return &ffmpegSink{command: command, stdin: stdin}, nil
}

func (s *ffmpegSink) WriteFrame(frame *image.RGBA) error {
	if frame.Rect != image.Rect(0, 0, videoWidth, videoHeight) || frame.Stride != videoWidth*4 {
		return fmt.Errorf("unexpected frame layout: rect=%v stride=%d", frame.Rect, frame.Stride)
	}
	return writeAll(s.stdin, frame.Pix)
}

func (s *ffmpegSink) Close() error {
	if err := s.stdin.Close(); err != nil {
		_ = s.command.Process.Kill()
		_ = s.command.Wait()
		return err
	}
	return s.command.Wait()
}

func (s *ffmpegSink) Abort() {
	_ = s.stdin.Close()
	if s.command.Process != nil {
		_ = s.command.Process.Kill()
	}
	_ = s.command.Wait()
}

func writeAll(writer io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := writer.Write(data)
		if err != nil {
			return err
		}
		if n == 0 {
			return io.ErrShortWrite
		}
		data = data[n:]
	}
	return nil
}

func humanBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	divisor, exponent := int64(unit), 0
	for value := bytes / unit; value >= unit; value /= unit {
		divisor *= unit
		exponent++
	}
	return fmt.Sprintf("%.1f %ciB", float64(bytes)/float64(divisor), "KMGTPE"[exponent])
}
