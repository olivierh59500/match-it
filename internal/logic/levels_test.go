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
	defer f.Close()

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
