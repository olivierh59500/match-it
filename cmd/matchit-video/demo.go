package main

import (
	"fmt"
	"image"
	"math"

	resources "github.com/olivierh59500/match-it/assets"
	"github.com/olivierh59500/match-it/internal/logic"
)

const (
	splashFrames        = 90
	menuIdleFrames      = 20
	menuMoveFrames      = 45
	menuClickFrames     = 18
	gameIntroFrames     = 8
	moveToFirstFrames   = 6
	firstSelectedFrames = 7
	moveToSecondFrames  = 6
	removeFrames        = 15
	settleFrames        = 2
	summaryFrames       = 180
)

type tilePoint struct {
	x, y int
}

type demoMove struct {
	x1, y1 int
	x2, y2 int
	tile1  byte
	tile2  byte
	path   []byte
}

type demoPlan struct {
	level int
	board logic.Board
	moves []demoMove
}

func buildDemoPlan() (*demoPlan, error) {
	f, err := resources.Files.Open("png/gamearea.img.png")
	if err != nil {
		return nil, err
	}
	levels, decodeErr := logic.DecodeLevels(f)
	closeErr := f.Close()
	if decodeErr != nil {
		return nil, decodeErr
	}
	if closeErr != nil {
		return nil, closeErr
	}

	bestInterest := -1
	var best *demoPlan
	for level := range levels.Boards {
		var initial logic.Board
		initial.FromLevel(levels.Boards[level], levels.PosList[level])
		board := initial
		moves := make([]demoMove, 0, 48)
		interest := 0
		remaining := countTiles(board)
		for remaining > 0 {
			x1, y1, x2, y2, path, ok := board.HelpSearch()
			if !ok {
				break
			}
			moves = append(moves, demoMove{
				x1:    x1,
				y1:    y1,
				x2:    x2,
				y2:    y2,
				tile1: board.Get(x1, y1),
				tile2: board.Get(x2, y2),
				path:  append([]byte(nil), path...),
			})
			interest += len(path)
			board.RemovePair(x1, y1, x2, y2)
			remaining -= 2
		}
		if remaining == 0 && len(moves) > 0 && interest > bestInterest {
			bestInterest = interest
			best = &demoPlan{level: level, board: initial, moves: moves}
		}
	}
	if best == nil {
		return nil, fmt.Errorf("none of the embedded levels can be completed")
	}
	return best, nil
}

func countTiles(board logic.Board) int {
	count := 0
	for _, tile := range board.Tiles {
		if tile != 0 {
			count++
		}
	}
	return count
}

func totalDemoFrames(moveCount int) int {
	perMove := moveToFirstFrames + firstSelectedFrames + moveToSecondFrames + removeFrames + settleFrames
	return splashFrames + menuIdleFrames + menuMoveFrames + menuClickFrames + gameIntroFrames + moveCount*perMove + summaryFrames
}

type frameSink interface {
	WriteFrame(*image.RGBA) error
}

func generateDemo(assets *demoAssets, plan *demoPlan, sink frameSink) error {
	frame := image.NewRGBA(image.Rect(0, 0, videoWidth, videoHeight))
	frameNumber := 0
	gameFrame := 0
	cursor := cursorState{x: 520, y: 335}

	emit := func() error {
		frameNumber++
		return sink.WriteFrame(frame)
	}
	emitMenu := func(cursor cursorState) error {
		assets.renderMenu(frame)
		drawCursor(frame, cursor)
		return emit()
	}
	emitGame := func(board *logic.Board, score int, first *tilePoint, move *demoMove, removeFrame int, cursor cursorState) error {
		timeLeft := 200 - gameFrame/70
		if timeLeft < 0 {
			timeLeft = 0
		}
		assets.renderGame(frame, gameRenderState{
			board:       board,
			score:       score,
			timeLeft:    timeLeft,
			helpCount:   2,
			first:       first,
			move:        move,
			removeFrame: removeFrame,
			cursor:      cursor,
			pulse:       gameFrame,
		})
		gameFrame++
		return emit()
	}

	for range splashFrames {
		assets.renderSplash(frame)
		if err := emit(); err != nil {
			return err
		}
	}
	for range menuIdleFrames {
		if err := emitMenu(cursor); err != nil {
			return err
		}
	}
	menuTarget := cursorState{x: 320, y: 90}
	for i := 0; i < menuMoveFrames; i++ {
		cursor = interpolateCursor(cursorState{x: 520, y: 335}, menuTarget, i, menuMoveFrames)
		if err := emitMenu(cursor); err != nil {
			return err
		}
	}
	for i := 0; i < menuClickFrames; i++ {
		cursor = menuTarget
		cursor.click = true
		cursor.progress = normalized(i, menuClickFrames)
		if err := emitMenu(cursor); err != nil {
			return err
		}
	}
	cursor = menuTarget

	board := plan.board
	score := 0
	for range gameIntroFrames {
		if err := emitGame(&board, score, nil, nil, -1, cursor); err != nil {
			return err
		}
	}

	for moveIndex := range plan.moves {
		move := &plan.moves[moveIndex]
		firstCursor := cursorState{x: float64(boardX + move.x1*tileWidth + tileWidth/2), y: float64(boardY + move.y1*tileHeight + tileHeight/2)}
		secondCursor := cursorState{x: float64(boardX + move.x2*tileWidth + tileWidth/2), y: float64(boardY + move.y2*tileHeight + tileHeight/2)}
		startCursor := cursor
		for i := 0; i < moveToFirstFrames; i++ {
			cursor = interpolateCursor(startCursor, firstCursor, i, moveToFirstFrames)
			if err := emitGame(&board, score, nil, nil, -1, cursor); err != nil {
				return err
			}
		}
		first := &tilePoint{x: move.x1, y: move.y1}
		for i := 0; i < firstSelectedFrames; i++ {
			cursor = firstCursor
			cursor.click = true
			cursor.progress = normalized(i, firstSelectedFrames)
			if err := emitGame(&board, score, first, nil, -1, cursor); err != nil {
				return err
			}
		}
		for i := 0; i < moveToSecondFrames; i++ {
			cursor = interpolateCursor(firstCursor, secondCursor, i, moveToSecondFrames)
			if err := emitGame(&board, score, first, nil, -1, cursor); err != nil {
				return err
			}
		}
		for removalFrame := 0; removalFrame < removeFrames; removalFrame++ {
			cursor = secondCursor
			if removalFrame < 8 {
				cursor.click = true
				cursor.progress = normalized(removalFrame, 8)
			}
			if err := emitGame(&board, score, nil, move, removalFrame, cursor); err != nil {
				return err
			}
		}
		board.RemovePair(move.x1, move.y1, move.x2, move.y2)
		score++
		cursor = secondCursor
		for range settleFrames {
			if err := emitGame(&board, score, nil, nil, -1, cursor); err != nil {
				return err
			}
		}
		if (moveIndex+1)%8 == 0 || moveIndex+1 == len(plan.moves) {
			fmt.Printf("pairs: %d/%d\n", moveIndex+1, len(plan.moves))
		}
	}

	timeBonus := 200 - gameFrame/70
	if timeBonus < 0 {
		timeBonus = 0
	}
	for range summaryFrames {
		assets.renderSummary(frame, 1, score, timeBonus, 200)
		if err := emit(); err != nil {
			return err
		}
	}

	expected := totalDemoFrames(len(plan.moves))
	if frameNumber != expected {
		return fmt.Errorf("generated %d frames, expected %d", frameNumber, expected)
	}
	return nil
}

func interpolateCursor(from, to cursorState, frame, frames int) cursorState {
	progress := normalized(frame, frames)
	progress = progress * progress * (3 - 2*progress)
	return cursorState{
		x: from.x + (to.x-from.x)*progress,
		y: from.y + (to.y-from.y)*progress,
	}
}

func normalized(frame, frames int) float64 {
	if frames <= 1 {
		return 1
	}
	return math.Min(1, float64(frame)/float64(frames-1))
}
