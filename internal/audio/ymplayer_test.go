package audiox

import (
	"testing"

	resources "github.com/olivierh59500/match-it/assets"
)

func TestYMPlayerReadProducesStereoWithoutAllocating(t *testing.T) {
	data, err := resources.Files.ReadFile("music/Chambers of Shaolin - Trapped in China.ym")
	if err != nil {
		t.Fatal(err)
	}

	player, err := NewYMPlayer(data, 48000, true)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := player.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	pcm := make([]byte, 4096*4)
	n, err := player.Read(pcm)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	if n != len(pcm) {
		t.Fatalf("Read returned %d bytes, want %d", n, len(pcm))
	}

	nonSilent := false
	for i := 0; i < n; i += 4 {
		left := uint16(pcm[i]) | uint16(pcm[i+1])<<8
		right := uint16(pcm[i+2]) | uint16(pcm[i+3])<<8
		if left != right {
			t.Fatalf("frame %d differs between channels: left=%d right=%d", i/4, left, right)
		}
		if left != 0 {
			nonSilent = true
		}
	}
	if !nonSilent {
		t.Fatal("rendered PCM block is silent")
	}

	var readN int
	var readErr error
	allocs := testing.AllocsPerRun(100, func() {
		readN, readErr = player.Read(pcm)
	})
	if readErr != nil {
		t.Fatalf("allocation run Read: %v", readErr)
	}
	if readN != len(pcm) {
		t.Fatalf("allocation run Read returned %d bytes, want %d", readN, len(pcm))
	}
	if allocs != 0 {
		t.Fatalf("Read allocated %.2f times per call, want 0", allocs)
	}
}

func TestYMPlayerCloseIsIdempotent(t *testing.T) {
	data, err := resources.Files.ReadFile("music/Chambers of Shaolin - Trapped in China.ym")
	if err != nil {
		t.Fatal(err)
	}
	player, err := NewYMPlayer(data, 48000, true)
	if err != nil {
		t.Fatal(err)
	}
	if err := player.Close(); err != nil {
		t.Fatal(err)
	}
	if err := player.Close(); err != nil {
		t.Fatal(err)
	}
}
