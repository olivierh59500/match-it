package logic

import "testing"

func TestFindPath(t *testing.T) {
	tests := []struct {
		name      string
		board     [boardWidth * boardHeight]byte
		x1, y1    int
		x2, y2    int
		want      bool
		wantPath  []byte
		wantBends int
	}{
		{
			name:  "straight",
			board: boardWithTiles(map[[2]int]byte{{1, 1}: 1, {4, 1}: 1}),
			x1:    1, y1: 1, x2: 4, y2: 1, want: true,
			wantPath: []byte{1, 1, 1}, wantBends: 0,
		},
		{
			name: "outside border",
			board: boardWithTiles(map[[2]int]byte{
				{0, 0}: 1, {1, 0}: 2, {2, 0}: 1,
				{0, 1}: 2, {1, 1}: 2, {2, 1}: 2,
			}),
			x1: 0, y1: 0, x2: 2, y2: 0, want: true,
			wantPath: []byte{4, 1, 1, 3}, wantBends: 2,
		},
		{
			name: "enclosed",
			board: boardWithTiles(map[[2]int]byte{
				{3, 3}: 1, {5, 3}: 1,
				{2, 2}: 2, {3, 2}: 2, {4, 2}: 2, {5, 2}: 2, {6, 2}: 2,
				{2, 3}: 2, {4, 3}: 2, {6, 3}: 2,
				{2, 4}: 2, {3, 4}: 2, {4, 4}: 2, {5, 4}: 2, {6, 4}: 2,
			}),
			x1: 3, y1: 3, x2: 5, y2: 3, want: false,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var path []byte
			got := FindPath(test.board, test.x1, test.y1, test.x2, test.y2, &path)
			if got != test.want {
				t.Fatalf("FindPath returned %t, want %t (path=%v)", got, test.want, path)
			}
			if !got {
				return
			}
			if test.wantPath != nil && !equalDirections(path, test.wantPath) {
				t.Fatalf("path=%v, want %v", path, test.wantPath)
			}
			if bends(path) != test.wantBends {
				t.Fatalf("path %v has %d bends, want %d", path, bends(path), test.wantBends)
			}
			validatePath(t, test.board, test.x1, test.y1, test.x2, test.y2, path)
			if !HasPath(test.board, test.x1, test.y1, test.x2, test.y2) {
				t.Fatal("HasPath disagrees with FindPath")
			}
		})
	}
}

func TestFindPathDoesNotAllocateAfterDestinationReuse(t *testing.T) {
	board := boardWithTiles(map[[2]int]byte{{1, 1}: 1, {4, 1}: 1})
	path := make([]byte, 0, maxPathLength)
	if !FindPath(board, 1, 1, 4, 1, &path) {
		t.Fatal("expected path")
	}
	if allocs := testing.AllocsPerRun(100, func() {
		if !FindPath(board, 1, 1, 4, 1, &path) {
			panic("path disappeared")
		}
	}); allocs != 0 {
		t.Fatalf("FindPath allocated %.2f times per call, want 0", allocs)
	}
}

func boardWithTiles(tiles map[[2]int]byte) (board [boardWidth * boardHeight]byte) {
	for position, tile := range tiles {
		board[position[1]*boardWidth+position[0]] = tile
	}
	return board
}

func equalDirections(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func bends(path []byte) int {
	count := 0
	for i := 1; i < len(path); i++ {
		if path[i] != path[i-1] {
			count++
		}
	}
	return count
}

func validatePath(t *testing.T, board [boardWidth * boardHeight]byte, x, y, targetX, targetY int, path []byte) {
	t.Helper()
	for _, direction := range path {
		switch direction {
		case 1:
			x++
		case 2:
			x--
		case 3:
			y++
		case 4:
			y--
		default:
			t.Fatalf("invalid direction %d", direction)
		}
		if x < -1 || x > boardWidth || y < -1 || y > boardHeight {
			t.Fatalf("path leaves padded board at (%d,%d)", x, y)
		}
		if x >= 0 && x < boardWidth && y >= 0 && y < boardHeight &&
			(x != targetX || y != targetY) && board[y*boardWidth+x] != 0 {
			t.Fatalf("path crosses occupied tile at (%d,%d)", x, y)
		}
	}
	if x != targetX || y != targetY {
		t.Fatalf("path ends at (%d,%d), want (%d,%d)", x, y, targetX, targetY)
	}
}
