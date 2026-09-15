package logic

const (
	boardWidth       = 18
	boardHeight      = 8
	paddedWidth      = boardWidth + 2
	paddedHeight     = boardHeight + 2
	paddedCells      = paddedWidth * paddedHeight
	pathDirections   = 5 // none, right, left, down, up
	pathBendStates   = 3 // zero, one, or two bends
	pathSearchStates = paddedCells * pathDirections * pathBendStates
	maxPathLength    = paddedCells
)

type pathState struct {
	position  uint16
	previous  int16
	direction byte
	bends     byte
}

var pathSteps = [...]struct {
	dx, dy    int
	direction byte
}{
	{dx: 1, direction: 1},
	{dx: -1, direction: 2},
	{dy: 1, direction: 3},
	{dy: -1, direction: 4},
}

// FindPath finds the shortest path between two tiles through empty cells,
// including the one-cell border around the board, with at most two bends.
// Directions are encoded as 1=right, 2=left, 3=down, and 4=up.
func FindPath(board [boardWidth * boardHeight]byte, x1, y1, x2, y2 int, dst *[]byte) bool {
	var path [maxPathLength]byte
	length, ok := findPath(board, x1, y1, x2, y2, path[:])
	if !ok {
		*dst = (*dst)[:0]
		return false
	}
	*dst = append((*dst)[:0], path[:length]...)
	return true
}

// HasPath is the allocation-free reachability variant used when a caller does
// not need the path itself.
func HasPath(board [boardWidth * boardHeight]byte, x1, y1, x2, y2 int) bool {
	_, ok := findPath(board, x1, y1, x2, y2, nil)
	return ok
}

func findPath(board [boardWidth * boardHeight]byte, x1, y1, x2, y2 int, path []byte) (int, bool) {
	if x1 < 0 || x1 >= boardWidth || y1 < 0 || y1 >= boardHeight ||
		x2 < 0 || x2 >= boardWidth || y2 < 0 || y2 >= boardHeight {
		return 0, false
	}

	start := (y1+1)*paddedWidth + x1 + 1
	target := (y2+1)*paddedWidth + x2 + 1
	if start == target {
		return 0, true
	}

	// The outer border stays free. Occupied board cells are blocked except for
	// the two endpoints.
	var blocked [paddedCells]bool
	for y := 0; y < boardHeight; y++ {
		for x := 0; x < boardWidth; x++ {
			position := (y+1)*paddedWidth + x + 1
			blocked[position] = board[y*boardWidth+x] != 0
		}
	}
	blocked[start] = false
	blocked[target] = false

	var queue [pathSearchStates]pathState
	var visited [pathSearchStates]bool
	queue[0] = pathState{position: uint16(start), previous: -1}
	visited[pathVisitedIndex(start, 0, 0)] = true
	head, tail := 0, 1
	found := -1

	for head < tail {
		currentIndex := head
		current := queue[head]
		head++
		position := int(current.position)
		x, y := position%paddedWidth, position/paddedWidth

		for _, step := range pathSteps {
			nextX, nextY := x+step.dx, y+step.dy
			if nextX < 0 || nextX >= paddedWidth || nextY < 0 || nextY >= paddedHeight {
				continue
			}
			nextPosition := nextY*paddedWidth + nextX
			if blocked[nextPosition] {
				continue
			}

			bends := current.bends
			if current.direction != 0 && current.direction != step.direction {
				bends++
			}
			if bends > 2 {
				continue
			}
			visitedIndex := pathVisitedIndex(nextPosition, step.direction, bends)
			if visited[visitedIndex] {
				continue
			}
			visited[visitedIndex] = true
			queue[tail] = pathState{
				position:  uint16(nextPosition),
				previous:  int16(currentIndex),
				direction: step.direction,
				bends:     bends,
			}
			if nextPosition == target {
				found = tail
				break
			}
			tail++
		}
		if found >= 0 {
			break
		}
	}

	if found < 0 {
		return 0, false
	}
	if path == nil {
		return 0, true
	}

	length := 0
	for index := found; queue[index].previous >= 0; index = int(queue[index].previous) {
		path[length] = queue[index].direction
		length++
	}
	for left, right := 0, length-1; left < right; left, right = left+1, right-1 {
		path[left], path[right] = path[right], path[left]
	}
	return length, true
}

func pathVisitedIndex(position int, direction, bends byte) int {
	return (position*pathDirections+int(direction))*pathBendStates + int(bends)
}
