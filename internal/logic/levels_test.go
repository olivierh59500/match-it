package logic_test

import (
	"testing"

	resources "github.com/olivierh59500/match-it/assets"
	"github.com/olivierh59500/match-it/internal/logic"
)

func TestDecodeEmbeddedLevels(t *testing.T) {
	f, err := resources.Files.Open("png/gamearea.img.png")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})

	levels, err := logic.DecodeLevels(f)
	if err != nil {
		t.Fatal(err)
	}
	board, _ := levels.NextBoard()
	nonEmpty := 0
	for _, tile := range board {
		if tile > 43 {
			t.Fatalf("invalid tile ID %d", tile)
		}
		if tile != 0 {
			nonEmpty++
		}
	}
	if nonEmpty == 0 {
		t.Fatal("decoded board is empty")
	}
}

func TestEveryEmbeddedLevelStartsWithAMove(t *testing.T) {
	f, err := resources.Files.Open("png/gamearea.img.png")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := f.Close(); err != nil {
			t.Errorf("Close: %v", err)
		}
	})
	levels, err := logic.DecodeLevels(f)
	if err != nil {
		t.Fatal(err)
	}

	for level := range levels.Boards {
		var board logic.Board
		board.FromLevel(levels.Boards[level], levels.PosList[level])
		if _, _, _, _, _, ok := board.HelpSearch(); !ok {
			t.Errorf("level %d starts without a removable pair", level)
		}
	}
}
