package audiox

import (
	"fmt"
	"io"
	"sync"

	"github.com/olivierh59500/ym-player/pkg/stsound"
)

// YMPlayer streams mono YM synthesis as stereo 16-bit little-endian PCM.
type YMPlayer struct {
	player *stsound.StSound
	buffer []int16
	mu     sync.Mutex
	loop   bool
}

func NewYMPlayer(data []byte, sampleRate int, loop bool) (*YMPlayer, error) {
	player := stsound.CreateWithRate(sampleRate)
	if err := player.LoadMemory(data); err != nil {
		player.Destroy()
		return nil, fmt.Errorf("load YM: %w", err)
	}
	player.SetLoopMode(loop)
	return &YMPlayer{
		player: player,
		buffer: make([]int16, 4096),
		loop:   loop,
	}, nil
}

func (y *YMPlayer) Read(p []byte) (n int, err error) {
	y.mu.Lock()
	defer y.mu.Unlock()

	if y.player == nil {
		return 0, io.ErrClosedPipe
	}

	samplesNeeded := len(p) / 4
	processed := 0
	for processed < samplesNeeded {
		chunk := samplesNeeded - processed
		if chunk > len(y.buffer) {
			chunk = len(y.buffer)
		}

		if !y.player.Compute(y.buffer[:chunk], chunk) {
			if !y.loop {
				clear(p[processed*4 : samplesNeeded*4])
				err = io.EOF
				break
			}
		}

		for i := 0; i < chunk; i++ {
			sample := y.buffer[i] / 2
			offset := (processed + i) * 4
			p[offset] = byte(sample)
			p[offset+1] = byte(sample >> 8)
			p[offset+2] = byte(sample)
			p[offset+3] = byte(sample >> 8)
		}
		processed += chunk
	}

	return samplesNeeded * 4, err
}

func (y *YMPlayer) Close() error {
	y.mu.Lock()
	defer y.mu.Unlock()
	if y.player != nil {
		y.player.Destroy()
		y.player = nil
	}
	return nil
}
