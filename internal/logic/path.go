package logic

// PathFinder reproduces such_weg: find a path between two tiles in an 18x8 board
// using an expanded 20x10 buffer with a one-tile border, allowing at most 2 bends.
// Returns the path as a sequence of directions: 1=right, 2=left, 3=down, 4=up.

// FindPath returns true if a valid path exists and writes the path into dst (len up to 18*8).
func FindPath(board [18 * 8]byte, x1, y1, x2, y2 int, dst *[]byte) bool {
    // Build padded buffer: 20x10, -1=blocked, 0=free.
    const W, H = 18, 8
    var pad [20 * 10]byte
    for i := range pad {
        pad[i] = 0xFF
    }
    // Copy board into pad at offset +1,+1 (index = (y+1)*20 + (x+1)). Free cells are 0; occupied cells are 0xFF.
    for y := 0; y < H; y++ {
        for x := 0; x < W; x++ {
            v := board[y*W+x]
            if v == 0 {
                pad[(y+1)*20+(x+1)] = 0
            } else {
                pad[(y+1)*20+(x+1)] = 0xFF
            }
        }
    }
    // Treat the two tiles as free for path search.
    sx, sy := x1+1, y1+1
    tx, ty := x2+1, y2+1
    pad[sy*20+sx] = 0
    pad[ty*20+tx] = 0

    // Iterative search mirroring the original: explore straight lines in four directions,
    // tracking the number of bends so far and recording paths; keep shortest.
    type node struct{ x, y, dir, bends int; path []byte }
    best := []byte(nil)
    // Seed from start with no previous direction (0) and 0 bends.
    stack := []node{{x: sx, y: sy, dir: 0, bends: 0, path: make([]byte, 0, 32)}}
    visited := make(map[[3]int]int) // key: x,y,dir -> bends

    push := func(nd node) {
        k := [3]int{nd.x, nd.y, nd.dir}
        if b, ok := visited[k]; ok && b <= nd.bends {
            return
        }
        visited[k] = nd.bends
        stack = append(stack, nd)
    }

    for len(stack) > 0 {
        nd := stack[len(stack)-1]
        stack = stack[:len(stack)-1]

        // If reached target, record best (shortest path).
        if nd.x == tx && nd.y == ty {
            if best == nil || len(nd.path) < len(best) {
                cp := make([]byte, len(nd.path))
                copy(cp, nd.path)
                best = cp
            }
            continue
        }
        if nd.bends >= 2 {
            continue
        }

        // Explore 4 directions per original encoding: 1=right,2=left,3=down,4=up.
        dirs := [][3]int{
            {1, 0, 1},  // right
            {-1, 0, 2}, // left
            {0, 1, 3},  // down
            {0, -1, 4}, // up
        }
        for _, d := range dirs {
            dx, dy, code := d[0], d[1], d[2]
            bends := nd.bends
            if nd.dir != 0 && nd.dir != code {
                bends++
            }
            if bends > 2 {
                continue
            }
            // Walk straight until blocked; push each step to stack (like recursive advance + backtrack).
            x, y := nd.x, nd.y
            p := nd.path
            for {
                x += dx
                y += dy
                if x < 0 || y < 0 || x >= 20 || y >= 10 {
                    break
                }
                if pad[y*20+x] != 0 && !(x == tx && y == ty) {
                    break
                }
                // extend path by one step
                p = append(p, byte(code))
                push(node{x: x, y: y, dir: code, bends: bends, path: append([]byte(nil), p...)})
                // stop at target; deeper steps will be handled by pop order.
                if x == tx && y == ty {
                    break
                }
            }
        }
    }

    if best == nil {
        return false
    }
    *dst = append((*dst)[:0], best...)
    return true
}
