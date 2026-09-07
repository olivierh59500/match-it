package logic

// Tile IDs follow the original 1..43 scheme in matchitbuff.
// 35..38 are seasons, 39..42 are flowers, and 43 is an ordinary identical
// pair. 0 means empty (no tile).

// MatchInfo returns whether two tiles can be removed and the special delay ticks (4)
// to pause the timer when removing seasons/flowers or those equal tiles.
func MatchInfo(a, b byte) (ok bool, delayTicks int) {
    if a == 0 || b == 0 {
        return false, 0
    }
    if a == b {
        if a >= 35 && a <= 42 {
            return true, 4
        }
        return true, 0
    }
    // Seasons 35..38 can match across that group.
    if a >= 35 && a <= 38 && b >= 35 && b <= 38 {
        return true, 4
    }
    // Flowers 39..42 can match across that group.
    if a >= 39 && a <= 42 && b >= 39 && b <= 42 {
        return true, 4
    }
    return false, 0
}

// MatchOK is a convenience wrapper.
func MatchOK(a, b byte) bool { ok, _ := MatchInfo(a, b); return ok }
