package logic

import (
    "time"
)

// parkMiller implements the Atari ST RNG observed in match_it.s (Park–Miller minimal standard)
// with Schrage method (127773, 2836) and modulus 2^31-1 (2147483647). The original routine
// seeds using the VBL counter then folds with the previous value; here we seed with time.
//
// This RNG is deterministic per instance and matches the arithmetic of the original code.
type parkMiller struct {
    seed int32 // always kept in [1, 2147483646]
}

func newRNG() *parkMiller {
    // Seed with current time similar to VBL noise; never allow 0.
    s := int32(time.Now().UnixNano() & 0x7fffffff)
    if s == 0 {
        s = 1
    }
    return &parkMiller{seed: s}
}

// next returns the next 31-bit positive integer in [1, 2147483646].
func (r *parkMiller) next() int32 {
    // Equivalent to: s = 16807 * (s % 127773) - 2836 * (s / 127773)
    // If s <= 0 then s += 2147483647
    const (
        a   = int32(16807)
        m   = int32(2147483647)
        q   = int32(127773)
        rem = int32(2836)
    )
    hi := r.seed / q
    lo := r.seed % q
    s := a*lo - rem*hi
    if s <= 0 {
        s += m
    }
    r.seed = s
    return s
}

// NextUint32 returns a non-negative 32-bit value.
func (r *parkMiller) NextUint32() uint32 { return uint32(r.next()) }

// Intn mirrors and.w #63 in assembly: returns x in [0, n).
func (r *parkMiller) Intn(n int) int {
    if n <= 0 {
        return 0
    }
    return int(r.next() % int32(n))
}

