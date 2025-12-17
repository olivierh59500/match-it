package audiox

import (
    "fmt"
    "io"
    "sync"

    "github.com/olivierh59500/ym-player/pkg/stsound"
)

// YMPlayer streams YM sound data via github.com/olivierh59500/ym-player.
type YMPlayer struct {
    player       *stsound.StSound
    sampleRate   int
    buffer       []int16
    mu           sync.Mutex
    position     int64
    totalSamples int64
    loop         bool
    volume       float64
}

func NewYMPlayer(data []byte, sampleRate int, loop bool) (*YMPlayer, error) {
    p := stsound.CreateWithRate(sampleRate)
    if err := p.LoadMemory(data); err != nil {
        p.Destroy()
        return nil, fmt.Errorf("load YM: %w", err)
    }
    p.SetLoopMode(loop)
    info := p.GetInfo()
    total := int64(info.MusicTimeInMs) * int64(sampleRate) / 1000
    return &YMPlayer{
        player:       p,
        sampleRate:   sampleRate,
        buffer:       make([]int16, 4096),
        totalSamples: total,
        loop:         loop,
        volume:       0.7,
    }, nil
}

func (y *YMPlayer) Read(p []byte) (n int, err error) {
    y.mu.Lock(); defer y.mu.Unlock()
    samplesNeeded := len(p) / 4 // stereo 16-bit
    out := make([]int16, samplesNeeded*2)
    processed := 0
    for processed < samplesNeeded {
        chunk := samplesNeeded - processed
        if chunk > len(y.buffer) { chunk = len(y.buffer) }
        if !y.player.Compute(y.buffer[:chunk], chunk) {
            if !y.loop {
                // fill remaining with zeros
                for i := processed * 2; i < len(out); i++ { out[i] = 0 }
                err = io.EOF
                break
            }
        }
        for i := 0; i < chunk; i++ {
            s := int16(float64(y.buffer[i]) * y.volume)
            out[(processed+i)*2] = s
            out[(processed+i)*2+1] = s
        }
        processed += chunk
        y.position += int64(chunk)
    }
    // interleave into byte slice
    bi := 0
    for i := 0; i < processed*2 && bi+1 < len(p); i++ {
        s := out[i]
        p[bi] = byte(s)
        p[bi+1] = byte(s >> 8)
        bi += 2
    }
    return bi, err
}

func (y *YMPlayer) Seek(offset int64, whence int) (int64, error) {
    y.mu.Lock(); defer y.mu.Unlock()
    var np int64
    switch whence {
    case io.SeekStart:
        np = offset
    case io.SeekCurrent:
        np = y.position + offset
    case io.SeekEnd:
        np = y.totalSamples + offset
    default:
        return 0, fmt.Errorf("invalid whence")
    }
    if np < 0 { np = 0 }
    if np > y.totalSamples { np = y.totalSamples }
    y.position = np
    return np, nil
}

func (y *YMPlayer) Close() error {
    y.mu.Lock(); defer y.mu.Unlock()
    if y.player != nil { y.player.Destroy(); y.player = nil }
    return nil
}

func (y *YMPlayer) GetVolume() float64 { y.mu.Lock(); defer y.mu.Unlock(); return y.volume }
func (y *YMPlayer) SetVolume(v float64) { y.mu.Lock(); y.volume = v; y.mu.Unlock() }

